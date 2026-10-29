package incidenttriage

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/bjo163/mwx-isp/internal/redact"
)

// Case is one labeled incident in the dataset. The lines are written as raw log
// records on purpose: they carry a NAS secret, a customer e-mail, a full IPv4
// and an inline password assignment, so the redaction step has real work to do
// before any rule sees them.
type Case struct {
	ID        string
	Namespace string
	Job       string
	Records   []Record
	Labels    []string
	Why       string
}

// Record is one log line before sanitization.
type Record struct {
	Level  string
	Msg    string
	Fields map[string]any
	Offset time.Duration
}

// Unresolved labels the cases where the honest answer is "the lines do not say".
const Unresolved = "unresolved"

// base is an arbitrary hour; the dataset is synthetic and says so. Only the
// gaps between lines matter here.
var base = time.Date(2026, 10, 29, 3, 0, 0, 0, time.UTC)

// Dataset is the labeled corpus: 20 incidents - 17 with one cause, 3 that must
// abstain. Every message string is copied from a template the product writes,
// and claims.go proves each token still resolves to a code line.
//
// The namespace on each case is the literal the code that produces that message
// attaches: the auth pipeline and its handlers log "radius", the admin API logs
// "adminapi", and the paths that run through internal/domain or internal/notify
// log no namespace at all. No case carries a job, because no product line
// attaches a job field - claims.go and the report both measure that.
func Dataset() []Case {
	return []Case{
		{ID: "c01", Namespace: "radius", Labels: []string{"credential-rejected"},
			Why: "the reject tag is the only thing the auth validators return for a bad credential",
			Records: []Record{{Level: "error", Msg: "authentication rejected: radus_reject_passwd_error",
				Fields: map[string]any{"user_name": "sutarmi-04", "nas_secret": "s3cr3t-nas-1", "framed_ip": "10.20.30.44"},
				Offset: 0}}},
		{ID: "c02", Namespace: "radius", Labels: []string{"credential-rejected"},
			Why: "an MS-CHAP2-Response whose peer value did not match is still a credential rejection",
			Records: []Record{{Level: "error", Msg: "MS-CHAP2-Response is 49 octets, expected 50",
				Fields: map[string]any{"user_name": "wahyu-11", "email": "wahyu@example.com"},
				Offset: 0}}},
		{ID: "c03", Labels: []string{"profile-unavailable"},
			Why: "the profile cache is the only producer of this message",
			Records: []Record{{Level: "error", Msg: "failed to get profile from cache",
				Fields: map[string]any{"user_name": "rani-07", "profile_id": 12},
				Offset: 0}}},
		{ID: "c04", Namespace: "adminapi", Labels: []string{"profile-unavailable"},
			Why: "a subscription whose package has no linked profile cannot authorize anything",
			Records: []Record{{Level: "error", Msg: "subscription cannot be activated: package has no linked RADIUS profile",
				Fields: map[string]any{"customer_id": 42, "password": "hunter2"},
				Offset: 0}}},
		{ID: "c05", Namespace: "radius", Labels: []string{"directory-unavailable"},
			Why: "the service bind is the directory's own failure string",
			Records: []Record{{Level: "error", Msg: "ldap: service bind: dial tcp 10.0.4.9:636: connect: connection refused",
				Fields: map[string]any{"customername": "Budi Santoso"},
				Offset: 0}}},
		{ID: "c06", Namespace: "radius", Labels: []string{"malformed-packet"},
			Why: "an AVP header running past the flight is a wire-format fault, not an auth fault",
			Records: []Record{{Level: "error", Msg: "ttls: truncated AVP header (7 bytes remaining)",
				Fields: map[string]any{"nas_ip": "10.20.30.1", "user_name": "odp-118"},
				Offset: 0}}},
		{ID: "c07", Namespace: "radius", Labels: []string{"tunnel-negotiation"},
			Why: "EAP-TLS with no trust anchor is a configuration refusal, not a client failure",
			Records: []Record{{Level: "error", Msg: "tls engine: ClientCAs is required for EAP-TLS peer authentication",
				Fields: map[string]any{"token": "abc.def.ghi"},
				Offset: 0}}},
		{ID: "c08", Namespace: "adminapi", Labels: []string{"tenant-integrity"},
			Why: "the restore gate is the only place that names a cross-tenant reference",
			Records: []Record{{Level: "error", Msg: "restore aborted: radius_users row 88 references record 19 from another tenant",
				Fields: map[string]any{"tenant_id": 3, "apikey": "k-1234567890"},
				Offset: 0}}},
		{ID: "c09", Namespace: "adminapi", Labels: []string{"backup-failed"},
			Why: "the backup endpoint validates before it writes",
			Records: []Record{{Level: "error", Msg: "Backup failed validation",
				Fields: map[string]any{"email": "ops@example.com"},
				Offset: 0}}},
		{ID: "c10", Labels: []string{"notification-failure"},
			Why: "the WhatsApp device store is the only producer of this string",
			Records: []Record{{Level: "error", Msg: "notify: load WhatsApp device: no such file or directory",
				Fields: map[string]any{"phone": "081234567890"},
				Offset: 0}}},
		{ID: "c11", Namespace: "radius", Labels: []string{"nas-unknown"},
			Why: "the stage name in the pipeline log is the NAS lookup",
			Records: []Record{{Level: "error", Msg: "auth pipeline: stage nas_lookup failed: client 10.20.30.7 is not a registered NAS",
				Fields: map[string]any{"nas_secret": "another-secret"},
				Offset: 0}}},
		{ID: "c12", Namespace: "radius", Labels: []string{"rate-limited"},
			Why: "the rate-limit stage names itself, and shedding is not an auth failure",
			Records: []Record{{Level: "warn", Msg: "auth pipeline: stage auth_rate_limit shed 41 attempts",
				Fields: map[string]any{},
				Offset: 0}}},
		{ID: "c13", Namespace: "radius", Labels: []string{"credential-rejected"},
			Why: "three lines of one rejected session still carry one cause",
			Records: []Record{
				{Level: "info", Msg: "auth attempt for MS-CHAP2-Response identity", Fields: map[string]any{"user_name": "rani-07"}, Offset: 0},
				{Level: "error", Msg: "MS-CHAP2-Response Ident 7 does not match the derived Ident 9", Fields: map[string]any{"framed_ip": "10.20.30.44"}, Offset: time.Minute},
				{Level: "error", Msg: "authentication rejected: radus_reject_passwd_error", Fields: map[string]any{"nas_secret": "s3cr3t-nas-1"}, Offset: 2 * time.Minute}}},
		{ID: "c14", Namespace: "adminapi", Labels: []string{"profile-unavailable"},
			Why: "a disabled profile behind a live subscription is the same class as a missing one",
			Records: []Record{{Level: "error", Msg: "activation refused: linked RADIUS profile is disabled",
				Fields: map[string]any{"customer_id": 91, "address": "Jl. Melati 12, Bandung"},
				Offset: 0}}},
		{ID: "c15", Namespace: "adminapi", Labels: []string{"tenant-integrity"},
			Why: "an unresolvable tenant id is the boundary class, not a packet class",
			Records: []Record{{Level: "error", Msg: "handler rejected request: invalid tenant ID 0",
				Fields: map[string]any{"tenant_id": 0},
				Offset: 0}}},
		{ID: "c16", Labels: []string{Unresolved},
			Why:     "a bare error with no taxonomy token must not be given a cause",
			Records: []Record{{Level: "error", Msg: "request failed", Fields: map[string]any{"status": 500}, Offset: 0}}},
		{ID: "c17", Labels: []string{Unresolved},
			Why: "two causes match one token each, so naming either would be a coin flip",
			Records: []Record{
				{Level: "error", Msg: "failed to get profile from cache", Fields: map[string]any{"user_name": "x-1"}, Offset: 0},
				{Level: "error", Msg: "ttls: truncated AVP header (3 bytes remaining)", Fields: map[string]any{"user_name": "x-1"}, Offset: time.Minute}}},
		{ID: "c18", Labels: []string{Unresolved},
			Why: "re-binding a pool is not the directory's service bind; the match has to be whole",
			Records: []Record{{Level: "info", Msg: "address pool re-binding completed for 12 prefixes",
				Fields: map[string]any{"pool": "pool-bandung-2"},
				Offset: 0}}},
		{ID: "c19", Namespace: "radius", Labels: []string{"malformed-packet"},
			Why: "a packet shorter than its authenticator is a wire fault",
			Records: []Record{{Level: "error", Msg: "radius: payload shorter than authenticator",
				Fields: map[string]any{"secret": "nas-shared-9"}, //nolint:gosec // G101: a planted value the leak check expects to see redacted; it authenticates nothing.
				Offset: 0}}},
		{ID: "c20", Labels: []string{"notification-failure"},
			Why: "the second notification incident keeps the class non-degenerate",
			Records: []Record{
				{Level: "error", Msg: "notify: load WhatsApp device: device store is locked", Fields: map[string]any{"msisdn": "62812345678"}, Offset: 0},
				{Level: "warn", Msg: "notification queue drained without a sender", Fields: map[string]any{"queued": 6}, Offset: time.Minute}}},
	}
}

