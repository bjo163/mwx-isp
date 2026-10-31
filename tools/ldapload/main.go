// Command ldapload measures what one LDAP-backed RADIUS authentication
// actually costs, so the M14.5 connection-robustness decision is made from
// numbers instead of intuition.
//
// The production path (internal/ldapauth) opens a fresh directory connection
// per authentication and closes it afterwards. Whether that is worth changing
// depends on how much of a single authentication is connection setup versus
// the Bind round trip, and on whether a canceled RADIUS request can interrupt
// a directory call that is already in flight. This command measures both:
//
//   - dial:       TCP (and, for ldaps://, TLS) setup with no LDAP operation
//   - verify:     the production path — dial, Bind (plus the service Bind and
//     Search in search mode), close
//   - warm bind:  a Bind issued on an already-open connection, i.e. the floor
//     a connection pool could reach
//   - throughput: verify at several concurrency levels, errors counted
//   - cancellation: a request whose context expires while the directory stays
//     silent, timed against the configured operation timeout
//
// verify minus warm bind is the price of the missing pool; the cancellation
// stage shows whether the request context reaches the directory call.
//
// The command only reads: it binds with the credentials you pass and never
// writes to the directory, so it is safe against a real server — though it does
// add load, so keep -requests modest.
//
// Usage against the OpenLDAP test service:
//
//	docker compose -f docker-compose.test.yml up -d openldap
//	go run ./tools/ldapload \
//	  -url ldap://127.0.0.1:1389 \
//	  -bind-mode template -bind-dn-template 'cn=%s,ou=users,dc=example,dc=org' \
//	  -user alice -password alice-password \
//	  -search-bind-dn cn=admin,dc=example,dc=org -search-bind-password adminpassword \
//	  -base-dn dc=example,dc=org -user-filter '(cn=%s)'
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/bjo163/mwx-isp/internal/ldapauth"
)

type options struct {
	url                string
	bindMode           string
	bindDNTemplate     string
	baseDN             string
	userFilter         string
	searchBindDN       string
	searchBindPassword string
	user               string
	password           string
	startTLS           bool
	skipVerify         bool
	timeout            time.Duration
	requests           int
	warmup             int
	concurrencies      concurrencyList
	cancelDeadline     time.Duration
}

func main() {
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "ldapload: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags(args []string) (options, error) {
	opts := options{concurrencies: concurrencyList{1, 10, 50}}

	fs := flag.NewFlagSet("ldapload", flag.ContinueOnError)
	fs.StringVar(&opts.url, "url", envDefault("TEST_LDAP_URL", "ldap://127.0.0.1:1389"), "ldap://, ldaps:// or ldapi:// directory URL")
	fs.StringVar(&opts.bindMode, "bind-mode", "template", "template or search — the ldapauth bind mode to measure")
	fs.StringVar(&opts.bindDNTemplate, "bind-dn-template", envDefault("TEST_LDAP_BIND_DN_TEMPLATE", "cn=%s,ou=users,dc=example,dc=org"), "DN template with a single %s (template mode)")
	fs.StringVar(&opts.baseDN, "base-dn", envDefault("TEST_LDAP_BASE_DN", "dc=example,dc=org"), "search base (search mode)")
	fs.StringVar(&opts.userFilter, "user-filter", envDefault("TEST_LDAP_USER_FILTER", "(cn=%s)"), "user filter with a single %s (search mode)")
	fs.StringVar(&opts.searchBindDN, "search-bind-dn", envDefault("TEST_LDAP_ADMIN_DN", "cn=admin,dc=example,dc=org"), "service-account DN (search mode)")
	fs.StringVar(&opts.searchBindPassword, "search-bind-password", envDefault("TEST_LDAP_ADMIN_PASSWORD", "adminpassword"), "service-account password (search mode)")
	fs.StringVar(&opts.user, "user", envDefault("TEST_LDAP_USER", "alice"), "username to authenticate as")
	fs.StringVar(&opts.password, "password", envDefault("TEST_LDAP_PASSWORD", "alice-password"), "password for -user; a wrong value measures the reject path")
	fs.BoolVar(&opts.startTLS, "starttls", false, "upgrade an ldap:// connection with StartTLS before binding")
	fs.BoolVar(&opts.skipVerify, "tls-skip-verify", false, "skip TLS certificate verification (lab directories only)")
	fs.DurationVar(&opts.timeout, "timeout", 3*time.Second, "per-operation timeout, mirroring ldapauth Config.Timeout")
	fs.IntVar(&opts.requests, "requests", 200, "measurements per stage (after warmup)")
	fs.IntVar(&opts.warmup, "warmup", 20, "ignored measurements per latency stage")
	fs.Var(&opts.concurrencies, "concurrency", "comma-separated concurrency levels for the throughput stage")
	fs.DurationVar(&opts.cancelDeadline, "cancel-deadline", 200*time.Millisecond, "context deadline used by the cancellation stage")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if opts.bindMode != "template" && opts.bindMode != "search" {
		return opts, fmt.Errorf("-bind-mode must be template or search, got %q", opts.bindMode)
	}
	if opts.requests <= 0 || opts.warmup < 0 || len(opts.concurrencies) == 0 {
		return opts, fmt.Errorf("-requests and -concurrency must be positive")
	}
	return opts, nil
}

