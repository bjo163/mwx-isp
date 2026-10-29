package migrationcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// importFile is the one handler that turns a customer file into accounts. The
// measured surface is read from it, not recited from a document.
const importFile = "internal/adminapi/users_import.go"

// fileReader is the shared upload reader the import delegates to.
const fileReader = "internal/webserver/server.go"

// ImportField is one column the validator reads, with the header spellings it
// accepts for it.
type ImportField struct {
	// Column is the canonical header, the first spelling the validator looks for.
	Column string
	// Aliases lists every header spelling that satisfies the column, in the order
	// the validator tries them.
	Aliases []string
}

// MigrationSurface is the vocabulary a client's export has to be written against.
type MigrationSurface struct {
	// Columns are the fields read with importMapString, ordered as first seen.
	Columns []ImportField
	// Rejections are the distinct reasons a row can be refused.
	Rejections []string
	// Formats are the upload types the shared reader dispatches on.
	Formats []string
	// Sheet is the worksheet name the spreadsheet reader is handed.
	Sheet string
	// ConditionalFields are columns that fall back to the billing profile when the
	// row leaves them empty.
	ConditionalFields []string
	// ProfileFields are columns always copied from the billing profile.
	ProfileFields []string
	// Defaults are values the validator fills in when the row omits a column.
	Defaults []string
}

// HeaderCount is how many distinct header spellings the validator accepts.
func (s MigrationSurface) HeaderCount() int {
	seen := map[string]bool{}
	for _, f := range s.Columns {
		for _, a := range f.Aliases {
			seen[a] = true
		}
	}
	return len(seen)
}

// aliasRE matches one importMapString call and captures its header spellings.
var aliasRE = regexp.MustCompile(`importMapString\(item,\s*("(?:[^"\\]|\\.)*"(?:,\s*"(?:[^"\\]|\\.)*")*)\)`)

// stringLitRE pulls the quoted values out of a captured argument list.
var stringLitRE = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)

// rejectRE matches one row rejection and captures the reason. A reason built by
// fmt.Sprintf keeps its literal template, which is the wording an operator sees.
var rejectRE = regexp.MustCompile(`reject\([^,]+,\s*(?:fmt\.Sprintf\()?"((?:[^"\\]|\\.)*)"`)

// conditionalRE matches a column filled from the profile only when the row left
// it empty.
var conditionalRE = regexp.MustCompile(`(\w+) = profile\.(\w+)$`)

// defaultRE matches a value the validator supplies when the row omits the column.
var defaultRE = regexp.MustCompile(`(\w+) = common\.(\w+)$`)

// expireDefaultRE matches the expiry horizon applied to a row without a date.
var expireDefaultRE = regexp.MustCompile(`defaultExpire := now\.AddDate\((\d+), 0, 0\)`)

// profileCopyRE matches a column copied straight from the profile in the row
// builder.
var profileCopyRE = regexp.MustCompile(`(\w+):\s*profile\.(\w+),\s*$`)

// sheetRE matches the worksheet name handed to the shared reader.
var sheetRE = regexp.MustCompile(`ImportData\(c,\s*"([^"]+)"\)`)

// formatRE matches the filename suffixes the shared reader dispatches on.
var formatRE = regexp.MustCompile(`HasSuffix\(file\.Filename,\s*"([^"]+)"\)`)

// MeasureSurface reads the import handler and the upload reader and reports the
// vocabulary they actually accept. It is the input to the migration checklist
// and to every effort driver, so a widened validator shows up as a changed count
// rather than as a stale paragraph.
func MeasureSurface(root string) (MigrationSurface, error) {
	//nolint:gosec // G304: paths from the catalog, resolved under the repository root.
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(importFile)))
	if err != nil {
		return MigrationSurface{}, fmt.Errorf("migrationcheck: %s: %w", importFile, err)
	}
	//nolint:gosec // G304: paths from the catalog, resolved under the repository root.
	reader, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fileReader)))
	if err != nil {
		return MigrationSurface{}, fmt.Errorf("migrationcheck: %s: %w", fileReader, err)
	}

	var surface MigrationSurface
	for _, m := range aliasRE.FindAllStringSubmatch(string(src), -1) {
		var aliases []string
		for _, lit := range stringLitRE.FindAllStringSubmatch(m[1], -1) {
			aliases = append(aliases, lit[1])
		}
		if len(aliases) == 0 {
			continue
		}
		surface.Columns = append(surface.Columns, ImportField{Column: aliases[0], Aliases: aliases})
	}

	seenReject := map[string]bool{}
	for _, m := range rejectRE.FindAllStringSubmatch(string(src), -1) {
		if seenReject[m[1]] {
			continue
		}
		seenReject[m[1]] = true
		surface.Rejections = append(surface.Rejections, m[1])
	}

	for _, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if m := conditionalRE.FindStringSubmatch(trimmed); m != nil {
			surface.ConditionalFields = append(surface.ConditionalFields, m[1]+"<-profile."+m[2])
		}
		if m := profileCopyRE.FindStringSubmatch(trimmed); m != nil {
			surface.ProfileFields = append(surface.ProfileFields, m[1]+"<-profile."+m[2])
		}
		if m := defaultRE.FindStringSubmatch(trimmed); m != nil {
			surface.Defaults = append(surface.Defaults, m[1]+"="+m[2])
		}
		if m := expireDefaultRE.FindStringSubmatch(trimmed); m != nil {
			surface.Defaults = append(surface.Defaults, "expire_time=now+"+m[1]+"y")
		}
	}

	if m := sheetRE.FindStringSubmatch(string(src)); m != nil {
		surface.Sheet = m[1]
	}
	for _, m := range formatRE.FindAllStringSubmatch(string(reader), -1) {
		surface.Formats = append(surface.Formats, m[1])
	}
	surface.Formats = append(surface.Formats, "xlsx (default when the name is neither JSON nor CSV)")

	sort.SliceStable(surface.Columns, func(i, j int) bool { return surface.Columns[i].Column < surface.Columns[j].Column })
	return surface, nil
}
