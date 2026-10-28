// Command sbom writes a reproducible CycloneDX 1.5 software bill of materials
// for the artifacts MWX-ISP ships: the Go module graph and the frontend's
// production npm dependencies (the built frontend is embedded in the Go binary).
//
//	go run ./tools/sbom          # write sbom/go.cdx.json and sbom/web.cdx.json
//	go run ./tools/sbom -check   # fail when the committed copies are stale
//
// The output deliberately contains no timestamp, no random serial number, no
// machine-local module cache paths and no tool versions, and every list is
// sorted. The same go.mod/go.sum and web/package-lock.json therefore always
// produce the same bytes, so a diff means the dependency set really changed.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	goBOMName  = "go.cdx.json"
	webBOMName = "web.cdx.json"
	// noAssertion is the CycloneDX marker for a license the generator could not
	// classify; it means "needs human review", not "no license".
	noAssertion = "NOASSERTION"
)

func main() {
	outDir := flag.String("out", "sbom", "directory for the generated SBOM files")
	check := flag.Bool("check", false, "verify the existing files instead of writing them")
	npmDir := flag.String("npm-dir", "web", "frontend directory that holds package-lock.json")
	flag.Parse()

	if err := run(*outDir, *npmDir, *check); err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
}

func run(outDir, npmDir string, check bool) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}

	goDoc, err := buildGoBOM(root)
	if err != nil {
		return err
	}

	npmRaw, err := runTool(filepath.Join(root, npmDir), npmCommand(), "sbom", "--sbom-format=cyclonedx", "--omit=dev")
	if err != nil {
		return fmt.Errorf("npm sbom: %w", err)
	}
	webDoc, err := normalizeNPMBOM(npmRaw)
	if err != nil {
		return err
	}

	outputs := []struct {
		name string
		data []byte
	}{
		{goBOMName, goDoc},
		{webBOMName, webDoc},
	}
	for _, out := range outputs {
		path := filepath.Join(root, outDir, out.name)
		if check {
			if err := checkFile(path, out.data); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(path, out.data, 0o644); err != nil { //nolint:gosec // G306: the SBOM is a committed, world-readable artifact
			return err
		}
		fmt.Printf("wrote %s (%d bytes)\n", filepath.ToSlash(filepath.Join(outDir, out.name)), len(out.data))
	}
	return nil
}

// checkFile fails when the file on disk differs from the freshly generated
// document, which is how a stale committed SBOM is detected.
func checkFile(path string, want []byte) error {
	got, err := os.ReadFile(path) //nolint:gosec // G304: path is the generated SBOM file inside the checkout
	if err != nil {
		return fmt.Errorf("%s: %w (run: go run ./tools/sbom)", filepath.ToSlash(path), err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s is stale: regenerate with `go run ./tools/sbom`", filepath.ToSlash(path))
	}
	fmt.Printf("%s is up to date\n", filepath.ToSlash(path))
	return nil
}

// moduleRoot returns the directory of the module the command runs in, so the
// tool works from any subdirectory and from `go run ./tools/sbom`.
func moduleRoot() (string, error) {
	out, err := runTool("", "go", "env", "GOMOD")
	if err != nil {
		return "", err
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return "", errors.New("not inside a Go module")
	}
	return filepath.Dir(gomod), nil
}

func npmCommand() string {
	if runtime.GOOS == "windows" {
		return "npm.cmd"
	}
	return "npm"
}

func runTool(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...) //nolint:gosec // G204: the only commands ever launched are go and npm
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
		}
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
	}
	return out, nil
}

type bom struct {
	Schema      string   `json:"$schema"`
	BOMFormat   string   `json:"bomFormat"`
	SpecVersion string   `json:"specVersion"`
	Version     int      `json:"version"`
	Metadata    metadata `json:"metadata"`
	Components  []comp   `json:"components"`
}

type metadata struct {
	Tools     []tool `json:"tools"`
	Component comp   `json:"component"`
}

type tool struct {
	Vendor string `json:"vendor"`
	Name   string `json:"name"`
}

type comp struct {
	BOMRef     string          `json:"bom-ref"`
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	Version    string          `json:"version,omitempty"`
	PURL       string          `json:"purl,omitempty"`
	Scope      string          `json:"scope,omitempty"`
	Licenses   []licenseChoice `json:"licenses,omitempty"`
	Properties []property      `json:"properties,omitempty"`
}

type licenseChoice struct {
	License license `json:"license"`
}

type license struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func newBOM(main comp) bom {
	return bom{
		Schema:      "http://cyclonedx.org/schema/bom-1.5.schema.json",
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.5",
		Version:     1,
		Metadata: metadata{
			Tools:     []tool{{Vendor: "bjo163", Name: "tools/sbom"}},
			Component: main,
		},
	}
}

// buildGoBOM lists the effective module graph and renders it. Modules that are
// in the graph but not in the local cache have no license text to inspect, so
// they are downloaded first; a license the detector still cannot classify ends
// up as NOASSERTION rather than being guessed.
func buildGoBOM(root string) ([]byte, error) {
	mods, err := listModules(root)
	if err != nil {
		return nil, err
	}
	// go list leaves Dir empty for modules that were never extracted, and the
	// license text only exists in the extracted source. Anything that still
	// cannot be resolved stays NOASSERTION and shows up as a review gap.
	dirs := resolveModuleDirs(root, undownloadedModules(mods))
	for i := range mods {
		if effectiveDir(mods[i]) != "" {
			continue
		}
		dir := dirs[mods[i].Path+"@"+mods[i].Version]
		if dir == "" {
			continue
		}
		if mods[i].Replace != nil {
			mods[i].Replace.Dir = dir
		} else {
			mods[i].Dir = dir
		}
	}
	return renderGoBOM(root, mods)
}