// Secrets lists the raw values that must not survive sanitization. The list is
// derived from the dataset through the product's own key classifier, so a new
// row cannot carry a secret the leak check does not know about.
func Secrets(cases []Case) []string {
	var out []string
	for _, c := range cases {
		for _, r := range c.Records {
			keys := make([]string, 0, len(r.Fields))
			for k := range r.Fields {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if redact.ClassifyKey(k) == redact.Plain {
					continue
				}
				if s, ok := r.Fields[k].(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	sort.Strings(out)
	return dedupe(out)
}

// RawLines flattens the dataset into the stream the export would hand back,
// spacing the cases ten minutes apart so each one groups as its own incident.
func RawLines(cases []Case) []string {
	var out []string
	for i, c := range cases {
		for _, r := range c.Records {
			ts := base.Add(time.Duration(i)*10*time.Minute + r.Offset)
			rec := map[string]any{"ts": float64(ts.Unix()), "level": r.Level, "msg": r.Msg}
			if c.Namespace != "" {
				rec["namespace"] = c.Namespace
			}
			if c.Job != "" {
				rec["job"] = c.Job
			}
			for k, v := range r.Fields {
				rec[k] = v
			}
			b, err := json.Marshal(rec)
			if err != nil {
				out = append(out, r.Level+" "+r.Msg)
				continue
			}
			out = append(out, string(b))
		}
	}
	return out
}

// Pairing is one dataset case and the incident the grouping produced for it.
type Pairing struct {
	Case     Case
	Incident Incident
}

// Pair refuses rather than guessing which incident belongs to which label: the
// corpus and the grouping must line up exactly.
func Pair(cases []Case, incidents []Incident) ([]Pairing, error) {
	if len(cases) != len(incidents) {
		return nil, fmt.Errorf("dataset has %d cases but grouping produced %d incidents", len(cases), len(incidents))
	}
	out := make([]Pairing, 0, len(cases))
	for i, c := range cases {
		inc := incidents[i]
		if len(c.Records) != len(inc.Lines) {
			return nil, fmt.Errorf("%s declares %d lines but its incident holds %d", c.ID, len(c.Records), len(inc.Lines))
		}
		out = append(out, Pairing{Case: c, Incident: inc})
	}
	return out, nil
}

// LabelCount tallies how many cases carry each label, which the criterion asks
// to be reported rather than assumed.
func LabelCount(cases []Case) map[string]int {
	out := map[string]int{}
	for _, c := range cases {
		for _, l := range c.Labels {
			out[l]++
		}
	}
	return out
}