// concurrencyList is the flag.Value behind -concurrency.
type concurrencyList []int

func (c concurrencyList) String() string {
	parts := make([]string, 0, len(c))
	for _, n := range c {
		parts = append(parts, strconv.Itoa(n))
	}
	return strings.Join(parts, ",")
}

func (c *concurrencyList) Set(raw string) error {
	var parsed concurrencyList
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		n, err := strconv.Atoi(entry)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid concurrency level %q", entry)
		}
		parsed = append(parsed, n)
	}
	if len(parsed) == 0 {
		return fmt.Errorf("no concurrency levels given")
	}
	*c = parsed
	return nil
}

func envDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (o options) config() ldapauth.Config {
	return ldapauth.Config{
		Enabled:            true,
		ServerURL:          o.url,
		BaseDN:             o.baseDN,
		BindMode:           ldapauth.BindMode(o.bindMode),
		BindDNTemplate:     o.bindDNTemplate,
		SearchBindDN:       o.searchBindDN,
		SearchBindPassword: o.searchBindPassword,
		UserFilter:         o.userFilter,
		StartTLS:           o.startTLS,
		TLSSkipVerify:      o.skipVerify,
		Timeout:            o.timeout,
	}
}

func run(opts options) error {
	cfg := opts.config()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	if err := ldapauth.New(cfg).Verify(context.Background(), opts.user, opts.password); err != nil {
		return fmt.Errorf("the directory does not authenticate %q: %w", opts.user, err)
	}

	fmt.Printf("ldapload %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("url=%s bind-mode=%s timeout=%s requests=%d warmup=%d\n\n",
		opts.url, opts.bindMode, opts.timeout, opts.requests, opts.warmup)

	dial, err := measureDial(opts)
	if err != nil {
		return fmt.Errorf("dial stage: %w", err)
	}
	warm, err := measureWarmBind(opts)
	if err != nil {
		return fmt.Errorf("warm-bind stage: %w", err)
	}
	verify, err := measureVerify(opts)
	if err != nil {
		return fmt.Errorf("verify stage: %w", err)
	}

	fmt.Printf("%-28s %8s %8s %8s %8s %7s\n", "stage", "p50", "p95", "p99", "max", "errors")
	for _, s := range []stageStat{dial, warm, verify} {
		fmt.Printf("%-28s %8s %8s %8s %8s %7d\n", s.name, s.p50, s.p95, s.p99, s.max, s.errors)
	}
	setup := verify.mean - warm.mean
	fmt.Printf("\none verify averages %s, of which %s (%s of it) is connection setup"+
		" — that is what a pool could remove\n",
		duration(verify.mean), duration(setup), relative(setup, verify.mean))

	throughput, err := measureThroughput(opts)
	if err != nil {
		return fmt.Errorf("throughput stage: %w", err)
	}
	fmt.Printf("\n%-13s %10s %10s %8s\n", "concurrency", "auth/s", "p95", "errors")
	for _, t := range throughput {
		fmt.Printf("%-13d %10.1f %10s %8d\n", t.concurrency, t.rate, duration(t.p95), t.errors)
	}

	canceled, err := measureCancellation(opts)
	if err != nil {
		return fmt.Errorf("cancellation stage: %w", err)
	}
	fmt.Printf("\ncancellation: directory accepts the connection, then says nothing\n")
	fmt.Printf("  context deadline %s, operation timeout %s, Verify returned after %s\n",
		duration(opts.cancelDeadline), duration(opts.timeout), duration(canceled.waited))
	fmt.Printf("  error: %s\n", canceled.errText)
	if canceled.honored {
		fmt.Printf("  verdict: the caller stopped inside its own deadline — the request" +
			" context reaches the in-flight operation (dial races ctx, the socket timeout" +
			" is capped to the remaining deadline, and each Bind/Search is preceded by" +
			" a ctx check)\n")
	} else {
		fmt.Printf("  verdict: the caller waited on the directory instead of its deadline — the" +
			" request context does not reach the in-flight operation\n")
	}
	return nil
}

