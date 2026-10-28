package radiusd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// radsecTestCA is a throwaway certificate authority used to sign the server and
// client certificates that the rotation tests hand to the reloader.
type radsecTestCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
}

func newRadsecTestCA(t *testing.T, cn string) *radsecTestCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}
	return &radsecTestCA{
		cert: cert,
		key:  key,
		pem:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}
}

func (ca *radsecTestCA) pool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(ca.cert)
	return pool
}

// issue signs a leaf certificate with the given serial and extended key usage.
// Server leaves carry the DNS name the tests dial; client leaves carry none.
func (ca *radsecTestCA) issue(t *testing.T, cn string, serial int64, usage x509.ExtKeyUsage, notAfter time.Time) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{usage},
		BasicConstraintsValid: true,
	}
	if usage == x509.ExtKeyUsageServerAuth {
		tmpl.DNSNames = []string{"radsec.test"}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("create leaf certificate: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal leaf key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

// writeRadsecFile writes data and forces a distinct, monotonically increasing
// modification time so the reloader's change detection is deterministic even on
// filesystems with coarse timestamps.
func writeRadsecFile(t *testing.T, path string, data []byte, seq int) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	stamp := time.Now().Add(time.Duration(seq) * time.Second)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatalf("touch %s: %v", path, err)
	}
}

// radsecTLSFixture holds the three configured RadSec paths for one test.
type radsecTLSFixture struct {
	dir      string
	caPath   string
	certPath string
	keyPath  string
}

func newRadsecTLSFixture(t *testing.T) *radsecTLSFixture {
	t.Helper()
	dir := t.TempDir()
	return &radsecTLSFixture{
		dir:      dir,
		caPath:   filepath.Join(dir, "ca.crt"),
		certPath: filepath.Join(dir, "radsec.tls.crt"),
		keyPath:  filepath.Join(dir, "radsec.tls.key"),
	}
}

func newRadsecTestServer(t *testing.T, caPath, certPath, keyPath string) string {
	t.Helper()
	server := &RadsecPacketServer{}
	cfg, err := server.initTLSConfig(caPath, certPath, keyPath)
	if err != nil {
		t.Fatalf("initTLSConfig: %v", err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_ = c.(*tls.Conn).Handshake()
			}(conn)
		}
	}()

	return ln.Addr().String()
}

// radsecHandshake completes a TLS handshake against addr and returns the serial
// of the certificate the server presented.
func radsecHandshake(t *testing.T, addr string, roots *x509.CertPool, clientCert *tls.Certificate) (string, error) {
	t.Helper()
	cfg := &tls.Config{
		RootCAs:    roots,
		ServerName: "radsec.test",
		MinVersion: tls.VersionTLS12,
	}
	if clientCert != nil {
		cfg.Certificates = []tls.Certificate{*clientCert}
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 3 * time.Second}, "tcp", addr, cfg)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	peers := conn.ConnectionState().PeerCertificates
	if len(peers) == 0 {
		return "", errors.New("server presented no certificate")
	}
	return peers[0].SerialNumber.String(), nil
}

func mustClientCert(t *testing.T, ca *radsecTestCA, serial int64) *tls.Certificate {
	t.Helper()
	certPEM, keyPEM := ca.issue(t, "radsec-client", serial, x509.ExtKeyUsageClientAuth, time.Now().Add(24*time.Hour))
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("client key pair: %v", err)
	}
	return &cert
}

