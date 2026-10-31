package hillstone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// vsaVendorCount returns how many Vendor-Specific attributes on p carry the
// given IANA enterprise number. The generated codec packs each VSA under its
// own RFC 2865 attribute-26 envelope, so this counts only this dictionary's
// attributes and ignores foreign vendors.
func vsaVendorCount(t *testing.T, p *radius.Packet, wantVendorID uint32) int {
	t.Helper()
	var n int
	for _, avp := range p.Attributes {
		if avp.Type != rfc2865.VendorSpecific_Type {
			continue
		}
		vendorID, _, err := radius.VendorSpecific(avp.Attribute)
		require.NoError(t, err)
		if vendorID == wantVendorID {
			n++
		}
	}
	return n
}

// TestHillstoneVPNPoolStringVSARoundTrip pins the wire contract for the
// Hillstone VSA dictionary: Hillstone-VPN-DHCP-Pool (102) encodes under a
// Hillstone (28557) Vendor-Specific attribute and decodes the same string back.
func TestHillstoneVPNPoolStringVSARoundTrip(t *testing.T) {
	p := &radius.Packet{}

	const value = "pool-h"
	require.NoError(t, HillstoneVPNDHCPPool_SetString(p, value))

	assert.Equal(t, 1, vsaVendorCount(t, p, _Hillstone_VendorID))
	assert.Equal(t, value, HillstoneVPNDHCPPool_GetString(p))

	HillstoneVPNDHCPPool_Del(p)
	assert.Equal(t, "", HillstoneVPNDHCPPool_GetString(p))
	assert.Equal(t, 0, vsaVendorCount(t, p, _Hillstone_VendorID), "the Hillstone VSA should be gone after delete")
}

// TestHillstoneVSAVendorIsolation ensures the Hillstone accessors ignore a
// foreign vendor's attribute carried in the same packet.
func TestHillstoneVSAVendorIsolation(t *testing.T) {
	p := &radius.Packet{}

	foreign, err := radius.NewVendorSpecific(14988, radius.Attribute{1, 4, 0xAA, 0xBB})
	require.NoError(t, err)
	p.Add(rfc2865.VendorSpecific_Type, foreign)

	assert.Equal(t, 0, vsaVendorCount(t, p, _Hillstone_VendorID))
	assert.Equal(t, "", HillstoneVPNDHCPPool_GetString(p), "a foreign VSA must not be read as a Hillstone attribute")
	assert.Equal(t, "", HillstoneUserRoleBame_GetString(p), "a foreign VSA must not be read as a Hillstone attribute")
}
