package profilewhatif

import (
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
	"layeh.com/radius/rfc3162"
	"layeh.com/radius/rfc4818"
	"layeh.com/radius/rfc6911"
)

// vendorDir is where the product keeps one generated dictionary per vendor.
const vendorDir = "internal/radiusd/vendors"

// attributeTypeVendor is the RADIUS type that wraps a vendor-specific attribute
// (RFC 2865 section 5.26). It is the only number this file hardcodes, and it is
// a protocol constant, not a naming choice.
const attributeTypeVendor = 26

// entryOf identifies one attribute exactly as the wire does. A vendor attribute
// becomes "<vendor-id>-<vendor-type>", everything else becomes "type-<n>".
func entryOf(avp *radius.AVP) (key, value string, typ, vendor, vendorTy int) {
	typ = int(avp.Type)
	raw := []byte(avp.Attribute)
	if typ == attributeTypeVendor && len(raw) >= 6 {
		vendor = int(be32(raw[:4]))
		vendorTy = int(raw[4])
		key = strconv.Itoa(vendor) + "-" + strconv.Itoa(vendorTy)
		return key, render(raw[6:]), typ, vendor, vendorTy
	}
	key = "type-" + strconv.Itoa(typ)
	return key, render(raw), typ, 0, 0
}

// rawLedger collapses a packet to key -> value. A key that appears more than once
// keeps the last instance; the ledger is a diff surface, not a serializer.
func rawLedger(p *radius.Packet) map[string]string {
	out := map[string]string{}
	for _, avp := range p.Attributes {
		k, v, _, _, _ := entryOf(avp)
		out[k] = v
	}
	return out
}

// ledger turns a packet into the report's attribute rows, naming each key from
// the dictionaries the study measured and flagging the ones held out of the diff.
func (s *study) ledger(p *radius.Packet, producer map[string]string) []Attr {
	var out []Attr
	for _, avp := range p.Attributes {
		k, v, typ, vendor, vty := entryOf(avp)
		out = append(out, Attr{
			Key: k, Name: s.names[k], Type: typ, Vendor: vendor, VendorTy: vty,
			Value: v, Producer: producer[k], Stable: !s.unstable[k],
		})
	}
	return out
}

// render makes a payload printable by one rule, applied to every attribute the
// same way: text stays text, a four-byte number prints as a decimal, anything
// else prints as hex.
func render(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	plain := true
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			plain = false
			break
		}
	}
	switch {
	case plain:
		return string(b)
	case len(b) == 4:
		return strconv.FormatUint(uint64(be32(b)), 10)
	default:
		return hex.EncodeToString(b)
	}
}

func be32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// vendorDict builds key -> dictionary identifier from the product's own generated
// vendor dictionaries. Nothing in this map was typed by hand.
func vendorDict(root string) map[string]string {
	out := map[string]string{}
	for _, path := range dictFiles(root) {
		src, ok := readFile(path)
		if !ok {
			continue
		}
		vendor := 0
		current := ""
		for _, ln := range blankLines(src) {
			if t := between(ln, "_VendorID = ", ""); t != "" {
				if n, err := strconv.Atoi(t); err == nil {
					vendor = n
				}
				continue
			}
			if strings.HasPrefix(ln, "func ") {
				current = attributeName(ln)
				continue
			}
			if vendor == 0 || current == "" {
				continue
			}
			if t := vendorAddType(ln); t > 0 {
				key := strconv.Itoa(vendor) + "-" + strconv.Itoa(t)
				if _, seen := out[key]; !seen {
					out[key] = current
				}
			}
		}
	}
	return out
}

// dictFiles lists the generated dictionary sources the product ships.
func dictFiles(root string) []string {
	var out []string
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(vendorDir)))
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(vendorDir), e.Name(), "generated.go")
		if _, ok := readFile(p); ok {
			out = append(out, p)
		}
	}
	return out
}

// dictSourceCount is how many generated dictionaries the names came from.
func dictSourceCount(root string) int { return len(dictFiles(root)) }

