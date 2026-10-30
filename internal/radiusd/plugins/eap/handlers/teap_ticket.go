package handlers

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
)

// RFC 9930 §3.11 makes the Phase 2 credentials part of a resumed session:
//
//	"the new credentials MUST be associated with the session ticket. If this
//	 association cannot be done, then the server MUST invalidate any session
//	 tickets for the current session."
//
// crypto/tls only knows the TLS session it resumed; which RADIUS identity was
// authenticated, and with which MSK, is TEAP's business. The two callbacks
// tlsengine.Config exposes (WrapSession and UnwrapSession) are crypto/tls's own
// tls.Config fields, so the association lives here: a ticket's opaque identity
// carries a reference to a record this package keeps, and the record names the
// credentials only after Phase 2 finished successfully.
//
// A ticket whose Phase 2 never completed has no record state and no credentials,
// so UnwrapSession refuses it (a nil session and a nil error, crypto/tls's way of
// falling back to a full handshake). That is the §3.11 invalidation, without a
// second call into crypto/tls.

const (
	// teapTicketPendingTTL bounds how long a ticket waits for the Phase 2 that
	// would complete it. A TEAP Phase 2 runs inside one EAP conversation, so a
	// record that is still pending after this window belongs to a failed or
	// abandoned exchange and must not be resumable.
	teapTicketPendingTTL = 5 * time.Minute
	// teapTicketCompleteTTL matches the 604800-second ticket lifetime crypto/tls
	// sends, so a record outlives its ticket by no more than the time the peer
	// needs to present it.
	teapTicketCompleteTTL = 7 * 24 * time.Hour
	// teapTicketIDLen is the number of random octets behind a ticket identity.
	// 32 octets make an identity unguessable, which matters because the store is
	// keyed by it and the record names the credentials it authorizes.
	teapTicketIDLen = 32
	// teapTicketMaxEntries caps the store so a peer that opens many handshakes
	// cannot grow it without bound.
	teapTicketMaxEntries = 4096
)

// teapTicketRecord is what a TEAP session ticket stands for: the TLS session
// state crypto/tls seals into the ticket, plus the credentials Phase 2 proved.
type teapTicketRecord struct {
	// state is the serialized tls.SessionState, or nil until WrapSession runs.
	state []byte
	// username and msk are the authenticated identity and the outer MSK, set
	// only when complete is true. RFC 9190 §5.7: "Authorization during resumption
	// MUST be based on such cached data."
	username string
	msk      []byte
	// complete reports that Phase 2 succeeded for this ticket.
	complete bool
	// expires bounds the record's life; it is refreshed when a ticket completes.
	expires time.Time
}

// teapTicketStore keeps the TEAP records behind the ticket identities this
// server issued. It is process-local: a deployment with several servers
// terminating the same peers would need a shared store, and until it has one a
// ticket issued by one server is simply not resumable on another.
type teapTicketStore struct {
	mu      sync.Mutex
	records map[string]*teapTicketRecord
	now     func() time.Time
}

// teapTickets is the store every TEAP handler shares. Tickets are only useful
// across handlers (the EAP state that issued a ticket is gone by the time the
// peer resumes), so the store cannot belong to a handler instance.
var teapTickets = newTEAPTicketStore()

// newTEAPTicketStore creates an empty store.
func newTEAPTicketStore() *teapTicketStore {
	return &teapTicketStore{records: make(map[string]*teapTicketRecord), now: time.Now}
}

