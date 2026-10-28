package radiusd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"layeh.com/radius"
)

// radsecHandshakeSetup is one running RadsecPacketServer with a trusted CA and a
// handler counter shared by the handshake matrix.
type radsecHandshakeSetup struct {
	ca      *radsecTestCA
	roots   *x509.CertPool
	addr    string
	handled *int32
	server  *RadsecPacketServer
}

func newRadsecHandshakeSetup(t *testing.T) *radsecHandshakeSetup {
	t.Helper()
	ca := newRadsecTestCA(t, "radsec-matrix-ca")
	fx := newRadsecTLSFixture(t)

	serverCert, serverKey := ca.issue(t, "radsec-server", 7001, x509.ExtKeyUsageServerAuth, time.Now().Add(24*time.Hour))
	writeRadsecFile(t, fx.certPath, serverCert, 1)
	writeRadsecFile(t, fx.keyPath, serverKey, 1)
	writeRadsecFile(t, fx.caPath, ca.pem, 1)

	handled := new(int32)
	server, addr := startRadsecPacketListener(t, fx, handled)
	return &radsecHandshakeSetup{
		ca:      ca,
		roots:   ca.pool(),
		addr:    addr,
		handled: handled,
		server:  server,
	}
}

// startRadsecPacketListener runs the real RadsecPacketServer behind a TLS
// listener built by initTLSConfig, so the matrix exercises the same
// configuration and connection handling as production. handled counts the
// packets that reached the RADIUS handler.
func startRadsecPacketListener(t *testing.T, fx *radsecTLSFixture, handled *int32) (*RadsecPacketServer, string) {
	t.Helper()
	server := &RadsecPacketServer{
		Handler: radsecHandlerFunc(func(radius.ResponseWriter, *radius.Request) {
			atomic.AddInt32(handled, 1)
		}),
		SecretSource: radsecSecretSourceFunc(func(context.Context, net.Addr) ([]byte, error) {
			return []byte("radsec"), nil
		}),
		RadsecWorker: 4,
	}
	cfg, err := server.initTLSConfig(fx.caPath, fx.certPath, fx.keyPath)
	if err != nil {
		t.Fatalf("initTLSConfig: %v", err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = ln.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("RadsecPacketServer.Shutdown: %v (a closed or rejected connection must not keep Serve running)", err)
		}
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) { _ = server.Serve(c) }(conn)
		}
	}()

	return server, ln.Addr().String()
}

// radsecExchange completes the handshake, sends one synthetic Access-Request and
// waits briefly for the handler. It returns the handshake error, if any.
func radsecExchange(t *testing.T, addr string, roots *x509.CertPool, clientCert *tls.Certificate, minVersion, maxVersion uint16) error {
	t.Helper()
	cfg := &tls.Config{
		RootCAs:    roots,
		ServerName: "radsec.test",
		MinVersion: minVersion,
		MaxVersion: maxVersion,
	}
	if clientCert != nil {
		// Present the certificate even when its issuer is missing from the
		// server's advertised CA list. Go's own client would otherwise decline
		// to send it, while a hostile or misconfigured peer would send it — and
		// the server-side verification is exactly what this matrix exercises.
		cfg.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return clientCert, nil
		}
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 3 * time.Second}, "tcp", addr, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write(radsecTestPacket(7)); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return nil
}

// clientCert issues a client certificate signed by the setup CA.
func (s *radsecHandshakeSetup) clientCert(t *testing.T, serial int64, usage x509.ExtKeyUsage, notAfter time.Time) *tls.Certificate {
	t.Helper()
	certPEM, keyPEM := s.ca.issue(t, "radsec-client", serial, usage, notAfter)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("client key pair: %v", err)
	}
	return &cert
}

