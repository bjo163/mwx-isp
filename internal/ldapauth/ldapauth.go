// Package ldapauth verifies user credentials against an LDAP or Active
// Directory server with the LDAP Bind operation defined in RFC 4511 section
// 4.2.
//
// It is a PAP-family authentication backend. Verifying a password requires
// MWX-ISP to perform a Bind with the user's cleartext password, so this
// backend can only support authentication methods where the server already
// holds that cleartext: bare PAP and the inner PAP of EAP-TTLS (RFC 5281).
// Challenge/response methods (CHAP, MS-CHAP, MS-CHAPv2, EAP-MD5,
// PEAP-MSCHAPv2) are physically impossible to back here because the directory
// never discloses the stored password and the server would need it (or its
// NT hash) to compute the challenge response. Callers must reject those
// methods explicitly rather than pretend to support them.
//
// This package is the standalone, unit-testable directory verifier plus its
// configuration loader. The live RADIUS authentication path adapts Verifier
// through internal/radiusd so bare PAP and EAP-TTLS inner PAP can share the
// same LDAP Bind authority while authorization still comes from the local
// RadiusUser record.
//
// Feature: TR-F025. Specs: RFC 4511 (LDAP protocol, Bind) and RFC 4513
// (LDAP authentication methods and security, in particular section 5.1.2 on
// unauthenticated bind and section 3 on StartTLS).
package ldapauth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// settingsCategory is the configuration category (the prefix before the dot in
// every "category.Name" key, see internal/app/config_schemas.json) that holds
// the LDAP backend settings.
const settingsCategory = "ldap"

// defaultTimeout is used when the configured timeout is missing or not
// positive. It bounds both the TCP dial and each LDAP operation.
const defaultTimeout = 5 * time.Second

// Sentinel errors returned by Verify. Callers compare with errors.Is.
//
// ErrInvalidCredentials and ErrUserNotFound are authentication rejections (the
// supplied identity is wrong); the remaining errors describe a backend that is
// disabled, misconfigured, or unreachable and should be surfaced as a
// server-side condition rather than a credential rejection.
var (
	// ErrDisabled indicates the LDAP backend is turned off (ldap.Enabled=false).
	ErrDisabled = errors.New("ldapauth: backend disabled")
	// ErrNotConfigured indicates required settings are missing for the selected bind mode.
	ErrNotConfigured = errors.New("ldapauth: backend not configured")
	// ErrEmptyCredential indicates an empty username or password was supplied;
	// it is rejected before any Bind to avoid an unauthenticated bind (RFC 4513 §5.1.2).
	ErrEmptyCredential = errors.New("ldapauth: empty username or password")
	// ErrInvalidCredentials indicates the directory rejected the user's Bind (wrong password).
	ErrInvalidCredentials = errors.New("ldapauth: invalid credentials")
	// ErrUserNotFound indicates search mode found no single matching directory entry.
	ErrUserNotFound = errors.New("ldapauth: user not found")
	// ErrUnavailable indicates the directory could not be reached or returned an unexpected error.
	ErrUnavailable = errors.New("ldapauth: directory unavailable")
)

// BindMode selects how a username is resolved to the directory entry whose
// password is verified.
type BindMode string

const (
	// BindModeTemplate substitutes the username into BindDNTemplate to form the
	// user's bind DN directly (for example uid=%s,ou=people,dc=example,dc=com or
	// the Active Directory userPrincipalName form %s@example.com). No directory
	// search is performed.
	BindModeTemplate BindMode = "template"
	// BindModeSearch first binds as a service account (SearchBindDN /
	// SearchBindPassword), searches BaseDN with UserFilter to locate the user's
	// DN, then re-binds as that DN with the supplied password. Use it when the
	// user's DN is not a fixed function of the username (for example Active
	// Directory sAMAccountName lookups).
	BindModeSearch BindMode = "search"
)

