// Package profilewhatif is the FUT-007 research prototype: it answers "what would
// this Access-Accept say if the profile changed" by running the product's own
// inheritance getters and response enhancers over in-memory fixtures, and reports
// the wire-level diff.
//
// It is read-only in the strongest sense the tree allows. The cache it injects has
// exactly one method, it holds no database handle, and the user and profile rows it
// studies are compared before and after every run to prove nothing moved. Nothing in
// the product imports it, and it reaches no NAS - the package that declares the CoA
// and Disconnect clients is not in its import set, which is a test, not a claim.
//
// The report it produces is a what-if ledger, not a decision: three of the ten
// inherited columns change whether a session is accepted rather than what the accept
// says, and this package can only show the second half.
package profilewhatif

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
)

// Field is one profile column an operator can edit, named exactly as the domain
// struct names it, with the column the management API writes.
type Field struct {
	Name    string
	Column  string
	Kind    string // "int" or "string"
	Note    string
	Reaches string // which product getter resolves it during authentication
}

// Fields is the profile vocabulary the estimator can change: every inherited column
// the domain getters resolve, each pinned in claims.go to its getter.
func Fields() []Field {
	return []Field{
		{Name: "UpRate", Column: "up_rate", Kind: "int", Reaches: "GetUpRate",
			Note: "kilobits, emitted per vendor as its own rate attribute"},
		{Name: "DownRate", Column: "down_rate", Kind: "int", Reaches: "GetDownRate",
			Note: "kilobits, emitted per vendor as its own rate attribute"},
		{Name: "ActiveNum", Column: "active_num", Kind: "int", Reaches: "GetActiveNum",
			Note: "concurrent-session ceiling read by a checker, not by any enhancer"},
		{Name: "AddrPool", Column: "addr_pool", Kind: "string", Reaches: "GetAddrPool",
			Note: "NA and empty string both mean unset"},
		{Name: "Domain", Column: "domain", Kind: "string", Reaches: "GetDomain",
			Note: "read as Huawei-Domain-Name and reused as Aruba-User-Role"},
		{Name: "RadiusClass", Column: "radius_class", Kind: "string", Reaches: "GetClass",
			Note: "RFC 2865 Class, emitted verbatim when set"},
		{Name: "IPv6PrefixPool", Column: "ipv6_prefix_pool", Kind: "string", Reaches: "GetIPv6PrefixPool",
			Note: "Framed-IPv6-Pool, the SLAAC side"},
		{Name: "DelegatedIpv6PrefixPool", Column: "delegated_ipv6_prefix_pool", Kind: "string", Reaches: "GetDelegatedIPv6PrefixPool",
			Note: "DHCPv6-PD pool, a different column with a different attribute"},
		{Name: "BindMac", Column: "bind_mac", Kind: "int", Reaches: "GetBindMac",
			Note: "gates acceptance, never reaches the accept's attributes"},
		{Name: "BindVlan", Column: "bind_vlan", Kind: "int", Reaches: "GetBindVlan",
			Note: "gates acceptance, never reaches the accept's attributes"},
	}
}

// Edit is one change to one profile column.
type Edit struct {
	Field string
	Value string
}

// Scenario is one what-if question about one fixture subscriber.
type Scenario struct {
	ID       string
	Question string
	Why      string
	Username string
	Vendor   string
	Edit     *Edit
	MoveTo   int64
	ToStatic bool
	Clear    bool // the linked profile cannot be resolved
}

// Attr is one attribute on a rendered Access-Accept, identified by what the wire
// actually says rather than by a hand-written name table.
type Attr struct {
	Key      string // "14988-8" for a vendor attribute, "type-51" for a standard one
	Name     string // dictionary identifier, or "" when nothing in the tree names it
	Type     int
	Vendor   int
	VendorTy int
	Value    string
	Producer string // which registered enhancer wrote it, measured by running them one at a time
	Stable   bool   // false when the value comes from the clock or from tenant config
}

// String renders the entry for the report.
func (a Attr) String() string {
	name := a.Key
	if a.Name != "" {
		name = a.Key + " " + a.Name
	}
	return name + " = " + a.Value
}

// Change is one line of a before/after diff.
type Change struct {
	Key    string
	Kind   string // added | removed | changed
	Before string
	After  string
	From   Attr
}

// Outcome is one scenario run against one NAS vendor.
type Outcome struct {
	Scenario string
	Question string
	Vendor   string
	Before   []Attr
	After    []Attr
	Changes  []Change
	Lookups  int
	HeldOut  []Attr
}

// FieldEffect records which wire attributes moved when only one profile column
// changed on a subscriber that inherits everything.
type FieldEffect struct {
	Field  string
	Vendor string
	Keys   []string
}

