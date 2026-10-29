package intentsearch

import (
	"fmt"
	"sort"
	"strings"
)

// Kind is the only three things a resolution may be. There is no fourth: a prompt
// that cannot be placed is a refusal, never a guess.
type Kind string

const (
	// KindNavigate points at a menu entry with no filter attached.
	KindNavigate Kind = "navigate"
	// KindFilter points at a menu entry plus the filters the operator named.
	KindFilter Kind = "filter"
	// KindRefuse names nothing and carries the reason why.
	KindRefuse Kind = "refuse"
)

// Intent is one cataloged operator goal: a word set pointing at a measured menu
// entry and, when the page declares one, a measured filter group.
type Intent struct {
	// ID names the goal in reports and fixtures.
	ID string
	// Label is what the operator is asking for, in one line.
	Label string
	// Resource must be a <Resource> name found in web/src/App.tsx.
	Resource string
	// FilterGroup must be a filter array found in web/src/resources, or empty.
	FilterGroup string
	// Keywords are the words that place the prompt in this intent.
	Keywords []string
}

// AppliedFilter is one query key the parser composed for a prompt.
type AppliedFilter struct {
	// Param is the query key the product reads, e.g. status.
	Param string
	// Value is filled when the page's own vocabulary contains it; empty means the
	// operator still has to supply a value.
	Value string
	// Why states which words in the prompt produced this filter.
	Why string
}

// Resolution is the complete output of the parser. It is inert data: it names a
// read route and query keys, and carries no verb, body or credential that a
// caller could execute.
type Resolution struct {
	Prompt   string
	Kind     Kind
	IntentID string
	Resource string
	Route    string
	Filters  []AppliedFilter
	// Refusals are the reasons the parser named nothing.
	Refusals []string
}

// Targets is every route this resolution could lead to. A refusal has none, which
// is the mechanical form of "a misread prompt triggers no side effect".
func (r Resolution) Targets() []string {
	if r.Kind == KindRefuse || r.Route == "" {
		return nil
	}
	return []string{r.Route}
}

// String renders one resolution as a report row.
func (r Resolution) String() string {
	if r.Kind == KindRefuse {
		return fmt.Sprintf("%s -> refuse (%s)", r.Prompt, strings.Join(r.Refusals, "; "))
	}
	parts := make([]string, 0, len(r.Filters))
	for _, f := range r.Filters {
		if f.Value == "" {
			parts = append(parts, f.Param+"=?")
			continue
		}
		parts = append(parts, f.Param+"="+f.Value)
	}
	return fmt.Sprintf("%s -> %s [%s]", r.Prompt, r.Route, strings.Join(parts, ", "))
}

// Catalog is the intent table. The keyword side of it is the only hand-written
// vocabulary in this prototype, because the console names its screens in English
// while operators ask in Indonesian; the target side is measured and validated
// against the scan by TestEveryCatalogedTargetExistsInTheTree.
func Catalog() []Intent {
	return []Intent{
		{ID: "customers", Label: "the customer register", Resource: "isp/customers",
			FilterGroup: "customerFilters", Keywords: []string{"pelanggan", "customer", "customers", "abonen", "pelanggannya"}},
		{ID: "packages", Label: "the commercial package list", Resource: "isp/packages",
			FilterGroup: "packageFilters", Keywords: []string{"paket", "packages", "package", "harga", "tarif"}},
		{ID: "subscriptions", Label: "the subscription register", Resource: "isp/subscriptions",
			FilterGroup: "subscriptionFilters", Keywords: []string{"langganan", "subscription", "subscriptions", "aktivasi"}},
		{ID: "invoices", Label: "the invoice list", Resource: "isp/invoices",
			FilterGroup: "invoiceFilters", Keywords: []string{"invoice", "invoices", "tagihan", "faktur"}},
		{ID: "payments", Label: "the payment ledger", Resource: "isp/payments",
			FilterGroup: "paymentFilters", Keywords: []string{"pembayaran", "payment", "payments", "bayaran"}},
		{ID: "radius-users", Label: "the RADIUS account list", Resource: "radius/users",
			Keywords: []string{"user", "users", "akun", "account", "radius", "kredensial"}},
		{ID: "online-sessions", Label: "the live session list", Resource: "radius/online",
			Keywords: []string{"sesi", "session", "sessions", "online", "terhubung"}},
		{ID: "accounting", Label: "the accounting records", Resource: "radius/accounting",
			Keywords: []string{"accounting", "rekaman", "catatan", "trafiking"}},
		{ID: "profiles", Label: "the RADIUS profile list", Resource: "radius/profiles",
			Keywords: []string{"profil", "profile", "profiles", "bandwidth", "kuota", "speed"}},
		{ID: "nas", Label: "the NAS inventory", Resource: "network/nas",
			Keywords: []string{"nas", "access", "server", "router"}},
		{ID: "nodes", Label: "the network node tree", Resource: "network/nodes",
			Keywords: []string{"node", "nodes", "olt", "odp", "cabang", "wilayah"}},
		{ID: "operators", Label: "the operator accounts", Resource: "system/operators",
			Keywords: []string{"operator", "operators", "staff", "petugas", "admin"}},
		{ID: "certificates", Label: "the certificate store", Resource: "system/certificate",
			Keywords: []string{"sertifikat", "certificate", "certificates", "cert", "radsec"}},
	}
}

