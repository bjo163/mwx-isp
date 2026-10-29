package incidentcard

import (
	"sort"
	"strings"
)

// The print census asks what the product already declares about paper, because a
// card that cannot reach a page has no legibility to be measured for. Every number
// here is a count of rules the tree does or does not write.

const webDir = "web/src"

// incidentRoutes are the paths the incident response types are served on. Each one
// is pinned to its ApiGET line in claims.go, so a renamed route breaks a statement
// instead of quietly shrinking the census.
var incidentRoutes = []string{
	"/isp/tickets",
	"/network/impact",
	"/network/monitor-incidents",
	"/network/monitor-targets",
	"/network/diagnostics/flapping",
	"/timeline",
}

// printToken is one declaration that changes what lands on the sheet.
type printToken struct {
	Token   string
	Channel string
	// Required marks the tokens the criterion depends on, so a zero count reads as
	// the finding it is rather than a missing row.
	Required bool
}

// printTokens are the declarations the paper census searches web/src for.
//
//nolint:gosec // G101: a Token is a CSS or DOM marker matched against source text, never a credential used to authenticate.
var printTokens = []printToken{
	{Token: "@media print", Channel: "paper rules written", Required: true},
	{Token: "window.print()", Channel: "print action wired", Required: true},
	{Token: "@page", Channel: "page geometry declared", Required: true},
	{Token: "size: A4", Channel: "A4 named on the sheet rule", Required: true},
	{Token: "print-color-adjust", Channel: "backgrounds asked of the browser"},
	{Token: "-webkit-print-color-adjust", Channel: "backgrounds asked, webkit spelling"},
	{Token: "pageBreakInside", Channel: "card kept whole across a break"},
	{Token: "page-break-inside", Channel: "card kept whole, css spelling"},
	{Token: "background-color: #fff !important", Channel: "sheet forced white"},
	{Token: "color: #000 !important", Channel: "ink forced black"},
	{Token: "background-color: transparent !important", Channel: "panel fill erased on paper"},
	{Token: "backgroundColor: 'transparent !important'", Channel: "panel fill erased in the sheet region"},
	{Token: "visibility: hidden", Channel: "everything but the region erased"},
	{Token: "display: none !important", Channel: "chrome removed"},
}

// PrintSite is one file's print declarations.
type PrintSite struct {
	File   string
	Tokens []string
	// Incident marks a file that renders incident data, decided by the routes it
	// fetches rather than by its name.
	Incident bool
}

// PrintCensus is the paper surface of the console, measured.
type PrintCensus struct {
	// Counts is how many code lines in web/src carry each channel's token.
	Counts map[string]int
	// Sites lists the files that write any print rule.
	Sites []PrintSite
	// IncidentFiles are the surfaces that show an incident to an operator.
	IncidentFiles []string
	// IncidentWithPrint are the incident surfaces with a print path at all.
	IncidentWithPrint []string
	// PrintBlocks holds the text of every @media print rule the tree writes.
	PrintBlocks []string
}

// measurePrint counts the paper rules the console writes and which surfaces can
// reach them.
func measurePrint(root string) PrintCensus {
	c := PrintCensus{Counts: map[string]int{}}
	files := walkFiles(root, webDir, ".tsx", ".ts", ".css")
	incidents := map[string]bool{}
	for _, rel := range files {
		for _, route := range incidentRoutes {
			if countMatches(root, rel, "'"+route) > 0 || countMatches(root, rel, "`"+route) > 0 ||
				countMatches(root, rel, route+"?") > 0 || countMatches(root, rel, "NocImpactPanel") > 0 {
				incidents[rel] = true
				break
			}
		}
	}
	for _, rel := range files {
		site := PrintSite{File: rel, Incident: incidents[rel]}
		for _, t := range printTokens {
			n := countMatches(root, rel, t.Token)
			if n == 0 {
				continue
			}
			c.Counts[t.Channel] += n
			site.Tokens = append(site.Tokens, t.Channel)
		}
		if len(site.Tokens) == 0 {
			continue
		}
		c.Sites = append(c.Sites, site)
		if site.Incident {
			c.IncidentWithPrint = append(c.IncidentWithPrint, rel)
		}
	}
	for _, t := range printTokens {
		if _, ok := c.Counts[t.Channel]; !ok {
			c.Counts[t.Channel] = 0
		}
	}
	for _, rel := range files {
		c.PrintBlocks = append(c.PrintBlocks, printBlocks(lines(root, rel))...)
	}
	for f := range incidents {
		c.IncidentFiles = append(c.IncidentFiles, f)
	}
	sort.Strings(c.IncidentFiles)
	sort.Slice(c.Sites, func(i, j int) bool { return c.Sites[i].File < c.Sites[j].File })
	return c
}

