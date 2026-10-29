package opstats

import (
	"sort"
	"strings"
)

// surfaceSpec is one statistics surface the console already serves. The route, the
// handler and the served type are all checked against the tree at run time: a spec
// whose route or type cannot be found is reported as broken, so the table cannot
// quietly shrink.
type surfaceSpec struct {
	route string
	// RouteFile is where the path is registered. It is the handler's own file unless the
	// server wires the route from a sibling file.
	routeFile string
	handler   string
	file      string
	typeFile  string
	typeName  string
	// Audience is who the response is served to. "tenant-operator" means the values
	// stay inside the tenant that owns them; "platform-admin" is the only surface
	// class that can see across tenants.
	audience string
}

func surfaceSpecs() []surfaceSpec {
	return []surfaceSpec{
		{route: "/dashboard/stats", handler: "GetDashboardStats", file: "internal/adminapi/dashboard.go", typeName: "DashboardStats", audience: "tenant-operator"},
		{route: "/dashboard/isp-stats", handler: "getISPDashboardStats", file: "internal/adminapi/isp.go", typeName: "ispDashboardStats", audience: "tenant-operator"},
		{route: "/network/impact", routeFile: "internal/adminapi/network_monitor.go", handler: "getNetworkImpact", file: "internal/adminapi/noc_impact.go", typeName: "nocImpactCounts", audience: "tenant-operator"},
		{route: "/network/monitor-targets", handler: "listMonitorTargets", file: "internal/adminapi/network_monitor.go", typeName: "monitorTargetDTO", audience: "tenant-operator"},
		{route: "/network/monitor-targets/:id/traffic", handler: "getMonitorTargetTraffic", file: "internal/adminapi/telemetry.go", typeName: "TrafficSeriesDTO", audience: "tenant-operator"},
		{route: "/radius/traffic/user/:username", handler: "getUserTraffic", file: "internal/adminapi/telemetry.go", typeName: "TrafficSeriesDTO", audience: "tenant-operator"},
		{route: "/network/ipam/pools", handler: "listIPAMPools", file: "internal/adminapi/isp_operations.go", typeFile: "internal/domain/isp_operations.go", typeName: "IPAMPool", audience: "tenant-operator"},
		{route: "/network/diagnostics/flapping", handler: "detectFlappingSubscribers", file: "internal/adminapi/isp_operations.go", typeName: "flappingSubscriber", audience: "tenant-operator"},
		{route: "/isp/customers/:id/timeline", routeFile: "internal/adminapi/isp.go", handler: "getCustomerTimeline", file: "internal/adminapi/customer_timeline.go", typeName: "customerTimelineResponse", audience: "tenant-operator"},
		{route: "/platform/tenants", handler: "listPlatformTenants", file: "internal/adminapi/platform_tenants.go", typeFile: "internal/domain/tenant.go", typeName: "Tenant", audience: "platform-admin"},
		{route: "/system/backup", handler: "backupSystem", file: "internal/adminapi/system_backup.go", typeFile: "internal/adminapi/backup_envelope.go", typeName: "backupManifest", audience: "platform-admin"},
	}
}

// fieldRow is one numeric-or-labeled slot of a served statistics response, with
// the statement that produces it.
type fieldRow struct {
	Surface  string
	Audience string
	Go       string
	Key      string
	Type     string
	// Shape is scalar, series or map: how many numbers one response carries under
	// this key.
	Shape string
	// Measure is what the number counts: count, money, bytes, rate, volume, label,
	// flag, id, time or other.
	Measure string
	Verb    string
	Column  string
	Table   string
	// Cell is the rule that says how many stored rows one published value covers.
	Cell string
	// Ratio names the two numbers a rate field divides, for a rate only.
	Ratio  string
	File   string
	Line   int
	Proven string
}

// labelKeys are the sibling keys that make a published number a bucket rather than
// a tenant total. The first one present in the same struct decides the cell, in
// this order.
var labelKeys = []string{"username", "customer_no", "profile_name", "date", "hour", "table"}

