package profilewhatif

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bjo163/mwx-isp/internal/domain"
)

// Markdown renders one run as the tables the memo splices in. Every number here is
// produced by this run; the memo adds no table of its own.
func Markdown(r *Result) string {
	var b strings.Builder
	section(&b, "The measured numbers")
	dump(&b, totalsRows(r.Totals))
	b.WriteString("\nreproducibility digest: `" + r.Digest + "`\n")

	section(&b, "The fixture profiles")
	dump(&b, profileRows())

	section(&b, "The fixture subscribers")
	dump(&b, userRows())

	section(&b, "What one Access-Accept says before any edit")
	dump(&b, attrRows(r.Producers, true))

	section(&b, "What the ledger refuses to diff")
	dump(&b, attrRows(r.HeldOut, false))
	if len(r.ConfigReads) > 0 {
		b.WriteString("\nThe config reads found in the enhancer source:\n\n")
		dump(&b, rows([]string{"config key", "source default", "wire attribute", "emitted", "equals default"},
			func(i int) []string {
				c := r.ConfigReads[i]
				return []string{c.Symbol, c.Default, c.Key, c.Observed, yesNo(c.Matched)}
			}, len(r.ConfigReads)))
	}

	section(&b, "Which wire attributes move when exactly one profile column changes")
	dump(&b, rows([]string{"profile column", "NAS vendor code", "attributes that moved"},
		func(i int) []string {
			e := r.Effects[i]
			return []string{e.Field, e.Vendor, strings.Join(e.Keys, ", ")}
		}, len(r.Effects)))

	section(&b, "The scenarios, one diff each")
	dump(&b, outcomeRows(r.Outcomes))

	section(&b, "The statements, pinned to code")
	dump(&b, claimRows(r.Claims))

	section(&b, "What this study does not close")
	dump(&b, gapRows(r.Gaps))

	section(&b, "The import set this package builds on")
	dump(&b, importRows(r))
	return b.String()
}

func section(b *strings.Builder, title string) {
	b.WriteString("\n## " + title + "\n\n")
}

func dump(b *strings.Builder, rows [][]string) {
	if len(rows) == 0 {
		b.WriteString("(no rows)\n")
		return
	}
	for i, ln := range rows {
		b.WriteString("| " + strings.Join(ln, " | ") + " |")
		if i == 0 {
			b.WriteString("\n|")
			for range ln {
				b.WriteString(" --- |")
			}
		}
		b.WriteString("\n")
	}
}

func totalsRows(t Totals) [][]string {
	pairs := [][2]string{
		{"fixture subscribers", itoa(t.Users)},
		{"fixture profiles", itoa(t.Profiles)},
		{"NAS vendors probed", itoa(t.Vendors)},
		{"profile columns", itoa(t.Fields)},
		{"one-column probes run", itoa(t.Probes)},
		{"probes that moved nothing", itoa(t.SilentProbes)},
		{"column/vendor pairs that moved something", itoa(t.Effects)},
		{"scenarios", itoa(t.Scenarios)},
		{"scenarios whose ledger moved", itoa(t.ScenariosMoved)},
		{"attribute lines across all scenario diffs", itoa(t.Changes)},
		{"profile lookups total", itoa(t.Lookups)},
		{"profile lookups in one enhancement pass", itoa(t.MaxLookups)},
		{"stable attributes in the baseline accept", itoa(t.StableAttrs)},
		{"attribute keys held out", itoa(t.HeldOutKeys)},
		{"runtime config reads found in the enhancer", itoa(t.ConfigReads)},
		{"of those, emitted exactly the source default", itoa(t.ConfigMatched)},
		{"vendor dictionary entries scanned", itoa(t.DictEntries)},
		{"vendor dictionary sources scanned", itoa(t.DictSources)},
		{"standard attribute names probed", itoa(t.StdNames)},
		{"enhancers registered in the product", itoa(t.EnhancersSeen)},
		{"enhancers this study ran", itoa(t.EnhancersRun)},
		{"columns read by a checker and by no enhancer", itoa(t.CheckerOnly)},
		{"imports of this package", itoa(t.Imports)},
		{"imports on the ban list", itoa(t.BannedHits)},
		{"client calls into the wire library", itoa(t.SendCalls)},
		{"package that owns CoA and Disconnect", t.CoAPackage},
		{"rate-asserting enhancer tests with no cache", itoa(t.NoCacheTests)},
		{"statements checked", itoa(t.Claims)},
		{"statements confirmed", itoa(t.ClaimsConfirmed)},
		{"gaps registered", itoa(t.Gaps)},
		{"gaps still open", itoa(t.GapsOpen)},
	}
	out := [][]string{{"measure", "value"}}
	for _, p := range pairs {
		out = append(out, []string{p[0], p[1]})
	}
	return out
}

func profileRows() [][]string {
	out := [][]string{{"profile", "column set as configured"}}
	ids := make([]int64, 0)
	for id := range Profiles() {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		p := Profiles()[id]
		var cells []string
		for _, f := range Fields() {
			cells = append(cells, f.Name+"="+columnValue(p, f))
		}
		out = append(out, []string{p.Name, strings.Join(cells, ", ")})
	}
	return out
}