// TestRadsecTLSFiles_RotatesWithoutRestart covers the readiness criterion: a
// new server certificate, then a rollback to the previous one, both take effect
// on the next handshake of a running listener.
func TestRadsecTLSFiles_RotatesWithoutRestart(t *testing.T) {
	ca := newRadsecTestCA(t, "radsec-test-ca")
	fx := newRadsecTLSFixture(t)

	oldCert, oldKey := ca.issue(t, "radsec-server-old", 1001, x509.ExtKeyUsageServerAuth, time.Now().Add(24*time.Hour))
	newCert, newKey := ca.issue(t, "radsec-server-new", 1002, x509.ExtKeyUsageServerAuth, time.Now().Add(48*time.Hour))

	writeRadsecFile(t, fx.certPath, oldCert, 1)
	writeRadsecFile(t, fx.keyPath, oldKey, 1)
	writeRadsecFile(t, fx.caPath, ca.pem, 1)

	addr := newRadsecTestServer(t, fx.caPath, fx.certPath, fx.keyPath)
	roots := ca.pool()
	client := mustClientCert(t, ca, 5001)

	serial, err := radsecHandshake(t, addr, roots, client)
	if err != nil {
		t.Fatalf("handshake with the original certificate: %v", err)
	}
	if serial != "1001" {
		t.Fatalf("expected the original serial 1001, got %s", serial)
	}

	writeRadsecFile(t, fx.certPath, newCert, 2)
	writeRadsecFile(t, fx.keyPath, newKey, 2)

	serial, err = radsecHandshake(t, addr, roots, client)
	if err != nil {
		t.Fatalf("handshake after rotation: %v", err)
	}
	if serial != "1002" {
		t.Fatalf("expected the rotated serial 1002 without a restart, got %s", serial)
	}

	// Rollback: restoring the previous files must take effect the same way.
	writeRadsecFile(t, fx.certPath, oldCert, 3)
	writeRadsecFile(t, fx.keyPath, oldKey, 3)

	serial, err = radsecHandshake(t, addr, roots, client)
	if err != nil {
		t.Fatalf("handshake after rollback: %v", err)
	}
	if serial != "1001" {
		t.Fatalf("expected the rolled-back serial 1001, got %s", serial)
	}
}

// TestRadsecTLSFiles_KeepsLastGoodOnInvalidReload asserts that a half-written or
// unparsable file does not take RadSec down: the previous certificate keeps
// serving until the operator fixes or restores the file.
func TestRadsecTLSFiles_KeepsLastGoodOnInvalidReload(t *testing.T) {
	ca := newRadsecTestCA(t, "radsec-test-ca")
	fx := newRadsecTLSFixture(t)

	goodCert, goodKey := ca.issue(t, "radsec-server", 2001, x509.ExtKeyUsageServerAuth, time.Now().Add(24*time.Hour))
	writeRadsecFile(t, fx.certPath, goodCert, 1)
	writeRadsecFile(t, fx.keyPath, goodKey, 1)
	writeRadsecFile(t, fx.caPath, ca.pem, 1)

	addr := newRadsecTestServer(t, fx.caPath, fx.certPath, fx.keyPath)
	roots := ca.pool()
	client := mustClientCert(t, ca, 5002)

	// Truncate the certificate as if a rotation write were interrupted.
	writeRadsecFile(t, fx.certPath, []byte("-----BEGIN CERTIFICATE-----\nnot a certificate\n"), 2)

	serial, err := radsecHandshake(t, addr, roots, client)
	if err != nil {
		t.Fatalf("handshake must keep working with the last known-good certificate: %v", err)
	}
	if serial != "2001" {
		t.Fatalf("expected the last known-good serial 2001, got %s", serial)
	}

	// A CA bundle with no PEM blocks is also a reload error, not a silent switch
	// to an empty trust store.
	writeRadsecFile(t, fx.caPath, []byte("garbage\n"), 3)

	serial, err = radsecHandshake(t, addr, roots, client)
	if err != nil {
		t.Fatalf("handshake must keep working after an invalid CA bundle: %v", err)
	}
	if serial != "2001" {
		t.Fatalf("expected serial 2001 while the CA bundle is invalid, got %s", serial)
	}
}

// TestRadsecTLSFiles_RejectsExpiredCertificate asserts the fail-closed startup
// rule: an already expired leaf is refused before the listener is created.
func TestRadsecTLSFiles_RejectsExpiredCertificate(t *testing.T) {
	ca := newRadsecTestCA(t, "radsec-test-ca")
	fx := newRadsecTLSFixture(t)

	expiredCert, expiredKey := ca.issue(t, "radsec-server-expired", 3001, x509.ExtKeyUsageServerAuth, time.Now().Add(-time.Minute))
	writeRadsecFile(t, fx.certPath, expiredCert, 1)
	writeRadsecFile(t, fx.keyPath, expiredKey, 1)
	writeRadsecFile(t, fx.caPath, ca.pem, 1)

	if _, err := newRadsecTLSFiles(fx.caPath, fx.certPath, fx.keyPath); err == nil {
		t.Fatal("expected an expired server certificate to be rejected")
	}
}