// Config is the resolved LDAP backend configuration. Use LoadConfig to build
// it from the application settings store.
type Config struct {
	Enabled            bool          // whether the backend is active
	ServerURL          string        // ldap:// or ldaps:// URL, e.g. ldaps://dc.example.com:636
	BaseDN             string        // search base for BindModeSearch, e.g. dc=example,dc=com
	BindMode           BindMode      // template or search
	BindDNTemplate     string        // BindModeTemplate DN template with a single %s for the username
	SearchBindDN       string        // service-account DN for BindModeSearch
	SearchBindPassword string        // service-account password for BindModeSearch
	UserFilter         string        // BindModeSearch filter with a single %s for the username, e.g. (uid=%s)
	StartTLS           bool          // upgrade an ldap:// connection with StartTLS before binding
	TLSSkipVerify      bool          // skip TLS certificate verification (lab use only)
	Timeout            time.Duration // per-operation timeout, and the budget the RADIUS adapter arms as the deadline for one whole exchange
}

// Settings is the read-only configuration accessor LoadConfig needs. It is the
// subset of *app.Application used here, declared as an interface so this
// package stays decoupled from internal/app (avoiding an import cycle once the
// pipeline wiring imports ldapauth) and can be exercised with a fake in tests.
type Settings interface {
	GetSettingsStringValue(category, key string) string
	GetSettingsInt64Value(category, key string) int64
	GetSettingsBoolValue(category, key string) bool
}

// LoadConfig reads the ldap.* settings into a Config. A missing or
// non-positive timeout falls back to defaultTimeout, and any bind mode other
// than search is normalized to template so an unset value yields the safe
// default rather than an invalid mode.
func LoadConfig(s Settings) Config {
	timeout := time.Duration(s.GetSettingsInt64Value(settingsCategory, "Timeout")) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	mode := BindMode(strings.TrimSpace(s.GetSettingsStringValue(settingsCategory, "BindMode")))
	if mode != BindModeSearch {
		mode = BindModeTemplate
	}
	return Config{
		Enabled:            s.GetSettingsBoolValue(settingsCategory, "Enabled"),
		ServerURL:          strings.TrimSpace(s.GetSettingsStringValue(settingsCategory, "ServerURL")),
		BaseDN:             strings.TrimSpace(s.GetSettingsStringValue(settingsCategory, "BaseDN")),
		BindMode:           mode,
		BindDNTemplate:     strings.TrimSpace(s.GetSettingsStringValue(settingsCategory, "BindDNTemplate")),
		SearchBindDN:       strings.TrimSpace(s.GetSettingsStringValue(settingsCategory, "SearchBindDN")),
		SearchBindPassword: s.GetSettingsStringValue(settingsCategory, "SearchBindPassword"),
		UserFilter:         strings.TrimSpace(s.GetSettingsStringValue(settingsCategory, "UserFilter")),
		StartTLS:           s.GetSettingsBoolValue(settingsCategory, "StartTLS"),
		TLSSkipVerify:      s.GetSettingsBoolValue(settingsCategory, "TLSSkipVerify"),
		Timeout:            timeout,
	}
}

// Validate reports whether the configuration is usable. It returns ErrDisabled
// when the backend is off, and ErrNotConfigured when a setting required by the
// selected bind mode is missing.
func (c Config) Validate() error {
	if !c.Enabled {
		return ErrDisabled
	}
	if c.ServerURL == "" {
		return fmt.Errorf("%w: ServerURL is empty", ErrNotConfigured)
	}
	switch c.BindMode {
	case BindModeTemplate:
		if c.BindDNTemplate == "" {
			return fmt.Errorf("%w: BindDNTemplate is empty", ErrNotConfigured)
		}
	case BindModeSearch:
		if c.BaseDN == "" || c.UserFilter == "" {
			return fmt.Errorf("%w: BaseDN and UserFilter are required for search mode", ErrNotConfigured)
		}
	default:
		return fmt.Errorf("%w: unknown bind mode %q", ErrNotConfigured, c.BindMode)
	}
	return nil
}

// tlsConfig builds the TLS configuration applied to ldaps:// connections and
// to StartTLS upgrades. ServerName is derived from the URL host so certificate
// verification works; it is ignored when TLSSkipVerify is set.
func (c Config) tlsConfig() *tls.Config {
	host := c.ServerURL
	if u, err := url.Parse(c.ServerURL); err == nil && u.Host != "" {
		host = u.Hostname()
	}
	return &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: c.TLSSkipVerify, //nolint:gosec // operator opt-in for lab/self-signed directories
	}
}

