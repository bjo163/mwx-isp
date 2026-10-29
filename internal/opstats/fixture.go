package opstats

import (
	"sort"
)

// The fixture is this study's model of a running deployment: synthetic tenants with
// known populations, each stored row carrying a value no other row shares. It exists
// because the criterion is about fixture data - "a tenant's unique fixture value must
// not be reconstructable from a response" - and because the repository has no live
// tenant populations to measure. Every number in the reconstructability tables is an
// arithmetic fact about this fixture and the aggregate verb the product uses, never a
// claim about a customer's real balance.

// tenantShapes are the six populations the fixture holds. The sizes are the point: a
// statistic that is safe over a town-sized tenant can be a name over a one-customer
// RT/RW Net, and only the small end exercises a cell of one.
var tenantShapes = []struct {
	Slug      string
	Customers int
	Invoices  int
	Sessions  int
	Tickets   int
	Profiles  int
	Pools     int
}{
	{Slug: "solo", Customers: 1, Invoices: 1, Sessions: 1, Tickets: 1, Profiles: 2, Pools: 1},
	{Slug: "pair", Customers: 2, Invoices: 1, Sessions: 2, Tickets: 0, Profiles: 2, Pools: 1},
	{Slug: "hamlet", Customers: 5, Invoices: 1, Sessions: 5, Tickets: 2, Profiles: 3, Pools: 2},
	{Slug: "village", Customers: 12, Invoices: 2, Sessions: 9, Tickets: 3, Profiles: 4, Pools: 2},
	{Slug: "district", Customers: 40, Invoices: 1, Sessions: 22, Tickets: 5, Profiles: 5, Pools: 3},
	{Slug: "town", Customers: 120, Invoices: 1, Sessions: 60, Tickets: 9, Profiles: 6, Pools: 4},
}

// fact is one stored row: which tenant owns it, which table it lives in, the buckets a
// grouped statistic would put it in, and the value only that row carries.
type fact struct {
	Tenant string
	Table  string
	Marker int64
	// Labels are the grouping keys the dashboard's series use. A row carries them all;
	// a cell rule picks one, so the same session is never counted twice.
	Labels map[string]string
}

// Fixture is the whole synthetic installation.
type Fixture struct {
	Tenants []string
	Facts   []fact
}

// tableOrder gives each table a stable digit in a marker, so a value is unique across the
// whole fixture rather than only inside one table.
var tableOrder = []string{
	"isp_customer",
	"isp_subscription",
	"isp_invoice",
	"isp_payment",
	"isp_trouble_ticket",
	"isp_ipam_pool",
	"radius_user",
	"radius_profile",
	"radius_online",
	"radius_accounting",
	"net_monitor_target",
	"tenant",
}

func marker(tenantIdx int, table string, row int) int64 {
	ti := 1
	for i, t := range tableOrder {
		if t == table {
			ti = i + 1
			break
		}
	}
	return int64(tenantIdx+1)*10_000_000 + int64(ti)*100_000 + int64(row) + 1
}

// buildFixture lays out every synthetic row. It reads no file and no database: the
// fixture is a constant of the study, and a test pins its marker uniqueness.
func buildFixture() *Fixture {
	fx := &Fixture{}
	for ti, shape := range tenantShapes {
		fx.Tenants = append(fx.Tenants, shape.Slug)
		add := func(table string, n int, labels func(row int) map[string]string) {
			if labels == nil {
				labels = func(int) map[string]string { return nil }
			}
			for r := 0; r < n; r++ {
				fx.Facts = append(fx.Facts, fact{
					Tenant: shape.Slug,
					Table:  table,
					Marker: marker(ti, table, r),
					Labels: labels(r),
				})
			}
		}
		profileAt := func(r int) map[string]string {
			return map[string]string{"profile_name": profileName(shape.Slug, r%shape.Profiles)}
		}
		buckets := func(r int) map[string]string {
			return map[string]string{"username": userName(shape.Slug, r%shape.Customers),
				"profile_name": profileName(shape.Slug, r%shape.Profiles),
				"date":         "d" + itoa(r%7+1), "hour": "h" + itoa(r%24+1)}
		}
		add("tenant", 1, nil)
		add("isp_customer", shape.Customers, nil)
		add("radius_user", shape.Customers, profileAt)
		add("isp_subscription", shape.Customers, profileAt)
		add("isp_invoice", shape.Customers*shape.Invoices, nil)
		add("isp_payment", shape.Customers, nil)
		add("isp_trouble_ticket", shape.Tickets, nil)
		add("isp_ipam_pool", shape.Pools, func(r int) map[string]string {
			return map[string]string{"table": "pool-" + itoa(r+1)}
		})
		add("radius_profile", shape.Profiles, func(r int) map[string]string {
			return map[string]string{"profile_name": profileName(shape.Slug, r)}
		})
		add("radius_online", shape.Sessions, buckets)
		add("radius_accounting", shape.Sessions, buckets)
		add("net_monitor_target", 3, nil)
	}
	return fx
}

