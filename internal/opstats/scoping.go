package opstats

import (
	"sort"
	"strings"
)

// A statistic is only private while the query that computes it stays inside the tenant
// that owns the rows. This file reads how each surface reaches the database, because the
// criterion's promise is a property of the query, not of the response shape.
//
// scopeRow is one surface's route to the data.
type scopeRow struct {
	Surface string
	// Class is callback (the ORM injects the tenant predicate), explicit (a written
	// tenant_id comparison), fallback (a default tenant stands in when no context is
	// present), or platform (the row is meant to cross tenants).
	Class    string
	Evidence string
	// ModelQueries counts statements that go through a domain model, where the shipped
	// callback can reach. RawQueries counts Table/Joins/Raw uses the callback cannot see.
	ModelQueries int
	RawQueries   int
}

// scopeTerms are the code markers that decide a surface's class, in decision order. The
// weakest evidence wins: a handler that can fall back to the default tenant is reported
// as a fallback surface even though it also calls the tenant-aware getter, because the
// fallback is the branch that decides whose rows the number covers.
var scopeTerms = []struct {
	term  string
	class string
	rank  int
}{
	{term: "context.Background()", class: "platform", rank: 4},
	{term: "DefaultTenantID", class: "fallback", rank: 3},
	{term: "tenancy.TenantID(", class: "explicit", rank: 2},
	{term: "tenant_id = ?", class: "explicit", rank: 2},
	{term: "GetDB(c)", class: "callback", rank: 1},
}

// readScoping classifies every surface from the code the handler can actually execute:
// its own body plus the file-local functions it calls, one hop deep. The whole file is
// not the unit - a handler file can hold a dozen routes, and a fallback three functions
// away belongs to a different surface.
func readScoping(root string) []scopeRow {
	specs := surfaceSpecs()
	out := make([]scopeRow, 0, len(specs))
	for _, sp := range specs {
		row := scopeRow{Surface: sp.route, Class: "unattributed"}
		row.ModelQueries = countMatches(root, sp.file, ".Model(&domain.")
		row.RawQueries = countMatches(root, sp.file, ".Table(")
		src := lines(root, sp.file)
		reach := handlerReach(src, sp.handler)
		bestRank := -1
		for _, t := range scopeTerms {
			ln := firstReachable(src, reach, t.term)
			if ln == 0 {
				continue
			}
			if sp.audience == "platform-admin" && t.class == "platform" {
				row.Class = t.class
				row.Evidence = sp.file + ":" + itoa(ln)
				break
			}
			if t.rank > bestRank {
				bestRank = t.rank
				row.Class = t.class
				row.Evidence = sp.file + ":" + itoa(ln)
			}
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Surface < out[j].Surface })
	return out
}

// handlerReach returns the 1-based lines one handler can run: its body and the bodies of
// the file-local functions called from it.
func handlerReach(src []string, handler string) []int {
	lo, hi := funcRange(src, handler)
	if lo < 0 {
		return nil
	}
	out := make([]int, 0, hi-lo)
	for i := lo; i < hi; i++ {
		out = append(out, i+1)
	}
	for _, i := range append([]int(nil), out...) {
		for _, callee := range calledFuncs(src[i-1]) {
			cLo, cHi := funcRange(src, callee)
			if cLo < 0 {
				continue
			}
			for j := cLo; j < cHi; j++ {
				out = append(out, j+1)
			}
		}
	}
	return out
}

// firstReachable gives the first line of the reachable set carrying the token, or 0.
func firstReachable(src []string, reach []int, tok string) int {
	best := 0
	for _, i := range reach {
		if i > len(src) {
			continue
		}
		if !containsBounded(src[i-1], tok) {
			continue
		}
		if best == 0 || i < best {
			best = i
		}
	}
	return best
}

// funcRange returns the half-open, 0-based body range of a file-local function.
func funcRange(src []string, name string) (lo, hi int) {
	lo = -1
	for i, ln := range src {
		if strings.Contains(ln, "func "+name+"(") {
			lo = i + 1
			break
		}
	}
	if lo < 0 {
		return -1, -1
	}
	for i := lo; i < len(src); i++ {
		if strings.HasPrefix(src[i], "}") && strings.TrimLeft(src[i], " \t") == "}" {
			return lo, i
		}
	}
	return lo, len(src)
}

// crossTenantServed counts the aggregate fields on a surface that is not tenant-scoped,
// which is the number the criterion turns on: a value computed from one tenant's rows
// that a different tenant can read.
func crossTenantServed(exps []exposure, scope map[string]scopeRow) []exposure {
	var out []exposure
	for _, e := range exps {
		s, ok := scope[e.Surface]
		if !ok {
			continue
		}
		if s.Class == "platform" {
			out = append(out, e)
		}
	}
	return out
}