type downloadResult struct {
	Path    string
	Version string
	Dir     string
}

// resolveModuleDirs downloads the given path@version specs and returns the
// extracted directory of each, keyed by path@version.
func resolveModuleDirs(root string, specs []string) map[string]string {
	if len(specs) == 0 {
		return nil
	}
	raw, err := runTool(root, "go", append([]string{"mod", "download", "-json"}, specs...)...)
	if err != nil && len(raw) == 0 {
		return nil
	}
	// A partial failure still prints one JSON object per module tried, so the
	// objects that were resolved are worth keeping.
	dirs := make(map[string]string, len(specs))
	dec := json.NewDecoder(bytes.NewReader(raw))
	for {
		var res downloadResult
		if err := dec.Decode(&res); err != nil {
			break
		}
		if res.Dir != "" {
			dirs[res.Path+"@"+res.Version] = res.Dir
		}
	}
	return dirs
}

func listModules(root string) ([]goModule, error) {
	raw, err := runTool(root, "go", "list", "-m", "-json", "all")
	if err != nil {
		return nil, err
	}
	return parseModuleStream(bytes.NewReader(raw))
}

func parseModuleStream(r io.Reader) ([]goModule, error) {
	dec := json.NewDecoder(r)
	var mods []goModule
	for {
		var mod goModule
		if err := dec.Decode(&mod); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("parse go list output: %w", err)
		}
		mods = append(mods, mod)
	}
	return mods, nil
}

// undownloadedModules returns path@version specs for modules whose source is
// missing from the module cache, which is where the license file is read from.
func undownloadedModules(mods []goModule) []string {
	var specs []string
	for _, mod := range mods {
		if mod.Main || mod.Version == "" || effectiveDir(mod) != "" {
			continue
		}
		specs = append(specs, mod.Path+"@"+mod.Version)
	}
	sort.Strings(specs)
	return specs
}

func renderGoBOM(root string, mods []goModule) ([]byte, error) {
	mainComp, comps, err := componentsFrom(root, mods)
	if err != nil {
		return nil, err
	}
	sortComponents(comps)

	doc := newBOM(mainComp)
	doc.Components = comps
	return marshal(doc)
}

func componentsFrom(root string, mods []goModule) (comp, []comp, error) {
	var mainComp comp
	var comps []comp
	for _, mod := range mods {
		if mod.Main {
			mainComp = comp{
				BOMRef:   "pkg:golang/" + mod.Path,
				Type:     "application",
				Name:     mod.Path,
				Version:  "(devel)",
				PURL:     "pkg:golang/" + mod.Path,
				Licenses: []licenseChoice{{License: detectLicense(root)}},
			}
			continue
		}
		comps = append(comps, moduleComponent(mod))
	}
	if mainComp.Name == "" {
		return comp{}, nil, errors.New("go list output has no main module")
	}
	return mainComp, comps, nil
}

type goModule struct {
	Path     string
	Version  string
	Dir      string
	Main     bool
	Indirect bool
	Replace  *goModule
}

// effectiveDir is the directory of the source that is actually compiled, which
// differs from Dir when the module is replaced by a fork.
func effectiveDir(mod goModule) string {
	if mod.Replace != nil {
		return mod.Replace.Dir
	}
	return mod.Dir
}

func moduleComponent(mod goModule) comp {
	path, version, dir := mod.Path, mod.Version, effectiveDir(mod)
	var props []property
	if mod.Replace != nil {
		props = append(props, property{Name: "go:replaces", Value: mod.Path + "@" + mod.Version})
		path, version = mod.Replace.Path, mod.Replace.Version
	}
	if mod.Indirect {
		props = append(props, property{Name: "go:indirect", Value: "true"})
	}
	return comp{
		BOMRef:     goPURL(path, version),
		Type:       "library",
		Name:       path,
		Version:    version,
		PURL:       goPURL(path, version),
		Scope:      "required",
		Licenses:   []licenseChoice{{License: detectLicense(dir)}},
		Properties: props,
	}
}

// goPURL builds a package URL. A "+" in a Go version (v2+incompatible) must be
// percent-encoded to stay a valid purl version.
func goPURL(path, version string) string {
	return "pkg:golang/" + path + "@" + strings.ReplaceAll(version, "+", "%2B")
}

func sortComponents(comps []comp) {
	sort.Slice(comps, func(i, j int) bool {
		if comps[i].Name != comps[j].Name {
			return comps[i].Name < comps[j].Name
		}
		return comps[i].Version < comps[j].Version
	})
}

// normalizeNPMBOM strips the fields npm sbom regenerates on every run
// (serialNumber, timestamp, tool version) and sorts the components, so the
// document only changes when the dependency set changes.
func normalizeNPMBOM(raw []byte) ([]byte, error) {
	var doc map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse npm sbom output: %w", err)
	}
	delete(doc, "serialNumber")
	if md, ok := doc["metadata"].(map[string]any); ok {
		delete(md, "timestamp")
		md["tools"] = []any{map[string]any{"vendor": "npm", "name": "cli"}}
	}
	if comps, ok := doc["components"].([]any); ok {
		sort.SliceStable(comps, func(i, j int) bool {
			return ref(comps[i]) < ref(comps[j])
		})
	}
	return marshal(doc)
}

func ref(v any) string {
	c, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := c["bom-ref"].(string)
	return s
}

func marshal(v any) ([]byte, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
