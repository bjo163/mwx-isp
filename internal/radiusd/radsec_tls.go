package radiusd

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/bjo163/mwx-isp/pkg/common"
	"go.uber.org/zap"
)

// radsecFileStamp identifies one on-disk version of a certificate input. A
// reload is attempted only when one of the three inputs changed, so a normal
// handshake costs three stats and no PEM parsing.
type radsecFileStamp struct {
	exists  bool
	size    int64
	modTime time.Time
}

func statRadsecFileStamp(path string) (radsecFileStamp, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return radsecFileStamp{}, nil
		}
		return radsecFileStamp{}, err
	}
	return radsecFileStamp{exists: true, size: info.Size(), modTime: info.ModTime()}, nil
}

// radsecTLSFiles owns the RadSec server key pair and client-CA bundle and keeps
// them reloadable without a process restart.
//
// Rotating a RadSec certificate used to require restarting the daemon, which
// also drops every live RADIUS-over-TLS connection. This type instead keeps the
// parsed material behind a mutex and checks the three configured files on each
// TLS handshake: when a file changed, the key pair and CA bundle are re-parsed
// and swapped in before the handshake continues. Unchanged inputs return the
// previous snapshot without parsing anything.
//
// A failed reload (partially written file, unparsable PEM, expired leaf) keeps
// the last known-good snapshot and logs the error, so a botched rotation does
// not take RadSec down mid-handshake; the operator sees the failure in the log
// and can restore the previous files. Only the initial load is fail-closed.
type radsecTLSFiles struct {
	caPath   string
	certPath string
	keyPath  string

	now func() time.Time

	mu        sync.Mutex
	certStamp radsecFileStamp
	keyStamp  radsecFileStamp
	caStamp   radsecFileStamp
	current   *tls.Config
	leaf      *x509.Certificate
}

// newRadsecTLSFiles loads the initial TLS material. It fails when the key pair
// cannot be loaded, the leaf is already expired, or a present CA bundle holds no
// PEM certificates, so the server never starts with material it cannot present.
func newRadsecTLSFiles(capath, crtfile, keyfile string) (*radsecTLSFiles, error) {
	f := &radsecTLSFiles{
		caPath:   capath,
		certPath: crtfile,
		keyPath:  keyfile,
		now:      time.Now,
	}
	if err := f.reload(); err != nil {
		return nil, err
	}
	return f, nil
}

// baseConfig returns the configuration handed to tls.Listen. It carries no
// certificate itself: every handshake resolves the current snapshot through
// GetConfigForClient, which is where a rotation is picked up. The snapshot is
// read-only and shared between concurrent handshakes, as crypto/tls expects.
func (f *radsecTLSFiles) baseConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.VerifyClientCertIfGiven,
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			return f.snapshot(), nil
		},
	}
}

// snapshot returns the current TLS configuration, reloading first when one of
// the inputs changed. A failed reload is logged and the previous configuration
// is returned so in-flight rotation mistakes do not interrupt handshakes.
func (f *radsecTLSFiles) snapshot() *tls.Config {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.changedLocked() {
		return f.current
	}
	if err := f.reloadLocked(); err != nil {
		zap.S().Errorf("radius: radsec tls reload failed, keeping previous certificate: %v", err)
	}
	return f.current
}

// reload forces a reload attempt regardless of file timestamps. It is used for
// the initial load.
func (f *radsecTLSFiles) reload() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reloadLocked()
}

// changedLocked reports whether any input differs from the version that
// produced the current snapshot. It returns true when the stamps cannot be
// read, letting reloadLocked surface the actual stat error.
func (f *radsecTLSFiles) changedLocked() bool {
	certStamp, err := statRadsecFileStamp(f.certPath)
	if err != nil {
		return true
	}
	keyStamp, err := statRadsecFileStamp(f.keyPath)
	if err != nil {
		return true
	}
	caStamp, err := statRadsecFileStamp(f.caPath)
	if err != nil {
		return true
	}
	return f.current == nil || certStamp != f.certStamp || keyStamp != f.keyStamp || caStamp != f.caStamp
}

