package incidenttriage

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// This file repeats the three evidence rules the ENG-006, ENG-007, ECO-006,
// FUT-001 and FUT-017 studies use: a citation must land on a code line, an
// identifier must match as a whole, and drift is reported instead of dropped.
// It is a package-local copy on purpose - two research packages that import
// each other stop being independent evidence.

// ownPackage names this prototype. Every scan of the product tree skips it, so
// no claim or gap is settled by the study's own fixtures.
const ownPackage = "incidenttriage"

// productFiles walks the product directories, skipping this prototype's own
// package and every _test.go, so a claim is proven by product code and never by
// a fixture next to it.
func productFiles(root string) ([]string, error) {
	var out []string
	for _, dir := range claimDirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if filepath.Base(path) == ownPackage {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			out = append(out, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// blankComments turns every comment into spaces of the same length, so a symbol
// mentioned only in prose cannot confirm a claim. Quote-aware: a // inside a
// string literal does not end the line.
func blankComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	runes := []rune(src)
	for i := 0; i < len(runes); {
		switch {
		case runes[i] == '"' || runes[i] == '\'':
			quote := runes[i]
			b.WriteRune(runes[i])
			i++
			for i < len(runes) && runes[i] != quote {
				if runes[i] == '\\' && i+1 < len(runes) {
					b.WriteRune(runes[i])
					b.WriteRune(runes[i+1])
					i += 2
					continue
				}
				b.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				b.WriteRune(runes[i])
				i++
			}
		case runes[i] == '/' && i+1 < len(runes) && runes[i+1] == '/':
			for i < len(runes) && runes[i] != '\n' {
				runes[i] = ' '
				i++
			}
		case runes[i] == '/' && i+1 < len(runes) && runes[i+1] == '*':
			for i < len(runes) {
				if runes[i] == '*' && i+1 < len(runes) && runes[i+1] == '/' {
					break
				}
				if runes[i] != '\n' {
					runes[i] = ' '
				}
				i++
			}
			if i < len(runes) {
				runes[i] = ' '
				i++
				if i < len(runes) {
					runes[i] = ' '
					i++
				}
			}
		default:
			b.WriteRune(runes[i])
			i++
		}
	}
	return b.String()
}

// locate returns the 1-based line holding the first code-line occurrence of
// needle, or 0.
func locate(lines []string, needle string) int {
	for i, ln := range lines {
		if strings.Contains(ln, needle) {
			return i + 1
		}
	}
	return 0
}

// isIdentRune is the one alphabet both the boundary test and the identifier
// test share.
func isIdentRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

// isIdent reports whether the whole token is identifier-shaped, which decides
// between a bounded match and a phrase match.
func isIdent(token string) bool {
	for _, r := range token {
		if !isIdentRune(r) {
			return false
		}
	}
	return len(token) > 0
}

func isWordBoundary(r rune) bool { return !isIdentRune(r) }

// containsBounded matches token as a whole word when it is identifier-shaped,
// and as a phrase otherwise. A phrase match still refuses when the character
// before or after is a word character, so "bind" cannot be proven by "binding"
// in the other direction.
func containsBounded(text, token string) bool {
	if text == "" || token == "" {
		return false
	}
	if !isIdent(token) {
		return strings.Contains(text, token)
	}
	for i := 0; i+len(token) <= len(text); i++ {
		if text[i:i+len(token)] != token {
			continue
		}
		if i > 0 && !isWordBoundary(rune(text[i-1])) {
			continue
		}
		if j := i + len(token); j < len(text) && !isWordBoundary(rune(text[j])) {
			continue
		}
		return true
	}
	return false
}

// readFile is the only file access in this package; it is read-only.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: caller passes a path from the scanned tree.
	return string(b), err
}
