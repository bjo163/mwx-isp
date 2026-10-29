package profilewhatif

import "strings"

// ClaimStatus is one statement the memo makes, pinned to a code line. Line is the
// 1-based line found now; 0 means the citation has drifted ("broken") and -1
// means the file is gone ("vacated").
type ClaimStatus struct {
	ID        string
	Statement string
	File      string
	Line      int
	Measured  string
	Confirmed bool
}

// Status names the drift state for the report.
func (c ClaimStatus) Status() string {
	switch {
	case c.Line < 0:
		return "vacated"
	case c.Line == 0:
		return "broken"
	case c.Confirmed:
		return "confirmed"
	default:
		return "contradicted"
	}
}

// GapStatus is one thing this study does not close, with the measurement that
// shows it is still open.
type GapStatus struct {
	ID       string
	Gap      string
	Evidence string
	Open     bool
}

type claimSpec struct {
	id, statement, file string
	tokens              []string
	measure             func(*evidence) string
	holds               func(*evidence) bool
}

// evidence carries the numbers the claims are written from, computed once.
type evidence struct {
	root              string
	getters           int
	enhancerReads     map[string]int
	checkerReads      map[string]int
	inheritedDirect   []string
	profileHandlers   []string
	cacheReadsAcct    int
	invalidateCalls   int
	setCalls          int
	coPkg             string
	registryLines     int
	dictSources       int
	stdDictSites      int
	previewEndpoints  int
	rateTestsNoCache  int
	checkerOnly       int
	probeSymbolHits   int
	probeSymbolsTotal int
	naSentinelGetters int
	fallbackLines     int
}

func gatherEvidence(root string) *evidence {
	e := &evidence{root: root}
	e.getters = getterCount(root)
	e.enhancerReads = map[string]int{}
	e.checkerReads = map[string]int{}
	for _, f := range Fields() {
		e.enhancerReads[f.Name] = readsColumnInEnhancers(root, "."+f.Reaches)
		e.checkerReads[f.Name] = readsColumnInCheckers(root, "."+f.Reaches)
	}
	e.cacheReadsAcct = accountingCacheReads(root)
	e.inheritedDirect = inheritedColumnsReadDirect(root)
	e.profileHandlers = profileAPIHandlers(root)
	e.invalidateCalls = invalidateSites(root)
	e.setCalls = countSubstring(corpus(goFiles(root, adminapiDir, false, false)), "ProfileCache().Set(")
	e.coPkg = coaDeclaringPackage(root)
	e.registryLines = len(registeredEnhancerNames(root))
	e.dictSources = dictSourceCount(root)
	e.stdDictSites = countToken(corpus(productFiles(root)), "Framed-Pool")
	e.previewEndpoints = countSubstring(corpus(goFiles(root, adminapiDir, false, false)), "what-if") +
		countSubstring(corpus(goFiles(root, adminapiDir, false, false)), "WhatIf")
	e.rateTestsNoCache = len(rateTestFilesWithoutCache(root))
	e.checkerOnly = len(checkerOnlyFields(root))
	e.probeSymbolsTotal = len(stdProbes)
	e.probeSymbolHits = probeSymbolsInProduct(root)
	e.naSentinelGetters = countSubstring([][]string{productLines(root, gettersFile)}, `!= "NA"`)
	e.fallbackLines = countToken([][]string{productLines(root, gettersFile)}, "failed to get profile from cache")
	return e
}

// probeSymbolsInProduct counts how many of the standard-attribute names the report
// uses are setters the product itself calls, so no name is invented by the study.
func probeSymbolsInProduct(root string) int {
	n := 0
	for _, pr := range stdProbes {
		if countSubstring(corpus(enhancerFiles(root)), "."+pr.symbol+"_") > 0 {
			n++
		}
	}
	return n
}

