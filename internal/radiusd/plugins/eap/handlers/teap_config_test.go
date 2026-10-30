package handlers

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
)

// teapReader selects the managed server certificate the tunnel providers build
// their config from, with the client CA also named so the EAP-TLS provider
// produces a config in the same table.
func teapReader(values map[string]string) TLSSettingsReader {
	base := map[string]string{"radius." + SettingEapTlsServerCert: "srv", "radius." + SettingEapTlsClientCa: "ca"}
	for k, v := range values {
		base[k] = v
	}
	return newFakeReader(base)
}

func TestNewSettingsTEAPConfigProvider_NotConfigured(t *testing.T) {
	certPEM, keyPEM := genTestCertPEM(t)
	resolver := &fakeCertResolver{certPEM: certPEM, keyPEM: keyPEM}

	cases := []struct {
		name     string
		reader   TLSSettingsReader
		resolver CertResolver
	}{
		{name: "nil reader", reader: nil, resolver: resolver},
		{name: "resolver but no ref", reader: newFakeReader(map[string]string{}), resolver: resolver},
		{name: "ref but no resolver", reader: newFakeReader(map[string]string{"radius." + SettingEapTlsServerCert: "srv"}), resolver: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewSettingsTEAPConfigProvider(tc.reader, tc.resolver)()
			require.NoError(t, err)
			assert.Nil(t, cfg, "an unconfigured tunnel must reject rather than open a tunnel with no certificate")
		})
	}
}

// TestTunnelProvidersKeepTheirOwnTLSBounds is the M11.2 regression pin: TEAP gets
// a TLS 1.3 tunnel of its own, and the three tunnels that were already serving are
// recorded here field by field so a later change to any of them has to be a
// decision rather than a side effect.
func TestTunnelProvidersKeepTheirOwnTLSBounds(t *testing.T) {
	certPEM, keyPEM := genTestCertPEM(t)
	resolver := &fakeCertResolver{certPEM: certPEM, keyPEM: keyPEM, caPEM: certPEM}
	reader := teapReader(nil)

	for _, tc := range []struct {
		name       string
		provider   TLSConfigProvider
		serverOnly bool
		clientCAs  bool
		maxVersion uint16
		tickets    bool
	}{
		{name: "EAP-TLS", provider: NewSettingsTLSConfigProvider(reader, resolver), serverOnly: false, clientCAs: true},
		{name: "PEAP", provider: NewSettingsPEAPConfigProvider(reader, resolver), serverOnly: true},
		{name: "EAP-TTLS", provider: NewSettingsTTLSConfigProvider(reader, resolver), serverOnly: true, maxVersion: tls.VersionTLS12},
		{name: "TEAP", provider: NewSettingsTEAPConfigProvider(reader, resolver), serverOnly: true, tickets: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := tc.provider()
			require.NoError(t, err)
			require.NotNil(t, cfg)

			assert.Equal(t, tc.serverOnly, cfg.ServerOnly)
			assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion, "the floor is the configured default for every tunnel")
			assert.Equal(t, tc.maxVersion, cfg.MaxVersion, "only a method that needs the cap may carry it")
			assert.Equal(t, tc.tickets, cfg.EnableSessionTickets)

			if tc.tickets {
				assert.Len(t, cfg.SessionTicketKey, tlsengine.SessionTicketKeySize, "a tunnel that resumes needs a key to seal its tickets")
			} else {
				assert.Empty(t, cfg.SessionTicketKey, "a tunnel that never resumes has no ticket to seal")
			}

			assert.Equal(t, tc.clientCAs, cfg.ClientCAs != nil, "only EAP-TLS verifies an outer client certificate")
			assert.Nil(t, cfg.CipherSuites, "the default profile leaves crypto/tls to choose modern suites")
		})
	}
}