type stageStat struct {
	name                     string
	p50, p95, p99, max, mean time.Duration
	errors                   int
}

type throughputStat struct {
	concurrency int
	rate        float64
	p95         time.Duration
	errors      int
}

type cancellationStat struct {
	waited  time.Duration
	honored bool
	errText string
}

func summarize(name string, samples []time.Duration, errs int) stageStat {
	sorted := sortedCopy(samples)
	s := stageStat{name: name, errors: errs}
	if len(sorted) == 0 {
		return s
	}
	s.p50 = percentile(sorted, 0.50)
	s.p95 = percentile(sorted, 0.95)
	s.p99 = percentile(sorted, 0.99)
	s.max = sorted[len(sorted)-1]
	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	s.mean = sum / time.Duration(len(sorted))
	return s
}

// percentile reads the q-th quantile from an ascending sample set using the
// nearest-rank definition (ceil(q*n)), so a small sample set reports its worst
// observation rather than rounding a p99 down to the median.
func percentile(ascending []time.Duration, q float64) time.Duration {
	if len(ascending) == 0 {
		return 0
	}
	if q <= 0 {
		return ascending[0]
	}
	if q >= 1 {
		return ascending[len(ascending)-1]
	}
	rank := int(math.Ceil(q * float64(len(ascending))))
	if rank < 1 {
		rank = 1
	}
	return ascending[rank-1]
}

func sortedCopy(samples []time.Duration) []time.Duration {
	out := append([]time.Duration(nil), samples...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func duration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dus", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func relative(part, whole time.Duration) string {
	if whole <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(part)/float64(whole))
}

// dialRaw mirrors ldapauth's production dialer so the setup measurement is the
// same code path an authentication pays.
func dialRaw(opts options) (*ldap.Conn, error) {
	c, err := ldap.DialURL(opts.url,
		ldap.DialWithDialer(&net.Dialer{Timeout: opts.timeout}),
		ldap.DialWithTLSConfig(tlsConfigFor(opts)),
	)
	if err != nil {
		return nil, err
	}
	c.SetTimeout(opts.timeout)
	return c, nil
}

func tlsConfigFor(opts options) *tls.Config {
	host := opts.url
	if u, err := url.Parse(opts.url); err == nil && u.Host != "" {
		host = u.Hostname()
	}
	return &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: opts.skipVerify, //nolint:gosec // G402: explicit operator flag for lab directories
	}
}

// measureDial times connection setup alone: no LDAP operation is sent.
func measureDial(opts options) (stageStat, error) {
	samples, errs, err := sample(opts, func() error {
		c, err := dialRaw(opts)
		if err != nil {
			return err
		}
		return c.Close()
	})
	if err != nil {
		return stageStat{}, err
	}
	return summarize("dial (no operation)", samples, errs), nil
}

// measureVerify times the production path.
func measureVerify(opts options) (stageStat, error) {
	verifier := ldapauth.New(opts.config())
	ctx := context.Background()
	samples, errs, err := sample(opts, func() error {
		return verifier.Verify(ctx, opts.user, opts.password)
	})
	if err != nil {
		return stageStat{}, err
	}
	return summarize("verify (fresh connection)", samples, errs), nil
}

