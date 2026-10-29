// Package policycheck classifies raw SQL statements by table-level tenant
// scope.
//
// It is a bounded research prototype behind FUT-010, not a product gate: the
// production ratchet in internal/tenancy accepts a statement when the string
// "tenant_id" appears anywhere in it, which a comment or an ORDER BY clause
// satisfies without bounding a single row. This package instead asks, per
// table, whether a predicate with a constant value scopes it — directly or
// through an equality chain to an already scoped table.
//
// Only statements whose raw-SQL arguments are all string literals can be
// analyzed. Anything assembled at runtime is reported as dynamic and left to
// review, because there is no statement text to inspect.
package policycheck

import (
	"strings"
)

// Table is one relation referenced by a statement.
type Table struct {
	Name   string
	Alias  string
	Scoped bool
}

// Report is the scope classification of one statement.
type Report struct {
	Tables []Table
	// Unscoped lists "table AS alias" for relations without a scoping
	// predicate. Empty means every referenced relation is bounded to a tenant.
	Unscoped []string
}

// Scoped reports whether the statement bounds every referenced relation.
func (r Report) Scoped() bool {
	return len(r.Unscoped) == 0
}

type tokenKind int

const (
	kindIdent tokenKind = iota
	kindString
	kindPlaceholder
	kindNumber
	kindPunct
)

type sqlToken struct {
	kind tokenKind
	text string
}

func (t sqlToken) isIdent(word string) bool {
	return t.kind == kindIdent && t.text == word
}

// keywords that end a predicate region. They are matched only outside
// parentheses, so a subquery's own clauses do not leak into the outer region.
var predicateTerminators = map[string]bool{
	"where": true, "on": true, "group": true, "order": true, "limit": true,
	"having": true, "union": true, "offset": true, "returning": true,
	"set": true, "values": true, "select": true, "from": true, "join": true,
	"left": true, "right": true, "inner": true, "outer": true, "full": true,
	"cross": true,
}

// clauseKeywords start a table list or predicate region.
var tableIntroducers = map[string]bool{"from": true, "join": true, "update": true, "into": true}

var aliasStopWords = map[string]bool{
	"where": true, "join": true, "left": true, "right": true, "inner": true,
	"outer": true, "full": true, "cross": true, "on": true, "group": true,
	"order": true, "limit": true, "having": true, "union": true, "set": true,
	"values": true, "select": true, "returning": true, "as": true, "using": true,
	"offset": true, "and": true, "or": true,
}

// Analyze classifies every relation in statement. Parenthesised subqueries are
// not descended into; their tables are the responsibility of the statement
// that owns them.
func Analyze(statement string) Report {
	tokens := lex(statement)
	var report Report
	aliases := make(map[string]string) // alias (or name) -> name
	order := make([]string, 0, 4)

	addTable := func(name, alias string) {
		if name == "" {
			return
		}
		if alias == "" {
			alias = name
		}
		report.Tables = append(report.Tables, Table{Name: name, Alias: alias})
		aliases[alias] = name
		order = append(order, alias)
	}

	scoped := make(map[string]bool)
	var relational [][2]string
	bareScoped := false

	depth := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.kind == kindPunct {
			switch tok.text {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				}
			}
			continue
		}
		if depth > 0 || tok.kind != kindIdent {
			continue
		}

		if tableIntroducers[tok.text] {
			name, alias, next := readRelation(tokens, i+1)
			if name != "" {
				addTable(name, alias)
			}
			i = next - 1
			continue
		}
		if tok.text == "where" || tok.text == "on" {
			end := predicateEnd(tokens, i+1)
			scanPredicate(tokens[i+1:end], scoped, &relational, &bareScoped)
			i = end - 1
		}
	}

	// A bare "tenant_id = ?" has a single possible column owner. In a
	// multi-table statement it is ambiguous, so only a one-table statement is
	// scoped by it.
	if bareScoped && len(report.Tables) == 1 {
		scoped[report.Tables[0].Alias] = true
	}

	// A relational predicate only scopes its table when the other side is
	// already scoped; iterate until no new table is reachable.
	for changed := true; changed; {
		changed = false
		for _, edge := range relational {
			left, right := edge[0], edge[1]
			if scoped[right] && !scoped[left] {
				scoped[left] = true
				changed = true
			}
			if scoped[left] && !scoped[right] {
				scoped[right] = true
				changed = true
			}
		}
	}

	for i := range report.Tables {
		table := &report.Tables[i]
		table.Scoped = scoped[table.Alias]
		if !table.Scoped {
			report.Unscoped = append(report.Unscoped, describe(*table))
		}
	}
	return report
}

func describe(table Table) string {
	if table.Alias != table.Name {
		return table.Name + " AS " + table.Alias
	}
	return table.Name
}

