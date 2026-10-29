package incidenttriage

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Signal is one observable string that a rule keys on. Every signal here was
// copied from a message or a name the product itself writes; claims.go proves
// each one still resolves to a code line, so a rule cannot survive a rename in
// silence.
type Signal struct {
	Token string
	Note  string
}

// Cause is a root-cause class an incident may be triaged into.
type Cause struct {
	ID      string
	Label   string
	Signals []Signal
}

// Causes is the taxonomy. Ten classes, each anchored in log text the code
// emits - not in a general theory of ISP incidents.
//
//nolint:gosec // G101: every string in the table is a log token the rules match on, not a credential.
func Causes() []Cause {
	return []Cause{
		{ID: "credential-rejected", Label: "credential rejected by the auth pipeline", Signals: []Signal{
			{Token: "radus_reject_passwd_error", Note: "the reject tag the auth validators return for a bad credential"},
			{Token: "MS-CHAP2-Response", Note: "the inner response whose Ident or peer value failed to match"},
		}},
		{ID: "directory-unavailable", Label: "the directory behind authentication is unavailable", Signals: []Signal{
			{Token: "service bind", Note: "the LDAP service bind failed"},
			{Token: "BindDNTemplate is empty", Note: "the directory config cannot bind at all"},
			{Token: "starttls", Note: "the directory connection could not be upgraded"},
		}},
		{ID: "profile-unavailable", Label: "the RADIUS profile behind the account cannot be resolved", Signals: []Signal{
			{Token: "failed to get profile from cache", Note: "the profile cache returned nothing for the user"},
			{Token: "linked RADIUS profile is disabled", Note: "the subscription points at a disabled profile"},
			{Token: "package has no linked RADIUS profile", Note: "the package was never linked to a profile"},
		}},
		{ID: "malformed-packet", Label: "the packet or AVP flight is malformed", Signals: []Signal{
			{Token: "truncated AVP header", Note: "an AVP header runs past the end of the flight"},
			{Token: "AVP Length", Note: "the declared AVP length disagrees with the buffer"},
			{Token: "payload shorter than authenticator", Note: "the packet is smaller than its own authenticator"},
		}},
		{ID: "tunnel-negotiation", Label: "tunnel or TLS negotiation fell back or refused", Signals: []Signal{
			{Token: "ClientCAs is required", Note: "EAP-TLS peer authentication has no trust anchor configured"},
			{Token: "preferred EAP method disabled", Note: "the method the deployment wanted was not available"},
		}},
		{ID: "tenant-integrity", Label: "a row crosses a tenant boundary", Signals: []Signal{
			{Token: "from another tenant", Note: "a restored or exported row references a record of a different tenant"},
			{Token: "invalid tenant ID", Note: "the request names a tenant the process cannot resolve"},
		}},
		{ID: "backup-failed", Label: "the backup job failed its own gate", Signals: []Signal{
			{Token: "Backup failed validation", Note: "the uploaded backup did not validate"},
			{Token: "Backup encryption passphrase is too short", Note: "the configured passphrase is below the floor"},
		}},
		{ID: "notification-failure", Label: "the outbound notification channel is broken", Signals: []Signal{
			{Token: "load WhatsApp device", Note: "the stored device for the notification channel could not be opened"},
		}},
		{ID: "nas-unknown", Label: "the NAS that sent the request is not in the inventory", Signals: []Signal{
			{Token: "nas_lookup", Note: "the authentication stage that resolves the client into a NAS record"},
		}},
		{ID: "rate-limited", Label: "authentication was rate limited", Signals: []Signal{
			{Token: "auth_rate_limit", Note: "the pipeline stage that sheds excessive auth attempts"},
		}},
	}
}

// CauseIDs lists the taxonomy for reporting.
func CauseIDs() []string {
	var ids []string
	for _, c := range Causes() {
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	return ids
}

// Incident is a run of sanitized lines the export shows together: same
// namespace and job, no gap wider than the window. The operator reads the log
// export as a stream, so this is the unit a triage verdict should speak for.
type Incident struct {
	ID        string
	Namespace string
	Job       string
	Lines     []SanitizedLine
	Window    time.Duration
}

// Text is everything the rules may match on for this incident.
func (inc Incident) Text() string {
	var b strings.Builder
	for _, l := range inc.Lines {
		b.WriteString(l.Text())
		b.WriteByte('\n')
	}
	return b.String()
}

// Group turns a sanitized batch into incidents. Lines with no timestamp at all
// stay together only if they are adjacent, so an unparseable batch cannot be
// silently merged into one giant incident.
func Group(lines []SanitizedLine, window time.Duration) []Incident {
	var out []Incident
	var cur *Incident
	var last time.Time
	flush := func() {
		if cur == nil {
			return
		}
		cur.ID = "inc-" + pad(len(out)+1)
		out = append(out, *cur)
		cur = nil
	}
	for _, l := range lines {
		ts, ok := parseStamp(l.Timestamp)
		same := cur != nil && cur.Namespace == l.Namespace && cur.Job == l.Job &&
			(!ok || !last.IsZero() && ts.Sub(last) <= window)
		if !same {
			flush()
			cur = &Incident{Namespace: l.Namespace, Job: l.Job, Window: window}
		}
		cur.Lines = append(cur.Lines, l)
		if ok {
			last = ts
		}
	}
	flush()
	return out
}

func parseStamp(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func pad(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
