package opstats

import "sort"

// Claim is one statement the run checks against the tree. The check is a whole
// identifier match on a code line of a named file, after comments are blanked, so a
// sentence in a doc comment cannot confirm a claim and a moved symbol breaks one.
type Claim struct {
	ID     string
	Text   string
	File   string
	Symbol string
	Line   int
	Status string
}

// Gap is something the criterion asks for that a repository-only round cannot answer,
// with the reason printed next to it.
type Gap struct {
	ID   string
	Need string
	// Status is always no-go here: the item needs an asset or a decision, not more reading.
	Status string
	Note   string
}

// claimSpec is a statement plus the file and symbol that decides it.
type claimSpec struct {
	id, text, file, symbol string
}

func claimSpecs() []claimSpec {
	return []claimSpec{
		{"A01", "the dashboard snapshot is a served type, so its numbers are candidates rather than inventions",
			"internal/adminapi/dashboard.go", "type DashboardStats struct"},
		{"A02", "the tenant's user total is a model query, the one path the shipped tenant callback can reach",
			"internal/adminapi/dashboard.go", ".Count(&stats.TotalUsers)"},
		{"A03", "the ISP dashboard publishes an outstanding money sum",
			"internal/adminapi/isp.go", "COALESCE(SUM(balance), 0)"},
		{"A04", "the column that sum reads is a stored invoice balance",
			"internal/domain/isp.go", "Balance"},
		{"A05", "the tenant predicate is injected by a GORM callback registered once",
			"internal/tenancy/scope.go", "func RegisterCallbacks("},
		{"A06", "the injected read scope is one function over the ORM query path",
			"internal/tenancy/scope.go", "func scopeQuery("},
		{"A07", "a tenant id arrives from the signed claim, not from a request parameter",
			"internal/adminapi/auth.go", `claims["tenant_id"]`},
		{"A08", "the request tenant is set in one place, from the authenticated operator",
			"internal/adminapi/auth.go", `c.Set("tenant_id"`},
		{"A09", "a handler without a tenant context stands in the default tenant instead of refusing",
			"internal/adminapi/context.go", "DefaultTenantID"},
		{"A10", "the NOC impact counts are their own served sub-struct",
			"internal/adminapi/noc_impact.go", "type nocImpactCounts struct"},
		{"A11", "the impacted roster counts over an explicitly written tenant comparison",
			"internal/adminapi/noc_impact.go", "s.tenant_id = ?"},
		{"A12", "the traffic series carries absolute byte sums, not a bucketed band",
			"internal/adminapi/telemetry.go", "total_in_bytes"},
		{"A13", "the per-subscriber traffic route takes a username from the path",
			"internal/adminapi/telemetry.go", "/radius/traffic/user/:username"},
		{"A14", "a flapping diagnosis groups accounting by username",
			"internal/adminapi/isp_operations.go", `Group("username")`},
		{"A15", "the IPAM pool numbers are capacity over a CIDR",
			"internal/domain/isp_operations.go", "total_ips"},
		{"A16", "the platform tenant list serves the persisted tenant row, not a statistics view",
			"internal/adminapi/platform_tenants.go", "domain.Tenant"},
		{"A17", "the tenant row itself carries a tax identifier and a billing address",
			"internal/domain/tenant.go", "tax_id"},
		{"A18", "the platform operator view exposes another tenant's operator real name",
			"internal/adminapi/platform_tenants.go", "realname"},
		{"A19", "platform access is a level check, not a permission string",
			"internal/adminapi/platform_tenants.go", "PlatformAdmin"},
		{"A20", "the backup manifest holds per-table record counts for the whole installation",
			"internal/adminapi/backup_envelope.go", "RecordCounts"},
		{"A21", "the shipped pseudonymiser exists and is a log and bundle tool",
			"internal/redact/redact.go", "func Pseudonymize"},
		{"A22", "the config registry the system config page renders is a flat list of keyed items",
			"internal/app/config_schemas.json", `"key"`},
		{"A23", "the product already ships a per-tenant consent row with a stored risk acknowledgement",
			"internal/domain/network_monitor.go", "RiskAcknowledgedAt"},
		{"A24", "a ratchet test already enforces tenant scope on raw SQL by column name",
			"internal/tenancy/rawsql_ratchet_test.go", "func TestRawSQLTenantScopeRatchet"},
		{"A25", "the dashboard test file asserts the response, and this run counts its tenant references separately",
			"internal/adminapi/dashboard_test.go", "func TestGetDashboardStats"},
		{"A26", "the tenancy policy checker classifies a relation but is not wired as a gate",
			"internal/tenancy/policycheck/policy.go", "Scoped()"},
	}
}

func evalClaims(root string) []Claim {
	specs := claimSpecs()
	out := make([]Claim, 0, len(specs))
	for _, s := range specs {
		c := Claim{ID: s.id, Text: s.text, File: s.file, Symbol: s.symbol}
		if src := lines(root, s.file); len(src) == 0 {
			c.Status = "vacated"
		} else if ln := findLine(src, s.symbol); ln > 0 {
			c.Line = ln
			c.Status = "confirmed"
		} else {
			c.Status = "broken"
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	// The claim set has to stay the size the memo prints. A dropped row would look
	// like a passing run.
	if len(out) != len(specs) {
		out = append(out, Claim{ID: "Z00", Text: "claim count drifted", Status: "broken"})
	}
	return out
}

// findLine returns the 1-based line of the first code line carrying the symbol.
func findLine(src []string, symbol string) int {
	for i, ln := range src {
		if containsBounded(ln, symbol) {
			return i + 1
		}
	}
	return 0
}

// gaps records the half of the criterion the repository cannot answer.
func gaps() []Gap {
	return []Gap{
		{"G01", "tenant populations from a running deployment",
			"no-go", "the cell sizes here are the fixture's; a one-customer RT/RW Net and a thousand-line ISP behave differently and only stored rows say which is which"},
		{"G02", "a governance decision about publishing an aggregate outside its tenant",
			"no-go", "TR-F033 as written says nothing about an aggregate leaving a tenant, so the permission is a scope change in docs/feature-checklist.md, not something a research round can grant"},
		{"G03", "the database's own answer to each parsed statement",
			"no-go", "this model applies the verb a handler writes to a synthetic table; a real query plan, a collation or a NULL edge only shows up against PostgreSQL"},
		{"G04", "the suppression threshold itself",
			"no-go", "k is a policy number. The run prices k at 1, 2, 3 and 5 and reports what each costs a tenant in coverage; choosing one is a product decision"},
		{"G05", "a collision study of real per-tenant values",
			"no-go", "every fixture tenant has a distinct aggregate because every marker is distinct, so the fingerprint result is a property of the fixture, not of real IDR sums"},
		{"G06", "the platform-admin trust boundary",
			"no-go", "the platform surface serves a tenant's tax id, billing address and contact phone to a super operator. Whether that is acceptable is a governance question the tree records no answer to"},
		{"G07", "closing the default-tenant fallback",
			"no-go", "the sites are counted and cited, but making a context-less job refuse instead of reporting the default tenant edits internal/adminapi, which is product code and outside a RISET round"},
	}
}

// statusCounts summarizes the claim table for the totals row.
func statusCounts(claims []Claim) (ok, broken, vacated int) {
	for _, c := range claims {
		switch c.Status {
		case "confirmed":
			ok++
		case "broken":
			broken++
		case "vacated":
			vacated++
		}
	}
	return ok, broken, vacated
}
