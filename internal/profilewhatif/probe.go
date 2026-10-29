package profilewhatif

import (
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
)

// enhancerFile is the one enhancer that reads runtime state the fixture cannot
// pin down: the clock through the expiry column and the tenant's own config.
const enhancerFile = "internal/radiusd/plugins/auth/enhancers/default_enhancer.go"

// configRead is one runtime-config read the product performs while building an
// Access-Accept, found by parsing the enhancer source, together with the wire key
// it lands on and the value the study observed.
type configRead struct {
	Symbol   string // the config key constant the source passes in
	Default  string // the fallback literal the source passes in
	Variable string // the local variable the enhancer holds it in
	Key      string // the wire attribute it is written to, joined via the library probe
	Observed string // the value this run actually emitted
	Matched  bool   // observed equals the source default
}

// clockShift is how far the expiry column is moved to find clock-dependent
// attributes. The size does not matter; any shift that moves Session-Timeout
// reveals it.
const clockShift = time.Hour

// computeUnstable derives, by measurement, which attribute keys must be held out
// of the diff ledger: those that move when only the clock moves, and those the
// source writes from a runtime config read.
func (s *study) computeUnstable(users []User, profiles map[int64]*domain.RadiusProfile) []configRead {
	base := findUser(users, "dyn-full")
	if base == nil {
		return nil
	}
	unstable := map[string]bool{}
	for _, code := range vendorCodes() {
		left := cloneUser(base)
		right := cloneUser(base)
		right.ExpireTime = left.ExpireTime.Add(clockShift)
		a := rawValueMap(s.packet(left, code, newReadOnlyCache(profiles)))
		b := rawValueMap(s.packet(right, code, newReadOnlyCache(profiles)))
		for k, v := range b {
			if old, ok := a[k]; ok && old != v {
				unstable[k] = true
			}
		}
	}
	s.unstable = unstable
	s.baseline = s.packet(cloneUser(base), vendors.CodeMikrotik, newReadOnlyCache(profiles)).attrs
	reads := s.detectConfigReads()
	// A read that emitted its own fallback literal is not this study's answer: the
	// product would have written whatever the tenant's config says, so the key joins
	// the clock-driven set and leaves the ledger.
	for _, r := range reads {
		if r.Matched {
			s.unstable[r.Key] = true
		}
	}
	for i := range s.baseline {
		s.baseline[i].Stable = !s.unstable[s.baseline[i].Key]
	}
	return reads
}

// detectConfigReads parses the enhancer source for its runtime-config reads and
// joins each one to the wire attribute it writes. A read whose emitted value
// equals the source's own fallback literal is held out of the ledger, because the
// product would have written whatever the tenant config says instead.
func (s *study) detectConfigReads() []configRead {
	lines := blankLinesOf(enhancerFile, s.root)
	if lines == nil {
		return nil
	}
	var out []configRead
	for _, ln := range lines {
		if !strings.Contains(ln, "getIntConfig(") || !strings.Contains(ln, ":=") {
			continue
		}
		parts := strings.Split(ln, ",")
		if len(parts) < 3 {
			continue
		}
		pair := strings.SplitN(parts[0], ":=", 2)
		if len(pair) != 2 {
			continue
		}
		r := configRead{
			Variable: strings.TrimSpace(pair[0]),
			Symbol:   strings.TrimPrefix(strings.TrimSpace(parts[1]), "app."),
			Default:  strings.TrimRight(strings.TrimSpace(parts[2]), " )"),
		}
		r.Key, r.Observed, r.Matched = s.joinConfigWrite(lines, r.Variable, r.Default)
		out = append(out, r)
	}
	return out
}

// joinConfigWrite finds the line that writes the variable into an attribute and
// maps that attribute's Go symbol to its wire key through the library probe.
func (s *study) joinConfigWrite(lines []string, variable, def string) (string, string, bool) {
	for _, ln := range lines {
		if !strings.Contains(ln, variable) || !strings.Contains(ln, "_Set(") {
			continue
		}
		symbol := setterSymbol(ln)
		if symbol == "" {
			continue
		}
		key := s.keyOfSymbol(symbol)
		if key == "" {
			continue
		}
		observed := s.valueOfKey(key)
		return key, observed, observed == def
	}
	return "", "", false
}

// keyOfSymbol maps a library accessor symbol to the wire key the probe wrote it on.
func (s *study) keyOfSymbol(symbol string) string {
	for key, name := range s.std {
		if name == symbol {
			return key
		}
	}
	return ""
}

// valueOfKey reads the emitted value of one wire key from the baseline ledger.
func (s *study) valueOfKey(key string) string {
	for _, a := range s.baseline {
		if a.Key == key {
			return a.Value
		}
	}
	return ""
}

// setterSymbol returns the accessor symbol of a "rfc2869.AcctInterimInterval_Set"
// style call.
func setterSymbol(ln string) string {
	i := strings.Index(ln, "_Set(")
	if i < 0 {
		return ""
	}
	head := ln[:i]
	start := strings.LastIndexAny(head, ". (")
	if start < 0 {
		return ""
	}
	return head[start+1:]
}

// probeFields changes exactly one profile column at a time for a subscriber that
// inherits everything, once per NAS vendor, and records which wire attributes move.
func (s *study) probeFields(users []User, profiles map[int64]*domain.RadiusProfile) ([]FieldEffect, int, int) {
	base := findUser(users, "dyn-full")
	if base == nil {
		return nil, 0, 0
	}
	var effects []FieldEffect
	probes := 0
	silent := 0
	for _, code := range vendorCodes() {
		before := s.packet(cloneUser(base), code, newReadOnlyCache(profiles))
		for _, f := range Fields() {
			probes++
			edited := cloneProfiles(profiles)
			applyEdit(edited[base.ProfileId], Edit{Field: f.Name, Value: probeValue(f.Name)})
			after := s.packet(cloneUser(base), code, newReadOnlyCache(edited))
			changes := diff(stableOnly(before.attrs), stableOnly(after.attrs))
			if len(changes) == 0 {
				silent++
				continue
			}
			keys := make([]string, 0, len(changes))
			for _, c := range changes {
				keys = append(keys, c.Key)
			}
			sortStrings(keys)
			effects = append(effects, FieldEffect{Field: f.Name, Vendor: code, Keys: keys})
		}
	}
	return effects, probes, silent
}

func rawValueMap(p packetResult) map[string]string {
	out := map[string]string{}
	for _, a := range p.attrs {
		out[a.Key] = a.Value
	}
	return out
}

func cloneUser(u *User) *User {
	c := *u
	return &c
}

// blankLinesOf returns one product file with comments blanked, or nil when the
// file is gone.
func blankLinesOf(rel, root string) []string {
	src, ok := readFile(root + "/" + rel)
	if !ok {
		return nil
	}
	return strings.Split(blankComments(src), "\n")
}
