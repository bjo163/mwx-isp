package handlers

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapinner"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
)

// SettingEapTeapPasswordPrompt is the RADIUS setting that turns on the
// Basic-Password-Auth exchange of RFC 9930 §3.6.3 as the second Inner Method of a
// TEAP session. Its value is the prompt shown to the user; an empty value keeps
// Phase 2 to the inner EAP-TLS method alone.
const SettingEapTeapPasswordPrompt = "EapTeapPasswordPrompt"

// TEAPOptions configures the Phase 2 half of a TEAP handler.
//
// Phase 2 is opt-in because it needs materials the outer tunnel does not: an
// inner TLS configuration with a client CA (the Inner Method that proves a
// machine certificate) and, when the site wants a second factor, the prompt of
// the Basic-Password-Auth exchange. A handler built without options keeps its
// Phase 1 behavior and fails closed at the Phase 2 gate.
type TEAPOptions struct {
	// InnerTLSConfigProvider supplies the TLS material of the inner EAP-TLS
	// method. Unlike the outer tunnel this configuration must authenticate the
	// peer, so it needs a client CA; the provider is consulted once per inner
	// handshake.
	InnerTLSConfigProvider TLSConfigProvider
	// PasswordPrompt returns the prompt of the Basic-Password-Auth-Req TLV. An
	// empty string leaves password authentication out of the plan, which is the
	// safe default: RFC 9930 §3.6.3 requires the first request to carry the
	// prompt the user is asked for.
	PasswordPrompt func() string
	// Policy tightens or relaxes the Inner Method checks of RFC 9930 §6.2.3. The
	// zero value selects teapinner.DefaultPolicy, the strict reading.
	Policy teapinner.Policy
}

// NewSettingsTEAPConfigProvider returns a TLSConfigProvider for the outer tunnel
// of RFC 9930 (TEAP Version 1).
//
// The tunnel is server-authenticated: TEAP proves the peer with its Inner Methods
// and the Crypto-Binding TLV inside the tunnel, so the outer handshake asks for no
// client certificate and needs no client CA, exactly as PEAP and EAP-TTLS do. It
// differs from them in two places, and RFC 9930 is why.
//
// First, it caps nothing. EAP-TTLS pins its tunnel to TLS 1.2 because that method's
// phase-2 completion relies on the TLS 1.2 final-flight framing, but RFC 9930 §1
// says of its own definitions that they "are only applicable for TLS 1.2 and
// TLS 1.3", so a TEAP tunnel that cannot reach 1.3 implements half of the transport
// it was specified for. MaxVersion therefore stays unset and the negotiation is
// bounded only by the operator's radius.EapTlsMinVersion.
//
// Second, it enables session tickets. RFC 9930 §2 deprecates the Protected Access
// Credential that RFC 7170 borrowed from EAP-FAST and resumes a TEAP session with
// the TLS 1.3 NewSessionTicket message instead, and RFC 9190 §2.1.2 requires a
// server that means to resume to issue the ticket during the initial
// authentication. crypto/tls meets that section's two conditions by construction:
// it sends one ticket per connection with the 604800-second lifetime the section
// caps the lifetime at, and a ticket it issues carries no early-data advertisement,
// which the section forbids.
//
// A ticket is only worth sending if a later handshake can open it, so the provider
// generates one ticket key when it is constructed and passes it to every handshake
// it configures. crypto/tls would otherwise randomize the key per tls.Config, and
// because the engine builds a fresh config for each EAP exchange, every ticket would
// be sealed with a secret that dies with its own handshake.
//
// Like the other providers it reuses the managed server certificate selected by
// radius.EapTlsServerCert; until one is selected it returns a nil config so the
// handler rejects safely with eap.ErrTLSNotConfigured instead of opening an
// unauthenticated tunnel. What resumption still needs on top of this config is the
// handshake that owns it: RFC 9930 §3.11 requires the credentials authenticated in
// Phase 2 to be associated with the ticket, or the ticket invalidated when that
// cannot be done.
func NewSettingsTEAPConfigProvider(reader TLSSettingsReader, resolvers ...CertResolver) TLSConfigProvider {
	resolver := firstResolver(resolvers)

	var (
		keyOnce   sync.Once
		ticketKey []byte
		keyErr    error
	)

	return func() (*tlsengine.Config, error) {
		if reader == nil {
			return nil, nil
		}

		if !serverCertConfigured(reader, resolver) {
			return nil, nil
		}

		keyOnce.Do(func() {
			ticketKey = make([]byte, tlsengine.SessionTicketKeySize)
			_, keyErr = rand.Read(ticketKey)
		})
		if keyErr != nil {
			return nil, fmt.Errorf("TEAP session ticket key: %w", keyErr)
		}

		cert, err := loadServerCertificate(reader, resolver, "TEAP")
		if err != nil {
			return nil, err
		}

		cfg := &tlsengine.Config{
			ServerCertificate:    cert,
			ServerOnly:           true,
			EnableSessionTickets: true,
			SessionTicketKey:     ticketKey,
		}
		if err := applyTLSPolicy(cfg, reader); err != nil {
			return nil, err
		}
		return cfg, nil
	}
}

// NewSettingsTEAPInnerTLSProvider returns a TLSConfigProvider for the inner
// EAP-TLS method of TEAP Phase 2.
//
// The inner method is a second, complete EAP-TLS conversation inside the outer
// tunnel, and it is where the peer actually proves who it is: the outer TEAP
// tunnel is server-authenticated only, so the client certificate this provider
// verifies is the credential the session is granted on. It therefore requires
// both the managed server certificate and the managed client CA, exactly as
// standalone EAP-TLS does, and returns a nil config (a safe reject) until both
// are selected.
//
// The inner handshake is pinned to TLS 1.3. teapinner.EAPTLS claims an EMSK
// because RFC 9190 §2.3 defines one only for TLS 1.3 (MSK = Key_Material(0,63),
// EMSK = Key_Material(64,127)); a TLS 1.2 inner handshake would export a
// different key schedule, and RFC 9930 §3.6.5 requires the EMSK Compound MAC
// this server cannot compute from it. Session tickets stay off inside the
// tunnel: RFC 9930 §3.6.5 requires every Inner Method to run a full
// authentication, so the inner EAP-TLS may not resume.
func NewSettingsTEAPInnerTLSProvider(reader TLSSettingsReader, resolvers ...CertResolver) TLSConfigProvider {
	resolver := firstResolver(resolvers)
	return func() (*tlsengine.Config, error) {
		if reader == nil {
			return nil, nil
		}

		if !serverCertConfigured(reader, resolver) || !clientCAConfigured(reader, resolver) {
			return nil, nil
		}

		cert, err := loadServerCertificate(reader, resolver, "TEAP inner EAP-TLS")
		if err != nil {
			return nil, err
		}
		pool, err := loadClientCAs(reader, resolver)
		if err != nil {
			return nil, err
		}

		cfg := &tlsengine.Config{
			ServerCertificate: cert,
			ClientCAs:         pool,
			MinVersion:        tls.VersionTLS13,
		}
		if err := applyTLSPolicy(cfg, reader); err != nil {
			return nil, err
		}
		// applyTLSPolicy honors the operator floor, which may be TLS 1.2; the
		// inner method needs 1.3 for its EMSK, so the floor is raised, never
		// lowered.
		cfg.MinVersion = tls.VersionTLS13
		return cfg, nil
	}
}