// reloadLocked builds a new snapshot from the files currently on disk. The
// stamps are recorded only after a successful parse, so a half-written file is
// retried on the next handshake instead of being remembered as the current
// version.
func (f *radsecTLSFiles) reloadLocked() error {
	certStamp, err := statRadsecFileStamp(f.certPath)
	if err != nil {
		return fmt.Errorf("stat RadSec certificate %q: %w", f.certPath, err)
	}
	keyStamp, err := statRadsecFileStamp(f.keyPath)
	if err != nil {
		return fmt.Errorf("stat RadSec private key %q: %w", f.keyPath, err)
	}
	caStamp, err := statRadsecFileStamp(f.caPath)
	if err != nil {
		return fmt.Errorf("stat RadSec CA bundle %q: %w", f.caPath, err)
	}

	cfg, leaf, err := loadRadsecTLSConfig(f.caPath, f.certPath, f.keyPath, f.now())
	if err != nil {
		return err
	}

	previousSerial := ""
	if f.leaf != nil {
		previousSerial = f.leaf.SerialNumber.String()
	}
	f.current = cfg
	f.leaf = leaf
	f.certStamp = certStamp
	f.keyStamp = keyStamp
	f.caStamp = caStamp

	// Record the observable handshake identity (serial, subject, expiry) so a
	// rotation can be confirmed from the log without dumping key material.
	zap.S().Infof(
		"radius: radsec tls material loaded: subject=%q serial=%s not_after=%s previous_serial=%s",
		leaf.Subject.String(),
		leaf.SerialNumber.String(),
		leaf.NotAfter.UTC().Format(time.RFC3339),
		previousSerial,
	)
	return nil
}

// loadRadsecTLSConfig parses the key pair and optional CA bundle. An expired
// leaf is rejected: crypto/tls would otherwise complete the handshake and leave
// the peer to fail, which is harder to diagnose than a clear load error.
func loadRadsecTLSConfig(capath, crtfile, keyfile string, now time.Time) (*tls.Config, *x509.Certificate, error) {
	crt, err := tls.LoadX509KeyPair(crtfile, keyfile)
	if err != nil {
		return nil, nil, fmt.Errorf("load RadSec key pair: %w", err)
	}

	leaf, err := x509.ParseCertificate(crt.Certificate[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parse RadSec server certificate: %w", err)
	}
	if !now.Before(leaf.NotAfter) {
		return nil, nil, fmt.Errorf("RadSec server certificate %q expired at %s",
			leaf.Subject.CommonName, leaf.NotAfter.UTC().Format(time.RFC3339))
	}
	crt.Leaf = leaf

	warnOnLooseRadsecKeyPermissions(keyfile)

	cfg := &tls.Config{
		Certificates: []tls.Certificate{crt},
		Time:         time.Now,
		Rand:         rand.Reader,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS12,
	}

	if common.FileExists(capath) {
		caBytes, err := os.ReadFile(capath) //nolint:gosec // G304: path is from validated config
		if err != nil {
			return nil, nil, fmt.Errorf("read RadSec CA bundle %q: %w", capath, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caBytes) {
			return nil, nil, fmt.Errorf("parse RadSec CA bundle %q: no PEM certificates found", capath)
		}
		cfg.ClientCAs = pool
	} else {
		// Without a bundle the server cannot advertise a client CA; peers that
		// present a certificate are still verified (against the system roots),
		// but a peer presenting none is accepted. Say so at load time instead of
		// letting an operator discover it from a packet trace.
		zap.S().Warnf(
			"radius: RadSec client CA bundle %q not found; client certificates are not required for incoming connections",
			capath,
		)
	}

	return cfg, leaf, nil
}

// warnOnLooseRadsecKeyPermissions logs when the private key is readable by
// anyone other than its owner. Errors are ignored: a permission warning must
// never block serving traffic. Windows ACLs are not inspected; deployments rely
// on directory ACLs there.
func warnOnLooseRadsecKeyPermissions(keyfile string) {
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(keyfile)
	if err != nil {
		return
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		zap.S().Warnf(
			"radius: radsec private key %s is accessible to group/other (mode %04o); restrict it to the service account",
			keyfile, mode,
		)
	}
}
