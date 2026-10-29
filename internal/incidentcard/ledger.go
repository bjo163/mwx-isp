package incidentcard

import (
	"regexp"
	"sort"
	"strings"
)

// The card ledger is the answer to "what may this card print?". A slot is on it
// only when four separate readings of the product agree: the response type serves
// the key, the shipped redaction rules do not name it sensitive, nothing in the
// handlers proves the value is a subscriber column or a sentence built out of one,
// and - for a key that speaks of a named party - the assignment behind it reads a row
// that is not the subscriber's.

const (
	disprint    = "print"
	dRefused    = "refused"
	dStructural = "structural"
)

// taint is one proven case of the product putting a column of an identity table
// into prose. Guarded says whether the shipped rules already protect that column.
type taint struct {
	Field   string
	Table   string
	Column  string
	Guarded bool
	File    string
	Line    int
}

var reFieldFromSprintf = regexp.MustCompile(`^([A-Z]\w*):\s*(?:fmt\.)?Sprintf`)

// composeTaints pairs a tainted Sprintf with the struct field it writes: the format
// call and the field name it is assigned to open on the same source line.
func composeTaints(root string, prose []projection) []taint {
	var out []taint
	for _, p := range prose {
		src := lines(root, p.File)
		if p.Line < 1 || p.Line > len(src) {
			continue
		}
		m := reFieldFromSprintf.FindStringSubmatch(strings.TrimSpace(src[p.Line-1]))
		if m == nil {
			continue
		}
		out = append(out, taint{Field: m[1], Table: p.Table, Column: p.Col,
			Guarded: p.Guarded, File: p.File, Line: p.Line})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		if out[i].Column != out[j].Column {
			return out[i].Column < out[j].Column
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// buildLedger reads the incident response types and applies the three checks.
func buildLedger(root string, rules Rules, tables []identityTable, projs []projection, taints []taint, fills []filler) ([]slot, error) {
	byTable := map[string]identityTable{}
	for _, t := range tables {
		byTable[t.Table] = t
	}
	bound := map[string]projection{}
	for _, p := range projs {
		if _, isIdentity := byTable[p.Table]; !isIdentity {
			continue
		}
		if _, ok := bound[p.Key]; !ok {
			bound[p.Key] = p
		}
	}
	tainted := map[string]taint{}
	for _, t := range taints {
		if _, ok := tainted[t.Field]; !ok {
			tainted[t.Field] = t
		}
	}

	var out []slot
	seenStruct := 0
	for _, src := range incidentSources {
		want := map[string]bool{src.name: true}
		defs := readStructs(root, src.file, want)
		def, ok := defs[src.name]
		if !ok {
			return nil, structError{file: src.file, name: src.name}
		}
		seenStruct++
		for _, s := range def.fields {
			s.Class = rules.Classify(s.Key)
			switch s.Class {
			case "Secret":
				s.Disposition, s.Note, s.Guarded = dRefused, "credential: "+s.Class+" by key", true
			case "CustomerPII":
				s.Disposition, s.Note, s.Guarded = dRefused, "subscriber PII by key", true
			case "Address":
				s.Disposition, s.Note, s.Guarded = dRefused, "address or IP by key", true
			default:
				if p, ok := bound[s.Key]; ok {
					s.Disposition = dRefused
					s.From = p.Table + "." + p.Col
					s.Guarded = p.Guarded
					s.Proven = p.File + ":" + itoa(p.Line)
					s.Note = "query projection: holds " + s.From + ", " + guardWord(p.Guarded)
				} else if t, ok := tainted[s.Go]; ok {
					s.Disposition = dRefused
					s.Composed = t.File + ":" + itoa(t.Line)
					s.Guarded = t.Guarded
					s.Note = "prose slot: the product formats " + t.Table + "." + t.Column +
						" into it, " + guardWord(t.Guarded)
				} else if isNameShape(s.Key) {
					v := nameDecision(fillsFor(fills, s.Struct, s.Go))
					if v.OK {
						s.Disposition = disprint
						s.Note = v.Why
					} else {
						s.Disposition = dRefused
						s.Held = v.Held
						s.Guarded = v.Guarded
						s.Note = v.Why
					}
				} else {
					s.Disposition = disprint
					s.Note = structuralNote(s)
				}
			}
			out = append(out, s)
		}
	}
	if seenStruct != len(incidentSources) {
		return nil, structError{file: "count", name: itoa(seenStruct) + " of " + itoa(len(incidentSources))}
	}
	if len(out) == 0 {
		return nil, structError{file: "ledger", name: "empty"}
	}
	return out, nil
}

// guardWord tells the reader which side of the shipped policy a refusal sits on: a
// rule already names the column, or the card would have to reach past every rule and
// this ledger is the only thing standing in the way.
func guardWord(covered bool) string {
	if covered {
		return "a column the shipped rules name"
	}
	return "a column no shipped rule names"
}

// structuralNote records the slots a card cannot omit rather than ones it may not
// print: a count, a page handle or an id has no operational reading without it.
func structuralNote(s slot) string {
	switch {
	case strings.HasPrefix(s.Type, "map["):
		return "free-form field map: the card prints the keys it recognizes only"
	case s.Key == "page" || s.Key == "page_size" || s.Key == "total":
		return "pagination handle, not incident content"
	case strings.HasSuffix(s.Key, "_id") || s.Key == "id":
		return "opaque handle: needed to correlate, carries no subscriber value"
	}
	return ""
}

type structError struct {
	file string
	name string
}

func (e structError) Error() string {
	return "incident response type " + e.name + " not readable in " + e.file
}

// allowed returns the slots a card may print, in ledger order.
func allowed(slots []slot) []slot {
	var out []slot
	for _, s := range slots {
		if s.Disposition == disprint {
			out = append(out, s)
		}
	}
	return out
}

// refused returns the slots the three checks turned away.
func refused(slots []slot) []slot {
	var out []slot
	for _, s := range slots {
		if s.Disposition == dRefused {
			out = append(out, s)
		}
	}
	return out
}
