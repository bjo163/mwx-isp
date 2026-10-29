package opstats

import (
	"sort"
	"strings"
)

// personalTables hold a subscriber's or an invoice's own row. A statistic over one of
// these describes a person; radius_profile, isp_ipam_pool, net_monitor_target and tenant
// describe a thing the operator owns, which is why the two classes are scored apart.
var personalTables = []string{
	"isp_customer",
	"isp_subscription",
	"isp_invoice",
	"isp_payment",
	"isp_trouble_ticket",
	"radius_user",
	"radius_online",
	"radius_accounting",
}

// exposure is one published number measured against the fixture.
type exposure struct {
	Surface string
	Key     string
	// GoName is the struct field the key is served from, so a claim about a composition
	// can be re-read against the assignment itself.
	GoName    string
	Audience  string
	Measure   string
	Verb      string
	Table     string
	Cell      string
	MinRows   int
	Singleton int
	// Blank counts the fixture tenants for which this field would publish nothing at
	// all once a suppression floor is applied - the utility half of the trade-off.
	Blank int
	// Verdict is the answer to the criterion for this one field.
	Verdict     string
	Personal    bool
	Denominator string
	Fingerprint bool
	Line        int
	Proven      string
}

// floors are the suppression thresholds the study prices. A floor of k withholds every
// cell that covers fewer than k stored rows.
var floors = []int{1, 2, 3, 5}

// measure runs the fixture against every aggregate field the product serves.
func measure(rows []fieldRow, fx *Fixture) []exposure {
	out := make([]exposure, 0, len(rows))
	for _, r := range rows {
		switch r.Measure {
		case "count", "money", "bytes", "rate":
		default:
			continue
		}
		e := exposure{
			Surface:  r.Surface,
			Key:      r.Key,
			GoName:   r.Go,
			Audience: r.Audience,
			Measure:  r.Measure,
			Verb:     r.Verb,
			Table:    r.Table,
			Cell:     r.Cell,
			Line:     r.Line,
			Proven:   r.Proven,
			Personal: contains(personalTables, r.Table),
		}
		// A ratio is priced through the two numbers it divides, not through a table the
		// study deliberately leaves unattributed.
		if r.Measure == "rate" {
			e.Denominator = r.Ratio
			e.Verdict = verdict(e)
			out = append(out, e)
			continue
		}
		if r.Table == "" || !fx.hasTable(r.Table) {
			e.Verdict = "no source table attributed"
			out = append(out, e)
			continue
		}
		e.MinRows = fx.minRows(r.Table, r.Cell)
		if e.MinRows == 0 {
			// The query was found and its table too, but the fixture has no row that the
			// cell rule groups - the study says so rather than inventing a population.
			e.Verdict = "the fixture groups no " + r.Table + " row by " + orNone(r.Cell)
			out = append(out, e)
			continue
		}
		e.Singleton = fx.singletonTenants(r.Table, r.Cell)
		e.Blank = fx.blankTenants(r.Table, r.Cell)
		e.Fingerprint = fx.fingerprintable(r.Table, r.Measure == "money" || r.Measure == "bytes")
		e.Verdict = verdict(e)
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Surface != out[j].Surface {
			return out[i].Surface < out[j].Surface
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// verdict applies the criterion's test to one field. The wording is the finding: an
// absolute number over a single row is that row's own value, and a count of one over a
// subscriber table says the tenant has exactly one.
func verdict(e exposure) string {
	if e.Measure == "rate" {
		switch {
		case e.Denominator == "":
			return "a ratio no served number is composed from"
		case strings.Contains(e.Denominator, unresolvedOperand):
			return "a ratio with an operand no served key answers to: " + e.Denominator
		default:
			return "a count in disguise - it divides " + e.Denominator + ", both served in the same response"
		}
	}
	if !e.Personal {
		if e.MinRows == 1 {
			return "one device or catalog row, not a subscriber"
		}
		return "aggregate over a thing the operator owns"
	}
	if e.MinRows == 1 && (e.Measure == "money" || e.Measure == "bytes") {
		return "publishes one stored value verbatim"
	}
	if e.MinRows == 1 {
		return "publishes that the tenant has exactly one"
	}
	return "aggregate over " + itoa(e.MinRows) + " stored rows"
}

// floorRow is one suppression threshold priced across every exposure.
type floorRow struct {
	K          int
	Exposed    int
	Suppressed int
	BlankAll   int
	Fields     int
}

// priceFloors answers the question a design decision actually needs: how high must the
// cell floor be before no fixture value survives, and how much of the tenant base does
// that take with it.
func priceFloors(exps []exposure) []floorRow {
	var out []floorRow
	for _, k := range floors {
		fr := floorRow{K: k, Fields: len(exps)}
		for _, e := range exps {
			if e.MinRows == 0 {
				continue
			}
			if e.MinRows < k {
				fr.Suppressed++
				continue
			}
			if reconstructs(e) {
				fr.Exposed++
			}
		}
		out = append(out, fr)
	}
	// The last column is the blunt half: at floor k, how many fields lose every value
	// for at least one tenant.
	for i := range out {
		k := out[i].K
		n := 0
		for _, e := range exps {
			if e.MinRows == 0 || e.MinRows >= k {
				continue
			}
			if e.Blank > 0 {
				n++
			}
		}
		out[i].BlankAll = n
	}
	return out
}

func reconstructs(e exposure) bool {
	if !e.Personal || e.MinRows != 1 {
		return false
	}
	return e.Measure == "money" || e.Measure == "bytes" || e.Measure == "count"
}

func contains(list []string, s string) bool {
	if s == "" {
		return false
	}
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