// Totals carries the numbers the memo quotes.
type Totals struct {
	Users           int
	Profiles        int
	Vendors         int
	Scenarios       int
	ScenariosMoved  int
	Fields          int
	Effects         int
	Probes          int
	SilentProbes    int
	Changes         int
	Lookups         int
	MaxLookups      int
	StableAttrs     int
	HeldOutKeys     int
	DictEntries     int
	DictSources     int
	EnhancersRun    int
	EnhancersSeen   int
	Imports         int
	BannedHits      int
	ConfigReads     int
	ConfigMatched   int
	CheckerOnly     int
	Claims          int
	ClaimsConfirmed int
	Gaps            int
	GapsOpen        int
	StdNames        int
	NoCacheTests    int
	SendCalls       int
	CoAPackage      string
}

// Result is one full run of the study.
type Result struct {
	Totals      Totals
	Digest      string
	Outcomes    []Outcome
	Effects     []FieldEffect
	Producers   []Attr
	HeldOut     []Attr
	Names       map[string]string
	ConfigReads []configRead
	Claims      []ClaimStatus
	Gaps        []GapStatus
	Banned      []string
	Imports     []string
}

// study holds the measured inputs every run shares: the roster read from the
// product's registry, the dictionaries, and the keys held out of the ledger.
type study struct {
	root     string
	roster   []auth.ResponseEnhancer
	dict     map[string]string
	std      map[string]string
	names    map[string]string
	unstable map[string]bool
	baseline []Attr
	failures []string
}

// Run executes the whole study against the repository rooted at root, over freshly
// built fixtures.
func Run(root string) (*Result, error) {
	return run(root, Fixtures(), Profiles())
}

// run is the same study over caller-supplied tables. The non-mutation test hands it
// its own copies, so a scenario that wrote through to the shared table would be
// caught rather than absorbed by the next Run.
func run(root string, users []User, profiles map[int64]*domain.RadiusProfile) (*Result, error) {
	s := &study{root: root, roster: enhancerRoster(root)}
	s.dict = vendorDict(root)
	s.std = stdNames()
	s.names = map[string]string{}
	for k, v := range s.dict {
		s.names[k] = v
	}
	for k, v := range s.std {
		s.names[k] = v
	}
	reads := s.computeUnstable(users, profiles)

	effects, probes, silent := s.probeFields(users, profiles)

	var outcomes []Outcome
	lookups, maxLookups, changes, moved := 0, 0, 0, 0
	for _, sc := range Scenarios() {
		base := findUser(users, sc.Username)
		if base == nil {
			continue
		}
		user := cloneUser(base)
		code := sc.Vendor
		if code == "" {
			code = vendors.CodeStandard
		}
		before := s.packet(cloneUser(user), code, newReadOnlyCache(profiles))

		afterProfiles := cloneProfiles(profiles)
		if sc.MoveTo != 0 {
			user.ProfileId = sc.MoveTo
		}
		if sc.ToStatic {
			user.ProfileLinkMode = 0
			snapshotOnto(user, afterProfiles[user.ProfileId])
		}
		if sc.Edit != nil {
			applyEdit(afterProfiles[user.ProfileId], *sc.Edit)
		}
		var view cacheView = newReadOnlyCache(afterProfiles)
		if sc.Clear {
			view = brokenCache{}
		}
		after := s.packet(cloneUser(user), code, view)

		o := Outcome{
			Scenario: sc.ID, Question: sc.Question, Vendor: code,
			Before:  stableOnly(before.attrs),
			After:   stableOnly(after.attrs),
			Changes: diff(stableOnly(before.attrs), stableOnly(after.attrs)),
			Lookups: before.lookups + after.lookups,
			HeldOut: heldOutOf(before.attrs, after.attrs),
		}
		lookups += o.Lookups
		if before.lookups > maxLookups {
			maxLookups = before.lookups
		}
		if after.lookups > maxLookups {
			maxLookups = after.lookups
		}
		changes += len(o.Changes)
		if len(o.Changes) > 0 {
			moved++
		}
		outcomes = append(outcomes, o)
	}

	imports := packageImports(ownPackageDir(root))
	banned := bannedPaths(banRules())
	checkerOnly := checkerOnlyFields(root)
	held := s.heldOutAttrs()

	res := &Result{
		Outcomes:    outcomes,
		Effects:     effects,
		Producers:   s.baseline,
		HeldOut:     held,
		Names:       s.names,
		ConfigReads: reads,
		Imports:     imports,
		Banned:      banned,
		Totals: Totals{
			Users: len(users), Profiles: len(profiles), Vendors: len(vendorCodes()),
			Scenarios: len(Scenarios()), ScenariosMoved: moved,
			Fields: len(Fields()), Effects: len(effects),
			Probes: probes, SilentProbes: silent,
			Changes: changes, Lookups: lookups, MaxLookups: maxLookups,
			StableAttrs: len(stableOnly(s.baseline)), HeldOutKeys: len(s.unstable),
			DictEntries: len(s.dict), DictSources: dictSourceCount(root), StdNames: len(s.std),
			EnhancersRun: len(s.roster), EnhancersSeen: len(registeredEnhancerNames(root)),
			Imports: len(imports), BannedHits: countBanned(imports, banRules()),
			SendCalls:   isImportSend(root),
			ConfigReads: len(reads), ConfigMatched: countMatched(reads),
			CheckerOnly:  len(checkerOnly),
			NoCacheTests: len(rateTestFilesWithoutCache(root)),
			CoAPackage:   coaDeclaringPackage(root),
		},
	}
	res.Claims = verifyClaims(root)
	res.Gaps = verifyGaps(root)
	for _, c := range res.Claims {
		if c.Confirmed {
			res.Totals.ClaimsConfirmed++
		}
	}
	res.Totals.Claims = len(res.Claims)
	for _, g := range res.Gaps {
		if g.Open {
			res.Totals.GapsOpen++
		}
	}
	res.Totals.Gaps = len(res.Gaps)
	res.Digest = digest(res)
	return res, nil
}

