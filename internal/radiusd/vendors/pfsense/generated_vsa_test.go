package pfsense

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

// TestPfSenseBandwidthIntegerVSARoundTrip pins the wire contract for the
// pfSense VSA dictionary: pfSense-Bandwidth-Max-Up (1) and -Down (2) encode as
// pfSense (13644) integer VSAs and decode the same values back.
func TestPfSenseBandwidthIntegerVSARoundTrip(t *testing.T) {
	p := &radius.Packet{}

	require.NoError(t, PfSenseBandwidthMaxUp_Set(p, 10240))
	require.NoError(t, PfSenseBandwidthMaxDown_Set(p, 51200))

	assert.Equal(t, 2, vsaVendorCount(t, p, _PfSense_VendorID))
	assert.Equal(t, PfSenseBandwidthMaxUp(10240), PfSenseBandwidthMaxUp_Get(p))
	assert.Equal(t, PfSenseBandwidthMaxDown(51200), PfSenseBandwidthMaxDown_Get(p))

	PfSenseBandwidthMaxUp_Del(p)
	PfSenseBandwidthMaxDown_Del(p)
	assert.Equal(t, PfSenseBandwidthMaxUp(0), PfSenseBandwidthMaxUp_Get(p))
	assert.Equal(t, 0, vsaVendorCount(t, p, _PfSense_VendorID), "the pfSense VSAs should be gone after delete")
}

// TestPfSenseVSAVendorIsolation ensures the pfSense accessors ignore a foreign
// vendor's attribute carried in the same packet.
func TestPfSenseVSAVendorIsolation(t *testing.T) {
	p := &radius.Packet{}

	foreign, err := radius.NewVendorSpecific(14988, radius.Attribute{1, 4, 0xAA, 0xBB})
	require.NoError(t, err)
	p.Add(rfc2865.VendorSpecific_Type, foreign)

	assert.Equal(t, 0, vsaVendorCount(t, p, _PfSense_VendorID))
	assert.Equal(t, PfSenseBandwidthMaxUp(0), PfSenseBandwidthMaxUp_Get(p), "a foreign VSA must not be read as a pfSense attribute")
}
