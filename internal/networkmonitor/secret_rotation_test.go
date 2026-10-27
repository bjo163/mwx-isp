package networkmonitor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/securestore"
)

func sealSecret(t *testing.T, key, value string) []byte {
	t.Helper()
	sealed, err := securestore.Seal(key, []byte(value))
	require.NoError(t, err)
	return sealed
}

func rotationTarget(name, probeType, version string, community, auth, privacy []byte) domain.NetMonitorTarget {
	return domain.NetMonitorTarget{
		Name: name, Kind: "router", Address: "192.0.2.10", ProbeType: probeType,
		SNMPVersion: version, SNMPCommunityEncrypted: community,
		SNMPAuthEncrypted: auth, SNMPPrivacyEncrypted: privacy,
		IntervalSeconds: 60, TimeoutMilliseconds: 1000, FailureThreshold: 2,
	}
}

func TestRotateSNMPSecretsDryRunKeepsCiphertext(t *testing.T) {
	db := monitorTestDB(t)
	const oldSecret, newSecret = "old-web-secret", "new-web-secret"
	target := rotationTarget("edge-router", "snmp", "v2c", sealSecret(t, oldSecret, "community-ro"), nil, nil)
	require.NoError(t, db.Create(&target).Error)

	report, err := RotateSNMPSecrets(context.Background(), db, oldSecret, newSecret, false)
	require.NoError(t, err)
	assert.Equal(t, SecretRotationReport{Targets: 1, Fields: 1}, report)

	var stored domain.NetMonitorTarget
	require.NoError(t, db.First(&stored, target.ID).Error)
	assert.Equal(t, target.SNMPCommunityEncrypted, stored.SNMPCommunityEncrypted,
		"dry run writes must be rolled back")
	plaintext, err := securestore.Open(oldSecret, stored.SNMPCommunityEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "community-ro", string(plaintext))
}

func TestRotateSNMPSecretsReencryptsAllCredentialColumns(t *testing.T) {
	db := monitorTestDB(t)
	const oldSecret, newSecret = "old-web-secret", "new-web-secret"

	v2c := rotationTarget("v2c-router", "snmp", "v2c", sealSecret(t, oldSecret, "community-ro"), nil, nil)
	v3 := rotationTarget("v3-router", "snmp", "v3", nil,
		sealSecret(t, oldSecret, "auth-pass"), sealSecret(t, oldSecret, "privacy-pass"))
	icmp := rotationTarget("plain-host", "icmp", "", nil, nil, nil)
	for _, target := range []*domain.NetMonitorTarget{&v2c, &v3, &icmp} {
		require.NoError(t, db.Create(target).Error)
	}

	report, err := RotateSNMPSecrets(context.Background(), db, oldSecret, newSecret, true)
	require.NoError(t, err)
	assert.Equal(t, SecretRotationReport{Targets: 2, Fields: 3}, report)

	var reloadedV2C domain.NetMonitorTarget
	require.NoError(t, db.First(&reloadedV2C, v2c.ID).Error)
	community, err := securestore.Open(newSecret, reloadedV2C.SNMPCommunityEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "community-ro", string(community))
	_, err = securestore.Open(oldSecret, reloadedV2C.SNMPCommunityEncrypted)
	assert.Error(t, err, "the old secret must no longer open the rotated value")

	var reloadedV3 domain.NetMonitorTarget
	require.NoError(t, db.First(&reloadedV3, v3.ID).Error)
	auth, err := securestore.Open(newSecret, reloadedV3.SNMPAuthEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "auth-pass", string(auth))
	privacy, err := securestore.Open(newSecret, reloadedV3.SNMPPrivacyEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "privacy-pass", string(privacy))

	var reloadedICMP domain.NetMonitorTarget
	require.NoError(t, db.First(&reloadedICMP, icmp.ID).Error)
	assert.Empty(t, reloadedICMP.SNMPCommunityEncrypted)
	assert.Empty(t, reloadedICMP.SNMPAuthEncrypted)
	assert.Empty(t, reloadedICMP.SNMPPrivacyEncrypted)
}

func TestRotateSNMPSecretsAbortsWithoutPartialWrites(t *testing.T) {
	db := monitorTestDB(t)
	const oldSecret, newSecret = "old-web-secret", "new-web-secret"

	first := rotationTarget("first-router", "snmp", "v2c", sealSecret(t, oldSecret, "community-ro"), nil, nil)
	// A row sealed with a secret the operator did not supply makes the whole
	// rotation impossible; it must not leave the earlier rows half-migrated.
	stale := rotationTarget("stale-router", "snmp", "v2c", sealSecret(t, "another-secret", "stale-community"), nil, nil)
	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&stale).Error)

	_, err := RotateSNMPSecrets(context.Background(), db, oldSecret, newSecret, true)
	require.ErrorContains(t, err, "cannot be decrypted with the old secret")
	assert.NotContains(t, err.Error(), "community-ro")
	assert.NotContains(t, err.Error(), oldSecret)

	var stored domain.NetMonitorTarget
	require.NoError(t, db.First(&stored, first.ID).Error)
	assert.Equal(t, first.SNMPCommunityEncrypted, stored.SNMPCommunityEncrypted,
		"the transaction must roll back every row when one row fails")
}

func TestRotateSNMPSecretsRejectsInvalidSecrets(t *testing.T) {
	db := monitorTestDB(t)
	for _, tc := range []struct {
		name     string
		old, new string
		want     string
	}{
		{name: "empty old", old: "", new: "new-web-secret", want: "required"},
		{name: "empty new", old: "old-web-secret", new: "", want: "required"},
		{name: "identical", old: "same-web-secret", new: "same-web-secret", want: "identical"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RotateSNMPSecrets(context.Background(), db, tc.old, tc.new, true)
			require.ErrorContains(t, err, tc.want)
		})
	}

	report, err := RotateSNMPSecrets(context.Background(), db, "old-web-secret", "new-web-secret", true)
	require.NoError(t, err)
	assert.Equal(t, SecretRotationReport{}, report, "an empty inventory rotates nothing")
}