// conn is the minimal subset of *ldap.Conn used by the verifier. Declaring it
// as an interface lets tests substitute a fake directory connection.
type conn interface {
	StartTLS(config *tls.Config) error
	Bind(username, password string) error
	Search(req *ldap.SearchRequest) (*ldap.SearchResult, error)
	Close() error
	SetTimeout(time.Duration)
}

// dialFunc opens a connection to the directory described by cfg. It receives
// the caller's context so a request whose deadline has already lapsed can bail
// out before the socket is opened and so a dial that outlives the caller is
// abandoned instead of leaking the connection.
type dialFunc func(ctx context.Context, cfg Config) (conn, error)

// openLDAP performs the socket-level dial dialReal races against the caller's
// context. It is separated so the abandonment path below is testable without a
// real directory standing still for a timeout.
func openLDAP(cfg Config) (conn, error) {
	opts := []ldap.DialOpt{
		ldap.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}),
		ldap.DialWithTLSConfig(cfg.tlsConfig()),
	}
	return ldap.DialURL(cfg.ServerURL, opts...)
}

// dialReal connects to cfg.ServerURL, honoring the dial timeout and (for
// ldaps://) the TLS configuration. It does not arm the per-operation timeout:
// Verify does that once it knows how much of the caller's deadline is left.
// The dial races against ctx so a stalled directory cannot pin a RADIUS worker
// beyond the caller's own deadline: when ctx expires first, the socket that
// openLDAP eventually returns is closed by a background drain and ctx.Err() is
// surfaced.
func dialReal(ctx context.Context, cfg Config) (conn, error) {
	return dialAbandonable(ctx, cfg, openLDAP)
}

// dialAbandonable runs open in a goroutine and returns as soon as ctx is done.
// The goroutine keeps running, so its result is drained in the background and
// any socket it produces is closed rather than leaked.
func dialAbandonable(ctx context.Context, cfg Config, open func(Config) (conn, error)) (conn, error) {
	type dialResult struct {
		c   conn
		err error
	}
	ch := make(chan dialResult, 1)
	go func() {
		c, err := open(cfg)
		ch <- dialResult{c, err}
	}()
	select {
	case <-ctx.Done():
		// The dial goroutine may still be in flight; drain its result in
		// the background and close the socket so a slow directory cannot
		// leak a connection that nobody waits on.
		go func() {
			r := <-ch
			if r.err == nil && r.c != nil {
				_ = r.c.Close()
			}
		}()
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}
		return r.c, nil
	}
}

// Verifier authenticates usernames and passwords against the configured
// directory. Construct it with New and call Verify per authentication attempt.
// A Verifier is safe for concurrent use; each Verify opens and closes its own
// connection. Reuse is deliberately not implemented: a Bind leaves the
// connection carrying that subscriber's identity, so pooling one across
// authentications risks serving a later request under the previous user's
// identity (M14.5 measured the cost and recorded the re-open trigger in
// docs/ldap-connection-load-evidence-2026-10-31.md).
type Verifier struct {
	cfg  Config
	dial dialFunc
}

// New returns a Verifier bound to cfg using the production directory dialer.
func New(cfg Config) *Verifier {
	return &Verifier{cfg: cfg, dial: dialReal}
}

// Config returns the verifier's configuration.
func (v *Verifier) Config() Config { return v.cfg }