// TestRadsecTLSFiles_KeepsLastGoodOnExpiredReload asserts that a rotation to an
// already expired certificate is refused and the current certificate keeps
// serving until the operator replaces it with valid material.
func TestRadsecTLSFiles_KeepsLastGoodOnExpiredReload(t *testing.T) {
	ca := newRadsecTestCA(t, "radsec-test-ca")
	fx := newRadsecTLSFixture(t)

	goodCert, goodKey := ca.issue(t, "radsec-server", 4001, x509.ExtKeyUsageServerAuth, time.Now().Add(24*time.Hour))
	expiredCert, expiredKey := ca.issue(t, "radsec-server-expired", 4002, x509.ExtKeyUsageServerAuth, time.Now().Add(-time.Minute))
	writeRadsecFile(t, fx.certPath, goodCert, 1)
	writeRadsecFile(t, fx.keyPath, goodKey, 1)
	writeRadsecFile(t, fx.caPath, ca.pem, 1)

	addr := newRadsecTestServer(t, fx.caPath, fx.certPath, fx.keyPath)
	client := mustClientCert(t, ca, 5003)

	writeRadsecFile(t, fx.certPath, expiredCert, 2)
	writeRadsecFile(t, fx.keyPath, expiredKey, 2)

	serial, err := radsecHandshake(t, addr, ca.pool(), client)
	if err != nil {
		t.Fatalf("handshake must keep working while the replacement is invalid: %v", err)
	}
	if serial != "4001" {
		t.Fatalf("expected the still-valid serial 4001, got %s", serial)
	}
}

// TestRadsecTLSFiles_ClientCAAOverlap covers a CA-bundle rotation: during the
// overlap window the bundle carries both the old and the new CA, so clients
// holding a certificate from either CA keep authenticating while they are
// migrated one at a time.
func TestRadsecTLSFiles_ClientCAAOverlap(t *testing.T) {
	oldCA := newRadsecTestCA(t, "radsec-old-ca")
	newCA := newRadsecTestCA(t, "radsec-new-ca")
	fx := newRadsecTLSFixture(t)

	serverCert, serverKey := oldCA.issue(t, "radsec-server", 6001, x509.ExtKeyUsageServerAuth, time.Now().Add(24*time.Hour))
	writeRadsecFile(t, fx.certPath, serverCert, 1)
	writeRadsecFile(t, fx.keyPath, serverKey, 1)
	writeRadsecFile(t, fx.caPath, oldCA.pem, 1)

	addr := newRadsecTestServer(t, fx.caPath, fx.certPath, fx.keyPath)
	roots := oldCA.pool()

	oldClient := mustClientCert(t, oldCA, 5101)
	newClient := mustClientCert(t, newCA, 5102)

	if _, err := radsecHandshake(t, addr, roots, oldClient); err != nil {
		t.Fatalf("handshake with the existing client CA: %v", err)
	}

	// Overlap: both CAs in the bundle.
	writeRadsecFile(t, fx.caPath, append(append([]byte{}, oldCA.pem...), newCA.pem...), 2)

	if _, err := radsecHandshake(t, addr, roots, oldClient); err != nil {
		t.Fatalf("handshake with the old client certificate during overlap: %v", err)
	}
	if _, err := radsecHandshake(t, addr, roots, newClient); err != nil {
		t.Fatalf("handshake with the new client certificate during overlap: %v", err)
	}

	// Rotate the bundle back to the old CA while handshakes are in flight: every
	// handshake must complete against either snapshot, never a half-loaded one.
	errCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_, err := radsecHandshake(t, addr, roots, oldClient)
			errCh <- err
		}()
	}
	writeRadsecFile(t, fx.caPath, oldCA.pem, 3)
	for i := 0; i < 8; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent handshake during rotation: %v", err)
		}
	}
}