// measureWarmBind times a user Bind on one long-lived connection: the floor a
// pool reaches, because setup is paid once.
func measureWarmBind(opts options) (stageStat, error) {
	c, err := dialRaw(opts)
	if err != nil {
		return stageStat{}, err
	}
	defer func() { _ = c.Close() }()

	if opts.startTLS {
		if err := c.StartTLS(tlsConfigFor(opts)); err != nil {
			return stageStat{}, err
		}
	}
	if opts.bindMode == "search" {
		// The service Bind is part of the search-mode flow, so keep it out of
		// the measured user Bind exactly as the verifier does.
		if err := c.Bind(opts.searchBindDN, opts.searchBindPassword); err != nil {
			return stageStat{}, err
		}
	}
	dn := strings.ReplaceAll(opts.bindDNTemplate, "%s", ldap.EscapeDN(opts.user))
	samples, errs, err := sample(opts, func() error {
		return c.Bind(dn, opts.password)
	})
	if err != nil {
		return stageStat{}, err
	}
	return summarize("bind on warm connection", samples, errs), nil
}

// sample runs op warmup+requests times, dropping the warmup measurements.
func sample(opts options, op func() error) ([]time.Duration, int, error) {
	samples := make([]time.Duration, 0, opts.requests)
	var errs int
	for i := 0; i < opts.warmup+opts.requests; i++ {
		begin := time.Now()
		err := op()
		elapsed := time.Since(begin)
		if err != nil {
			errs++
			continue
		}
		if i >= opts.warmup {
			samples = append(samples, elapsed)
		}
	}
	if len(samples) == 0 {
		return nil, errs, fmt.Errorf("every measurement failed (%d errors)", errs)
	}
	return samples, errs, nil
}

func measureThroughput(opts options) ([]throughputStat, error) {
	verifier := ldapauth.New(opts.config())
	ctx := context.Background()

	results := make([]throughputStat, 0, len(opts.concurrencies))
	for _, level := range opts.concurrencies {
		perWorker := opts.requests / level
		if perWorker < 1 {
			perWorker = 1
		}
		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			all   []time.Duration
			errs  int
			start = time.Now()
		)
		for w := 0; w < level; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				local := make([]time.Duration, 0, perWorker)
				var localErrs int
				for i := 0; i < perWorker; i++ {
					begin := time.Now()
					if err := verifier.Verify(ctx, opts.user, opts.password); err != nil {
						localErrs++
						continue
					}
					local = append(local, time.Since(begin))
				}
				mu.Lock()
				all = append(all, local...)
				errs += localErrs
				mu.Unlock()
			}()
		}
		wg.Wait()
		elapsed := time.Since(start)
		results = append(results, throughputStat{
			concurrency: level,
			rate:        float64(level*perWorker) / elapsed.Seconds(),
			p95:         percentile(sortedCopy(all), 0.95),
			errors:      errs,
		})
	}
	return results, nil
}

// measureCancellation points the verifier at a listener that accepts and then
// stays silent, expires the caller's context well before the operation
// timeout, and times how long Verify really takes to come back.
func measureCancellation(opts options) (cancellationStat, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return cancellationStat{}, err
	}
	defer func() { _ = listener.Close() }()

	// One slot is enough: the stage makes exactly one authentication attempt.
	// Any further accepted socket is closed by the loop itself. The channel is
	// never closed — the accept goroutine exits when the listener closes.
	accepted := make(chan net.Conn, 1)
	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			select {
			case accepted <- c:
			default:
				_ = c.Close()
			}
		}
	}()

	blackhole := ldapauth.Config{
		Enabled:        true,
		ServerURL:      "ldap://" + listener.Addr().String(),
		BindMode:       ldapauth.BindModeTemplate,
		BindDNTemplate: "cn=%s,ou=users,dc=example,dc=org",
		Timeout:        opts.timeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.cancelDeadline)
	defer cancel()

	begin := time.Now()
	err = ldapauth.New(blackhole).Verify(ctx, opts.user, opts.password)
	waited := time.Since(begin)

	// Closing the silent peer releases the verifier's socket if it is still
	// waiting on a read after the caller's deadline.
	select {
	case c := <-accepted:
		_ = c.Close()
	case <-time.After(time.Second):
	}

	deadline := opts.cancelDeadline + 100*time.Millisecond
	return cancellationStat{
		waited:  waited,
		honored: waited <= deadline,
		errText: errorText(err),
	}, nil
}

func errorText(err error) string {
	if err == nil {
		return "nil"
	}
	return err.Error()
}
