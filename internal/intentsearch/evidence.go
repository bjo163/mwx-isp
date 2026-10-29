package intentsearch

import "strings"

// The three rules every study in this portfolio uses, repeated here so the package
// stays standalone: a citation must land on a code line (comment lines are blanked
// first), a symbol must match as a whole identifier, and drift is reported rather
// than dropped.

// blankComments returns text with comment-only lines removed and trailing comments
// cut, keeping the line count so a found symbol still reports its real line.
func blankComments(text string) string {
	var out []string
	inBlock := false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if inBlock {
			if strings.Contains(trimmed, "*/") {
				inBlock = false
			}
			out = append(out, "")
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			inBlock = !strings.Contains(trimmed, "*/")
			out = append(out, "")
			continue
		}
		if strings.HasPrefix(trimmed, "*/") || strings.HasPrefix(trimmed, "*") {
			out = append(out, "")
			continue
		}
		if i := strings.Index(trimmed, "//"); i >= 0 &&
			strings.Count(trimmed[:i], "\"")%2 == 0 && strings.Count(trimmed[:i], "'")%2 == 0 {
			trimmed = strings.TrimRight(trimmed[:i], " \t")
			if trimmed == "" {
				out = append(out, "")
				continue
			}
			out = append(out, trimmed)
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// locate returns the one-based line where symbol appears as a whole identifier, or
// zero when it does not appear at all.
func locate(lines []string, symbol string) int {
	needle := strings.TrimSpace(symbol)
	for i, l := range lines {
		if containsBounded(l, needle) {
			return i + 1
		}
	}
	return 0
}

// isIdent reports whether b can continue a JavaScript or Go identifier.
func isIdent(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '$'
}

// containsBounded reports whether text holds needle with identifier edges, so a
// longer name can never prove a shorter claim.
func containsBounded(text, needle string) bool {
	if needle == "" {
		return false
	}
	needLeft := isIdent(needle[0])
	needRight := isIdent(needle[len(needle)-1])
	for i := 0; i+len(needle) <= len(text); i++ {
		if text[i:i+len(needle)] != needle {
			continue
		}
		if needLeft && i > 0 && isIdent(text[i-1]) {
			continue
		}
		if needRight && i+len(needle) < len(text) && isIdent(text[i+len(needle)]) {
			continue
		}
		return true
	}
	return false
}