// verifyClaims checks every statement the memo makes against the code that is
// there now.
func verifyClaims(root string) []ClaimStatus {
	e := gatherEvidence(root)
	specs := []claimSpec{
		{"C01", "the domain exposes one inheritance getter per column the study can change",
			gettersFile, []string{"func (u *RadiusUser) Get"},
			func(*evidence) string { return itoa(e.getters) }, nil},
		{"C02", "a getter only consults the profile when the subscriber is linked dynamically",
			gettersFile, []string{"u.ProfileLinkMode == ProfileLinkModeDynamic"}, nil, nil},
		{"C03", "a nonzero user value beats the profile",
			gettersFile, []string{"u.UpRate > 0"}, nil, nil},
		{"C04", "an empty string and NA both mean unset, so the subscriber inherits",
			gettersFile, []string{"u.AddrPool !=", "NA"},
			func(*evidence) string { return itoa(e.naSentinelGetters) + " getters test the sentinel" }, nil},
		{"C05", "when the profile cannot be read the getter returns the stale user value and logs",
			gettersFile, []string{"failed to get profile from cache"},
			func(*evidence) string { return itoa(e.fallbackLines) + " fallback log sites" }, nil},
		{"C06", "static link mode is 0 and dynamic is 1",
			gettersFile, []string{"ProfileLinkModeStatic"}, nil, nil},
		{"C07", "Mikrotik renders both rates as one string attribute",
			"internal/radiusd/plugins/auth/enhancers/mikrotik_enhancer.go",
			[]string{"MikrotikRateLimit_SetString"}, nil, nil},
		{"C08", "Huawei emits a peak as four times the average",
			"internal/radiusd/plugins/auth/enhancers/huawei_enhancer.go",
			[]string{"upPeak", "clampInt64"}, nil, nil},
		{"C09", "iKuai scales the same column by 1024*8",
			"internal/radiusd/plugins/auth/enhancers/ikuai_enhancer.go",
			[]string{"1024*8"}, nil, nil},
		{"C10", "Aruba reads the subscriber's own VLAN column, never a profile getter",
			"internal/radiusd/plugins/auth/enhancers/aruba_enhancer.go",
			[]string{"user.Vlanid1"}, nil, nil},
		{"C11", "the default enhancer derives Session-Timeout from the expiry column",
			enhancerFile, []string{"SessionTimeout_Set"}, nil, nil},
		{"C12", "the default enhancer writes Acct-Interim-Interval from a runtime config read",
			enhancerFile, []string{"AcctInterimInterval_Set"}, nil, nil},
		{"C13", "the profile cache has a five minute default TTL",
			cacheFile, []string{"DefaultProfileCacheTTL"}, nil, nil},
		{"C14", "the cache is keyed by profile id alone",
			cacheFile, []string{"map[int64]*profileCacheEntry"}, nil, nil},
		{"C15", "the profile update endpoint invalidates the cache after writing",
			profileAPI, []string{"ProfileCache().Invalidate"},
			func(*evidence) string { return itoa(e.invalidateCalls) + " invalidate sites in adminapi" }, nil},
		{"C16", "no production caller warms or writes the cache; only invalidation is used",
			profileAPI, []string{"ProfileCache().Invalidate"},
			func(*evidence) string { return itoa(e.setCalls) + " write-through call sites" },
			func(e *evidence) bool { return e.setCalls == 0 }},
		{"C17", "the profile cache is injected on the authentication side and never on the accounting side",
			runnerFile, []string{"profile_cache", "ProfileCache()"},
			func(*evidence) string { return itoa(e.cacheReadsAcct) + " cache reads under plugins/accounting" },
			func(ev *evidence) bool { return ev.cacheReadsAcct == 0 }},
		{"C18", "the CoA and Disconnect clients live in a package this prototype does not import",
			coaFile, []string{") CoA("},
			func(*evidence) string { return e.coPkg }, nil},
		{"C19", "the roster the study runs is the order the product registers",
			registryFile, []string{"RegisterResponseEnhancer"},
			func(*evidence) string { return itoa(e.registryLines) + " registrations" }, nil},
		{"C20", "the runner hands the real profile cache to the same metadata key the study injects",
			runnerFile, []string{"profile_cache", "ProfileCache()"}, nil, nil},
		{"C21", "every standard attribute name in the report is a setter the product calls",
			enhancerFile, []string{"_Set("},
			func(*evidence) string {
				return itoa(e.probeSymbolHits) + "/" + itoa(e.probeSymbolsTotal) + " probe symbols found in product code"
			},
			func(e *evidence) bool { return e.probeSymbolHits == e.probeSymbolsTotal }},
		{"C22", "the cache type the product injects has mutators; the interface the getters accept does not",
			gettersFile, []string{"ProfileCacheGetter"},
			func(*evidence) string {
				return itoa(len(cacheMutators(e.root))) + " mutators on ProfileCache: " +
					strings.Join(cacheMutators(e.root), ", ")
			}, nil},
		{"C23", "an enhancer reaches every inherited column through a getter, never as a field",
			"internal/radiusd/plugins/auth/enhancers/mikrotik_enhancer.go",
			[]string{"user.GetUpRate", "profileCache"},
			func(*evidence) string {
				return itoa(len(e.inheritedDirect)) +
					" inherited columns read as fields anywhere in the enhancer tree"
			},
			func(e *evidence) bool { return len(e.inheritedDirect) == 0 }},
	}
	out := make([]ClaimStatus, 0, len(specs))
	for _, s := range specs {
		c := ClaimStatus{ID: s.id, Statement: s.statement, File: s.file}
		c.Line = anchor(root, s.file, s.tokens...)
		if s.measure != nil {
			c.Measured = s.measure(e)
		}
		c.Confirmed = c.Line > 0
		if s.holds != nil {
			c.Confirmed = c.Line > 0 && s.holds(e)
		}
		out = append(out, c)
	}
	return out
}

// verifyGaps records what this study does not close.
func verifyGaps(root string) []GapStatus {
	e := gatherEvidence(root)
	return []GapStatus{
		{"G01", "the management API has no what-if or preview route for a profile edit",
			itoa(e.previewEndpoints) + " preview endpoints among the " +
				itoa(len(e.profileHandlers)) + " profile handlers: " +
				strings.Join(e.profileHandlers, ", "), e.previewEndpoints == 0},
		{"G02", "no product test runs a rate-bearing vendor enhancer with a profile cache",
			itoa(e.rateTestsNoCache) + " rate-asserting enhancer test files without a cache",
			e.rateTestsNoCache > 0},
		{"G03", "three inherited columns change whether a session is accepted, not what the accept says",
			itoa(e.checkerOnly) + " columns read by a checker and by no enhancer", e.checkerOnly > 0},
		{"G04", "the study ships no standard-attribute dictionary of its own",
			itoa(e.stdDictSites) + " code sites naming a standard attribute by its display name",
			e.stdDictSites == 0},
		{"G05", "no NAS is reachable from CI, so the ledger is never confirmed on real hardware",
			"asset-blocked: needs a lab NAS per vendor", true},
		{"G06", "re-authorization after a profile edit is out of reach",
			e.coPkg + " owns CoA and is not in the prototype import set", true},
		{"G07", "how many live subscribers a profile edit actually touches is unknown",
			"asset-blocked: needs production session counts", true},
	}
}