// moneyColumns and bytesColumns are the study's two interpretive lists: they decide
// whether a SUM publishes an amount of money or a volume of traffic. Each entry is
// pinned to a domain column declaration by a claim in this run.
var moneyColumns = []string{"balance", "total", "subtotal", "paid_amount", "amount", "unit_price", "price"}

var bytesColumns = []string{"acct_input_total", "acct_output_total", "total_in_bytes", "total_out_bytes"}

// modelTables maps the domain model named in the producing statement to the table it
// persists, which is what lets the fixture count rows behind a number.
var modelTables = map[string]string{
	"Customer":         "isp_customer",
	"Subscription":     "isp_subscription",
	"RadiusUser":       "radius_user",
	"RadiusOnline":     "radius_online",
	"RadiusAccounting": "radius_accounting",
	"RadiusProfile":    "radius_profile",
	"Invoice":          "isp_invoice",
	"Payment":          "isp_payment",
	"TroubleTicket":    "isp_trouble_ticket",
	"IPAMPool":         "isp_ipam_pool",
	"Tenant":           "tenant",
}

// columnTables maps an aggregated column back to its table when the statement does
// not name a model.
var columnTables = map[string]string{
	"balance":           "isp_invoice",
	"amount":            "isp_payment",
	"acct_input_total":  "radius_accounting",
	"acct_output_total": "radius_accounting",
	"disconnect_count":  "radius_accounting",
}

// readSurfaces expands every spec into its field rows.
func readSurfaces(root string) ([]fieldRow, []string) {
	var out []fieldRow
	var broken []string
	for _, sp := range surfaceSpecs() {
		tf := sp.typeFile
		if tf == "" {
			tf = sp.file
		}
		rf := sp.routeFile
		if rf == "" {
			rf = sp.file
		}
		if firstLine(root, rf, `"`+sp.route+`"`) == 0 {
			broken = append(broken, sp.route+": route not registered in "+rf)
		}
		if firstLine(root, sp.file, "func "+sp.handler+"(") == 0 {
			broken = append(broken, sp.route+": handler not found in "+sp.file)
		}
		fields, found := structFields(root, tf, sp.typeName)
		if !found {
			broken = append(broken, sp.route+": type "+sp.typeName+" not found in "+tf)
			continue
		}
		rows := expand(root, sp, tf, sp.typeName, "", fields, 0)
		// A response that serves a domain model serves the persisted row: every column
		// it prints is that row's own stored value, with no aggregate verb between them.
		if table, ok := modelTables[sp.typeName]; ok && strings.HasPrefix(tf, "internal/domain/") {
			for i := range rows {
				rows[i].Verb = "SELECT"
				rows[i].Column = lastKey(rows[i].Key)
				rows[i].Table = table
				rows[i].Cell = "row"
				rows[i].Proven = "persisted column"
			}
		}
		out = append(out, rows...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Surface != out[j].Surface {
			return out[i].Surface < out[j].Surface
		}
		return out[i].Key < out[j].Key
	})
	return out, broken
}

