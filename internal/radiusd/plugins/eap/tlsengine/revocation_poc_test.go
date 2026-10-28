package tlsengine

// Proof of concept for the EAP-TLS revocation policy (backlog item ENG-003).
//
// The production engine verifies the peer chain with ClientAuth=
// RequireAndVerifyClientCert and nothing else: crypto/tls does not fetch or
// evaluate CRLs or OCSP responses, so a client certificate that has been
// revoked is still accepted. The verifier below is the proposed policy, kept
// in the test tree on purpose — the decision to wire it into the EAP-TLS
// provider is recorded in docs/eap-tls-revocation-policy.md and is not claimed
// to be deployed.

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// crlVerifier returns a VerifyConnection hook that fails closed for every
// presented certificate whose issuer has a configured CRL:
//
//   - a listed serial number rejects the handshake (revoked);
//   - a CRL that is missing NextUpdate validity, not yet valid, or past
//     NextUpdate rejects the handshake (stale revocation data);
//   - an issuer without a CRL is accepted (coverage limit, see the ADR).
func crlVerifier(ca *x509.Certificate, lists []*x509.RevocationList, now time.Time) func(tls.ConnectionState) error {
	return func(state tls.ConnectionState) error {
		for _, cert := range state.PeerCertificates {
			for _, list := range lists {
				if err := list.CheckSignatureFrom(ca); err != nil {
					return err
				}
				if string(list.RawIssuer) != string(cert.RawIssuer) {
					continue
				}
				if now.Before(list.ThisUpdate) {
					return errRevocationNotYetValid
				}
				if !list.NextUpdate.IsZero() && now.After(list.NextUpdate) {
					return errRevocationExpired
				}
				for _, entry := range list.RevokedCertificateEntries {
					if entry.SerialNumber != nil && entry.SerialNumber.Cmp(cert.SerialNumber) == 0 {
						return errCertificateRevoked
					}
				}
			}
		}
		return nil
	}
}

var (
	errCertificateRevoked    = &revocationError{"client certificate is revoked"}
	errRevocationExpired     = &revocationError{"revocation list expired"}
	errRevocationNotYetValid = &revocationError{"revocation list not yet valid"}
)

type revocationError struct{ message string }

func (e *revocationError) Error() string { return e.message }

// rejectionReason returns the handshake error the engine recorded. When the
// client closes as soon as it sees the alert, the driver loop can observe the
// closed pipe before the handshake goroutine records the real error, so the
// engine is polled once more for the handshake outcome.
func rejectionReason(eng *Engine, serverErr error) error {
	if serverErr != nil && (errors.Is(serverErr, net.ErrClosed) || strings.Contains(serverErr.Error(), "closed network connection")) {
		if _, _, err := eng.Process(nil); err != nil {
			return err
		}
	}
	return serverErr
}

// issueCRL builds and parses a CRL signed by ca.
func issueCRL(t *testing.T, ca *testCA, thisUpdate, nextUpdate time.Time, revoked ...*x509.Certificate) *x509.RevocationList {
	t.Helper()
	entries := make([]x509.RevocationListEntry, 0, len(revoked))
	for _, cert := range revoked {
		entries = append(entries, x509.RevocationListEntry{
			SerialNumber:   cert.SerialNumber,
			RevocationTime: thisUpdate,
		})
	}
	tmpl := &x509.RevocationList{
		Number:                    big.NewInt(1),
		ThisUpdate:                thisUpdate,
		NextUpdate:                nextUpdate,
		RevokedCertificateEntries: entries,
	}
	der, err := x509.CreateRevocationList(rand.Reader, tmpl, ca.cert, ca.key)
	if err != nil {
		t.Fatalf("create CRL: %v", err)
	}
	list, err := x509.ParseRevocationList(der)
	if err != nil {
		t.Fatalf("parse CRL: %v", err)
	}
	return list
}

