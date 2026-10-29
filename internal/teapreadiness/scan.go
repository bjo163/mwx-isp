package teapreadiness

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// scanDirs are the product-code locations a readiness scan reads, relative to
// the repository root. They are named explicitly: a scan that walked the whole
// repository would find TEAP discussion in docs and research notes and report a
// mechanism as implemented because somebody wrote about it.
var scanDirs = []string{
	"internal/radiusd/plugins/eap",
	"internal/radiusd",
	"internal/app",
}

// Scan reads the product code under root and returns the comment-free tree the
// analyzer works on. It fails when a scan directory is absent rather than
// reporting an empty tree, because an empty tree makes every mechanism look
// missing and that would be the most convincing wrong answer this package can
// produce.
func Scan(root string) (Tree, error) {
	tree := Tree{Files: map[string][]string{}}
	for _, dir := range scanDirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		info, err := os.Stat(base)
		if err != nil {
			return Tree{}, fmt.Errorf("teapreadiness: scan directory %s: %w", dir, err)
		}
		if !info.IsDir() {
			return Tree{}, fmt.Errorf("teapreadiness: scan path %s is not a directory", dir)
		}
		if err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			slash := filepath.ToSlash(rel)
			if !productFile(entry, slash) {
				return nil
			}
			//nolint:gosec // G304: repository paths named by scanDirs, never user input.
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			tree.Files[slash] = strings.Split(stripComments(string(raw)), "\n")
			return nil
		}); err != nil {
			return Tree{}, fmt.Errorf("teapreadiness: reading %s: %w", dir, err)
		}
	}
	return tree, nil
}

// productFile reports whether a walked entry is product code the analyzer may
// cite. Test files are excluded because a test can define the very identifier a
// verdict looks for without the server ever using it, and the same identifier in
// a comment proves only that somebody read the RFC.
func productFile(entry fs.DirEntry, slashPath string) bool {
	if entry.IsDir() {
		return false
	}
	if !strings.HasSuffix(slashPath, ".go") {
		return false
	}
	if strings.HasSuffix(slashPath, "_test.go") {
		return false
	}
	return !strings.Contains(slashPath, "/testdata/")
}

// stripComments replaces every Go comment with spaces, keeping the line count so
// the line numbers the report prints still address the original file. String and
// rune literals are passed through untouched, since a "//" inside a string is
// not a comment.
func stripComments(src string) string {
	var out strings.Builder
	out.Grow(len(src))
	line := func(b byte) { out.WriteByte(b) }
	for i := 0; i < len(src); {
		switch c := src[i]; c {
		case '/':
			if i+1 < len(src) && src[i+1] == '/' {
				for i < len(src) && src[i] != '\n' {
					line(' ')
					i++
				}
				continue
			}
			if i+1 < len(src) && src[i+1] == '*' {
				for i < len(src) {
					if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
						line(' ')
						line(' ')
						i += 2
						break
					}
					if src[i] == '\n' {
						line('\n')
					} else {
						line(' ')
					}
					i++
				}
				continue
			}
			line(c)
			i++
		case '"':
			i = copyQuoted(&out, src, i, false)
		case '`':
			i = copyQuoted(&out, src, i, true)
		case '\'':
			i = copyQuoted(&out, src, i, false)
		default:
			line(c)
			i++
		}
	}
	return out.String()
}

// copyQuoted copies one literal from src starting at i to out and returns the
// index just past its closing delimiter. Comments cannot start inside a literal,
// so the scanner only has to find the matching end.
func copyQuoted(out *strings.Builder, src string, i int, raw bool) int {
	start := i
	out.WriteByte(src[i])
	i++
	for i < len(src) {
		c := src[i]
		out.WriteByte(c)
		i++
		if !raw && c == '\\' && i < len(src) {
			out.WriteByte(src[i])
			i++
			continue
		}
		if raw && c == '`' {
			return i
		}
		if !raw && (c == '"' || (c == '\'' && i > start+1)) {
			return i
		}
	}
	return i
}
