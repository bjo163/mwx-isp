package incidentcard

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

// The card's candidate field set is not chosen by this study. It is read off the
// response types the product already serves for an incident, then filtered through
// the shipped redaction rules and through what the server actually puts in them, so
// "data yang terbukti saja" means the slot is present in a served response AND no
// rule calls it sensitive AND no line of code proves it carries a subscriber value.

// incidentSources are the response types an operator sees when investigating an
// incident, named by file and type so a moved struct breaks a row instead of
// vanishing from the table.
var incidentSources = []struct{ file, name string }{
	{"internal/domain/isp_operations.go", "TroubleTicket"},
	{"internal/domain/network_monitor.go", "NetMonitorIncident"},
	{"internal/domain/network_monitor.go", "NetMonitorSample"},
	{"internal/domain/network_monitor.go", "NotificationOutbox"},
	{"internal/domain/mrtg_syslog.go", "SyslogEvent"},
	{"internal/adminapi/noc_impact.go", "nocImpactResponse"},
	{"internal/adminapi/noc_impact.go", "nocImpactIncident"},
	{"internal/adminapi/noc_impact.go", "nocImpactMonitoring"},
	{"internal/adminapi/noc_impact.go", "nocImpactScope"},
	{"internal/adminapi/noc_impact.go", "nocImpactCounts"},
	{"internal/adminapi/noc_impact.go", "nocImpactSubscriber"},
	{"internal/adminapi/noc_impact.go", "nocImpactPagination"},
	{"internal/adminapi/isp_operations.go", "flappingSubscriber"},
	{"internal/adminapi/customer_timeline.go", "customerTimelineEvent"},
	{"internal/adminapi/system_logs.go", "systemLogEntry"},
}

// domainDir holds the persisted models. A table is a subscriber-identity table for
// this study when the shipped rules already call one of its columns sensitive; its
// remaining columns are then the ones a card can still reach without a rule.
const domainDir = "internal/domain"

// apiDir holds the admin handlers, where a served slot gets its value.
const apiDir = "internal/adminapi"

// jsonSkip marks a field the response never carries.
const jsonSkip = "-"

// slot is one candidate card field with the four checks applied to it.
type slot struct {
	Struct string
	Go     string
	Key    string
	Type   string
	File   string
	Line   int
	// Class is what the shipped classifier says about the key ("" for a key that
	// is never served).
	Class string
	// From names the subscriber column a SELECT proves this slot carries, when the
	// slot is filled by a query rather than by the classifier's own list.
	From string
	// Proven is the file:line of that projection, so a rejection can be pointed at.
	Proven string
	// Guarded says whether the column behind a refusal is one the shipped rules
	// already name, or one the card would reach past them.
	Guarded bool
	// Composed is the file:line of a string assembly that puts a sensitive column
	// into this slot's free text.
	Composed string
	// Held is where the name trace puts the blame for a name-shaped slot that stays
	// off the sheet: the file:line of the assignment the study cannot clear, or the
	// handlers directory when no assignment writes the field at all.
	Held string
	// Disposition is the verdict: printed on the card, or why not.
	Disposition string
	// Note explains a disposition that a reader would otherwise have to derive.
	Note string
}

// identityTable is a persisted model that the shipped rules already partly cover.
type identityTable struct {
	Struct string
	Table  string
	File   string
	Line   int
	// Guarded are the columns the rules classify as secret or personal.
	Guarded []string
	// Unguarded are the columns of the same row that reach the default branch.
	Unguarded []string
	// Keys maps every served json key of the table to its column name.
	Keys map[string]string
	// Sensitive marks the Go field names the shipped rules already cover.
	Sensitive map[string]bool
}

type structDef struct {
	name   string
	file   string
	line   int
	table  string
	fields []slot
}

// readStructs parses the named structs of one file, with their table names and the
// json key, Go type and line of every field.
func readStructs(root, rel string, want map[string]bool) map[string]structDef {
	f, fset, ok := parseGo(root, rel)
	if !ok {
		return nil
	}
	out := map[string]structDef{}
	tables := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		fd, isFn := n.(*ast.FuncDecl)
		if !isFn || fd.Name.Name != "TableName" || fd.Recv == nil || fd.Body == nil || len(fd.Body.List) != 1 {
			return true
		}
		ret, isRet := fd.Body.List[0].(*ast.ReturnStmt)
		if !isRet || len(ret.Results) != 1 {
			return true
		}
		lit, isLit := ret.Results[0].(*ast.BasicLit)
		if !isLit {
			return true
		}
		if id, isStar := fd.Recv.List[0].Type.(*ast.StarExpr); isStar {
			if se, isSel := id.X.(*ast.SelectorExpr); isSel {
				tables[se.Sel.Name] = strings.Trim(lit.Value, `"`)
			}
			return true
		}
		if id, isIdent := fd.Recv.List[0].Type.(*ast.Ident); isIdent {
			tables[id.Name] = strings.Trim(lit.Value, `"`)
		}
		return true
	})

	for _, decl := range f.Decls {
		gd, isGen := decl.(*ast.GenDecl)
		if !isGen || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, isTS := spec.(*ast.TypeSpec)
			if !isTS || (want != nil && !want[ts.Name.Name]) {
				continue
			}
			st, isSt := ts.Type.(*ast.StructType)
			if !isSt {
				continue
			}
			def := structDef{
				name:  ts.Name.Name,
				file:  rel,
				line:  fset.Position(ts.Pos()).Line,
				table: tables[ts.Name.Name],
			}
			for _, fld := range st.Fields.List {
				if len(fld.Names) == 0 {
					continue
				}
				key := jsonSkip
				if fld.Tag != nil {
					key = jsonKey(strings.Trim(fld.Tag.Value, "`"))
				}
				if key == jsonSkip {
					// The field is not a card candidate; it is also not a leak path.
					continue
				}
				def.fields = append(def.fields, slot{
					Struct: def.name,
					Go:     fld.Names[0].Name,
					Key:    key,
					Type:   nodeString(fld.Type),
					File:   rel,
					Line:   fset.Position(fld.Pos()).Line,
				})
			}
			out[def.name] = def
		}
	}
	return out
}