func TestNewSettingsTEAPConfigProvider_HonorsTheConfiguredFloor(t *testing.T) {
	certPEM, keyPEM := genTestCertPEM(t)
	resolver := &fakeCertResolver{certPEM: certPEM, keyPEM: keyPEM}

	cfg, err := NewSettingsTEAPConfigProvider(teapReader(map[string]string{
		"radius." + SettingEapTlsMinVersion: "1.3",
	}), resolver)()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
	assert.Zero(t, cfg.MaxVersion)
}

func TestNewSettingsTEAPConfigProvider_PolicyFailuresSurface(t *testing.T) {
	ecCert, ecKey := genTestCertPEM(t)
	rsaCert, rsaKey := genRSATestCertPEM(t)

	cases := []struct {
		name     string
		cert     []byte
		key      []byte
		values   map[string]string
		resolver CertResolver
		want     error
	}{
		{
			name:   "legacy cipher profile needs an RSA certificate",
			cert:   ecCert,
			key:    ecKey,
			values: map[string]string{"radius." + SettingEapTlsCipherProfile: "legacy-rsa-cbc"},
			want:   tlsengine.ErrLegacyNeedsRSACertificate,
		},
		{
			name: "legacy cipher profile cannot be paired with a TLS 1.3 floor",
			cert: rsaCert,
			key:  rsaKey,
			values: map[string]string{
				"radius." + SettingEapTlsMinVersion:    "1.3",
				"radius." + SettingEapTlsCipherProfile: "legacy-rsa-cbc",
			},
			want: tlsengine.ErrLegacyIncompatibleTLS13,
		},
		{
			name:     "managed certificate cannot be resolved",
			resolver: &fakeCertResolver{err: errResolver},
			want:     errResolver,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := tc.resolver
			if resolver == nil {
				resolver = &fakeCertResolver{certPEM: tc.cert, keyPEM: tc.key}
			}
			_, err := NewSettingsTEAPConfigProvider(teapReader(tc.values), resolver)()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

// TestTEAPProviderKeepsOneTicketKeyAcrossHandshakes pins why the provider owns the
// key instead of letting crypto/tls randomize it: the engine builds a new tls.Config
// for every EAP exchange, so two handshakes from this provider must be sealed with
// the same key or the peer can never present its ticket back.
func TestTEAPProviderKeepsOneTicketKeyAcrossHandshakes(t *testing.T) {
	certPEM, keyPEM := genTestCertPEM(t)
	provider := NewSettingsTEAPConfigProvider(teapReader(nil), &fakeCertResolver{certPEM: certPEM, keyPEM: keyPEM})

	first, err := provider()
	require.NoError(t, err)
	require.NotNil(t, first)
	second, err := provider()
	require.NoError(t, err)
	require.NotNil(t, second)

	require.Len(t, first.SessionTicketKey, tlsengine.SessionTicketKeySize)
	assert.Equal(t, first.SessionTicketKey, second.SessionTicketKey, "a key per handshake would seal tickets no later handshake can open")
}

// TestTEAPTunnelNegotiatesTLS13AndResumes is the M11.2 claim end to end: the outer
// tunnel built from the provider's config negotiates TLS 1.3, hands the peer the
// single NewSessionTicket RFC 9190 §2.1.2 asks of a server that means to resume,
// and resumes the next handshake with it, which is how RFC 9930 §2 replaces the
// deprecated PAC.
func TestTEAPTunnelNegotiatesTLS13AndResumes(t *testing.T) {
	cfg := teapTunnelConfig(t)
	cache := tls.NewLRUClientSessionCache(4)

	first := driveTunnelHandshake(t, cfg, cache)
	assert.Equal(t, uint16(tls.VersionTLS13), first.Version)
	assert.False(t, first.DidResume, "the initial authentication is a full handshake")

	second := driveTunnelHandshake(t, cfg, cache)
	assert.Equal(t, uint16(tls.VersionTLS13), second.Version)
	assert.True(t, second.DidResume, "the ticket issued in the initial authentication must resume the next session")
}

// TestExistingTunnelsKeepTheirWireBehavior drives the same handshake pair for the two
// tunnels that existed before TEAP: EAP-TTLS stays on TLS 1.2, and neither PEAP nor
// EAP-TTLS spends an EAP round on a ticket, so neither can resume.
func TestExistingTunnelsKeepTheirWireBehavior(t *testing.T) {
	certPEM, keyPEM := genTestCertPEM(t)
	resolver := &fakeCertResolver{certPEM: certPEM, keyPEM: keyPEM}

	for _, tc := range []struct {
		name     string
		provider TLSConfigProvider
		version  uint16
	}{
		{name: "PEAP", provider: NewSettingsPEAPConfigProvider(teapReader(nil), resolver), version: tls.VersionTLS13},
		{name: "EAP-TTLS", provider: NewSettingsTTLSConfigProvider(teapReader(nil), resolver), version: tls.VersionTLS12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := tc.provider()
			require.NoError(t, err)
			require.NotNil(t, cfg)
			cfg.HandshakeTimeout = 5 * time.Second

			cache := tls.NewLRUClientSessionCache(4)
			if state := driveTunnelHandshake(t, cfg, cache); state.Version != tc.version {
				t.Fatalf("negotiated %#x, want %#x", state.Version, tc.version)
			}
			if state := driveTunnelHandshake(t, cfg, cache); state.DidResume {
				t.Fatal("a tunnel that issues no ticket must not resume")
			}
		})
	}
}

// teapTunnelConfig builds the TEAP provider's config from an in-memory managed
// certificate, so the test drives exactly what the serving path would hand the
// engine.
func teapTunnelConfig(t *testing.T) *tlsengine.Config {
	t.Helper()
	certPEM, keyPEM := genTestCertPEM(t)
	resolver := &fakeCertResolver{certPEM: certPEM, keyPEM: keyPEM}

	cfg, err := NewSettingsTEAPConfigProvider(teapReader(nil), resolver)()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	cfg.HandshakeTimeout = 5 * time.Second
	return cfg
}

// driveTunnelHandshake runs a server-side tunnel handshake against a real crypto/tls
// peer and returns the peer's view of it, which is where a TLS 1.3 ticket is
// observable: the server sends it inside an encrypted application-data record. The
// peer skips certificate verification because the material under test is the
// self-signed managed certificate; what the exchange is asked to prove is which
// version the tunnel selects and whether its ticket resumes.
func driveTunnelHandshake(t *testing.T, cfg *tlsengine.Config, cache tls.ClientSessionCache) tls.ConnectionState {
	t.Helper()

	eng, err := tlsengine.New(cfg)
	require.NoError(t, err)
	defer func() { _ = eng.Close() }()

	toClient, fromClient := newHSStream(), newHSStream()
	client := tls.Client(&hsConn{rd: toClient, wr: fromClient}, &tls.Config{
		//nolint:gosec // G402: the test certificate is self-signed; the tunnel settings are what is under test
		InsecureSkipVerify: true,
		ServerName:         "radius.example.com",
		MinVersion:         tls.VersionTLS12,
		ClientSessionCache: cache,
	})
	clientDone := make(chan error, 1)
	go func() { clientDone <- client.Handshake() }()

	buf := make([]byte, 16*1024)
	var in []byte
	for round := 0; round < 16; round++ {
		out, done, err := eng.Process(in)
		if len(out) > 0 {
			_, werr := toClient.Write(out)
			require.NoError(t, werr)
		}
		if done {
			break
		}
		require.NoError(t, err, "tunnel handshake round %d", round)

		n, rerr := fromClient.Read(buf)
		require.NoError(t, rerr)
		in = append([]byte(nil), buf[:n]...)
	}

	select {
	case err := <-clientDone:
		require.NoError(t, err, "peer handshake")
	case <-time.After(2 * time.Second):
		t.Fatal("peer handshake did not return")
	}

	// A TLS 1.3 peer stores the ticket only while it reads from the tunnel, so close
	// the server-to-peer direction and read to end of stream: the ticket is either
	// there or it is not, and the read finishes either way.
	toClient.close()
	for {
		if _, rerr := client.Read(buf); rerr != nil {
			break
		}
	}

	state := client.ConnectionState()
	_ = client.Close()
	return state
}
