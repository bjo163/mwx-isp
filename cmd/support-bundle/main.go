/*
 * MWX-ISP support bundle tool
 *
 * Stages a redacted copy of the configuration and the application log, then —
 * only after a reviewer approves the staged copy — writes a tar.gz archive for
 * handoff. Staging and approval are separate commands so no archive can be
 * produced without someone looking at the exact bytes first.
 *
 * Usage:
 *   go run ./cmd/support-bundle -c mwx-isp.yml -log data/logs/app.log -out ./support-out
 *   go run ./cmd/support-bundle -approve ./support-out/support-bundle-20261029T090000Z -reviewer alice@example.com
 *
 * The tool does not read the database, does not open a network connection and
 * does not upload anything. "Export" means the operator moves the archive.
 */
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/supportbundle"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("support-bundle", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("c", "", "configuration file to stage (optional)")
	logPath := flags.String("log", "", "application log file to stage (optional)")
	outDir := flags.String("out", "", "directory that receives the staged bundle")
	logTail := flags.Int64("log-tail", supportbundle.DefaultLogTailBytes, "bytes of the log tail to stage")
	appVersion := flags.String("version", "unknown", "MWX-ISP version recorded in the bundle")
	approveDir := flags.String("approve", "", "staged bundle directory to approve and archive")
	reviewer := flags.String("reviewer", "", "identity approving the bundle (required with -approve)")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*approveDir) != "" {
		if *configPath != "" || *logPath != "" || *outDir != "" {
			_, _ = fmt.Fprintln(stderr, "Error: -approve cannot be combined with staging flags")
			return 2
		}
		if strings.TrimSpace(*reviewer) == "" {
			_, _ = fmt.Fprintln(stderr, "Error: -reviewer is required with -approve")
			return 2
		}
		return approve(*approveDir, *reviewer, stdout, stderr)
	}
	if *configPath == "" && *logPath == "" {
		_, _ = fmt.Fprintln(stderr, "Error: provide -c, -log, or both")
		return 2
	}
	if strings.TrimSpace(*outDir) == "" {
		_, _ = fmt.Fprintln(stderr, "Error: -out is required for staging")
		return 2
	}

	staged, err := supportbundle.Stage(supportbundle.Options{
		OutputDir:    *outDir,
		ConfigPath:   *configPath,
		LogPath:      *logPath,
		LogTailBytes: *logTail,
		AppVersion:   *appVersion,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: stage support bundle: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Staged support bundle: %s\n", staged.Dir)
	_, _ = fmt.Fprintf(stdout, "Bundle checksum: %s\n", staged.Manifest.Checksum)
	_, _ = fmt.Fprintf(stdout, "Redacted: %d secrets, %d customer identifiers, %d addresses, %d free-text values\n",
		staged.Report.Totals.Secrets, staged.Report.Totals.PII,
		staged.Report.Totals.Addresses, staged.Report.Totals.FreeText)
	_, _ = fmt.Fprintln(stdout, "Review before export:")
	_, _ = fmt.Fprintf(stdout, "  1. Open %s and read the redaction report\n", staged.Dir)
	_, _ = fmt.Fprintln(stdout, "  2. Delete the bundle if any customer value or credential survived")
	_, _ = fmt.Fprintf(stdout, "  3. Approve: support-bundle -approve %s -reviewer <your-name>\n", staged.Dir)
	_, _ = fmt.Fprintln(stdout, "No archive was created and nothing was sent off-host.")
	return 0
}

func approve(dir, reviewer string, stdout, stderr io.Writer) int {
	bundle, err := supportbundle.Approve(dir, reviewer)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: approve support bundle: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "Approved by %s\n", strings.TrimSpace(reviewer))
	_, _ = fmt.Fprintf(stdout, "Archive: %s\n", bundle.ArchivePath)
	_, _ = fmt.Fprintf(stdout, "SHA256: %s\n", bundle.SHA256)
	_, _ = fmt.Fprintf(stdout, "Checksum file: %s\n", bundle.ChecksumPath)
	_, _ = fmt.Fprintln(stdout, "Share the archive only with the party handling the incident.")
	return 0
}
