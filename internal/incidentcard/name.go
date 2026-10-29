package incidentcard

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

// The criterion's last clause is about names, so a name-shaped slot gets its own
// measurement: not what the key looks like, but where the handler actually reads the
// value from. A card that prints "name" taken from a NAS record is not the leak the
// criterion forbids; a card that prints "name" taken from the customer row is, and
// the difference is only visible in the assignment.

// subscriberRows are the persisted models whose rows carry the subscriber's own name:
// isp_customer.name is the person the complaint is about, and radius_user.realname is
// the same person on the account record. Every other identity row names equipment, a
// site, or the operator's company.
var subscriberRows = map[string]bool{"isp_customer": true, "radius_user": true}

// isNameShape reports whether a served key speaks of a named party. The test is
// deliberately dumb - the key says name - because the point is to catch every slot
// where a name could be printed, including the one the classifier does not guard.
func isNameShape(key string) bool {
	return key == "name" || key == "realname" || strings.HasSuffix(key, "_name")
}

// filler is one assignment the product performs to a name-shaped slot, with the row
// the value is read out of.
type filler struct {
	Struct string
	Go     string
	// Table is the persisted model the value comes from when the right-hand side is a
	// read of a row this study can read; otherwise it is the Go type it came from.
	Table string
	// Column is the served key of that model, or the Go field name when untraceable.
	Column string
	// Traced says the value was read out of a persisted model the study knows.
	Traced bool
	// Subscriber marks the one source the criterion forbids.
	Subscriber bool
	// Guarded says whether the shipped rules already name the column read.
	Guarded bool
	File    string
	Line    int
}

func (f filler) source() string {
	if f.Table == "" {
		return "an expression the study cannot resolve"
	}
	return f.Table + "." + f.Column
}

func (f filler) where() string { return f.File + ":" + itoa(f.Line) }

// collectVars maps the identifiers of one file to the domain model they hold, so a
// selector read can be traced back to a table.
func collectVars(f *ast.File) map[string]string {
	vars := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.DeclStmt:
			gen, isGen := d.Decl.(*ast.GenDecl)
			if !isGen || gen.Tok != token.VAR {
				return true
			}
			for _, s := range gen.Specs {
				vs, isVS := s.(*ast.ValueSpec)
				if !isVS || len(vs.Names) == 0 {
					continue
				}
				if sel, isSel := vs.Type.(*ast.SelectorExpr); isSel {
					vars[vs.Names[0].Name] = sel.Sel.Name
				}
			}
		case *ast.AssignStmt:
			for i, rhs := range d.Rhs {
				if i >= len(d.Lhs) {
					continue
				}
				lhs, isIdent := d.Lhs[i].(*ast.Ident)
				if !isIdent {
					continue
				}
				t, ok := structOf(rhs)
				if !ok {
					continue
				}
				if sel, isSel := t.(*ast.SelectorExpr); isSel {
					vars[lhs.Name] = sel.Sel.Name
				}
			}
		}
		return true
	})
	return vars
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return typeName(t.X)
	}
	return ""
}

// selectorChain lists the names of a dotted expression from the field written down to
// the base identifier, so "resp.Scope.NasName" reads as NasName, Scope, resp.
func selectorChain(e ast.Expr) []string {
	var out []string
	for {
		sel, isSel := e.(*ast.SelectorExpr)
		if !isSel {
			if id, isIdent := e.(*ast.Ident); isIdent {
				out = append(out, id.Name)
			}
			return out
		}
		out = append(out, sel.Sel.Name)
		e = sel.X
	}
}

// nameWatch is the set of name-shaped Go fields the card could print, with the
// response struct that owns each one and the field names that point at that struct.
type nameWatch struct {
	// owners maps a name-shaped Go field to its single owning response struct. A
	// field two response types share is dropped rather than guessed, because a wrong
	// attribution is worse than an untraced slot the ledger refuses.
	owners map[string]string
	// parents maps a response struct to the fields of other response structs that hold
	// it, so "resp.Scope.NasName" is recognized as a write to nocImpactScope.
	parents map[string][]string
}

func newWatch(defs map[string]structDef) nameWatch {
	w := nameWatch{owners: map[string]string{}, parents: map[string][]string{}}
	counted := map[string][]string{}
	var names []string
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for _, fld := range defs[name].fields {
			if isNameShape(fld.Key) {
				counted[fld.Go] = append(counted[fld.Go], name)
			}
			target := strings.TrimPrefix(fld.Type, "*")
			if _, holds := defs[target]; holds && target != name {
				w.parents[target] = append(w.parents[target], fld.Go)
			}
		}
	}
	for goName, owners := range counted {
		if len(owners) == 1 {
			w.owners[goName] = owners[0]
		}
	}
	return w
}

// incidentDefs reads every response type an incident surfaces, once.
func incidentDefs(root string) (map[string]structDef, error) {
	defs := map[string]structDef{}
	var missing []string
	for _, src := range incidentSources {
		got := readStructs(root, src.file, map[string]bool{src.name: true})
		def, ok := got[src.name]
		if !ok {
			missing = append(missing, src.file+":"+src.name)
			continue
		}
		defs[src.name] = def
	}
	if len(missing) > 0 {
		return nil, structError{file: strings.Join(missing, ", "), name: "response types"}
	}
	return defs, nil
}