// attributeName extracts the dictionary identifier from a generated accessor's
// signature line, dropping the package prefix and the accessor suffix.
func attributeName(ln string) string {
	body := strings.TrimPrefix(ln, "func ")
	if i := strings.Index(body, "("); i > 0 {
		body = body[:i]
	}
	i := strings.LastIndex(body, "_")
	if i <= 0 {
		return ""
	}
	suffix := body[i+1:]
	switch suffix {
	case "Add", "AddString", "AddBytes", "AddIP", "Set", "SetString", "SetBytes", "SetIP":
		return body[:i]
	default:
		return ""
	}
}

// vendorAddType reads the vendor attribute number out of a generated helper call
// such as "return _Mikrotik_AddVendor(p, 8, a)".
func vendorAddType(ln string) int {
	for _, call := range []string{"_AddVendor(p, ", "_SetVendor(p, "} {
		i := strings.Index(ln, call)
		if i < 0 {
			continue
		}
		rest := ln[i+len(call):]
		j := strings.Index(rest, ",")
		if j < 0 {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(rest[:j])); err == nil {
			return n
		}
	}
	return 0
}

// stdProbe is one setter from the wire library the product itself calls. The
// attribute number is never typed here: the probe writes the attribute and reads
// the number back off the packet.
type stdProbe struct {
	symbol string
	set    func(*radius.Packet)
}

var stdProbes = []stdProbe{
	{"SessionTimeout", func(p *radius.Packet) { _ = rfc2865.SessionTimeout_Set(p, rfc2865.SessionTimeout(1)) }},
	{"Class", func(p *radius.Packet) { _ = rfc2865.Class_SetString(p, "x") }},
	{"FramedIPAddress", func(p *radius.Packet) { _ = rfc2865.FramedIPAddress_Set(p, net.IP{10, 0, 0, 1}) }},
	{"AcctInterimInterval", func(p *radius.Packet) { _ = rfc2869.AcctInterimInterval_Set(p, rfc2869.AcctInterimInterval(1)) }},
	{"FramedPool", func(p *radius.Packet) { _ = rfc2869.FramedPool_SetString(p, "x") }},
	{"FramedIPv6Prefix", func(p *radius.Packet) { _ = rfc3162.FramedIPv6Prefix_Set(p, cidr("2001:db8::/64")) }},
	{"FramedIPv6Pool", func(p *radius.Packet) { _ = rfc3162.FramedIPv6Pool_SetString(p, "x") }},
	{"DelegatedIPv6Prefix", func(p *radius.Packet) { _ = rfc4818.DelegatedIPv6Prefix_Set(p, cidr("2001:db8:1::/48")) }},
	{"FramedIPv6Address", func(p *radius.Packet) { _ = rfc6911.FramedIPv6Address_Set(p, net.ParseIP("2001:db8::1")) }},
	{"DelegatedIPv6PrefixPool", func(p *radius.Packet) { _ = rfc6911.DelegatedIPv6PrefixPool_SetString(p, "x") }},
}

func cidr(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		return nil
	}
	return n
}

// stdNames probes the wire library for the number behind every standard
// attribute the product's default enhancer can set.
func stdNames() map[string]string {
	out := map[string]string{}
	for _, pr := range stdProbes {
		p := radius.New(radius.CodeAccessAccept, []byte(sharedSecret))
		pr.set(p)
		for _, avp := range p.Attributes {
			k, _, _, _, _ := entryOf(avp)
			out[k] = pr.symbol
		}
	}
	return out
}

// blankLines returns the file with every comment replaced by spaces, one entry
// per line, so a symbol named only in prose cannot be mistaken for code.
func blankLines(src string) []string {
	return strings.Split(blankComments(src), "\n")
}

// between trims src around prefix and to the end of the line.
func between(src, prefix, suffix string) string {
	i := strings.Index(src, prefix)
	if i < 0 {
		return ""
	}
	rest := src[i+len(prefix):]
	if suffix != "" {
		if j := strings.Index(rest, suffix); j >= 0 {
			rest = rest[:j]
		}
	}
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	return strings.TrimSpace(rest)
}