// newTEAPTicketID returns a fresh, unguessable ticket identity.
func newTEAPTicketID() (string, error) {
	raw := make([]byte, teapTicketIDLen)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("TEAP session ticket identity: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// put stores a pending record for a freshly sealed ticket and returns nothing:
// the identity is the caller's.
func (s *teapTicketStore) put(id string, state []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	if len(s.records) >= teapTicketMaxEntries {
		s.evictOldestLocked()
	}
	s.records[id] = &teapTicketRecord{
		state:   append([]byte(nil), state...),
		expires: s.now().Add(teapTicketPendingTTL),
	}
}

// complete associates credentials with one ticket. It reports whether the
// ticket is still known, so a caller can tell a late Phase 2 from a ticket that
// was already swept.
func (s *teapTicketStore) complete(id, username string, msk []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.liveLocked(id)
	if !ok {
		return false
	}
	rec.username = username
	rec.msk = append([]byte(nil), msk...)
	rec.complete = true
	rec.expires = s.now().Add(teapTicketCompleteTTL)
	return true
}

// get returns the live record for an identity. An expired record is removed and
// reported absent.
func (s *teapTicketStore) get(id string) (teapTicketRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.liveLocked(id)
	if !ok {
		return teapTicketRecord{}, false
	}
	out := *rec
	out.state = append([]byte(nil), rec.state...)
	out.msk = append([]byte(nil), rec.msk...)
	return out, true
}

// delete removes a record, which is how a refused resumption invalidates the
// ticket a peer presented.
func (s *teapTicketStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, id)
}

// liveLocked returns the record if it exists and has not expired. The caller
// holds s.mu.
func (s *teapTicketStore) liveLocked(id string) (*teapTicketRecord, bool) {
	rec, ok := s.records[id]
	if !ok {
		return nil, false
	}
	if !s.now().Before(rec.expires) {
		delete(s.records, id)
		return nil, false
	}
	return rec, true
}

// sweepLocked drops every expired record. The caller holds s.mu.
func (s *teapTicketStore) sweepLocked() {
	now := s.now()
	for id, rec := range s.records {
		if !now.Before(rec.expires) {
			delete(s.records, id)
		}
	}
}

// evictOldestLocked drops one record to make room. The caller holds s.mu and has
// already swept, so every remaining record is live and the oldest expiry is the
// least valuable.
func (s *teapTicketStore) evictOldestLocked() {
	var oldestID string
	var oldest time.Time
	for id, rec := range s.records {
		if oldestID == "" || rec.expires.Before(oldest) {
			oldestID, oldest = id, rec.expires
		}
	}
	if oldestID != "" {
		delete(s.records, oldestID)
	}
}

// teapTicketHolder is the per-EAP-state half of the association. One holder is
// captured by the WrapSession and UnwrapSession closures of the outer TLS
// engine, so both callbacks see the same record of what this conversation
// issued and what it resumed.
//
// crypto/tls may call WrapSession from another goroutine, so every field is
// guarded by mu and the store is only touched while holding it.
type teapTicketHolder struct {
	// store is the process-wide ticket store; it is set once at construction.
	store *teapTicketStore

	mu sync.Mutex
	// issued lists the identities this handshake produced, so a Phase 2 that
	// finishes after the handshake can complete every one of them.
	issued []string
	// result is the credential pair Phase 2 proved, or nil while Phase 2 has not
	// succeeded. WrapSession consults it so a ticket sealed after the fact is
	// still associated.
	result *teapTicketCredentials
	// resumed is what a completed resumption brought back from the store. It
	// carries the cached credentials of the ticket the peer presented.
	resumed *teapTicketCredentials
	// refused lists identities whose resumption this handshake rejected, so the
	// ticket can be invalidated (§3.11).
	refused []string
}

// teapTicketCredentials are the values a ticket stands for.
type teapTicketCredentials struct {
	username string
	msk      []byte
}

// newTEAPTicketHolder creates the holder a TEAP state keeps in its Data map.
func newTEAPTicketHolder() *teapTicketHolder {
	return &teapTicketHolder{store: teapTickets}
}

// ticketHolder returns the holder stored on state, creating one on first use.
func ticketHolder(state *eap.EAPState) *teapTicketHolder {
	if h, ok := state.Data[stateKeyTEAPTicket].(*teapTicketHolder); ok && h != nil {
		return h
	}
	h := newTEAPTicketHolder()
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	state.Data[stateKeyTEAPTicket] = h
	return h
}

// wrap seals a TLS 1.3 session into a ticket identity this server can open
// again. It records the identity on the holder so a Phase 2 that finishes after
// the handshake can still associate its credentials, and it completes the ticket
// immediately when this handshake is itself a resumption: the credentials were
// cached, and §3.11 asks for the new credentials to be associated with the new
// ticket.
func (h *teapTicketHolder) wrap(_ tls.ConnectionState, ss *tls.SessionState) ([]byte, error) {
	id, err := newTEAPTicketID()
	if err != nil {
		return nil, err
	}
	state, err := ss.Bytes()
	if err != nil {
		return nil, fmt.Errorf("TEAP session ticket state: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.issued = append(h.issued, id)
	h.store.put(id, state)
	if h.result != nil {
		h.store.complete(id, h.result.username, h.result.msk)
		return []byte(id), nil
	}
	if h.resumed != nil {
		h.store.complete(id, h.resumed.username, h.resumed.msk)
	}
	return []byte(id), nil
}

// unwrap opens a ticket identity. It refuses a ticket whose Phase 2 never
// completed (and invalidates it), and otherwise records the cached credentials
// for the handshake to grant on.
func (h *teapTicketHolder) unwrap(idBytes []byte, _ tls.ConnectionState) (*tls.SessionState, error) {
	id := string(idBytes)

	h.mu.Lock()
	defer h.mu.Unlock()
	rec, ok := h.store.get(id)
	if !ok || !rec.complete {
		// §3.11: a ticket this server cannot associate credentials with must not
		// authorize anything, so it is invalidated rather than left to be tried
		// again.
		h.store.delete(id)
		h.refused = append(h.refused, id)
		return nil, nil
	}
	state, err := tls.ParseSessionState(rec.state)
	if err != nil {
		h.store.delete(id)
		h.refused = append(h.refused, id)
		return nil, nil
	}
	h.resumed = &teapTicketCredentials{username: rec.username, msk: rec.msk}
	return state, nil
}

// associate records the credentials Phase 2 proved and completes every ticket
// this handshake issued. §3.11's "new credentials MUST be associated with the
// session ticket" is this call; a handshake that never reaches it leaves its
// tickets pending, and a pending ticket never resumes.
func (h *teapTicketHolder) associate(username string, msk []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.result = &teapTicketCredentials{username: username, msk: append([]byte(nil), msk...)}
	for _, id := range h.issued {
		h.store.complete(id, username, msk)
	}
}

// resumedCredentials returns what a resumed handshake brought back, or nil.
func (h *teapTicketHolder) resumedCredentials() *teapTicketCredentials {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.resumed == nil {
		return nil
	}
	out := *h.resumed
	out.msk = append([]byte(nil), h.resumed.msk...)
	return &out
}