// nameFillers follows every name-shaped slot back to the assignments that fill it.
func nameFillers(root string, tables []identityTable, w nameWatch) ([]filler, error) {
	byStruct := map[string]identityTable{}
	columnOf := map[string]string{}
	for _, t := range tables {
		byStruct[t.Struct] = t
		for key, goField := range t.Keys {
			columnOf[t.Struct+"."+goField] = key
		}
	}
	if len(w.owners) == 0 {
		return nil, structError{file: apiDir, name: "no name-shaped slot is served"}
	}
	var out []filler
	for _, rel := range walkFiles(root, apiDir, ".go") {
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		f, fset, ok := parseGo(root, rel)
		if !ok {
			continue
		}
		vars := collectVars(f)
		add := func(structName, goName string, val ast.Expr, line int) {
			fl := filler{Struct: structName, Go: goName, File: rel, Line: line}
			if sel, isSel := val.(*ast.SelectorExpr); isSel {
				if id, isIdent := sel.X.(*ast.Ident); isIdent {
					model := vars[id.Name]
					t, known := byStruct[model]
					switch {
					case known:
						col := columnOf[model+"."+sel.Sel.Name]
						if col == "" {
							col = sel.Sel.Name
						}
						fl.Table, fl.Column, fl.Traced = t.Table, col, true
						fl.Subscriber = subscriberRows[t.Table]
						fl.Guarded = t.Sensitive[sel.Sel.Name]
					case model != "":
						fl.Table, fl.Column = model, sel.Sel.Name
					}
				}
			}
			out = append(out, fl)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch d := n.(type) {
			case *ast.CompositeLit:
				owner := typeName(d.Type)
				for _, e := range d.Elts {
					kv, isKV := e.(*ast.KeyValueExpr)
					if !isKV {
						continue
					}
					key, isIdent := kv.Key.(*ast.Ident)
					if !isIdent {
						continue
					}
					if s, tracked := w.owners[key.Name]; tracked && s == owner {
						add(s, key.Name, kv.Value, fset.Position(kv.Pos()).Line)
					}
				}
			case *ast.AssignStmt:
				if len(d.Lhs) != 1 || len(d.Rhs) != 1 {
					return true
				}
				chain := selectorChain(d.Lhs[0])
				if len(chain) < 2 {
					return true
				}
				s, tracked := w.owners[chain[0]]
				if !tracked {
					return true
				}
				for _, name := range chain[1:] {
					for _, parent := range w.parents[s] {
						if name == parent {
							add(s, chain[0], d.Rhs[0], fset.Position(d.Pos()).Line)
						}
					}
				}
			}
			return true
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Struct != out[j].Struct {
			return out[i].Struct < out[j].Struct
		}
		if out[i].Go != out[j].Go {
			return out[i].Go < out[j].Go
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	if len(out) == 0 {
		return nil, structError{file: apiDir, name: "no assignment fills a name-shaped slot"}
	}
	return out, nil
}

// fillsFor returns the assignments recorded for one slot.
func fillsFor(fills []filler, structName, goName string) []filler {
	var out []filler
	for _, fl := range fills {
		if fl.Struct == structName && fl.Go == goName {
			out = append(out, fl)
		}
	}
	return out
}

// nameVerdict is the criterion's name clause applied to one slot, with the evidence
// the memo prints next to it: the assignment that decides the answer, or the sweep
// that found none.
type nameVerdict struct {
	OK   bool
	Why  string
	Held string
	// Guarded reports whether the shipped rules already name the column behind the
	// refusing assignment.
	Guarded bool
}

// nameDecision decides one name-shaped slot: it may print only when every assignment
// the handlers perform for it reads a persisted row that is not a subscriber row.
// Anything else - a subscriber row, a value carried in from another response object,
// a name no handler accounts for - stays off the sheet.
func nameDecision(fills []filler) nameVerdict {
	if len(fills) == 0 {
		return nameVerdict{
			Held: apiDir,
			Why:  "no assignment in the served handlers accounts for this name",
		}
	}
	for _, fl := range fills {
		if fl.Subscriber {
			return nameVerdict{Held: fl.where(), Guarded: fl.Guarded,
				Why: "filled from " + fl.source() + " at " + fl.where()}
		}
		if !fl.Traced {
			return nameVerdict{Held: fl.where(), Guarded: fl.Guarded,
				Why: "filled from " + fl.source() + " at " + fl.where() +
					", which is not a row this study can read"}
		}
	}
	var srcs []string
	seen := map[string]bool{}
	for _, fl := range fills {
		if seen[fl.source()] {
			continue
		}
		seen[fl.source()] = true
		srcs = append(srcs, fl.source())
	}
	sort.Strings(srcs)
	return nameVerdict{OK: true,
		Why: "name-shaped; every assignment reads " + strings.Join(srcs, ", ")}
}