// Verify checks username and password against the directory.
//
// It returns nil when the directory accepts the user's Bind. It returns
// ErrInvalidCredentials or ErrUserNotFound for a failed authentication, and
// ErrDisabled, ErrNotConfigured, ErrEmptyCredential, or ErrUnavailable for
// conditions that are not credential rejections. Callers should compare the
// result with errors.Is.
//
// Empty usernames and passwords are rejected before any network operation:
// per RFC 4513 section 5.1.2 a Bind carrying a DN with an empty password is an
// unauthenticated bind that many servers complete as anonymous, which would
// otherwise falsely authenticate the user.
//
// The caller's context is honored at every checkpoint: a canceled or expired
// context bails out before the dial and before each Bind/Search, and the
// socket timeout is capped to the remaining deadline. A caller therefore bounds
// the whole directory exchange with one deadline instead of letting every round
// trip consume a timeout in full, and can abandon an in-flight authentication
// when its own budget runs out.
func (v *Verifier) Verify(ctx context.Context, username, password string) error {
	if err := v.cfg.Validate(); err != nil {
		return err
	}
	if username == "" || password == "" {
		return ErrEmptyCredential
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	c, err := v.dial(ctx, v.cfg)
	if err != nil {
		// A context error (canceled or deadline exceeded) is propagated as-is
		// so callers can distinguish "the request was abandoned" from "the
		// directory was unreachable". Wrapping it in ErrUnavailable would
		// collapse the two and report a directory outage for a request that
		// the caller had already given up on.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("%w: dial: %v", ErrUnavailable, err)
	}
	defer func() { _ = c.Close() }()

	// Arm the socket timeout to the smaller of the configured operation
	// timeout and the caller's remaining deadline, so a slow directory
	// cannot pin the worker past the request's own budget. armTimeout only
	// reports context errors, so the same "abandoned, not unreachable"
	// distinction as the dial path applies here.
	if err := armTimeout(c, ctx, v.cfg.Timeout); err != nil {
		return err
	}

	if v.cfg.StartTLS {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.StartTLS(v.cfg.tlsConfig()); err != nil {
			return fmt.Errorf("%w: starttls: %v", ErrUnavailable, err)
		}
	}

	switch v.cfg.BindMode {
	case BindModeSearch:
		return v.verifySearch(c, ctx, username, password)
	default:
		return v.verifyTemplate(c, ctx, username, password)
	}
}

// armTimeout sets the connection's per-operation timeout to the smaller of the
// configured timeout and the caller's remaining context deadline. It returns
// ctx.Err() when the deadline has already lapsed, so Verify fails fast instead
// of opening a socket for a request that has already been abandoned.
func armTimeout(c conn, ctx context.Context, timeout time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return ctx.Err()
		}
		if remaining < timeout {
			timeout = remaining
		}
	}
	c.SetTimeout(timeout)
	return nil
}

// verifyTemplate binds directly as the DN produced by substituting the
// username into BindDNTemplate.
func (v *Verifier) verifyTemplate(c conn, ctx context.Context, username, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	userDN := strings.ReplaceAll(v.cfg.BindDNTemplate, "%s", ldap.EscapeDN(username))
	if err := c.Bind(userDN, password); err != nil {
		return mapBindError(err)
	}
	return nil
}

// verifySearch binds as the service account, locates the user's DN with
// UserFilter, then re-binds as that DN with the supplied password. Each
// round trip is preceded by a context check so a deadline that lapses between
// the service bind and the search, or between the search and the user bind,
// aborts the authentication instead of continuing against a directory that is
// no longer answering the request.
func (v *Verifier) verifySearch(c conn, ctx context.Context, username, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.Bind(v.cfg.SearchBindDN, v.cfg.SearchBindPassword); err != nil {
		return fmt.Errorf("%w: service bind: %v", ErrUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// EscapeFilter neutralizes LDAP filter metacharacters in the
	// request-supplied username, preventing filter injection (RFC 4515 §3).
	filter := strings.ReplaceAll(v.cfg.UserFilter, "%s", ldap.EscapeFilter(username))
	req := ldap.NewSearchRequest(
		v.cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		2, int(v.cfg.Timeout.Seconds()), false,
		filter, []string{"dn"}, nil,
	)
	res, err := c.Search(req)
	if err != nil {
		return fmt.Errorf("%w: search: %v", ErrUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	switch {
	case len(res.Entries) == 0:
		return ErrUserNotFound
	case len(res.Entries) > 1:
		// An ambiguous filter must never authenticate an arbitrary match.
		return fmt.Errorf("%w: filter matched %d entries", ErrUserNotFound, len(res.Entries))
	}
	userDN := res.Entries[0].DN
	if userDN == "" {
		return ErrUserNotFound
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.Bind(userDN, password); err != nil {
		return mapBindError(err)
	}
	return nil
}

// mapBindError translates a user Bind failure into a sentinel: an
// invalid-credentials result code (RFC 4511 §4.1.9, code 49) becomes
// ErrInvalidCredentials; anything else is treated as a backend problem.
func mapBindError(err error) error {
	if err == nil {
		return nil
	}
	if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
		return ErrInvalidCredentials
	}
	return fmt.Errorf("%w: bind: %v", ErrUnavailable, err)
}
