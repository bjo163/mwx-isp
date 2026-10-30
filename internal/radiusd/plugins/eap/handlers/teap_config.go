package handlers

import (
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
)

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