// TestRadsecHandshake_TrustedClientAccepted is the positive half of the matrix:
// a client certificate from the configured CA authenticates and its packet is
// processed.
func TestRadsecHandshake_TrustedClientAccepted(t *testing.T) {
	setup := newRadsecHandshakeSetup(t)
	client := setup.clientCert(t, 7201, x509.ExtKeyUsageClientAuth, time.Now().Add(24*time.Hour))

	if err := radsecExchange(t, setup.addr, setup.roots, client, tls.VersionTLS12, 0); err != nil {
		t.Fatalf("handshake with a trusted client certificate: %v", err)
	}
	if got := atomic.LoadInt32(setup.handled); got != 1 {
		t.Fatalf("expected the packet to be handled once, got %d", got)
	}
}

// TestRadsecHandshake_UntrustedClientRejected covers the regression named by the
// item: a client certificate signed by a CA outside the configured bundle must
// not complete the handshake, and must never reach the RADIUS handler.
func TestRadsecHandshake_UntrustedClientRejected(t *testing.T) {
	setup := newRadsecHandshakeSetup(t)
	otherCA := newRadsecTestCA(t, "radsec-untrusted-ca")
	client := mustClientCert(t, otherCA, 7301)

	// TLS 1.2 reports the client-certificate rejection synchronously.
	if err := radsecExchange(t, setup.addr, setup.roots, client, tls.VersionTLS12, tls.VersionTLS12); err == nil {
		t.Fatal("expected the handshake with an untrusted client certificate to fail")
	}
	if got := atomic.LoadInt32(setup.handled); got != 0 {
		t.Fatalf("an untrusted client reached the handler %d time(s)", got)
	}

	// TLS 1.3 finishes the client side of the handshake before the server's
	// rejection alert arrives, so the guarantee here is that the packet is never
	// processed.
	_ = radsecExchange(t, setup.addr, setup.roots, client, tls.VersionTLS13, tls.VersionTLS13)
	if got := atomic.LoadInt32(setup.handled); got != 0 {
		t.Fatalf("an untrusted TLS 1.3 client reached the handler %d time(s)", got)
	}
}

// TestRadsecHandshake_ExpiredClientRejected verifies that an expired client
// certificate is rejected even though its CA is trusted.
func TestRadsecHandshake_ExpiredClientRejected(t *testing.T) {
	setup := newRadsecHandshakeSetup(t)
	client := setup.clientCert(t, 7401, x509.ExtKeyUsageClientAuth, time.Now().Add(-time.Minute))

	if err := radsecExchange(t, setup.addr, setup.roots, client, tls.VersionTLS12, tls.VersionTLS12); err == nil {
		t.Fatal("expected an expired client certificate to be rejected")
	}
	if got := atomic.LoadInt32(setup.handled); got != 0 {
		t.Fatalf("an expired client reached the handler %d time(s)", got)
	}
}

// TestRadsecHandshake_WrongKeyUsageRejected verifies that a certificate signed
// by the trusted CA but issued for server authentication is not accepted as a
// client identity.
func TestRadsecHandshake_WrongKeyUsageRejected(t *testing.T) {
	setup := newRadsecHandshakeSetup(t)
	client := setup.clientCert(t, 7501, x509.ExtKeyUsageServerAuth, time.Now().Add(24*time.Hour))

	if err := radsecExchange(t, setup.addr, setup.roots, client, tls.VersionTLS12, tls.VersionTLS12); err == nil {
		t.Fatal("expected a server-only client certificate to be rejected")
	}
	if got := atomic.LoadInt32(setup.handled); got != 0 {
		t.Fatalf("a wrong-EKU client reached the handler %d time(s)", got)
	}
}

