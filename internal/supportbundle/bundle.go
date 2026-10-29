// Package supportbundle writes a local, reviewed support bundle for incident
// handoff.
//
// It is a deliberately small proof of concept for TR-F022: the bundle is staged
// under an operator-chosen directory, every payload file is redacted with the
// rules in internal/redact, a manifest records a sha256 per file plus a bundle
// checksum, and an archive is only produced after a reviewer approves the
// staged copy. The package never opens a socket, never reads the database, and
// never writes outside the output directory, so "export" means "operator copies
// the archive somewhere", not "the tool sends it".
package supportbundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/redact"
	"gopkg.in/yaml.v3"
)

const (
	// Format identifies a bundle produced by this package.
	Format = "mwx-isp-support-bundle"
	// Version is the bundle layout version. It changes when a file name or the
	// manifest shape changes in a way older reviewers cannot read.
	Version = 1

	// DefaultLogTailBytes bounds how much of the log is staged. A support bundle
	// is for an incident window, not for an unbounded log archive.
	DefaultLogTailBytes = 2 << 20

	manifestFile = "manifest.json"
	reportFile   = "redaction-report.json"
	reviewFile   = "review.json"
	metaFile     = "meta.json"
	configFile   = "config.redacted.yaml"
	logFile      = "logs/app.log"
)

var (
	// ErrOutputDirRequired is returned when no output directory is given. The
	// tool never writes next to its own working directory implicitly.
	ErrOutputDirRequired = errors.New("support bundle output directory is required")
	// ErrNothingToStage is returned when neither a config file nor a log file was
	// supplied.
	ErrNothingToStage = errors.New("nothing to stage: provide a config file, a log file, or both")
	// ErrReviewerRequired is returned when an archive is requested without a
	// reviewer identity.
	ErrReviewerRequired = errors.New("approval requires a reviewer identity")
	// ErrChecksumMismatch is returned when a staged file changed after staging.
	ErrChecksumMismatch = errors.New("staged bundle does not match its manifest")
	// ErrAlreadyApproved is returned when a bundle directory already carries an
	// approval record.
	ErrAlreadyApproved = errors.New("support bundle was already approved")
)

// Options describes one staging run.
type Options struct {
	// OutputDir is the directory that receives the staged bundle. Required.
	OutputDir string
	// ConfigPath is the YAML configuration file to redact. Optional.
	ConfigPath string
	// LogPath is the application log file to redact. Optional.
	LogPath string
	// LogTailBytes limits the log read to the last N bytes. Zero means
	// DefaultLogTailBytes; negative values are rejected.
	LogTailBytes int64
	// AppVersion is the MWX-ISP version written to meta.json.
	AppVersion string
	// Now is the staging timestamp. Zero means time.Now().UTC().
	Now time.Time
}