func columnValue(p *domain.RadiusProfile, f Field) string {
	switch f.Name {
	case "UpRate":
		return itoa(p.UpRate)
	case "DownRate":
		return itoa(p.DownRate)
	case "ActiveNum":
		return itoa(p.ActiveNum)
	case "BindMac":
		return itoa(p.BindMac)
	case "BindVlan":
		return itoa(p.BindVlan)
	case "AddrPool":
		return quote(p.AddrPool)
	case "Domain":
		return quote(p.Domain)
	case "RadiusClass":
		return quote(p.RadiusClass)
	case "IPv6PrefixPool":
		return quote(p.IPv6PrefixPool)
	case "DelegatedIpv6PrefixPool":
		return quote(p.DelegatedIpv6PrefixPool)
	}
	return ""
}

func userRows() [][]string {
	out := [][]string{{"subscriber", "link mode", "profile", "columns set on the user row"}}
	for _, u := range Fixtures() {
		mode := "dynamic"
		if u.ProfileLinkMode == 0 {
			mode = "static"
		}
		var cells []string
		if u.UpRate != 0 {
			cells = append(cells, "UpRate="+itoa(u.UpRate))
		}
		if u.DownRate != 0 {
			cells = append(cells, "DownRate="+itoa(u.DownRate))
		}
		if u.ActiveNum != 0 {
			cells = append(cells, "ActiveNum="+itoa(u.ActiveNum))
		}
		for _, s := range []struct {
			name, value string
		}{
			{"AddrPool", u.AddrPool}, {"Domain", u.Domain}, {"RadiusClass", u.RadiusClass},
			{"IPv6PrefixPool", u.IPv6PrefixPool}, {"DelegatedIpv6PrefixPool", u.DelegatedIpv6PrefixPool},
			{"MacAddr", u.MacAddr}, {"IpAddr", u.IpAddr}, {"IpV6Addr", u.IpV6Addr},
			{"DelegatedIpv6Prefix", u.DelegatedIpv6Prefix},
		} {
			if s.value != "" {
				cells = append(cells, s.name+"="+s.value)
			}
		}
		if u.Vlanid1 != 0 || u.Vlanid2 != 0 {
			cells = append(cells, fmt.Sprintf("Vlanid1=%d,Vlanid2=%d", u.Vlanid1, u.Vlanid2))
		}
		if len(cells) == 0 {
			cells = append(cells, "(nothing: every column inherits)")
		}
		out = append(out, []string{u.Username, mode, Profiles()[u.ProfileId].Name, strings.Join(cells, ", ")})
	}
	return out
}

func attrRows(attrs []Attr, withProducer bool) [][]string {
	head := []string{"wire key", "dictionary name", "value", "written by"}
	if !withProducer {
		head = []string{"wire key", "dictionary name", "value", "why held out"}
	}
	out := [][]string{head}
	for _, a := range attrs {
		why := "clock or config"
		if withProducer {
			why = a.Producer
		}
		name := a.Name
		if name == "" {
			name = "(no name in the tree)"
		}
		if withProducer && !a.Stable {
			why = a.Producer + ", held out of every diff"
		}
		out = append(out, []string{a.Key, name, a.Value, why})
	}
	return out
}

func outcomeRows(os []Outcome) [][]string {
	out := [][]string{{"scenario", "NAS", "cache reads", "ledger diff"}}
	for _, o := range os {
		diff := "(empty: the accept says the same thing)"
		if len(o.Changes) > 0 {
			var cells []string
			for _, c := range o.Changes {
				cells = append(cells, c.Key+" "+c.Kind+" "+c.Before+" -> "+c.After)
			}
			diff = strings.Join(cells, "; ")
		}
		out = append(out, []string{o.Scenario, o.Vendor, itoa(o.Lookups), diff})
	}
	return out
}

func claimRows(cs []ClaimStatus) [][]string {
	out := [][]string{{"id", "statement", "code line", "status", "measured"}}
	for _, c := range cs {
		out = append(out, []string{c.ID, c.Statement, c.File + ":" + itoa(c.Line), c.Status(), c.Measured})
	}
	return out
}

func gapRows(gs []GapStatus) [][]string {
	out := [][]string{{"id", "gap", "evidence", "open"}}
	for _, g := range gs {
		out = append(out, []string{g.ID, g.Gap, g.Evidence, yesNo(g.Open)})
	}
	return out
}

func importRows(r *Result) [][]string {
	out := [][]string{{"import", "banned"}}
	for _, imp := range r.Imports {
		hit := "no"
		if bannedHit(imp, r.Banned) {
			hit = "YES"
		}
		out = append(out, []string{imp, hit})
	}
	return out
}

// rows builds a table from a callback, so each section keeps its own header.
func rows(head []string, f func(int) []string, n int) [][]string {
	out := [][]string{head}
	for i := 0; i < n; i++ {
		out = append(out, f(i))
	}
	return out
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func quote(s string) string {
	if s == "" {
		return "(empty)"
	}
	return `"` + s + `"`
}
