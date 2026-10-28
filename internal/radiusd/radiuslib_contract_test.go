package radiusd

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

// ipv6PrefixAttribute builds the wire form of RFC 3162 Framed-IPv6-Prefix and
// Delegated-IPv6-Prefix: reserved byte, prefix length, then 16 address octets.
func ipv6PrefixAttribute(t *testing.T, prefixLen byte, ip net.IP) radius.Attribute {
	t.Helper()
	v6 := ip.To16()
	require.NotNil(t, v6, "fixture address must be IPv6")
	attr := make(radius.Attribute, 2+net.IPv6len)
	attr[1] = prefixLen
	copy(attr[2:], v6)
	return attr
}

// TestRadiusLibraryToleratesIPv6PrefixPaddingBits pins the behavior of the
// pinned fork github.com/talkincode/radius v0.1.0: RFC 3162 (§2.3) requires the
// bits after the prefix length to be zero, but equipment commonly sends a full
// interface identifier with a /64 or leaves a trailing bit set on a non-octet
// prefix. The fork masks those bits; upstream layeh/radius rejects the
// attribute with "invalid prefix data". Enhancing Access-Accept with the framed
// or delegated prefix (huawei dictionary, auth enhancers) therefore depends on
// this fork behavior, so re-pinning to the upstream commit must fail here.
func TestRadiusLibraryToleratesIPv6PrefixPaddingBits(t *testing.T) {
	t.Run("host bits behind a /64", func(t *testing.T) {
		attr := ipv6PrefixAttribute(t, 64, net.ParseIP("2001:db8:1:2:aaaa:bbbb:cccc:dddd"))

		prefix, err := radius.IPv6Prefix(attr)
		require.NoError(t, err)
		assert.Equal(t, "2001:db8:1:2::/64", prefix.String())
	})

	t.Run("trailing bit behind a non-octet prefix", func(t *testing.T) {
		// /61 keeps five bits of the eighth octet (0x02), so the remaining three
		// bits are padding and must be cleared rather than rejected.
		attr := ipv6PrefixAttribute(t, 61, net.ParseIP("2001:db8:1:2:aaaa:bbbb:cccc:dddd"))

		prefix, err := radius.IPv6Prefix(attr)
		require.NoError(t, err)
		assert.Equal(t, "2001:db8:1::/61", prefix.String())
	})

	t.Run("malformed input is still rejected", func(t *testing.T) {
		_, err := radius.IPv6Prefix(ipv6PrefixAttribute(t, 129, net.ParseIP("2001:db8::1")))
		assert.Error(t, err, "an out-of-range prefix length must not be tolerated")

		_, err = radius.IPv6Prefix(radius.Attribute{64})
		assert.Error(t, err, "a prefix attribute without a length octet must not be tolerated")
	})
}
