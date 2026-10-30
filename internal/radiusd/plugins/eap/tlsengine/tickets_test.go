package tlsengine

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"sync"
	"testing"
	"time"
)

// ticketJar is a peer tls.ClientSessionCache that counts what it was given, so a
// test can ask how many NewSessionTicket records the server issued. Counting the
// server's own records cannot answer that: a TLS 1.3 ticket travels inside an
// encrypted application-data record, and only the peer ever sees it in the clear.
type ticketJar struct {
	tls.ClientSessionCache

	mu     sync.Mutex
	stored int
}

func newTicketJar() *ticketJar {
	return &ticketJar{ClientSessionCache: tls.NewLRUClientSessionCache(4)}
}

// Put counts a stored ticket. crypto/tls also calls Put with a nil state, which
// means "drop this session" and is not a ticket the server issued.
func (j *ticketJar) Put(key string, state *tls.ClientSessionState) {
	j.ClientSessionCache.Put(key, state)
	if state == nil {
		return
	}
	j.mu.Lock()
	j.stored++
	j.mu.Unlock()
}

func (j *ticketJar) count() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.stored
}

// tunnelConfig returns a server-only tunnel config - the shape PEAP, EAP-TTLS and
// TEAP hand the engine - with a managed certificate the test CA can issue.
func tunnelConfig(t *testing.T, ca *testCA) *Config {
	t.Helper()
	serverCert := ca.issue(t, "radius.example.com", func(c *x509.Certificate) {
		c.DNSNames = []string{"radius.example.com"}
	})
	return &Config{
		ServerCertificate: serverCert,
		ServerOnly:        true,
		MinVersion:        tls.VersionTLS12,
		HandshakeTimeout:  5 * time.Second,
	}
}

// runTunnelHandshake drives one tunnel handshake against an Engine built from cfg
// and returns the peer's connection state. The peer keeps its tickets in jar, so
// the second run of a pair can be told apart from a full handshake.
func runTunnelHandshake(t *testing.T, cfg *Config, ca *testCA, jar *ticketJar) tls.ConnectionState {
	t.Helper()

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = eng.Close() }()

	clientConn, toClient, fromClient := newClientPipe()
	client := tls.Client(clientConn, &tls.Config{
		RootCAs:            ca.pool,
		ServerName:         "radius.example.com",
		MinVersion:         tls.VersionTLS12,
		ClientSessionCache: jar,
	})

	clientDone := make(chan error, 1)
	go func() { clientDone <- client.Handshake() }()

	buf := make([]byte, 16*1024)
	var in []byte
	for round := 0; round < 16; round++ {
		out, done, err := eng.Process(in)
		if len(out) > 0 {
			if _, werr := toClient.Write(out); werr != nil {
				t.Fatalf("deliver server flight: %v", werr)
			}
		}
		if done {
			break
		}
		if err != nil {
			t.Fatalf("engine round %d: %v", round, err)
		}
		n, rerr := fromClient.Read(buf)
		if rerr != nil {
			t.Fatalf("read peer flight: %v", rerr)
		}
		in = append([]byte(nil), buf[:n]...)
	}

	select {
	case err := <-clientDone:
		if err != nil {
			t.Fatalf("peer handshake: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("peer handshake did not return")
	}

	// A TLS 1.3 peer only stores the ticket while it reads from the tunnel, so
	// close the server-to-peer direction and read to end of stream: every record
	// the server queued is processed and the read returns, rather than the test
	// waiting on a timeout that may or may not be long enough.
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

// TestEngine_SessionTicketsAreOptIn pins the M11.2 rule that the engine issues
// resumption tickets only when the method asks: the default leaves a TLS 1.3
// handshake free of the post-handshake record, and enabling it puts exactly one
// ticket in the server's final flight.
func TestEngine_SessionTicketsAreOptIn(t *testing.T) {
	ca := newTestCA(t, "Test Root CA")

	for _, tc := range []struct {
		name        string
		enable      bool
		wantTickets int
	}{
		{name: "off by default", enable: false, wantTickets: 0},
		{name: "on when the method resumes", enable: true, wantTickets: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tunnelConfig(t, ca)
			if tc.enable {
				cfg.EnableSessionTickets = true
				cfg.SessionTicketKey = newTicketKey(t)
			}

			jar := newTicketJar()
			state := runTunnelHandshake(t, cfg, ca, jar)
			if state.Version != tls.VersionTLS13 {
				t.Fatalf("negotiated %#x, want TLS 1.3", state.Version)
			}
			if got := jar.count(); got != tc.wantTickets {
				t.Fatalf("tickets issued = %d, want %d", got, tc.wantTickets)
			}
		})
	}
}

// TestEngine_TicketsResumeOnTheNextHandshake is why Config.SessionTicketKey is
// required rather than defaulted: the engine builds a fresh crypto/tls config per
// EAP exchange, so a key crypto/tls randomizes for itself seals a ticket no later
// engine can open. One key across the pair is what makes RFC 9930 §2 resumption
// real; a second key is the same tunnel restarted, whose ticket a peer cannot use.
func TestEngine_TicketsResumeOnTheNextHandshake(t *testing.T) {
	ca := newTestCA(t, "Test Root CA")
	key := newTicketKey(t)

	for _, tc := range []struct {
		name       string
		secondKey  []byte
		wantResume bool
	}{
		{name: "same key resumes", secondKey: key, wantResume: true},
		{name: "a per-handshake key does not", secondKey: newTicketKey(t), wantResume: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := tunnelConfig(t, ca)
			first.EnableSessionTickets = true
			first.SessionTicketKey = key

			jar := newTicketJar()
			if state := runTunnelHandshake(t, first, ca, jar); state.DidResume {
				t.Fatal("the initial handshake must not resume")
			}
			if got := jar.count(); got != 1 {
				t.Fatalf("tickets issued = %d, want 1", got)
			}

			second := tunnelConfig(t, ca)
			second.EnableSessionTickets = true
			second.SessionTicketKey = tc.secondKey
			state := runTunnelHandshake(t, second, ca, jar)
			if state.DidResume != tc.wantResume {
				t.Fatalf("second handshake DidResume = %v, want %v", state.DidResume, tc.wantResume)
			}
		})
	}
}

// TestEngine_NewRejectsTicketsWithoutAKey proves the pairing of the two fields is
// enforced, not merely documented: an engine cannot claim resumability with a key
// too short to seal a ticket.
func TestEngine_NewRejectsTicketsWithoutAKey(t *testing.T) {
	ca := newTestCA(t, "Test Root CA")

	for _, tc := range []struct {
		name string
		key  []byte
	}{
		{name: "no key", key: nil},
		{name: "short key", key: make([]byte, SessionTicketKeySize-1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tunnelConfig(t, ca)
			cfg.EnableSessionTickets = true
			cfg.SessionTicketKey = tc.key
			_, err := New(cfg)
			if !errors.Is(err, ErrNoSessionTicketKey) {
				t.Fatalf("got %v, want %v", err, ErrNoSessionTicketKey)
			}
		})
	}
}

// newTicketKey returns a session ticket key, the same construction a configuration
// provider uses once for its lifetime.
func newTicketKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, SessionTicketKeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate session ticket key: %v", err)
	}
	return key
}