// testLeaf issues a client certificate with a fixed serial so the CRL entry can
// reference it.
func testLeaf(t *testing.T, ca *testCA, commonName string, serial int64) tls.Certificate {
	t.Helper()
	return ca.issue(t, commonName, func(c *x509.Certificate) {
		c.SerialNumber = big.NewInt(serial)
		c.EmailAddresses = []string{commonName + "@example.com"}
	})
}

// TestRevocationPOC_RevokedCertificate covers the two halves of the gap: the
// current engine accepts the revoked certificate, and the proposed verifier
// rejects it without touching chain validation.
func TestRevocationPOC_RevokedCertificate(t *testing.T) {
	ca := newTestCA(t, "EAP Client CA")
	clientCert := testLeaf(t, ca, "alice", 4242)
	revoked := issueCRL(t, ca, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), clientCert.Leaf)

	t.Run("current engine accepts a revoked certificate", func(t *testing.T) {
		eng, err := New(serverConfig(t, ca, ca))
		if err != nil {
			t.Fatalf("new engine: %v", err)
		}
		defer func() { _ = eng.Close() }()

		clientErr, serverErr := driveHandshake(t, eng, clientTLSConfig(ca, clientCert))
		if clientErr != nil || serverErr != nil {
			t.Fatalf("baseline handshake should complete without revocation data: client=%v server=%v", clientErr, serverErr)
		}
	})

	t.Run("proposed verifier rejects the revoked certificate", func(t *testing.T) {
		cfg := serverConfig(t, ca, ca)
		cfg.VerifyConnection = crlVerifier(ca.cert, []*x509.RevocationList{revoked}, time.Now())

		eng, err := New(cfg)
		if err != nil {
			t.Fatalf("new engine: %v", err)
		}
		defer func() { _ = eng.Close() }()

		clientErr, serverErr := driveHandshake(t, eng, clientTLSConfig(ca, clientCert))
		serverErr = rejectionReason(eng, serverErr)
		if serverErr == nil || !strings.Contains(serverErr.Error(), errCertificateRevoked.Error()) {
			t.Fatalf("server should reject the revoked certificate, got client=%v server=%v", clientErr, serverErr)
		}
		// Under TLS 1.3 the client can finish its side before the server's
		// rejection alert arrives, so only the server error is asserted here.
	})
}

// TestRevocationPOC_StaleListFailsClosed pins the timeout side of the policy:
// revocation data that is past NextUpdate rejects the handshake even when the
// presented certificate is not listed.
func TestRevocationPOC_StaleListFailsClosed(t *testing.T) {
	ca := newTestCA(t, "EAP Client CA")
	clientCert := testLeaf(t, ca, "bob", 5150)
	stale := issueCRL(t, ca, time.Now().Add(-4*time.Hour), time.Now().Add(-2*time.Hour))

	cfg := serverConfig(t, ca, ca)
	cfg.VerifyConnection = crlVerifier(ca.cert, []*x509.RevocationList{stale}, time.Now())

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = eng.Close() }()

	_, serverErr := driveHandshake(t, eng, clientTLSConfig(ca, clientCert))
	serverErr = rejectionReason(eng, serverErr)
	if serverErr == nil || !strings.Contains(serverErr.Error(), errRevocationExpired.Error()) {
		t.Fatalf("stale revocation data must fail closed, got %v", serverErr)
	}
}

// TestRevocationPOC_IssuerWithoutCRLIsAccepted records the coverage boundary:
// the policy only speaks for issuers that have a CRL, so the hook alone is not
// a complete revocation guarantee.
func TestRevocationPOC_IssuerWithoutCRLIsAccepted(t *testing.T) {
	ca := newTestCA(t, "EAP Client CA")
	clientCert := testLeaf(t, ca, "carol", 6161)

	cfg := serverConfig(t, ca, ca)
	cfg.VerifyConnection = crlVerifier(ca.cert, nil, time.Now())

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = eng.Close() }()

	clientErr, serverErr := driveHandshake(t, eng, clientTLSConfig(ca, clientCert))
	if clientErr != nil || serverErr != nil {
		t.Fatalf("without a CRL for the issuer the handshake should complete: client=%v server=%v", clientErr, serverErr)
	}
}