// printBlocks returns the text inside every "@media print" block of a file, counted
// by braces so a nested rule stays inside the block that declares it.
func printBlocks(src []string) []string {
	var out []string
	for i, ln := range src {
		if !strings.Contains(ln, "@media print") {
			continue
		}
		depth := 0
		var body []string
		for j := i; j < len(src); j++ {
			depth += strings.Count(src[j], "{") - strings.Count(src[j], "}")
			body = append(body, src[j])
			if depth <= 0 && j > i {
				break
			}
		}
		out = append(out, strings.Join(body, "\n"))
	}
	return out
}

// inkChannel is one way the skin puts a mark on a surface, and what the shipped
// paper rules do about it.
type inkChannel struct {
	// Channel is the paint the theme writes, such as an ink outline or a dot grid.
	Channel string
	// Property is the CSS property a paper rule would have to name to reach it.
	Property string
	// Declared is how many code lines of theme.ts write it.
	Declared int
	// PaperRules is how many @media print blocks name the same property.
	PaperRules int
	// Disposition is what the census says about the mark on the sheet.
	Disposition string
}

// grungeChannels are the paints the manga-ink direction is made of. The list is the
// vocabulary of the skin, so a channel the theme never writes parses as declared 0
// rather than going missing.
var grungeChannels = []struct {
	name     string
	pattern  string
	property string
}{
	{name: "ink panel outline", pattern: "border: ", property: "border"},
	{name: "ink rule under a header", pattern: "borderBottom: ", property: "border"},
	{name: "ink divider above a footer", pattern: "borderTop: ", property: "border"},
	{name: "hard offset shadow", pattern: "boxShadow: inkShadow(", property: "box-shadow"},
	{name: "halftone dot grid", pattern: "radial-gradient(", property: "background-image"},
	{name: "paper tint", pattern: "backgroundColor: palette.background", property: "background-color"},
	{name: "header band", pattern: "backgroundColor: headerBg", property: "background-color"},
	{name: "uppercase display titles", pattern: "textTransform: 'uppercase'", property: "text-transform"},
	{name: "monospaced identifiers", pattern: "fontFamily: fontStacks.mono", property: "font-family"},
	{name: "reserved status color", pattern: "palette.success", property: "color"},
}

// measureInk asks the aesthetic half of the criterion. The grunge look is a set of
// paints; a paper rule either names the property it draws on or does not, and
// nothing in the repository can decide what a browser drops when it prints without
// background graphics. That last part is a gap, not a measurement, and is recorded
// as one.
func measureInk(root string, census PrintCensus) []inkChannel {
	src := lines(root, themeFile)
	var out []inkChannel
	for _, g := range grungeChannels {
		n := 0
		for _, ln := range src {
			if strings.Contains(ln, g.pattern) {
				n++
			}
		}
		rules := 0
		for _, block := range census.PrintBlocks {
			if strings.Contains(block, g.property) {
				rules++
			}
		}
		ch := inkChannel{Channel: g.name, Property: g.property, Declared: n, PaperRules: rules}
		switch {
		case n == 0:
			ch.Disposition = "the skin never writes it"
		case rules > 0:
			ch.Disposition = "a paper rule names the property"
		default:
			ch.Disposition = "no paper rule names the property"
		}
		out = append(out, ch)
	}
	return out
}