// expand walks one served type and emits a row for every leaf. A struct or slice of
// structs is opened, so a nested number is measured exactly like a top-level one;
// the container itself is not counted, which keeps the totals honest.
func expand(root string, sp surfaceSpec, file, typeName, prefix string, fields []declField, depth int) []fieldRow {
	if depth > 2 {
		return nil
	}
	cell := cellOf(fields)
	var out []fieldRow
	for _, f := range fields {
		key := f.JSON
		if key == "" {
			key = strings.ToLower(f.Name)
		}
		if prefix != "" {
			key = prefix + "." + key
		}
		inner := strings.TrimPrefix(strings.TrimPrefix(f.Type, "[]"), "*")
		if child, ok := structFields(root, file, inner); ok && !isBuiltin(inner) {
			shape := "series"
			if strings.HasPrefix(f.Type, "map[") {
				shape = "map"
			} else if !strings.HasPrefix(f.Type, "[]") {
				shape = "scalar"
			}
			for _, sub := range expand(root, sp, file, inner, key, child, depth+1) {
				sub.Shape = shape
				sub.Cell = cellOf(child)
				out = append(out, sub)
			}
			continue
		}
		shape := "scalar"
		if strings.HasPrefix(f.Type, "[]") {
			shape = "series"
		} else if strings.HasPrefix(f.Type, "map[") {
			shape = "map"
		}
		row := fieldRow{
			Surface:  sp.route,
			Audience: sp.audience,
			Go:       f.Name,
			Key:      key,
			Type:     f.Type,
			Shape:    shape,
			Measure:  measureOf(key, f.Type),
			File:     file,
			Line:     f.Line,
		}
		if shape == "map" {
			row.Cell = "map-key"
		} else {
			row.Cell = cell
		}
		out = append(out, row)
	}
	return out
}

// cellOf applies the bucket rule: the sibling label key carried next to the number,
// or the tenant as a whole when none is present.
func cellOf(fields []declField) string {
	present := map[string]bool{}
	for _, f := range fields {
		present[f.JSON] = true
	}
	for _, l := range labelKeys {
		if present[l] {
			return l
		}
	}
	return "tenant"
}

func isBuiltin(name string) bool {
	switch name {
	case "int", "int64", "float64", "string", "bool", "any", "byte", "Time", "unknown":
		return true
	}
	return false
}

func measureOf(key, typ string) string {
	last := key
	if i := strings.LastIndex(key, "."); i >= 0 {
		last = key[i+1:]
	}
	switch {
	case typ == "bool":
		return "flag"
	case typ == "string":
		if strings.HasSuffix(last, "_at") || last == "date" || last == "hour" || last == "timestamp" {
			return "time"
		}
		return "label"
	case strings.HasSuffix(last, "_gb"), strings.HasSuffix(last, "_bytes"):
		return "bytes"
	case strings.HasSuffix(last, "_percent"), strings.HasSuffix(last, "_rate"), last == "adoption_rate":
		return "rate"
	case strings.HasSuffix(last, "_id"), last == "id":
		return "id"
	case typ == "int", typ == "int64", typ == "float64":
		return "count"
	default:
		return "other"
	}
}

// attribute finds the statement that produces each number field, in the file the
// handler lives in. The rule is mechanical and reported per row: a line naming the
// field with an aggregate verb on it, an upward search from the assignment, or a
// bounded walk through the file-local helpers the assignment calls.
//
// A ratio is deliberately not traced to a query. What makes it dangerous is the pair it
// divides, so the run reads the division and names both sides by the served key behind
// them instead of borrowing a neighbour's statement.
func attribute(root string, rows []fieldRow) {
	for i := range rows {
		f := &rows[i]
		if f.Measure == "label" || f.Measure == "flag" || f.Measure == "id" || f.Measure == "time" || f.Measure == "other" {
			continue
		}
		if f.Measure == "rate" {
			line, ratio := ratioOf(root, f.File, f.Surface, rows, f.Go)
			if ratio == "" {
				f.Proven = "composition"
				continue
			}
			f.Line, f.Ratio, f.Proven = line, ratio, "ratio read from its own assignment"
			continue
		}
		a := produce(root, f.File, f.Go)
		f.Verb, f.Column, f.Proven, f.Line = a.verb, a.column, a.proven, a.line
		f.Measure = refineMeasure(f.Measure, a.verb, a.column)
		switch {
		case modelTables[a.model] != "":
			f.Table = modelTables[a.model]
		case columnTables[f.Column] != "":
			f.Table = columnTables[f.Column]
		case columnTables[lastKey(f.Key)] != "":
			f.Table = columnTables[lastKey(f.Key)]
		case a.Table != "":
			f.Table = a.Table
		}
	}
}

