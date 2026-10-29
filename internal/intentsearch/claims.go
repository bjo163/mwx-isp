package intentsearch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Claim is one statement the intent-search argument rests on, pinned to a symbol
// in product code. A claim is confirmed only when the symbol appears on a code
// line, not in a comment.
type Claim struct {
	// ID names the claim in the report.
	ID string
	// Statement is what is claimed, in one line.
	Statement string
	// File is the product file that must carry the symbol.
	File string
	// Symbol is the identifier the statement depends on.
	Symbol string
}

// Absence is one gap the criterion would have to close, recorded as a set of
// symbols that must not appear anywhere in the scanned scope.
type Absence struct {
	ID        string
	Statement string
	// Scope is a directory or file relative to the repository root; empty is the
	// whole product tree.
	Scope string
	// Symbols are the names whose absence the gap claims.
	Symbols []string
}

// ClaimStatus is the measured outcome of one claim.
type ClaimStatus struct {
	Claim
	// Confirmed means the symbol was found on a code line in the named file.
	Confirmed bool
	// Broken means the named file no longer exists - drift, not a dropped claim.
	Broken bool
	Hits   []string
}

// GapStatus is the measured outcome of one absence.
type GapStatus struct {
	Absence
	// Open means none of the symbols was found, so the gap is real today.
	Open bool
	// Vacated means the gap got filled since it was recorded.
	Vacated bool
	Hits    []string
}

// Totals are the numbers the report quotes, all measured from the scan.
type Totals struct {
	ReadRoutes      int
	MutationRoutes  int
	ActionVerbs     int
	GuardVerbs      int
	PathMapEntries  int
	UnmappedNames   int
	HazardAliases   int
	Fixtures        int
	Matched         int
	Navigations     int
	Filters         int
	Refusals        int
	SideEffectNames int
	StatusFills     int
	IntentsReached  int
	IntentsTotal    int
	Claims          int
	ClaimsConfirmed int
	Gaps            int
	GapsOpen        int
}

// Report is the whole study output.
type Report struct {
	Surface Surface
	Results []Result
	Claims  []ClaimStatus
	Gaps    []GapStatus
	Totals  Totals
	// StaleFiles lists the backup files the scan could not tell from real source.
	StaleFiles []string
}

// Claims is the positive catalog: every statement an intent-search proposal would
// otherwise have to assert from prose.
func Claims() []Claim {
	return []Claim{
		{ID: "customer-list-handler", Statement: "the customer register is a read handler the console already calls",
			File: "internal/adminapi/isp.go", Symbol: "listCustomers"},
		{ID: "invoice-list-handler", Statement: "invoices are listed through query parameters, not through a bespoke search",
			File: "internal/adminapi/isp.go", Symbol: "listInvoices"},
		{ID: "payment-list-handler", Statement: "the payment ledger is reachable by a read route",
			File: "internal/adminapi/isp.go", Symbol: "listPayments"},
		{ID: "subscription-list-handler", Statement: "subscriptions are a read surface with a status column",
			File: "internal/adminapi/isp.go", Symbol: "listSubscriptions"},
		{ID: "query-param-read", Statement: "the console reads its filters back as query parameters",
			File: "internal/adminapi/isp.go", Symbol: "QueryParam"},
		{ID: "status-widget", Statement: "the status filter is a widget with a declared value list",
			File: "web/src/resources/isp.tsx", Symbol: "StatusSelectInput"},
		{ID: "status-vocabulary", Statement: "the values a prompt may fill come from the page's own array",
			File: "web/src/resources/isp.tsx", Symbol: "customerStatusValues"},
		{ID: "customer-list-page", Statement: "the customer list page is the menu entry a navigation resolves to",
			File: "web/src/resources/isp.tsx", Symbol: "CustomerList"},
		{ID: "session-list-page", Statement: "the live session page is a declared resource, not a custom route",
			File: "web/src/resources/onlineSessions.tsx", Symbol: "OnlineSessionList"},
		{ID: "certificate-list-page", Statement: "the certificate page is a declared resource",
			File: "web/src/resources/certificates.tsx", Symbol: "CertificateList"},
		{ID: "active-filter-render", Statement: "the UI already renders the composed filter set back to the operator",
			File: "web/src/components/ActiveFilters.tsx", Symbol: "ActiveFilters"},
		{ID: "menu-name-map", Statement: "a menu name becomes an API path through one table the console already owns",
			File: "web/src/providers/dataProvider.ts", Symbol: "resourcePathMap"},
		{ID: "filter-becomes-query", Statement: "the data provider appends the filter object to the read URL, so a composed filter needs no new endpoint",
			File: "web/src/providers/dataProvider.ts", Symbol: "getList"},
		{ID: "role-gate", Statement: "an intent result still has to pass the role gate the admin API installs",
			File: "internal/adminapi/authz.go", Symbol: "requireAdmin"},
		{ID: "platform-gate", Statement: "platform-only screens are gated separately, so an intent cannot flatten them",
			File: "internal/adminapi/platform_tenants.go", Symbol: "requirePlatformAdmin"},
		{ID: "dry-run-precedent", Statement: "the product already has a no-write preview for a mutating path",
			File: "internal/adminapi/system_backup.go", Symbol: "restoreDryRun"},
		{ID: "lifecycle-action-words", Statement: "not every mutation verb is a route segment; the subscription lifecycle takes them as a value",
			File: "internal/adminapi/isp.go", Symbol: "reactivate"},
		{ID: "session-action-route", Statement: "the live session page has a mutating action route, so a read-only resolver must name it and stay off it",
			File: "internal/adminapi/sessions.go", Symbol: "DisconnectOnlineSession"},
	}
}

