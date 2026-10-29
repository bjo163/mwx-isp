package profilewhatif

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth/enhancers"
	"layeh.com/radius"
)

// sharedSecret is a fixture secret. The study never sends the packet anywhere;
// the secret only has to exist because the wire library requires one to build a
// packet.
const sharedSecret = "whatif-fixture-secret"

// registryFile is where the product decides which enhancers run, in which order.
const registryFile = "internal/radiusd/plugins/init.go"

// constructors names the hand-written half of the roster: the product's registry
// file is scanned for the constructor symbols, and this table is what turns a
// symbol into a live enhancer. A symbol the scan finds but the table does not
// know is reported as a gap between EnhancersSeen and EnhancersRun rather than
// being quietly skipped.
var constructors = map[string]func() auth.ResponseEnhancer{
	"NewDefaultAcceptEnhancer":  func() auth.ResponseEnhancer { return enhancers.NewDefaultAcceptEnhancer() },
	"NewHuaweiAcceptEnhancer":   func() auth.ResponseEnhancer { return enhancers.NewHuaweiAcceptEnhancer() },
	"NewH3CAcceptEnhancer":      func() auth.ResponseEnhancer { return enhancers.NewH3CAcceptEnhancer() },
	"NewZTEAcceptEnhancer":      func() auth.ResponseEnhancer { return enhancers.NewZTEAcceptEnhancer() },
	"NewMikrotikAcceptEnhancer": func() auth.ResponseEnhancer { return enhancers.NewMikrotikAcceptEnhancer() },
	"NewIkuaiAcceptEnhancer":    func() auth.ResponseEnhancer { return enhancers.NewIkuaiAcceptEnhancer() },
	"NewArubaAcceptEnhancer":    func() auth.ResponseEnhancer { return enhancers.NewArubaAcceptEnhancer() },
	"NewCiscoAcceptEnhancer":    func() auth.ResponseEnhancer { return enhancers.NewCiscoAcceptEnhancer() },
}

// registeredEnhancerNames reads the constructor symbols in registration order
// straight out of the product's own registry file.
func registeredEnhancerNames(root string) []string {
	src, ok := readFile(filepath.Join(root, filepath.FromSlash(registryFile)))
	if !ok {
		return nil
	}
	const call = "RegisterResponseEnhancer(enhancers."
	var out []string
	for _, ln := range strings.Split(src, "\n") {
		code := blankComments(ln)
		i := strings.Index(code, call)
		if i < 0 {
			continue
		}
		rest := code[i+len(call):]
		j := strings.Index(rest, ")")
		if j < 0 {
			continue
		}
		name := strings.TrimRight(strings.TrimSpace(rest[:j]), "(")
		if strings.HasPrefix(name, "New") {
			out = append(out, name)
		}
	}
	return out
}

// enhancerRoster returns the live enhancers, in the order the product registered
// them.
func enhancerRoster(root string) []auth.ResponseEnhancer {
	var out []auth.ResponseEnhancer
	for _, name := range registeredEnhancerNames(root) {
		if c, ok := constructors[name]; ok {
			out = append(out, c())
		}
	}
	return out
}

// packet runs the product's own enhancer roster over one fixture pair and
// returns the resulting Access-Accept as a ledger, plus the number of profile
// lookups the getters performed.
func (s *study) packet(u *User, nasCode string, cache cacheView) packetResult {
	resp := radius.New(radius.CodeAccessAccept, []byte(sharedSecret))
	ctx := &auth.AuthContext{
		Response: resp,
		User:     u,
		Nas:      VendorFor(nasCode),
		Metadata: map[string]interface{}{"profile_cache": cache},
	}
	prev := map[string]string{}
	producer := map[string]string{}
	for _, e := range s.roster {
		if err := e.Enhance(context.Background(), ctx); err != nil {
			s.failures = append(s.failures, e.Name()+": "+err.Error())
		}
		cur := rawLedger(resp)
		for k, v := range cur {
			if prev[k] != v {
				producer[k] = e.Name()
			}
			prev[k] = v
		}
	}
	return packetResult{attrs: s.ledger(resp, producer), lookups: lookupsOf(cache)}
}