// refineMeasure lets the producing statement, not the JSON key, say what a number is:
// a SUM over an amount column publishes money even when the key reads like a count.
func refineMeasure(measure, verb, column string) string {
	if measure != "count" || (verb != "SUM" && verb != "AVG") {
		return measure
	}
	if contains(moneyColumns, column) {
		return "money"
	}
	if contains(bytesColumns, column) {
		return "bytes"
	}
	return measure
}

func lastKey(key string) string {
	if i := strings.LastIndex(key, "."); i >= 0 {
		return key[i+1:]
	}
	return key
}

// statement is the code an aggregate is attributed to.
type statement struct {
	verb, column, model string
	// Table is the table a chained statement names when it queries without a domain
	// model, which is where the tenant callback cannot reach.
	Table  string
	proven string
	line   int
}

// produce applies the attribution rule for one field.
func produce(root, file, goName string) statement {
	src := lines(root, file)
	if len(src) == 0 {
		return statement{proven: "vacated"}
	}
	for i, ln := range src {
		if !namesField(ln, goName) {
			continue
		}
		if v, col, mdl := verbsIn(ln); v != "" {
			return resolve(root, file, src, i, v, col, mdl, "same line")
		}
	}
	// The field is written from a value computed elsewhere: look back from the
	// assignment for the statement, then follow a file-local helper if one is called.
	for i, ln := range src {
		if !namesField(ln, goName) {
			continue
		}
		for back := i; back > 0 && i-back < 14; back-- {
			if v, col, mdl := verbsIn(src[back-1]); v != "" {
				return resolve(root, file, src, back-1, v, col, mdl, "above line")
			}
		}
		near := modelAbove(src, i)
		for _, callee := range calledFuncs(ln) {
			if v, col, mdl, depth := follow(root, file, callee, 1, near); v != "" {
				st := resolve(root, file, src, i, v, col, mdl, "via "+callee+strings.Repeat("+", depth-1))
				st.line = i + 1
				if st.Table == "" && st.model == "" {
					st.Table = calleeTable(root, file, callee)
				}
				return st
			}
		}
		return statement{verb: "composed", proven: "assignment only", line: i + 1}
	}
	return statement{proven: "not found"}
}

// resolve finishes an attribution found on one line. A statement that names no domain
// model still names its tables, and a chain built by a file-local helper names them
// there rather than at the call site.
func resolve(root, file string, src []string, at int, verb, column, model, proven string) statement {
	st := statement{verb: verb, column: column, model: model, proven: proven, line: at + 1}
	if model != "" {
		return st
	}
	lo, hi := chain(src, at)
	primary, aliases := statementTables(src[lo : hi+1])
	if primary == "" {
		for _, callee := range calledFuncs(src[at]) {
			if t, a := calleeTables(root, file, callee); t != "" {
				primary, aliases = t, a
				st.proven = proven + ", chain built in " + callee
				break
			}
		}
	}
	if primary == "" && st.model == "" {
		// A counter written through a file-local query closure names its model a few lines
		// above the call, not on it.
		if m := modelAbove(src, at+1); m != "" {
			st.model = m
			st.proven = proven + ", model from the query built above"
		}
	}
	if alias := distinctAlias(src[at]); alias != "" {
		if t, ok := aliases[alias]; ok && t != "" {
			primary = t
		}
	}
	st.Table = primary
	return st
}

// chain gives the span of the chained statement one line belongs to: itself plus the
// continuation lines above it, which is how a GORM query written one method per line is
// read as one statement.
func chain(src []string, at int) (lo, hi int) {
	lo = at
	for lo > 0 && strings.HasSuffix(strings.TrimSpace(src[lo-1]), ".") {
		lo--
	}
	return lo, at
}

// calleeTables reads the tables named inside a file-local helper that builds the query.
func calleeTables(root, file, name string) (string, map[string]string) {
	body := funcBody(root, file, name)
	if len(body) == 0 {
		return "", nil
	}
	return statementTables(body)
}

func calleeTable(root, file, name string) string {
	t, _ := calleeTables(root, file, name)
	return t
}