// FileEntry is one payload file recorded in the manifest.
type FileEntry struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// ReviewState records whether an operator has approved the staged bundle.
type ReviewState struct {
	Status     string     `json:"status"`
	Reviewer   string     `json:"reviewer,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

// Manifest is manifest.json in a staged bundle.
type Manifest struct {
	Format     string      `json:"format"`
	Version    int         `json:"version"`
	CreatedAt  time.Time   `json:"created_at"`
	AppVersion string      `json:"app_version"`
	Files      []FileEntry `json:"files"`
	Checksum   string      `json:"checksum"`
	Review     ReviewState `json:"review"`
}

// ClassCounts counts redacted values per data class.
type ClassCounts struct {
	Secrets   int `json:"secrets"`
	PII       int `json:"pii"`
	Addresses int `json:"addresses"`
	FreeText  int `json:"free_text"`
}

// SourceReport describes what happened to one input file.
type SourceReport struct {
	Source      string      `json:"source"`
	File        string      `json:"file"`
	InputBytes  int64       `json:"input_bytes"`
	OutputBytes int64       `json:"output_bytes"`
	Counts      ClassCounts `json:"counts"`
	Keys        []string    `json:"redacted_keys,omitempty"`
	Notes       []string    `json:"notes,omitempty"`
}

// RedactionReport is redaction-report.json in a staged bundle.
type RedactionReport struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Rules       []string       `json:"rules"`
	Sources     []SourceReport `json:"sources"`
	Totals      ClassCounts    `json:"totals"`
}

// Staged is the result of Stage.
type Staged struct {
	// Dir is the staged bundle directory.
	Dir string
	// Manifest is the written manifest.
	Manifest Manifest
	// Report is the written redaction report.
	Report RedactionReport
}

// Bundle is the result of Approve.
type Bundle struct {
	// ArchivePath is the ".tar.gz" written for handoff.
	ArchivePath string
	// ChecksumPath is the sidecar holding the archive sha256.
	ChecksumPath string
	// SHA256 is the hex digest of the archive.
	SHA256 string
	// ReviewPath is the review record inside the staged directory.
	ReviewPath string
}

// Stage writes the redacted payload, manifest and redaction report into a new
// "support-bundle-<timestamp>" directory under Options.OutputDir. It does not
// create an archive; that requires Approve.
func Stage(opts Options) (*Staged, error) {
	outDir := strings.TrimSpace(opts.OutputDir)
	if outDir == "" {
		return nil, ErrOutputDirRequired
	}
	if opts.LogTailBytes < 0 {
		return nil, fmt.Errorf("log tail bytes must not be negative")
	}
	tail := opts.LogTailBytes
	if tail == 0 {
		tail = DefaultLogTailBytes
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	appVersion := strings.TrimSpace(opts.AppVersion)
	if appVersion == "" {
		appVersion = "unknown"
	}

	configPath := strings.TrimSpace(opts.ConfigPath)
	logPath := strings.TrimSpace(opts.LogPath)
	if configPath == "" && logPath == "" {
		return nil, ErrNothingToStage
	}

	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	report := RedactionReport{
		GeneratedAt: now,
		Rules: []string{
			"secret keys and credential assignments -> " + redact.Marker,
			"customer identifiers -> sha256 pseudonym",
			"ip addresses -> /24 (v4) or /48 (v6) prefix",
			"free text -> email and ipv4 sweep",
		},
	}
	type stagedFile struct {
		rel     string
		content []byte
	}
	files := make([]stagedFile, 0, 3)

	// Both inputs are redacted in memory before the staging directory exists, so
	// a parse or read failure cannot leave a half-written bundle behind.
	if configPath != "" {
		content, source, err := buildConfig(configPath)
		if err != nil {
			return nil, err
		}
		report.Sources = append(report.Sources, source)
		files = append(files, stagedFile{rel: configFile, content: content})
	}
	if logPath != "" {
		content, source, err := buildLog(logPath, tail)
		if err != nil {
			return nil, err
		}
		report.Sources = append(report.Sources, source)
		files = append(files, stagedFile{rel: logFile, content: content})
	}
	// Key paths are collected in map-iteration order; sort them so the report and
	// therefore the bundle checksum are reproducible for identical inputs.
	for i := range report.Sources {
		sort.Strings(report.Sources[i].Keys)
	}
	for _, source := range report.Sources {
		report.Totals.Secrets += source.Counts.Secrets
		report.Totals.PII += source.Counts.PII
		report.Totals.Addresses += source.Counts.Addresses
		report.Totals.FreeText += source.Counts.FreeText
	}
	encodedReport, err := encodeJSON(report)
	if err != nil {
		return nil, err
	}
	files = append(files, stagedFile{rel: reportFile, content: encodedReport})

	encodedMeta, err := encodeJSON(map[string]any{
		"app_version": appVersion,
		"created_at":  now,
		"go_version":  runtime.Version(),
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"format":      Format,
		"version":     Version,
	})
	if err != nil {
		return nil, err
	}
	files = append(files, stagedFile{rel: metaFile, content: encodedMeta})

	dir := filepath.Join(outDir, "support-bundle-"+now.Format("20060102T150405Z"))
	if err := os.Mkdir(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}

	manifest := Manifest{
		Format:     Format,
		Version:    Version,
		CreatedAt:  now,
		AppVersion: appVersion,
		Review:     ReviewState{Status: "pending_review"},
	}
	for _, file := range files {
		if err := writeFile(filepath.Join(dir, filepath.FromSlash(file.rel)), file.content); err != nil {
			return nil, err
		}
		sum := sha256.Sum256(file.content)
		manifest.Files = append(manifest.Files, FileEntry{
			Path:   file.rel,
			Bytes:  int64(len(file.content)),
			SHA256: hex.EncodeToString(sum[:]),
		})
	}
	manifest.Checksum = bundleChecksum(manifest.Files)
	encodedManifest, err := encodeJSON(manifest)
	if err != nil {
		return nil, err
	}
	if err := writeFile(filepath.Join(dir, manifestFile), encodedManifest); err != nil {
		return nil, err
	}
	return &Staged{Dir: dir, Manifest: manifest, Report: report}, nil
}

// Approve verifies the staged payload against its manifest, records the
// reviewer, and writes "<bundle>.tar.gz" plus a sha256 sidecar.
//
// Approval fails when the reviewer is empty, when the bundle was already
// approved, or when any staged file no longer matches the checksum recorded at
// staging time. That last check is the review gate: a reviewer must look at the
// exact bytes that go into the archive.
func Approve(dir, reviewer string) (*Bundle, error) {
	dir = strings.TrimSpace(dir)
	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		return nil, ErrReviewerRequired
	}

	manifest, err := readManifest(dir)
	if err != nil {
		return nil, err
	}
	if manifest.Format != Format || manifest.Version != Version {
		return nil, fmt.Errorf("unsupported bundle format %q version %d", manifest.Format, manifest.Version)
	}
	if manifest.Review.Status != "pending_review" {
		return nil, ErrAlreadyApproved
	}
	if _, err := os.Stat(filepath.Join(dir, reviewFile)); err == nil {
		return nil, ErrAlreadyApproved
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	entries := make([]FileEntry, 0, len(manifest.Files)+3)
	for _, want := range manifest.Files {
		got, err := fileEntry(dir, want.Path)
		if err != nil {
			return nil, err
		}
		if got.SHA256 != want.SHA256 || got.Bytes != want.Bytes {
			return nil, fmt.Errorf("%w: %s changed after staging", ErrChecksumMismatch, want.Path)
		}
		entries = append(entries, got)
	}
	if checksum := bundleChecksum(entries); checksum != manifest.Checksum {
		return nil, fmt.Errorf("%w: bundle checksum %s does not match %s", ErrChecksumMismatch, checksum, manifest.Checksum)
	}

	reviewedAt := time.Now().UTC()
	review := struct {
		Status         string    `json:"status"`
		Reviewer       string    `json:"reviewer"`
		ReviewedAt     time.Time `json:"reviewed_at"`
		BundleChecksum string    `json:"bundle_checksum"`
	}{Status: "approved", Reviewer: reviewer, ReviewedAt: reviewedAt, BundleChecksum: manifest.Checksum}
	reviewPath := filepath.Join(dir, reviewFile)
	if err := writeJSON(reviewPath, review); err != nil {
		return nil, err
	}

	manifest.Review = ReviewState{Status: "approved", Reviewer: reviewer, ReviewedAt: &reviewedAt}
	if err := writeJSON(filepath.Join(dir, manifestFile), manifest); err != nil {
		return nil, err
	}

	archiveFiles := append([]string{manifestFile}, payloadPaths(manifest.Files)...)
	archiveFiles = append(archiveFiles, reviewFile)
	name := filepath.Base(dir) + ".tar.gz"
	archivePath := filepath.Join(filepath.Dir(dir), name)
	digest, err := writeArchive(archivePath, dir, archiveFiles, manifest.CreatedAt)
	if err != nil {
		_ = os.Remove(reviewPath)
		return nil, err
	}
	checksumPath := archivePath + ".sha256"
	if err := os.WriteFile(checksumPath, []byte(digest+"  "+name+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("write archive checksum: %w", err)
	}
	return &Bundle{ArchivePath: archivePath, ChecksumPath: checksumPath, SHA256: digest, ReviewPath: reviewPath}, nil
}

func buildConfig(path string) ([]byte, SourceReport, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: the operator names the config file on the command line.
	if err != nil {
		return nil, SourceReport{}, fmt.Errorf("read config: %w", err)
	}
	var document any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return nil, SourceReport{}, fmt.Errorf("parse config %s: %w", filepath.Base(path), err)
	}
	source := SourceReport{Source: "config", File: configFile, InputBytes: int64(len(raw))}
	redacted := redactConfigValue("", document, "", &source)
	encoded, err := yaml.Marshal(redacted)
	if err != nil {
		return nil, SourceReport{}, fmt.Errorf("encode redacted config: %w", err)
	}
	header := "# Redacted by MWX-ISP support-bundle. Secrets are " + redact.Marker + ",\n" +
		"# customer identifiers are pseudonyms, addresses keep only their prefix.\n"
	content := append([]byte(header), encoded...)
	source.OutputBytes = int64(len(content))
	return content, source, nil
}

func redactConfigValue(key string, value any, path string, source *SourceReport) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for childKey, child := range typed {
			out[childKey] = redactConfigValue(childKey, child, joinKeyPath(path, childKey), source)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = redactConfigValue(key, child, path, source)
		}
		return out
	default:
		class := redact.ClassifyKey(key)
		switch class {
		case redact.Secret:
			recordRedaction(source, path, redact.Secret)
			return redact.Marker
		case redact.CustomerPII:
			recordRedaction(source, path, redact.CustomerPII)
			return redact.Pseudonymize(fmt.Sprint(value))
		case redact.Address:
			recordRedaction(source, path, redact.Address)
			return redact.MaskAddress(fmt.Sprint(value))
		}
		if text, ok := value.(string); ok {
			redacted := redact.Text(text)
			if redacted != text {
				source.Counts.FreeText++
			}
			return redacted
		}
		return value
	}
}

func buildLog(path string, tail int64) ([]byte, SourceReport, error) {
	file, err := os.Open(path) //nolint:gosec // G304: the operator names the log file on the command line.
	if err != nil {
		return nil, SourceReport{}, fmt.Errorf("open log: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, SourceReport{}, fmt.Errorf("stat log: %w", err)
	}
	size := info.Size()
	start := int64(0)
	if size > tail {
		start = size - tail
	}
	raw := make([]byte, size-start)
	if _, err := file.ReadAt(raw, start); err != nil && !errors.Is(err, io.EOF) {
		return nil, SourceReport{}, fmt.Errorf("read log: %w", err)
	}
	if start > 0 {
		if idx := strings.IndexByte(string(raw), '\n'); idx >= 0 {
			raw = raw[idx+1:]
		}
	}

	source := SourceReport{Source: "logs", File: logFile, InputBytes: int64(len(raw))}
	var out strings.Builder
	content := strings.TrimSuffix(string(raw), "\n")
	if content != "" {
		for _, line := range strings.Split(content, "\n") {
			out.WriteString(redactLogLine(line, &source))
			out.WriteString("\n")
		}
	}
	source.OutputBytes = int64(out.Len())
	return []byte(out.String()), source, nil
}

// redactLogLine keeps the JSON shape of a structured log line so the reviewer
// can still follow the incident, while removing values by class.
func redactLogLine(line string, source *SourceReport) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		redacted := redact.Text(line)
		if redacted != line {
			source.Counts.FreeText++
		}
		return redacted
	}
	for key, value := range fields {
		fields[key] = redactLogValue(key, value, source)
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return redact.Text(line)
	}
	return redact.Text(string(encoded))
}

func redactLogValue(key string, value any, source *SourceReport) any {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, child := range typed {
			typed[childKey] = redactLogValue(childKey, child, source)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = redactLogValue(key, child, source)
		}
		return typed
	}
	switch redact.ClassifyKey(key) {
	case redact.Secret:
		source.Counts.Secrets++
		return redact.Marker
	case redact.CustomerPII:
		source.Counts.PII++
		return redact.Pseudonymize(fmt.Sprint(value))
	case redact.Address:
		source.Counts.Addresses++
		return redact.MaskAddress(fmt.Sprint(value))
	}
	if text, ok := value.(string); ok {
		redacted := redact.Text(text)
		if redacted != text {
			source.Counts.FreeText++
		}
		return redacted
	}
	return value
}

func recordRedaction(source *SourceReport, path string, class redact.Class) {
	switch class {
	case redact.Secret:
		source.Counts.Secrets++
	case redact.CustomerPII:
		source.Counts.PII++
	case redact.Address:
		source.Counts.Addresses++
	}
	if path != "" {
		source.Keys = append(source.Keys, path)
	}
}

func joinKeyPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func readManifest(dir string) (Manifest, error) {
	var manifest Manifest
	raw, err := os.ReadFile(filepath.Join(dir, manifestFile)) //nolint:gosec // G304: dir is the staged bundle the operator names on the command line.
	if err != nil {
		return manifest, fmt.Errorf("read manifest: %w", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return manifest, fmt.Errorf("parse manifest: %w", err)
	}
	return manifest, nil
}

func fileEntry(root, rel string) (FileEntry, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(path) //nolint:gosec // G304: rel comes from the manifest written by Stage.
	if err != nil {
		return FileEntry{}, fmt.Errorf("read %s: %w", rel, err)
	}
	sum := sha256.Sum256(raw)
	return FileEntry{Path: rel, Bytes: int64(len(raw)), SHA256: hex.EncodeToString(sum[:])}, nil
}

// bundleChecksum hashes the sorted path/digest pairs so the value covers the
// set of payload files, not just each file in isolation.
func bundleChecksum(entries []FileEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.Path+"\x00"+entry.SHA256)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func payloadPaths(entries []FileEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return paths
}

func writeArchive(path, root string, files []string, mtime time.Time) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".support-bundle-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create archive: %w", err)
	}
	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}
	digest := sha256.New()
	gzipWriter := gzip.NewWriter(io.MultiWriter(temp, digest))
	tarWriter := tar.NewWriter(gzipWriter)

	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	for _, rel := range sorted {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) //nolint:gosec // G304: root is the staged bundle, rel comes from its manifest.
		if err != nil {
			cleanup()
			return "", fmt.Errorf("archive %s: %w", rel, err)
		}
		header := &tar.Header{
			Name:    rel,
			Mode:    0o644,
			Size:    int64(len(raw)),
			ModTime: mtime,
			Format:  tar.FormatPAX,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			cleanup()
			return "", fmt.Errorf("archive header %s: %w", rel, err)
		}
		if _, err := tarWriter.Write(raw); err != nil {
			cleanup()
			return "", fmt.Errorf("archive body %s: %w", rel, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		cleanup()
		return "", err
	}
	if err := gzipWriter.Close(); err != nil {
		cleanup()
		return "", err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return "", err
	}
	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("write archive: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func encodeJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeJSON(path string, value any) error {
	raw, err := encodeJSON(value)
	if err != nil {
		return err
	}
	return writeFile(path, raw)
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}