// TestRadsecHandshake_ClientWithoutCertificateAccepted pins the current policy:
// ClientAuth is VerifyClientCertIfGiven, so a client that presents no
// certificate still connects. Requiring one would tighten the policy and needs a
// separate scope decision, but no future change may silently relax it further
// (see TestRadsecTLSConfig_DoesNotRelaxClientAuth).
func TestRadsecHandshake_ClientWithoutCertificateAccepted(t *testing.T) {
	setup := newRadsecHandshakeSetup(t)

	if err := radsecExchange(t, setup.addr, setup.roots, nil, tls.VersionTLS12, 0); err != nil {
		t.Fatalf("handshake without a client certificate: %v", err)
	}
	if got := atomic.LoadInt32(setup.handled); got != 1 {
		t.Fatalf("expected the packet to be handled once, got %d", got)
	}
}

// TestRadsecHandshake_TLSVersionFloor verifies that the TLS 1.2 minimum holds
// end to end: a TLS 1.1 client cannot connect.
func TestRadsecHandshake_TLSVersionFloor(t *testing.T) {
	setup := newRadsecHandshakeSetup(t)
	client := setup.clientCert(t, 7601, x509.ExtKeyUsageClientAuth, time.Now().Add(24*time.Hour))

	if err := radsecExchange(t, setup.addr, setup.roots, client, tls.VersionTLS10, tls.VersionTLS11); err == nil {
		t.Fatal("expected a TLS 1.1 client to be rejected")
	}
	if got := atomic.LoadInt32(setup.handled); got != 0 {
		t.Fatalf("a TLS 1.1 client reached the handler %d time(s)", got)
	}
}

// TestRadsecHandshake_FailedHandshakeReleasesConnection is the regression guard
// for a rejected handshake: Serve used to treat the resulting transport error as
// a recoverable packet-parse error and re-enter the read loop, spinning on the
// same dead connection. It must instead return and release the connection.
func TestRadsecHandshake_FailedHandshakeReleasesConnection(t *testing.T) {
	setup := newRadsecHandshakeSetup(t)
	otherCA := newRadsecTestCA(t, "radsec-spin-ca")
	client := mustClientCert(t, otherCA, 7801)

	if err := radsecExchange(t, setup.addr, setup.roots, client, tls.VersionTLS12, tls.VersionTLS12); err == nil {
		t.Fatal("expected the handshake with an untrusted client certificate to fail")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&setup.server.activeCount) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Serve is still running (%d active) after a rejected handshake", atomic.LoadInt32(&setup.server.activeCount))
}

// TestRadsecTLSConfig_DoesNotRelaxClientAuth pins the security-relevant parts of
// the live configuration so a future refactor cannot quietly drop client
// certificate verification or lower the protocol floor.
func TestRadsecTLSConfig_DoesNotRelaxClientAuth(t *testing.T) {
	ca := newRadsecTestCA(t, "radsec-policy-ca")
	fx := newRadsecTLSFixture(t)
	serverCert, serverKey := ca.issue(t, "radsec-server", 7701, x509.ExtKeyUsageServerAuth, time.Now().Add(24*time.Hour))
	writeRadsecFile(t, fx.certPath, serverCert, 1)
	writeRadsecFile(t, fx.keyPath, serverKey, 1)
	writeRadsecFile(t, fx.caPath, ca.pem, 1)

	files, err := newRadsecTLSFiles(fx.caPath, fx.certPath, fx.keyPath)
	if err != nil {
		t.Fatalf("newRadsecTLSFiles: %v", err)
	}
	snapshot := files.snapshot()

	if snapshot.ClientAuth != tls.VerifyClientCertIfGiven {
		t.Fatalf("ClientAuth = %v, want VerifyClientCertIfGiven (a looser value would accept unverifiable peers)", snapshot.ClientAuth)
	}
	if snapshot.MinVersion < tls.VersionTLS12 {
		t.Fatalf("MinVersion = %#x, want at least TLS 1.2", snapshot.MinVersion)
	}
	if snapshot.ClientCAs == nil {
		t.Fatal("expected the configured CA bundle to be loaded for client verification")
	}
	if len(snapshot.Certificates) != 1 {
		t.Fatalf("expected exactly one server certificate, got %d", len(snapshot.Certificates))
	}
}