// jsonKey returns the response key of a struct tag, or "-" when the tag hides it.
func jsonKey(tag string) string {
	for _, part := range strings.Split(tag, " ") {
		if !strings.HasPrefix(part, "json:") {
			continue
		}
		val := strings.TrimPrefix(part, "json:")
		val = strings.Trim(val, `"`)
		first := strings.Split(val, ",")[0]
		if first == "" {
			return jsonSkip
		}
		return first
	}
	return jsonSkip
}

func nodeString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + nodeString(e.X)
	case *ast.SelectorExpr:
		return nodeString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + nodeString(e.Elt)
	case *ast.MapType:
		return "map[" + nodeString(e.Key) + "]" + nodeString(e.Value)
	}
	return "unknown"
}

// identityTables finds the persisted models the shipped rules already name a
// sensitive column on, and splits their remaining columns into the two sets the
// card has to reason about.
func identityTables(root string, rules Rules) ([]identityTable, error) {
	files := walkFiles(root, domainDir, ".go")
	if len(files) == 0 {
		return nil, fmt.Errorf("no domain files found under %s", domainDir)
	}
	var out []identityTable
	for _, rel := range files {
		for _, def := range readStructs(root, rel, nil) {
			if def.table == "" {
				continue
			}
			t := identityTable{Struct: def.name, Table: def.table, File: rel, Line: def.line,
				Keys: map[string]string{}, Sensitive: map[string]bool{}}
			for _, fld := range def.fields {
				t.Keys[fld.Key] = fld.Go
				class := rules.Classify(fld.Key)
				switch class {
				case "Secret", "CustomerPII", "Address":
					t.Guarded = append(t.Guarded, fld.Key)
					t.Sensitive[fld.Go] = true
				default:
					t.Unguarded = append(t.Unguarded, fld.Key)
				}
			}
			if len(t.Guarded) == 0 {
				continue
			}
			sort.Strings(t.Guarded)
			sort.Strings(t.Unguarded)
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no persisted model carries a column the rules in %s name sensitive", redactFile)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Table < out[j].Table })
	return out, nil
}

// projection is one column a served query moves into a response key, or one
// sensitive read the server formats into a string. Guarded records whether the
// shipped rules already protect that column: a refusal that reaches past them is
// the finding the card has to carry.
type projection struct {
	Key     string
	Table   string
	Col     string
	Guarded bool
	File    string
	Line    int
}

var (
	reJoinAlias  = mustRe(`(?:FROM|JOIN)\s+([a-z_][a-z0-9_]*) AS ([a-z])\b`)
	reTableAlias = mustRe(`^([a-z_][a-z0-9_]*) AS ([a-z])\b`)
	reDotted     = mustRe(`\b([a-z])\.([a-z_][a-z0-9_]*)`)
	reDottedAs   = mustRe(`\b([a-z])\.([a-z_][a-z0-9_]*) AS ([a-z_][a-z0-9_]*)`)
	reWrappedAs  = mustRe(`\b([a-z])\.([a-z_][a-z0-9_]*)[^)]*\) AS ([a-z_][a-z0-9_]*)`)
	reAsWord     = mustRe(`(?i)\bAS\b`)
)

// selectListItem reports whether the dotted reference that ends at "end" is an item
// of a select list: the next thing in the literal is a comma or nothing at all. A
// reference followed by an operator, a sort direction or an alias is a comparison
// operand, an ORDER BY key, or the input of a projection the aliased passes already
// recorded under its destination key. Only list items carry a value toward a
// response slot, and the study will not claim a provenance the query does not show.
func selectListItem(text string, end int) bool {
	for i := end; i < len(text); i++ {
		switch text[i] {
		case ' ', '\t':
		case ',':
			return true
		default:
			return false
		}
	}
	return true
}

// readProjections reads the table aliases and selected columns out of the product's
// own SQL strings. It is how a slot the key rules call harmless can still be
// refused: the query says where its value comes from. Only a select-list item counts
// as a projection, because that is the only position where the query shows the value
// leaving the table toward a response.
func readProjections(root string, tables []identityTable) []projection {
	served := map[string]string{} // table.column -> table
	guarded := map[string]bool{}  // table.column -> the shipped rules already name it
	for _, t := range tables {
		protects := map[string]bool{}
		for _, key := range t.Guarded {
			protects[key] = true
		}
		for key := range t.Keys {
			served[t.Table+"."+key] = t.Table
			if protects[key] {
				guarded[t.Table+"."+key] = true
			}
		}
	}
	var out []projection
	for _, rel := range append(walkFiles(root, apiDir, ".go"), walkFiles(root, domainDir, ".go")...) {
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		aliases := map[string]string{}
		for _, l := range stringLiterals(root, rel) {
			for _, m := range reJoinAlias.FindAllStringSubmatch(l.Text, -1) {
				aliases[m[2]] = m[1]
			}
			for _, m := range reTableAlias.FindAllStringSubmatch(l.Text, -1) {
				aliases[m[2]] = m[1]
			}
		}
		if len(aliases) == 0 {
			continue
		}
		seen := map[string]bool{}
		add := func(p projection) {
			p.Guarded = guarded[p.Table+"."+p.Col]
			k := p.Key + "|" + p.Table + "|" + p.Col + "|" + p.File + "|" + itoa(p.Line)
			if seen[k] {
				return
			}
			seen[k] = true
			out = append(out, p)
		}
		for _, l := range stringLiterals(root, rel) {
			for _, m := range reWrappedAs.FindAllStringSubmatch(l.Text, -1) {
				if t, ok := aliases[m[1]]; ok {
					add(projection{Key: m[3], Table: t, Col: m[2], File: rel, Line: l.Line})
				}
			}
			for _, m := range reDottedAs.FindAllStringSubmatch(l.Text, -1) {
				if t, ok := aliases[m[1]]; ok {
					add(projection{Key: m[3], Table: t, Col: m[2], File: rel, Line: l.Line})
				}
			}
			for _, m := range reDotted.FindAllStringSubmatchIndex(l.Text, -1) {
				if !reAsWord.MatchString(l.Text) || !selectListItem(l.Text, m[1]) {
					continue
				}
				alias, col := l.Text[m[2]:m[3]], l.Text[m[4]:m[5]]
				t, ok := aliases[alias]
				if !ok {
					continue
				}
				if _, isColumn := served[t+"."+col]; !isColumn {
					continue
				}
				add(projection{Key: col, Table: t, Col: col, File: rel, Line: l.Line})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		if out[i].Table != out[j].Table {
			return out[i].Table < out[j].Table
		}
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// proseAssembly finds every fmt.Sprintf that reads a column out of an identity row
// and formats it into a string. This is how a customer name reaches a card slot
// without the slot ever being named after a customer: the key says "description",
// the format string says otherwise. The conservative direction is taken - any
// column of an identity table counts, because the study cannot rate a column
// harmless on the same rules it is auditing.
func proseAssembly(root string, tables []identityTable) []projection {
	identity := map[string]identityTable{}
	for _, t := range tables {
		identity[t.Struct] = t
	}
	if len(identity) == 0 {
		return nil
	}
	var out []projection
	for _, rel := range walkFiles(root, apiDir, ".go") {
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		f, fset, ok := parseGo(root, rel)
		if !ok {
			continue
		}
		vars := collectVars(f)
		if len(vars) == 0 {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall || !isSprintf(call) {
				return true
			}
			line := fset.Position(call.Pos()).Line
			for _, a := range call.Args[1:] {
				sel, isSel := a.(*ast.SelectorExpr)
				if !isSel {
					continue
				}
				id, isIdent := sel.X.(*ast.Ident)
				if !isIdent {
					continue
				}
				structName, ok := vars[id.Name]
				if !ok {
					continue
				}
				m, isModel := identity[structName]
				if !isModel {
					continue
				}
				out = append(out, projection{Table: m.Table, Col: sel.Sel.Name,
					Guarded: m.Sensitive[sel.Sel.Name], File: rel, Line: line})
			}
			return true
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// structOf unwraps the concrete type an expression produces, so `customer :=
// &domain.Customer{}` and `var customer domain.Customer` name the same model.
func structOf(expr ast.Expr) (ast.Expr, bool) {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return e.Type, true
	case *ast.UnaryExpr:
		return structOf(e.X)
	case *ast.CallExpr:
		if sel, isSel := e.Fun.(*ast.SelectorExpr); isSel && sel.Sel.Name == "First" {
			return nil, false
		}
	}
	return nil, false
}

func isSprintf(call *ast.CallExpr) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Sprintf" {
		return false
	}
	pkg, isPkg := sel.X.(*ast.Ident)
	return isPkg && pkg.Name == "fmt"
}

func mustRe(pattern string) *regexp.Regexp { return regexp.MustCompile(pattern) }
