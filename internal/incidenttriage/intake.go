// Package incidenttriage is the FUT-002 research prototype: it takes the
// operator log export in its sanitized form, groups it into incidents, triages
// each incident into a cause taxonomy, and measures what the answer costs in
// accuracy and hallucination. Nothing in the product imports it, and it changes
// nothing: the only file access is a read of the tree it studies.
package incidenttriage

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/redact"
)

// RawLine is one line exactly as the log file holds it.
type RawLine struct {
	Index  int
	Text   string
	Fields map[string]any
}

// SanitizedLine is the form the operator log export hands out: the same shape
// internal/adminapi/system_logs.go reads, with internal/redact applied to
// every value and to the free-text message.
//
// It is a separate type on purpose. A triage engine can only be handed a
// []SanitizedLine, and the only way to build one is Sanitize, so "triage ran on
// sanitized logs" is a property of the type, not of a test.
type SanitizedLine struct {
	Index     int
	Timestamp string
	Level     string
	Namespace string
	Job       string
	Message   string
	Fields    map[string]any
}

// Text is everything a rule may match on: the message plus the field values,
// in a stable order.
func (l SanitizedLine) Text() string {
	keys := make([]string, 0, len(l.Fields))
	for k := range l.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(l.Level)
	b.WriteByte(' ')
	b.WriteString(l.Namespace)
	b.WriteByte(' ')
	b.WriteString(l.Job)
	b.WriteByte(' ')
	b.WriteString(l.Message)
	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(stringify(l.Fields[k]))
	}
	return b.String()
}

// Intake is the parsed, pre-sanitization view of one export batch.
type Intake struct {
	Lines []RawLine
	Bad   []int // indices that are not structured log lines at all
}

// Parse mirrors the export's own parser: a line that is not JSON survives as
// free text, because that is what querySystemLogs does with it.
func Parse(lines []string) Intake {
	in := Intake{Lines: make([]RawLine, 0, len(lines))}
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			in.Bad = append(in.Bad, i)
			continue
		}
		fields := map[string]any{}
		if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
			in.Lines = append(in.Lines, RawLine{Index: i, Text: trimmed})
			continue
		}
		in.Lines = append(in.Lines, RawLine{Index: i, Text: trimmed, Fields: fields})
	}
	return in
}

// SanitizeStats counts what redaction touched, so the memo can state that the
// corpus actually carried something to redact rather than being pre-cleaned.
type SanitizeStats struct {
	Structured    int
	SecretKeys    int
	PIIKeys       int
	AddressKeys   int
	MessagesSwept int
	LeakedValues  []string
}

// Sanitize runs internal/redact over the batch, the same path the operator
// export uses. secrets names the raw values that must not survive; any of them
// found in the sanitized output is reported as a leak rather than hidden.
func (in Intake) Sanitize(secrets []string) ([]SanitizedLine, SanitizeStats) {
	st := SanitizeStats{}
	out := make([]SanitizedLine, 0, len(in.Lines))
	for _, raw := range in.Lines {
		line := SanitizedLine{Index: raw.Index, Fields: map[string]any{}}
		if len(raw.Fields) == 0 {
			line.Message = redact.Text(raw.Text)
			if line.Message != raw.Text {
				st.MessagesSwept++
			}
			out = append(out, line)
			continue
		}
		st.Structured++
		line.Timestamp = timestampField(raw.Fields)
		line.Level = stringField(raw.Fields, "level", "Level", "l")
		line.Namespace = stringField(raw.Fields, "namespace", "Namespace")
		line.Job = stringField(raw.Fields, "job", "Job")
		line.Message = redact.Text(stringField(raw.Fields, "msg", "message", "Message"))
		if line.Message != stringField(raw.Fields, "msg", "message", "Message") {
			st.MessagesSwept++
		}
		for k, v := range raw.Fields {
			switch redact.ClassifyKey(k) {
			case redact.Secret:
				st.SecretKeys++
			case redact.CustomerPII:
				st.PIIKeys++
			case redact.Address:
				st.AddressKeys++
			}
			line.Fields[k] = redact.Value(k, v)
		}
		out = append(out, line)
	}
	st.LeakedValues = leaked(out, secrets)
	return out, st
}

// leaked reports every declared secret that still appears in sanitized text.
func leaked(lines []SanitizedLine, secrets []string) []string {
	var hits []string
	for _, s := range secrets {
		if strings.TrimSpace(s) == "" {
			continue
		}
		for _, l := range lines {
			if strings.Contains(l.Text(), s) {
				hits = append(hits, s)
				break
			}
		}
	}
	sort.Strings(hits)
	return dedupe(hits)
}

func stringField(fields map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := fields[k]; ok {
			return strings.TrimSpace(stringify(v))
		}
	}
	return ""
}

// timestampField mirrors the export's own timestamp handling: a zap numeric
// second becomes an RFC3339 stamp, a textual one is read as it stands. Without
// this conversion every line looks timestamp-less and grouping would merge the
// whole stream into one incident.
func timestampField(fields map[string]any) string {
	if ts, ok := fields["ts"].(float64); ok {
		return time.Unix(0, int64(ts*float64(time.Second))).Format(time.RFC3339)
	}
	return stringField(fields, "timestamp", "time", "Time")
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func dedupe(in []string) []string {
	var out []string
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
