package networkmonitor

import (
	"errors"
	"net/netip"
	"strings"
)

// limitedBroadcast is the IPv4 limited-broadcast address, which netip reports
// as neither multicast nor unspecified.
var limitedBroadcast = netip.MustParseAddr("255.255.255.255")

// CheckTargetAddress reports why addr cannot be used as a probe destination.
//
// A target must be an explicit IP literal. Hostnames are refused rather than
// resolved: a name can resolve to a different address than the one validated
// here (DNS rebinding), and a reachability probe is not a name resolver.
//
// Unspecified, loopback, link-local, multicast, "this network" and broadcast
// addresses are refused: they are never a monitored device, and loopback and
// link-local are where an SSRF attempt aims (the daemon's own management port,
// a cloud metadata endpoint). Private IPv4 ranges and IPv6 unique-local
// addresses stay allowed because the network an ISP monitors is private.
func CheckTargetAddress(addr string) error {
	ip, err := netip.ParseAddr(strings.TrimSpace(addr))
	if err != nil {
		return errors.New("address must be one explicit unicast IP address")
	}
	ip = ip.Unmap()
	switch {
	case ip.IsUnspecified(), ip.IsMulticast(), ip == limitedBroadcast, ip.Is4() && ip.As4()[0] == 0:
		return errors.New("address must be one explicit unicast IP address")
	case ip.IsLoopback():
		return errors.New("address must not be a loopback address")
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return errors.New("address must not be link-local")
	}
	return nil
}