// paramCues are the words that say an operator named a filter key without giving
// its value. Each key is a query parameter the scan found in the handler file.
// Content words only: "yang" and "dengan" carry no information about which key the
// operator means, so they are deliberately absent, and a phrasing that uses no
// content word resolves to a navigation with no filter rather than to a guess.
var paramCues = map[string][]string{
	"q":            {"cari", "search", "nama", "bernama"},
	"invoice_no":   {"nomor", "no", "number", "inv"},
	"customer_id":  {"pelanggan", "customer", "punya", "milik"},
	"invoice_id":   {"invoice", "tagihan", "atas"},
	"status":       {"status", "statusnya"},
	"type":         {"tipe", "jenis"},
	"name":         {"nama", "key"},
	"reference_id": {"rujukan"},
}

// hazardAliases is the Indonesian half of the deny list. The English half is
// measured from the API's own action routes; this table only adds the everyday
// words an operator would use for the same state change, and is disclosed as
// hand-written in the study memo.
func hazardAliases() []string {
	return []string{
		"hapus", "tambah", "tambahkan", "ubah", "ganti", "kirim", "putuskan",
		"matikan", "nyalakan", "restart", "buat", "buatkan", "batalkan",
		"tahan", "putus", "blokir", "aktifkan", "bayar", "tagih", "reset",
	}
}

// Parser resolves prompts against a scanned surface. It holds no connection, no
// credentials and no writer: everything it can do is return a Resolution.
type Parser struct {
	surface  Surface
	intents  []Intent
	verbs    []string
	aliases  []string
	readOnly map[string]bool
}

// NewParser binds the catalog to a measured surface. A cataloged target that the
// surface does not have is an error, so the table cannot point at a phantom menu.
func NewParser(s Surface) (*Parser, error) {
	read := map[string]bool{}
	for _, r := range s.ReadRoutes() {
		read[r.Display()] = true
	}
	p := &Parser{surface: s, verbs: s.MutationVerbs(), aliases: hazardAliases(), readOnly: read}
	for _, in := range Catalog() {
		if !s.HasResource(in.Resource) {
			return nil, fmt.Errorf("intent %s points at %q, which is not a resource in App.tsx", in.ID, in.Resource)
		}
		if _, ok := s.RouteFor(in.Resource); !ok {
			return nil, fmt.Errorf("intent %s has no read route for %q", in.ID, in.Resource)
		}
		if in.FilterGroup != "" && len(s.FiltersForGroup(in.FilterGroup)) == 0 {
			return nil, fmt.Errorf("intent %s points at filter group %q, which no page declares", in.ID, in.FilterGroup)
		}
		p.intents = append(p.intents, in)
	}
	return p, nil
}

// Surface returns the measured surface the parser is bound to.
func (p *Parser) Surface() Surface { return p.surface }

// MutationVocabulary is the full deny set: every action word the API itself uses
// for a state change, plus the documented Indonesian aliases.
func (p *Parser) MutationVocabulary() []string {
	out := append([]string{}, p.verbs...)
	out = append(out, p.aliases...)
	sort.Strings(out)
	return out
}

// Resolve turns one operator prompt into a Resolution.
func (p *Parser) Resolve(prompt string) Resolution {
	res := Resolution{Prompt: prompt, Kind: KindRefuse}
	norm := normalize(prompt)
	tokens := strings.Fields(norm)

	// Deny first, so a prompt that also names a screen still cannot aim at a write.
	for _, v := range p.verbs {
		if !hasToken(tokens, v) {
			continue
		}
		rt := p.surface.RoutesForVerb(v)
		if len(rt) > 0 {
			var disp []string
			for _, r := range rt {
				disp = append(disp, r.Display())
			}
			res.Refusals = append(res.Refusals, fmt.Sprintf(
				"%q is an action the console performs with %s", v, strings.Join(disp, ", ")))
			continue
		}
		res.Refusals = append(res.Refusals, fmt.Sprintf(
			"%q is an action the console accepts as a value at %s", v,
			strings.Join(p.surface.ActionSitesFor(v), ", ")))
	}
	for _, a := range p.aliases {
		if hasToken(tokens, a) {
			res.Refusals = append(res.Refusals, fmt.Sprintf("%q asks for a change of state, not a view", a))
		}
	}
	if len(res.Refusals) > 0 {
		return res
	}

	best, scored, ambiguous := p.score(tokens)
	if best == nil {
		if ambiguous != "" {
			res.Refusals = append(res.Refusals, fmt.Sprintf("ambiguous between %s; naming one of them would be a guess", ambiguous))
			return res
		}
		res.Refusals = append(res.Refusals, "no word in the prompt matches a cataloged menu entry, so the parser names nothing")
		return res
	}
	route, ok := p.surface.RouteFor(best.Resource)
	if !ok {
		res.Refusals = append(res.Refusals, fmt.Sprintf("menu entry %q has no read route", best.Resource))
		return res
	}
	res.IntentID = best.ID
	res.Resource = best.Resource
	res.Route = route.Display()
	res.Filters = p.compose(*best, tokens, scored)
	if len(res.Filters) > 0 {
		res.Kind = KindFilter
	} else {
		res.Kind = KindNavigate
	}
	return res
}