// Gaps is the negative catalog: what an intent search would have to add, stated as
// symbols that must be missing today.
func Gaps() []Absence {
	return []Absence{
		{ID: "no-command-palette", Scope: "web/src",
			Statement: "the console has no command palette or fuzzy launcher for an operator to type into",
			Symbols:   []string{"CommandPalette", "useSearch", "OpenWithPipeline"}},
		{ID: "no-intent-parser", Scope: "internal",
			Statement: "no product code parses a free-text operator intent",
			Symbols:   []string{"IntentParser", "ParseIntent", "parseIntent"}},
		{ID: "no-search-endpoint", Scope: "internal/adminapi",
			Statement: "no admin route takes a natural-language query; free text only arrives as a search box value",
			Symbols:   []string{"\"/search\"", "\"/ai/query\"", "\"/assistant\""}},
		{ID: "no-date-filter-on-customers", Scope: "internal/adminapi/isp.go",
			Statement: "the customer list offers no date range, so a prompt about last month cannot be composed",
			Symbols:   []string{"since", "date_from", "created_after"}},
		{ID: "dry-run-only-for-restore", Scope: "internal/adminapi",
			Statement: "the no-write preview exists for the restore path only, not for the ISP mutations an intent would refuse",
			Symbols:   []string{"isp/dry_run", "preview_only"}},
	}
}

// Verify scans the repository and runs the corpus. root is the repository checkout.
func Verify(root string) (Report, error) {
	s, err := Scan(root)
	if err != nil {
		return Report{}, err
	}
	r := Report{Surface: s, Results: Run(s)}
	r.Claims = verifyClaims(root, Claims())
	r.Gaps = verifyGaps(root, Gaps())
	r.StaleFiles = staleFiles(filepath.Join(root, "web", "src"), root)

	p, err := NewParser(s)
	if err != nil {
		return Report{}, err
	}
	t := Totals{}
	t.ReadRoutes = len(s.ReadRoutes())
	for _, rt := range s.Routes {
		if !rt.ReadOnly() {
			t.MutationRoutes++
		}
	}
	t.ActionVerbs = len(s.MutationVerbs())
	t.GuardVerbs = len(s.ActionSites)
	t.PathMapEntries = len(s.PathMap)
	for _, res := range s.Resources {
		if _, ok := s.PathMap[res.Name]; !ok {
			t.UnmappedNames++
		}
	}
	t.HazardAliases = len(p.aliases)
	for _, res := range r.Results {
		t.Fixtures++
		if res.Matches {
			t.Matched++
		}
		switch res.Resolution.Kind {
		case KindNavigate:
			t.Navigations++
		case KindFilter:
			t.Filters++
		case KindRefuse:
			t.Refusals++
		}
		t.SideEffectNames += len(res.Resolution.Targets())
		for _, f := range res.Resolution.Filters {
			if f.Value != "" {
				t.StatusFills++
			}
		}
	}
	t.IntentsReached, t.IntentsTotal = Coverage(r.Results)
	t.Claims = len(r.Claims)
	for _, c := range r.Claims {
		if c.Confirmed {
			t.ClaimsConfirmed++
		}
	}
	t.Gaps = len(r.Gaps)
	for _, g := range r.Gaps {
		if g.Open {
			t.GapsOpen++
		}
	}
	r.Totals = t
	return r, nil
}

func verifyClaims(root string, claims []Claim) []ClaimStatus {
	out := []ClaimStatus{}
	for _, c := range claims {
		st := ClaimStatus{Claim: c}
		//nolint:gosec // G304: path is a listed source file under the checkout root.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.File)))
		if err != nil {
			st.Broken = true
			out = append(out, st)
			continue
		}
		lines := strings.Split(blankComments(string(raw)), "\n")
		if at := locate(lines, c.Symbol); at > 0 {
			st.Confirmed = true
			st.Hits = []string{fmt.Sprintf("%s:%d", c.File, at)}
		}
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func verifyGaps(root string, gaps []Absence) []GapStatus {
	out := []GapStatus{}
	for _, g := range gaps {
		st := GapStatus{Absence: g, Open: true}
		scope := root
		if g.Scope != "" {
			scope = filepath.Join(root, filepath.FromSlash(g.Scope))
		}
		_ = filepath.Walk(scope, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel := relPath(root, p)
			if strings.HasPrefix(rel, "internal/intentsearch/") {
				return nil
			}
			if !strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, ".tsx") && !strings.HasSuffix(rel, ".ts") {
				return nil
			}
			//nolint:gosec // G304: path comes from the walk, filtered to source files.
			raw, rerr := os.ReadFile(p)
			if rerr != nil {
				return nil
			}
			lines := strings.Split(blankComments(string(raw)), "\n")
			for _, sym := range g.Symbols {
				if at := locate(lines, sym); at > 0 {
					st.Hits = append(st.Hits, fmt.Sprintf("%s:%d", rel, at))
					st.Open = false
					st.Vacated = true
				}
			}
			return nil
		})
		sort.Strings(st.Hits)
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// staleFiles reports the backup copies sitting inside source directories. The scan
// reads them as text, so the study records them instead of pretending they are not
// there.
func staleFiles(dir, root string) []string {
	var out []string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(filepath.Base(p), ".backup") || strings.HasSuffix(p, ".bak") {
			out = append(out, relPath(root, p))
		}
		return nil
	})
	sort.Strings(out)
	return out
}