// statementTables reads the table literals of a chained statement: the table it counts
// from and the alias each join binds.
func statementTables(block []string) (primary string, aliases map[string]string) {
	aliases = map[string]string{}
	for _, ln := range block {
		for _, lit := range quotedAfter(ln, "Table(") {
			tbl, alias := parseTableLit(lit)
			if tbl != "" && primary == "" {
				primary = tbl
			}
			if alias != "" {
				aliases[alias] = tbl
			}
		}
		for _, lit := range quotedAfter(ln, "Joins(") {
			tbl, alias := parseTableLit(lit)
			if alias != "" {
				aliases[alias] = tbl
			}
		}
	}
	return primary, aliases
}

// parseTableLit reads "isp_subscription AS s" into its table and alias.
func parseTableLit(s string) (table, alias string) {
	fields := strings.Fields(s)
	for len(fields) > 0 && isJoinWord(strings.ToUpper(fields[0])) {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return "", ""
	}
	table = strings.ToLower(fields[0])
	for i, f := range fields {
		if strings.EqualFold(f, "AS") && i+1 < len(fields) {
			alias = fields[i+1]
		}
	}
	return table, alias
}

func isJoinWord(w string) bool {
	switch w {
	case "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "CROSS":
		return true
	}
	return false
}

// distinctAlias names the alias a DISTINCT counts over, which is the row set a published
// count really covers when the query joins several tables.
func distinctAlias(ln string) string {
	lits := quotedAfter(ln, "Distinct(")
	if len(lits) == 0 {
		return ""
	}
	head := strings.Fields(lits[0])
	if len(head) == 0 {
		return ""
	}
	if i := strings.Index(head[0], "."); i > 0 {
		return head[0][:i]
	}
	return ""
}

// quotedAfter returns every string literal written directly after one of the call
// markers on a line.
func quotedAfter(ln, marker string) []string {
	var out []string
	for i := 0; i < len(ln); i++ {
		if !strings.HasPrefix(ln[i:], marker) {
			continue
		}
		j := i + len(marker)
		if j >= len(ln) || ln[j] != '"' {
			i = j - 1
			continue
		}
		k := strings.IndexByte(ln[j+1:], '"')
		if k < 0 {
			break
		}
		out = append(out, ln[j+1:j+1+k])
		i = j + k
	}
	return out
}

// namesField says a line writes or reads this field, rather than merely carrying an
// identifier that happens to share its name. Without the selector or the literal colon,
// a field called Count would be attributed to every .Count call in the file.
func namesField(ln, goName string) bool {
	return namesSelector(ln, goName) || strings.Contains(ln, goName+":")
}

// namesSelector finds a selector that reads or writes the field. A selector followed by
// an open paren is a method call, not a struct slot, so a field named Count stays out of
// the census even in a file full of .Count(...) calls.
func namesSelector(ln, goName string) bool {
	for at := 0; ; {
		i := strings.Index(ln[at:], "."+goName)
		if i < 0 {
			return false
		}
		j := at + i + 1 + len(goName)
		if j >= len(ln) || ln[j] != '(' {
			return true
		}
		at = j
	}
}

// modelAbove returns the domain model named by the nearest statement above the line,
// which is how a helper that only takes a *DB still gets a table.
func modelAbove(src []string, at int) string {
	for back := at; back > 0 && at-back < 14; back-- {
		if m := modelIn(src[back-1]); m != "" {
			return m
		}
	}
	return ""
}