// score returns the winning intent and the reason it refused when it cannot win.
//
// Three steps, in order: count the cataloged words each intent matched; drop an
// intent whose every matched word is a filter key or value of a rival intent's own
// filter group, because in that prompt the word names a column, not a second
// screen; if two survive with the same count the prompt is a coin flip and the
// parser refuses instead of guessing.
func (p *Parser) score(tokens []string) (*Intent, map[string]int, string) {
	matched := map[string][]string{}
	scored := map[string]int{}
	for i := range p.intents {
		var hits []string
		for _, kw := range p.intents[i].Keywords {
			if hasToken(tokens, kw) {
				hits = append(hits, kw)
			}
		}
		if len(hits) == 0 {
			continue
		}
		matched[p.intents[i].ID] = hits
		scored[p.intents[i].ID] = len(hits)
	}
	if len(scored) == 0 {
		return nil, scored, ""
	}
	best := 0
	var winners []string
	for _, in := range p.intents {
		if scored[in.ID] > best {
			best = scored[in.ID]
		}
	}
	for _, in := range p.intents {
		if scored[in.ID] == best {
			winners = append(winners, in.ID)
		}
	}
	if len(winners) > 1 {
		winners = p.absorb(winners, matched)
	}
	if len(winners) == 0 {
		return nil, scored, ""
	}
	if len(winners) > 1 {
		sort.Strings(winners)
		return nil, scored, strings.Join(winners, " and ")
	}
	for i := range p.intents {
		if p.intents[i].ID == winners[0] {
			return &p.intents[i], scored, ""
		}
	}
	return nil, scored, ""
}

// absorb removes a tied intent whose matched words all belong to another tied
// intent's filter vocabulary.
func (p *Parser) absorb(winners []string, matched map[string][]string) []string {
	var out []string
	for _, w := range winners {
		absorbed := false
		for _, rival := range winners {
			if rival == w {
				continue
			}
			if p.covers(rival, matched[w]) {
				absorbed = true
				break
			}
		}
		if !absorbed {
			out = append(out, w)
		}
	}
	return out
}

// covers reports whether every word is a parameter, cue or offered value of the
// rival intent's own filter group.
func (p *Parser) covers(rivalID string, words []string) bool {
	if len(words) == 0 {
		return false
	}
	for _, in := range p.intents {
		if in.ID != rivalID || in.FilterGroup == "" {
			continue
		}
		for _, w := range words {
			if !p.isCue(in.FilterGroup, w) {
				return false
			}
		}
		return true
	}
	return false
}

// isCue reports whether a filter input in the group is keyed or filled by the word.
func (p *Parser) isCue(group, word string) bool {
	for _, f := range p.surface.FiltersForGroup(group) {
		if f.Param == word {
			return true
		}
		for _, c := range paramCues[f.Param] {
			if strings.EqualFold(c, word) {
				return true
			}
		}
		for _, v := range f.Values {
			if strings.EqualFold(v, word) {
				return true
			}
		}
	}
	return false
}

// compose builds the filter set from the page's own inputs. A value only appears
// when the page offers it; otherwise the key is named with an empty value, which
// the operator must still fill in.
func (p *Parser) compose(in Intent, tokens []string, scored map[string]int) []AppliedFilter {
	if in.FilterGroup == "" {
		return nil
	}
	var out []AppliedFilter
	for _, f := range p.surface.FiltersForGroup(in.FilterGroup) {
		cues := append([]string{f.Param}, paramCues[f.Param]...)
		cues = append(cues, f.Values...)
		matched := []string{}
		for _, c := range cues {
			if hasToken(tokens, strings.ToLower(c)) {
				matched = append(matched, c)
			}
		}
		if len(matched) == 0 {
			continue
		}
		af := AppliedFilter{Param: f.Param, Why: fmt.Sprintf("matched %s in group %s", strings.Join(matched, ", "), f.Group)}
		for _, v := range f.Values {
			if hasToken(tokens, strings.ToLower(v)) {
				af.Value = v
				break
			}
		}
		out = append(out, af)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Param < out[j].Param })
	return out
}

func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func hasToken(tokens []string, want string) bool {
	want = strings.ToLower(want)
	for _, t := range tokens {
		if t == want {
			return true
		}
	}
	return false
}