func profileName(slug string, i int) string {
	return "profile-" + slug + "-" + itoa(i+1)
}

// userName is the subscriber a session row belongs to, which is the dimension a
// per-username statistic groups by.
func userName(slug string, i int) string {
	return "user-" + slug + "-" + itoa(i+1)
}

// labelFor picks the grouping key a cell rule reads.
func (f fact) labelFor(cellKey string) string {
	if cellKey == "tenant" || cellKey == "map-key" {
		return ""
	}
	return f.Labels[cellKey]
}

// cells returns the buckets of one tenant-table-cell rule, with the rows and the summed
// markers each holds. A cell rule of "tenant" is one bucket covering the whole tenant.
func (fx *Fixture) cells(tenant, table, cellKey string) []cellStat {
	grouped := cellKey != "tenant" && cellKey != "map-key"
	groups := map[string]*cellStat{}
	for i := range fx.Facts {
		f := fx.Facts[i]
		if f.Tenant != tenant || f.Table != table {
			continue
		}
		key := ""
		if grouped {
			if cellKey == "row" {
				key = itoa(int(f.Marker))
			} else {
				key = f.labelFor(cellKey)
			}
			if key == "" {
				continue
			}
		}
		cs, ok := groups[key]
		if !ok {
			groups[key] = &cellStat{Cell: key}
			cs = groups[key]
		}
		cs.Rows++
		cs.Sum += f.Marker
	}
	out := make([]cellStat, 0, len(groups))
	for _, c := range groups {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cell < out[j].Cell })
	return out
}

type cellStat struct {
	Cell string
	Rows int
	Sum  int64
}

// minRows is the smallest cell a published value for this field could cover. One is the
// case that matters: a statistic over a single row is that row's own value.
func (fx *Fixture) minRows(table, cellKey string) int {
	best := 0
	for _, t := range fx.Tenants {
		for _, c := range fx.cells(t, table, cellKey) {
			if best == 0 || c.Rows < best {
				best = c.Rows
			}
		}
	}
	return best
}

// singletonTenants counts the fixture tenants that hold at least one cell of one row.
func (fx *Fixture) singletonTenants(table, cellKey string) int {
	set := map[string]bool{}
	for _, t := range fx.Tenants {
		for _, c := range fx.cells(t, table, cellKey) {
			if c.Rows == 1 {
				set[t] = true
			}
		}
	}
	return len(set)
}

// blankTenants counts the tenants for which every cell of this field covers fewer than two
// stored rows, so at floor 2 they would publish nothing at all.
func (fx *Fixture) blankTenants(table, cellKey string) int {
	n := 0
	for _, t := range fx.Tenants {
		cells := fx.cells(t, table, cellKey)
		if len(cells) == 0 {
			n++
			continue
		}
		allSmall := true
		for _, c := range cells {
			if c.Rows >= 2 {
				allSmall = false
			}
		}
		if allSmall {
			n++
		}
	}
	return n
}

// fingerprintable reports whether publishing one value per tenant would give every tenant
// a distinct value - the aggregate then identifies the tenant itself.
func (fx *Fixture) fingerprintable(table string, sum bool) bool {
	seen := map[int64]bool{}
	for _, t := range fx.Tenants {
		var v int64
		for _, f := range fx.Facts {
			if f.Tenant != t || f.Table != table {
				continue
			}
			if sum {
				v += f.Marker
			} else {
				v++
			}
		}
		seen[v] = true
	}
	return len(seen) == len(fx.Tenants)
}

// markersUnique is the fixture's own guarantee: no two stored rows share a value, so a
// published number that equals one can only have come from that row.
func (fx *Fixture) markersUnique() bool {
	seen := map[int64]string{}
	for _, f := range fx.Facts {
		where := f.Tenant + "/" + f.Table
		if prev, dup := seen[f.Marker]; dup && prev != where {
			return false
		}
		seen[f.Marker] = where
	}
	return true
}

// rowTotal is the fixture's population, printed so the memo can state what was measured.
func (fx *Fixture) rowTotal() int { return len(fx.Facts) }

// count is how many stored rows one tenant holds in one table.
func (fx *Fixture) count(tenant, table string) int {
	n := 0
	for _, f := range fx.Facts {
		if f.Tenant == tenant && f.Table == table {
			n++
		}
	}
	return n
}

// tablesUsed lists the distinct tables in the fixture.
func (fx *Fixture) tablesUsed() []string {
	set := map[string]bool{}
	for _, f := range fx.Facts {
		set[f.Table] = true
	}
	return sortedTables(set)
}

// hasTable says whether the fixture can speak about a table at all.
func (fx *Fixture) hasTable(table string) bool {
	for _, t := range fx.tablesUsed() {
		if t == table {
			return true
		}
	}
	return false
}