// readRelation reads "schema.table [AS] alias" starting at start and returns
// the table name, alias, and the index after the consumed tokens.
func readRelation(tokens []sqlToken, start int) (name, alias string, next int) {
	i := start
	if i >= len(tokens) || tokens[i].kind != kindIdent || tableIntroducers[tokens[i].text] {
		return "", "", start
	}
	name = tokens[i].text
	i++
	if i < len(tokens) && tokens[i].isIdent("as") {
		i++
		if i < len(tokens) && tokens[i].kind == kindIdent && !aliasStopWords[tokens[i].text] {
			alias = tokens[i].text
			i++
		}
		return name, alias, i
	}
	if i < len(tokens) && tokens[i].kind == kindIdent && !aliasStopWords[tokens[i].text] {
		alias = tokens[i].text
		i++
	}
	return name, alias, i
}

// predicateEnd returns the index of the first token that closes the predicate
// region, honoring parentheses.
func predicateEnd(tokens []sqlToken, start int) int {
	depth := 0
	for i := start; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.kind == kindPunct {
			switch tok.text {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				}
			}
			continue
		}
		if depth == 0 && tok.kind == kindIdent && predicateTerminators[tok.text] {
			return i
		}
	}
	return len(tokens)
}

// scanPredicate records "alias.tenant_id = constant" as scoping,
// "alias.tenant_id = other.tenant_id" as a relational edge, and a bare
// "tenant_id = constant" as scoping for the statement's only table.
func scanPredicate(tokens []sqlToken, scoped map[string]bool, relational *[][2]string, bareScoped *bool) {
	for i := 0; i < len(tokens); i++ {
		left, ok := tenantRef(tokens[i])
		if !ok {
			continue
		}
		if i+2 >= len(tokens) || tokens[i+1].kind != kindPunct || tokens[i+1].text != "=" {
			continue
		}
		if right, isRef := tenantRef(tokens[i+2]); isRef {
			if left != "" && right != "" {
				*relational = append(*relational, [2]string{left, right})
			}
			i += 2
			continue
		}
		switch tokens[i+2].kind {
		case kindPlaceholder, kindNumber, kindString:
			if left == "" {
				*bareScoped = true
			} else {
				scoped[left] = true
			}
			i += 2
		}
	}
}

// tenantRef returns the alias owning a tenant_id column reference. A bare
// "tenant_id" returns "" and is scoped by the caller.
func tenantRef(tok sqlToken) (string, bool) {
	if tok.kind != kindIdent {
		return "", false
	}
	if tok.text == "tenant_id" {
		return "", true
	}
	if strings.HasSuffix(tok.text, ".tenant_id") {
		return strings.TrimSuffix(tok.text, ".tenant_id"), true
	}
	return "", false
}

func lex(statement string) []sqlToken {
	tokens := make([]sqlToken, 0, 32)
	for i := 0; i < len(statement); {
		c := statement[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '-' && i+1 < len(statement) && statement[i+1] == '-':
			for i < len(statement) && statement[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(statement) && statement[i+1] == '*':
			i += 2
			for i+1 < len(statement) && (statement[i] != '*' || statement[i+1] != '/') {
				i++
			}
			if i+1 < len(statement) {
				i += 2
			}
		case c == '\'':
			start := i
			i++
			for i < len(statement) {
				if statement[i] == '\'' {
					if i+1 < len(statement) && statement[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			tokens = append(tokens, sqlToken{kind: kindString, text: statement[start:i]})
		case c == '"' || c == '`':
			quote := c
			start := i
			i++
			for i < len(statement) && statement[i] != quote {
				i++
			}
			if i < len(statement) {
				i++
			}
			tokens = append(tokens, sqlToken{kind: kindIdent, text: strings.ToLower(strings.Trim(statement[start:i], "\"`"))})
		case c == '?':
			tokens = append(tokens, sqlToken{kind: kindPlaceholder, text: "?"})
			i++
		case c == '$':
			start := i
			i++
			for i < len(statement) && statement[i] >= '0' && statement[i] <= '9' {
				i++
			}
			tokens = append(tokens, sqlToken{kind: kindPlaceholder, text: statement[start:i]})
		case isDigit(c):
			start := i
			for i < len(statement) && (isDigit(statement[i]) || statement[i] == '.') {
				i++
			}
			tokens = append(tokens, sqlToken{kind: kindNumber, text: statement[start:i]})
		case isIdentStart(c):
			start := i
			for i < len(statement) && isIdentPart(statement[i]) {
				i++
			}
			tokens = append(tokens, sqlToken{kind: kindIdent, text: strings.ToLower(statement[start:i])})
		default:
			tokens = append(tokens, sqlToken{kind: kindPunct, text: string(c)})
			i++
		}
	}
	return tokens
}

func isDigit(c byte) bool      { return c >= '0' && c <= '9' }
func isIdentStart(c byte) bool { return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isIdentPart(c byte) bool {
	return isIdentStart(c) || isDigit(c) || c == '.' || c == '"' || c == '`'
}