func countMatched(reads []configRead) int {
	n := 0
	for _, r := range reads {
		if r.Matched {
			n++
		}
	}
	return n
}

// packetResult is one run of the enhancer roster over one fixture pair.
type packetResult struct {
	attrs   []Attr
	lookups int
}

// stableOnly drops the held-out attributes and sorts what is left.
func stableOnly(attrs []Attr) []Attr {
	var out []Attr
	for _, a := range attrs {
		if a.Stable {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// heldOutAttrs lists the attributes the ledger refuses to diff, with the value this
// run observed.
func (s *study) heldOutAttrs() []Attr {
	var out []Attr
	seen := map[string]bool{}
	for _, a := range s.baseline {
		if a.Stable || seen[a.Key] {
			continue
		}
		seen[a.Key] = true
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// heldOutOf returns the clock- or config-driven attributes of a packet pair.
func heldOutOf(pairs ...[]Attr) []Attr {
	seen := map[string]bool{}
	var out []Attr
	for _, list := range pairs {
		for _, a := range list {
			if a.Stable || seen[a.Key] {
				continue
			}
			seen[a.Key] = true
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// diff compares two sorted ledgers by attribute key.
func diff(before, after []Attr) []Change {
	index := func(list []Attr) map[string]Attr {
		m := map[string]Attr{}
		for _, a := range list {
			m[a.Key] = a
		}
		return m
	}
	b, a := index(before), index(after)
	var keys []string
	for k := range b {
		keys = append(keys, k)
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			keys = append(keys, k)
		}
	}
	sortStrings(keys)
	var out []Change
	for _, k := range keys {
		old, hadOld := b[k]
		nw, hasNew := a[k]
		switch {
		case !hadOld:
			out = append(out, Change{Key: k, Kind: "added", After: nw.Value, From: nw})
		case !hasNew:
			out = append(out, Change{Key: k, Kind: "removed", Before: old.Value, From: old})
		case old.Value != nw.Value:
			out = append(out, Change{Key: k, Kind: "changed", Before: old.Value, After: nw.Value, From: nw})
		}
	}
	return out
}

// digest is the reproducibility fingerprint: the same fixtures over the same code
// must produce the same string.
func digest(r *Result) string {
	var b strings.Builder
	for _, o := range r.Outcomes {
		b.WriteString(o.Scenario + "\t" + o.Vendor + "\n")
		for _, c := range o.Changes {
			b.WriteString("\t" + c.Kind + " " + c.Key + " " + c.Before + " -> " + c.After + "\n")
		}
		b.WriteString("\tlookups " + itoa(o.Lookups) + "\n")
	}
	for _, e := range r.Effects {
		b.WriteString(e.Field + "\t" + e.Vendor + "\t" + strings.Join(e.Keys, ",") + "\n")
	}
	for _, a := range r.Producers {
		if !a.Stable {
			continue
		}
		b.WriteString("ledger\t" + a.Key + "\t" + a.Name + "\t" + a.Value + "\t" + a.Producer + "\n")
	}
	// The held-out rows keep their keys but not their values: a value the study
	// refuses to promise cannot be part of the promise it repeats.
	for _, a := range r.HeldOut {
		b.WriteString("heldout\t" + a.Key + "\n")
	}
	for _, c := range r.Claims {
		b.WriteString("claim\t" + c.ID + "\t" + c.Status() + "\t" + c.File + ":" + itoa(c.Line) + "\n")
	}
	for _, g := range r.Gaps {
		b.WriteString("gap\t" + g.ID + "\t" + g.Evidence + "\t" + boolString(g.Open) + "\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func boolString(v bool) string {
	if v {
		return "open"
	}
	return "closed"
}

func findUser(users []User, username string) *User {
	for i := range users {
		if users[i].Username == username {
			u := users[i]
			return &u
		}
	}
	return nil
}