// follow searches a file-local function body for an aggregate verb, descending at
// most three hops so a helper that calls a helper is still read. When the verb line
// names no model, the body's own or the caller's is borrowed and reported as such.
func follow(root, file, name string, depth int, fallbackModel string) (verb, column, model string, hops int) {
	if depth > 3 {
		return "", "", "", 0
	}
	body := funcBody(root, file, name)
	own := ""
	for _, ln := range body {
		if m := modelIn(ln); m != "" {
			own = m
			break
		}
	}
	for _, ln := range body {
		if v, col, mdl := verbsIn(ln); v != "" {
			if mdl == "" {
				mdl = own
			}
			if mdl == "" {
				mdl = fallbackModel
			}
			return v, col, mdl, depth
		}
	}
	for _, ln := range body {
		for _, callee := range calledFuncs(ln) {
			if callee == name {
				continue
			}
			if v, col, mdl, hops := follow(root, file, callee, depth+1, fallbackModel); v != "" {
				return v, col, mdl, hops
			}
		}
	}
	return "", "", "", 0
}

// funcBody returns the code lines of a file-local function, or nil.
func funcBody(root, file, name string) []string {
	src := lines(root, file)
	lo, hi := funcRange(src, name)
	if lo < 0 {
		return nil
	}
	return src[lo:hi]
}

// verbsIn reads the aggregate verbs and their column and model out of one line.
func verbsIn(ln string) (verb, column, model string) {
	if i := strings.Index(ln, "COUNT("); i >= 0 {
		verb = "COUNT"
		column = inner(ln, i+len("COUNT("))
	}
	if i := strings.Index(ln, "SUM("); i >= 0 {
		if verb == "" {
			verb = "SUM"
		}
		column = inner(ln, i+len("SUM("))
	}
	if i := strings.Index(ln, "AVG("); i >= 0 {
		if verb == "" {
			verb = "AVG"
		}
		column = inner(ln, i+len("AVG("))
	}
	if strings.Contains(ln, ".Count(") || strings.Contains(ln, "Count(&") {
		if verb == "" {
			verb = "COUNT"
		}
	}
	if verb == "" {
		return "", "", ""
	}
	return verb, cleanColumn(column), modelIn(ln)
}

// modelIn reads the domain model a statement queries, or the empty string.
func modelIn(ln string) string {
	j := strings.Index(ln, "Model(&domain.")
	if j < 0 {
		return ""
	}
	rest := ln[j+len("Model(&domain."):]
	k := strings.IndexAny(rest, "{.")
	if k < 0 {
		return ""
	}
	return strings.TrimSuffix(rest[:k], "}")
}

// inner reads the argument of an aggregate call up to the closing parenthesis, and
// keeps the identifier a CASE expression finally sums or counts.
func inner(s string, from int) string {
	depth := 0
	var b []rune
	for _, r := range s[from:] {
		switch r {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return string(b)
			}
			depth--
		case ',':
			if depth == 0 {
				return string(b)
			}
		}
		b = append(b, r)
	}
	return string(b)
}

func cleanColumn(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return s
	}
	if i := strings.LastIndex(s, "THEN "); i >= 0 {
		s = s[i+len("THEN "):]
	}
	if i := strings.LastIndex(s, "then "); i >= 0 {
		s = s[i+len("then "):]
	}
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ')' || r == '(' || r == ',' })
	for i := len(fields) - 1; i >= 0; i-- {
		c := strings.ToLower(fields[i])
		for _, known := range append(append([]string{}, moneyColumns...), bytesColumns...) {
			if c == known {
				return known
			}
		}
	}
	if len(fields) > 0 {
		return strings.ToLower(strings.Trim(fields[len(fields)-1], "`\""))
	}
	return ""
}

