package intentsearch

import "sort"

// Fixture is one operator prompt with the reason it should map where it maps. The
// FUT-001 criterion asks for twenty of these: 20 prompt fixture mapped dengan
// alasan, salah tafsir tidak memicu side effect.
type Fixture struct {
	// Prompt is what the operator types, in the language they actually use.
	Prompt string
	// Why is the analyst's reason for the expected mapping, not the parser's.
	Why string
	// WantKind is navigate, filter or refuse.
	WantKind Kind
	// WantResource is the menu entry expected, empty for a refusal.
	WantResource string
	// WantParams are the query keys expected, sorted; an empty list means no filter.
	WantParams []string
}

// Fixtures is the corpus. It is ordered by outcome so the report reads as the three
// cases the criterion names: reach a screen, compose a filter, refuse.
func Fixtures() []Fixture {
	return []Fixture{
		{Prompt: "daftar user radius", WantKind: KindNavigate, WantResource: "radius/users",
			Why: "the account list is the only screen the words name; the page has no filter array, so nothing is composed"},
		{Prompt: "profil radius untuk bandwidth 10 Mbps", WantKind: KindNavigate, WantResource: "radius/profiles",
			Why: "profil and bandwidth both sit on the profile screen, which carries more mentions than the RADIUS account list"},
		{Prompt: "sesi yang sedang online", WantKind: KindNavigate, WantResource: "radius/online",
			Why: "the live session list is a screen, not a filter; the resource declares no filter array"},
		{Prompt: "daftar perangkat NAS", WantKind: KindNavigate, WantResource: "network/nas",
			Why: "NAS is the device inventory; perangkat is not a cataloged word and cannot add a second target"},
		{Prompt: "pohon node dan ODP", WantKind: KindNavigate, WantResource: "network/nodes",
			Why: "node and ODP are both words of the node tree, so one intent matches twice and no rival exists"},
		{Prompt: "operator yang punya akses admin", WantKind: KindNavigate, WantResource: "system/operators",
			Why: "the operator register is the screen; the words describe the list, not a query key the page offers"},
		{Prompt: "sertifikat radsec yang hampir kedaluwarsa", WantKind: KindNavigate, WantResource: "system/certificate",
			Why: "the certificate store is the only screen named; expiry is not a filter this page declares"},
		{Prompt: "list pembayaran bulan ini", WantKind: KindNavigate, WantResource: "isp/payments",
			Why: "the ledger is the screen; the payments page offers customer and invoice references, neither named here"},
		{Prompt: "rekaman accounting kemarin", WantKind: KindNavigate, WantResource: "radius/accounting",
			Why: "accounting records are a screen; yesterday is a date the page cannot filter on"},
		{Prompt: "cari pelanggan bernama rani", WantKind: KindFilter, WantResource: "isp/customers",
			WantParams: []string{"q"},
			Why:        "cari and bernama are the cues for the free-text key the customer page declares; the parser names the key and leaves the value to the operator"},
		{Prompt: "pelanggan yang statusnya suspended", WantKind: KindFilter, WantResource: "isp/customers",
			WantParams: []string{"status"},
			Why:        "suspended is one of the five values the customer page itself offers, so the filter can be filled, not just named"},
		{Prompt: "invoice status belum dilunasi", WantKind: KindFilter, WantResource: "isp/invoices",
			WantParams: []string{"status"},
			Why:        "status names the key, but the page's vocabulary is English (paid, overdue, partial) and no offered value matches"},
		{Prompt: "invoice nomor INV-2026-0001", WantKind: KindFilter, WantResource: "isp/invoices",
			WantParams: []string{"invoice_no"},
			Why:        "nomor is the cue for the invoice number input; the value is text the operator must confirm before the list runs"},
		{Prompt: "tagihan pelanggan 42", WantKind: KindFilter, WantResource: "isp/invoices",
			WantParams: []string{"customer_id"},
			Why:        "pelanggan would also open the customer register, but it is a reference input of the invoice page, so it reads as a column here"},
		{Prompt: "daftar langganan yang pending", WantKind: KindFilter, WantResource: "isp/subscriptions",
			WantParams: []string{"status"},
			Why:        "pending is offered by the subscription page's own status vocabulary"},
		{Prompt: "paket yang aktif", WantKind: KindNavigate, WantResource: "isp/packages",
			Why: "the page offers active and inactive in English; aktif is the same word in Indonesian and the parser will not invent a translation, so it opens the screen and composes nothing"},
		{Prompt: "suspend pelanggan macet", WantKind: KindRefuse,
			Why: "suspend is a mutation the console accepts only as an action value, not a route; the parser names where it found the word and aims at nothing"},
		{Prompt: "bayar invoice pelanggan 7", WantKind: KindRefuse,
			Why: "bayar is a change of state, and a payment the console records through a POST; refusing is the only safe reading"},
		{Prompt: "pelanggan dan user", WantKind: KindRefuse,
			Why: "one mention each for two screens, and neither word belongs to the other page's filter vocabulary, so picking one would be a guess"},
		{Prompt: "tampilkan ringkasan performa jaringan", WantKind: KindRefuse,
			Why: "no cataloged word matches: the dashboard is a custom route, not a Resource, so there is no menu entry to name"},
	}
}

// ParamsOf returns the sorted query keys a resolution composed.
func (r Resolution) ParamsOf() []string {
	out := make([]string, 0, len(r.Filters))
	for _, f := range r.Filters {
		out = append(out, f.Param)
	}
	sort.Strings(out)
	return out
}

// FilterValue reports whether a query key was named at all, and the value filled
// for it. An empty value with ok=true means "this key, and the operator still has
// to say what for".
func (r Resolution) FilterValue(param string) (string, bool) {
	for _, f := range r.Filters {
		if f.Param == param {
			return f.Value, true
		}
	}
	return "", false
}

// Coverage counts how many cataloged intents the corpus reaches at least once.
func Coverage(results []Result) (reached, total int) {
	seen := map[string]bool{}
	total = len(Catalog())
	for _, res := range results {
		if res.Resolution.IntentID != "" {
			seen[res.Resolution.IntentID] = true
		}
	}
	for range seen {
		reached++
	}
	return reached, total
}

// Result pairs a fixture with what the parser actually did.
type Result struct {
	Fixture    Fixture
	Resolution Resolution
	// Matches is the honest pass/fail of this fixture against its expectation.
	Matches bool
}

// Run resolves the whole corpus and scores each fixture.
func Run(s Surface) []Result {
	p, err := NewParser(s)
	if err != nil {
		panic(err)
	}
	out := []Result{}
	for _, f := range Fixtures() {
		r := p.Resolve(f.Prompt)
		out = append(out, Result{Fixture: f, Resolution: r, Matches: agrees(f, r)})
	}
	return out
}

func agrees(f Fixture, r Resolution) bool {
	if r.Kind != f.WantKind {
		return false
	}
	if f.WantKind == KindRefuse {
		return len(r.Targets()) == 0 && len(r.Refusals) > 0
	}
	if r.Resource != f.WantResource {
		return false
	}
	got := r.ParamsOf()
	want := append([]string{}, f.WantParams...)
	sort.Strings(want)
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
