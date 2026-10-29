package profilewhatif

import (
	"strconv"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
)

// User is the product's own subscriber row. The study fabricates it in memory;
// it never reads or writes a database.
type User = domain.RadiusUser

// expireBase is a fixed instant so the emitted Session-Timeout does not move
// between runs. The clock-dependent attribute is held out of the stable ledger
// anyway; pinning the fixture keeps the held-out set itself reproducible.
var expireBase = time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)

// Profiles returns the four fixture profiles, keyed by ID.
func Profiles() map[int64]*domain.RadiusProfile {
	return map[int64]*domain.RadiusProfile{
		1: {ID: 1, NodeId: 1, Name: "bronze", Status: "enabled",
			UpRate: 10240, DownRate: 5120, ActiveNum: 2,
			AddrPool: "pool-bronze", Domain: "bronze.example", RadiusClass: "OU=bronze",
			IPv6PrefixPool: "v6-bronze", DelegatedIpv6PrefixPool: "pd-bronze"},
		2: {ID: 2, NodeId: 1, Name: "silver", Status: "enabled",
			UpRate: 20480, DownRate: 10240, ActiveNum: 4,
			AddrPool: "pool-silver", Domain: "silver.example", RadiusClass: "OU=silver",
			IPv6PrefixPool: "v6-silver", DelegatedIpv6PrefixPool: "pd-silver",
			BindMac: 1, BindVlan: 1},
		3: {ID: 3, NodeId: 1, Name: "gold", Status: "enabled",
			UpRate: 51200, DownRate: 204800, ActiveNum: 8,
			AddrPool: "pool-gold", Domain: "gold.example", RadiusClass: "OU=gold;guest",
			IPv6PrefixPool: "v6-gold", DelegatedIpv6PrefixPool: "pd-gold"},
		4: {ID: 4, NodeId: 1, Name: "unpolished", Status: "enabled",
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA"},
	}
}

// Fixtures returns the subscriber rows the study runs. Each one isolates one
// rule of the inheritance contract, so a scenario can name the rule it tests.
func Fixtures() []User {
	return []User{
		// 1 - every inherited column left unset: the pure inheritance case.
		{ID: 101, Username: "dyn-full", ProfileId: 1, ProfileLinkMode: 1,
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
		// 2 - user rates set while linked dynamically: the override case.
		{ID: 102, Username: "dyn-rate-override", ProfileId: 1, ProfileLinkMode: 1,
			UpRate: 1024, DownRate: 2048,
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
		// 3 - "NA" sentinels on the user: the contract says unset, so it inherits.
		{ID: 103, Username: "dyn-na-sentinel", ProfileId: 2, ProfileLinkMode: 1,
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA", IPv6PrefixPool: "NA",
			ExpireTime: expireBase},
		// 4 - static link mode holding a snapshot of bronze.
		{ID: 104, Username: "static-snapshot", ProfileId: 1, ProfileLinkMode: 0,
			UpRate: 10240, DownRate: 5120, ActiveNum: 2,
			AddrPool: "pool-bronze", Domain: "bronze.example", RadiusClass: "OU=bronze",
			IPv6PrefixPool: "v6-bronze", DelegatedIpv6PrefixPool: "pd-bronze",
			ExpireTime: expireBase},
		// 5 - dynamic link at a profile with no rates at all.
		{ID: 105, Username: "dyn-zero-profile", ProfileId: 4, ProfileLinkMode: 1,
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
		// 6 - MAC present on the user, binding flag left to the profile.
		{ID: 106, Username: "dyn-bind-mac", ProfileId: 2, ProfileLinkMode: 1,
			MacAddr:  "00:11:22:33:44:55",
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
		// 7 - VLANs present on the user, binding flag left to the profile.
		{ID: 107, Username: "dyn-bind-vlan", ProfileId: 2, ProfileLinkMode: 1,
			Vlanid1: 100, Vlanid2: 200,
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
		// 8 - static addresses plus a delegated prefix.
		{ID: 108, Username: "dyn-v6-static", ProfileId: 3, ProfileLinkMode: 1,
			IpAddr: "10.10.10.10", IpV6Addr: "2001:db8::10",
			DelegatedIpv6Prefix: "2001:db8:10::/48",
			AddrPool:            "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
		// 9 - an Aruba subscriber whose VLAN lives only on the user row.
		{ID: 109, Username: "dyn-aruba", ProfileId: 3, ProfileLinkMode: 1, Vlanid1: 120,
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
		// 10 - a Cisco subscriber that names no address pool of its own.
		{ID: 110, Username: "dyn-cisco", ProfileId: 1, ProfileLinkMode: 1,
			AddrPool: "NA", Domain: "NA", RadiusClass: "NA",
			IPv6PrefixPool: "NA", DelegatedIpv6PrefixPool: "NA", ExpireTime: expireBase},
	}
}

// vendorCodes is the NAS vocabulary the study can speak, taken from the vendor
// codes the product registers an enhancer for.
func vendorCodes() []string {
	return []string{
		vendors.CodeMikrotik, vendors.CodeHuawei, vendors.CodeH3C, vendors.CodeZTE,
		vendors.CodeIkuai, vendors.CodeAruba, vendors.CodeCisco, vendors.CodeStandard,
	}
}

// VendorFor builds the NAS row the enhancers match on. Only the vendor code
// reaches an enhancer; the rest is what a real NAS record would carry.
func VendorFor(code string) *domain.NetNas {
	return &domain.NetNas{ID: 9001, Name: "whatif-nas", VendorCode: code, Status: "enabled"}
}

// cloneProfiles deep copies the fixture table so a scenario's edit cannot leak
// into the next scenario.
func cloneProfiles(in map[int64]*domain.RadiusProfile) map[int64]*domain.RadiusProfile {
	out := make(map[int64]*domain.RadiusProfile, len(in))
	for id, p := range in {
		c := *p
		out[id] = &c
	}
	return out
}

// fieldKind reports whether a profile column is numeric, from the same table
// the report prints.
func fieldKind(name string) string {
	for _, f := range Fields() {
		if f.Name == name {
			return f.Kind
		}
	}
	return ""
}

// applyEdit writes one what-if value onto one profile row.
func applyEdit(p *domain.RadiusProfile, e Edit) {
	if p == nil {
		return
	}
	if fieldKind(e.Field) == "int" {
		n, _ := strconv.Atoi(e.Value)
		switch e.Field {
		case "UpRate":
			p.UpRate = n
		case "DownRate":
			p.DownRate = n
		case "ActiveNum":
			p.ActiveNum = n
		case "BindMac":
			p.BindMac = n
		case "BindVlan":
			p.BindVlan = n
		}
		return
	}
	switch e.Field {
	case "AddrPool":
		p.AddrPool = e.Value
	case "Domain":
		p.Domain = e.Value
	case "RadiusClass":
		p.RadiusClass = e.Value
	case "IPv6PrefixPool":
		p.IPv6PrefixPool = e.Value
	case "DelegatedIpv6PrefixPool":
		p.DelegatedIpv6PrefixPool = e.Value
	}
}

// snapshotOnto freezes the inheritance by copying the effective value of every
// inherited column onto the user row, which is what the product does when an
// operator switches a subscriber to static link mode.
func snapshotOnto(u *User, p *domain.RadiusProfile) {
	if p == nil {
		return
	}
	u.UpRate = p.UpRate
	u.DownRate = p.DownRate
	u.ActiveNum = p.ActiveNum
	u.AddrPool = p.AddrPool
	u.Domain = p.Domain
	u.RadiusClass = p.RadiusClass
	u.IPv6PrefixPool = p.IPv6PrefixPool
	u.DelegatedIpv6PrefixPool = p.DelegatedIpv6PrefixPool
	u.BindMac = p.BindMac
	u.BindVlan = p.BindVlan
}

// probeValue is the single what-if value used for every field probe, so one
// edit is always distinguishable from the fixture default.
func probeValue(field string) string {
	switch field {
	case "UpRate":
		return "33333"
	case "DownRate":
		return "44444"
	case "ActiveNum":
		return "7"
	case "BindMac":
		return "1"
	case "BindVlan":
		return "1"
	case "AddrPool":
		return "whatif-pool"
	case "Domain":
		return "whatif.example"
	case "RadiusClass":
		return "OU=whatif"
	case "IPv6PrefixPool":
		return "whatif-v6"
	case "DelegatedIpv6PrefixPool":
		return "whatif-pd"
	}
	return ""
}

// Scenarios are the questions the study answers. Each one names the inheritance
// rule it exercises.
func Scenarios() []Scenario {
	return []Scenario{
		{ID: "S01", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "UpRate", Value: "33333"},
			Question: "profile upload rate 10240 -> 33333, subscriber inherits it",
			Why:      "the plain case an operator expects a preview for"},
		{ID: "S02", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "DownRate", Value: "44444"},
			Question: "profile download rate 5120 -> 44444",
			Why:      "download is a separate column and a separate attribute"},
		{ID: "S03", Username: "dyn-full", Vendor: vendors.CodeHuawei,
			Edit:     &Edit{Field: "UpRate", Value: "33333"},
			Question: "same upload edit, Huawei NAS",
			Why:      "Huawei emits an average and a peak per direction"},
		{ID: "S04", Username: "dyn-full", Vendor: vendors.CodeH3C,
			Edit:     &Edit{Field: "DownRate", Value: "44444"},
			Question: "same download edit, H3C NAS",
			Why:      "H3C uses its own VSA numbers for the same two columns"},
		{ID: "S05", Username: "dyn-full", Vendor: vendors.CodeZTE,
			Edit:     &Edit{Field: "UpRate", Value: "33333"},
			Question: "same upload edit, ZTE NAS",
			Why:      "ZTE emits only two rate attributes"},
		{ID: "S06", Username: "dyn-full", Vendor: vendors.CodeIkuai,
			Edit:     &Edit{Field: "UpRate", Value: "33333"},
			Question: "same upload edit, iKuai NAS",
			Why:      "iKuai scales by 1024*8, so the same column moves a different number"},
		{ID: "S07", Username: "dyn-rate-override", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "UpRate", Value: "33333"},
			Question: "profile upload edited while the user overrides upload",
			Why:      "precedence: the user value must win, so nothing should move"},
		{ID: "S08", Username: "dyn-na-sentinel", Vendor: vendors.CodeHuawei,
			Edit:     &Edit{Field: "Domain", Value: "whatif.example"},
			Question: "profile domain edited while the user carries the NA sentinel",
			Why:      "NA means unset, so the subscriber must follow the profile"},
		{ID: "S09", Username: "dyn-cisco", Vendor: vendors.CodeCisco,
			Edit:     &Edit{Field: "AddrPool", Value: "whatif-pool"},
			Question: "profile address pool renamed, Cisco NAS",
			Why:      "the pool reaches both a standard and a vendor attribute"},
		{ID: "S10", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "RadiusClass", Value: "OU=whatif"},
			Question: "profile Class renamed",
			Why:      "RFC 2865 Class is emitted verbatim"},
		{ID: "S11", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "IPv6PrefixPool", Value: "whatif-v6"},
			Question: "profile SLAAC pool renamed",
			Why:      "Framed-IPv6-Pool and the PD pool are different columns"},
		{ID: "S12", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "DelegatedIpv6PrefixPool", Value: "whatif-pd"},
			Question: "profile DHCPv6-PD pool renamed",
			Why:      "RFC 6911 keeps this distinct from the SLAAC pool"},
		{ID: "S13", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			MoveTo:   3,
			Question: "subscriber re-linked from bronze to gold",
			Why:      "the whole ledger moves at once, the common sales upsell"},
		{ID: "S14", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			ToStatic: true, Edit: &Edit{Field: "UpRate", Value: "33333"},
			Question: "detach to static, then edit the profile",
			Why:      "a snapshot must freeze the answer, so the diff should be empty"},
		{ID: "S15", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			Clear:    true,
			Question: "linked profile no longer resolvable at authentication time",
			Why:      "the getters fall back to the stale user row"},
		{ID: "S16", Username: "dyn-full", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "ActiveNum", Value: "7"},
			Question: "profile concurrent-session ceiling raised",
			Why:      "no enhancer reads it, so the accept says the same thing"},
		{ID: "S17", Username: "dyn-bind-mac", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "BindMac", Value: "1"},
			Question: "profile MAC binding turned on",
			Why:      "binding gates acceptance, it is not an attribute"},
		{ID: "S18", Username: "dyn-bind-vlan", Vendor: vendors.CodeAruba,
			Edit:     &Edit{Field: "BindVlan", Value: "1"},
			Question: "profile VLAN binding turned on, Aruba NAS",
			Why:      "Aruba reads the user VLAN directly, never the profile"},
		{ID: "S19", Username: "dyn-aruba", Vendor: vendors.CodeAruba,
			Edit:     &Edit{Field: "Domain", Value: "whatif.example"},
			Question: "profile domain edited, Aruba NAS",
			Why:      "Aruba reuses the domain column as the user role"},
		{ID: "S20", Username: "dyn-zero-profile", Vendor: vendors.CodeMikrotik,
			Edit:     &Edit{Field: "UpRate", Value: "33333"},
			Question: "first rate ever written on a profile nothing had rates on",
			Why:      "zero to nonzero is the edge an operator trips over"},
	}
}