// calledFuncs lists the identifiers called on a line, in order of appearance.
func calledFuncs(ln string) []string {
	var out []string
	for i := 0; i < len(ln); i++ {
		if ln[i] != '(' {
			continue
		}
		j := i
		for j > 0 && (isIdentRune(rune(ln[j-1])) || ln[j-1] == '.') {
			j--
		}
		name := ln[j:i]
		if name == "" || !isIdent(name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

// aggregateRows keeps the rows that publish a number a reader can act on.
func aggregateRows(rows []fieldRow) []fieldRow {
	var out []fieldRow
	for _, r := range rows {
		switch r.Measure {
		case "count", "money", "bytes", "rate":
			out = append(out, r)
		}
	}
	return out
}

// sortedTables is the fixture's table list in a stable order.
func sortedTables(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// unresolvedOperand marks a side of a rate that no served key is found for, so the
// finding says what it could not read rather than guessing.
const unresolvedOperand = "unserved:"

// ratioOf reads the division that composes a rate field and names both sides by the key
// the same response serves them under. A percentage is not safer than the count it
// divides when that count travels beside it, and a rate served with one operand hands
// the reader the other.
func ratioOf(root, file, surface string, rows []fieldRow, goName string) (int, string) {
	src := lines(root, file)
	for i, ln := range src {
		at := strings.Index(ln, "."+goName+" =")
		if at < 0 {
			continue
		}
		expr := ln[at+len(goName)+3:]
		slash := strings.Index(expr, "/")
		if slash < 0 {
			return i + 1, ""
		}
		num := servedKey(rows, surface, lastOperand(expr[:slash]), src)
		den := servedKey(rows, surface, lastOperand(expr[slash+1:]), src)
		return i + 1, num + " / " + den
	}
	return 0, ""
}

// servedKey names the operand by the response key it publishes, following a helper's
// parameter back to the argument the same file passes into it.
func servedKey(rows []fieldRow, surface, ident string, src []string) string {
	if ident == "" {
		return unresolvedOperand + "?"
	}
	for _, r := range rows {
		if r.Surface == surface && r.Go == ident {
			return r.Key
		}
	}
	if from := paramOrigin(src, ident); from != "" {
		for _, r := range rows {
			if r.Surface == surface && r.Go == from {
				return r.Key
			}
		}
	}
	return unresolvedOperand + ident
}

// lastOperand takes the identifier an expression side names - the field inside a
// float64(stats.OnlineWithIPv6) cast, not the cast or the *100 that scales it.
func lastOperand(expr string) string {
	var last string
	for _, tok := range identTokens(expr) {
		if !isNumeric(tok) {
			last = tok
		}
	}
	return last
}

// paramOrigin follows a function parameter to the argument the same file passes into it,
// so a denominator carried under a local name is reported as the served field behind it.
func paramOrigin(src []string, ident string) string {
	for _, ln := range src {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, "func ") {
			continue
		}
		params := argList(t[strings.IndexByte(t, '('):])
		idx := paramIndex(params, ident)
		if idx < 0 {
			continue
		}
		name := t[len("func "):strings.IndexByte(t, '(')]
		for _, call := range src {
			ct := strings.TrimSpace(call)
			if strings.HasPrefix(ct, "func ") || !strings.Contains(ct, name+"(") {
				continue
			}
			args := argList(ct[strings.Index(ct, name+"("):])
			if idx < len(args) {
				if a := lastOperand(args[idx]); a != "" {
					return a
				}
			}
		}
	}
	return ""
}

// argList splits the comma-separated list inside the first balanced pair of parentheses.
func argList(s string) []string {
	open := strings.IndexByte(s, '(')
	if open < 0 {
		return nil
	}
	var out []string
	depth, cur := 1, open+1
	for i := open + 1; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 {
				return append(out, s[cur:i])
			}
		case ',':
			if depth == 1 {
				out = append(out, s[cur:i])
				cur = i + 1
			}
		}
	}
	return out
}

// paramIndex locates one named parameter in a declaration list.
func paramIndex(params []string, ident string) int {
	for i, p := range params {
		if f := strings.Fields(p); len(f) >= 2 && f[0] == ident {
			return i
		}
	}
	return -1
}

func identTokens(s string) []string {
	var out []string
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if !isIdentRune(runes[i]) {
			i++
			continue
		}
		j := i
		for j < len(runes) && isIdentRune(runes[j]) {
			j++
		}
		out = append(out, string(runes[i:j]))
		i = j
	}
	return out
}

func isNumeric(tok string) bool {
	return tok != "" && tok[0] >= '0' && tok[0] <= '9'
}
