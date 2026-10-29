package incidenttriage

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// claimDirs are the product locations a citation may be proven in. Research
// packages written by earlier rounds are deliberately not in this list, so a
// claim about the product cannot be confirmed by another study's code.
var claimDirs = []string{
	"internal/adminapi", "internal/app", "internal/domain", "internal/radiusd",
	"internal/redact", "internal/notify", "internal/ldapauth", "internal/supportbundle",
	"internal/webserver", "pkg", "cmd",
}

// Claim is one statement this study rests on, pinned to a token inside a named
// product file. The line number is discovered by the scan, never written here,
// so no citation can be invented.
type Claim struct {
	ID        string
	Statement string
	Token     string
	File      string
}

// ClaimStatus is a claim after the scan.
type ClaimStatus struct {
	Claim
	Confirmed bool
	Broken    bool
	Hits      []string
}

// Gap is an absence the study depends on.
type Gap struct {
	ID        string
	Statement string
	Dirs      []string
	Symbols   []string
}

// GapStatus is a gap after the scan. Vacated means the tree no longer lacks it.
type GapStatus struct {
	Gap
	Open    bool
	Vacated bool
	Hits    []string
}

// Claims are the product behaviors the triage design assumes.
//
//nolint:gosec // G101: a Token is a string searched for in the tree, never a credential used to authenticate.
func Claims() []Claim {
	return []Claim{
		{ID: "export-redacts-before-return",
			Statement: "the operator log export redacts every entry before it leaves the process",
			Token:     "redactSystemLogEntry", File: "internal/adminapi/system_logs.go"},
		{ID: "export-parses-json-lines",
			Statement: "the export reads structured log lines and keeps unstructured ones as free text",
			Token:     "parseSystemLogLine", File: "internal/adminapi/system_logs.go"},
		{ID: "redact-one-rule-set",
			Statement: "one classifier decides secret, PII and address keys, so the export and the support bundle cannot drift",
			Token:     "func ClassifyKey", File: "internal/redact/redact.go"},
		{ID: "reject-tag-is-stable",
			Statement: "a credential rejection is tagged with one stable string the validators return",
			Token:     "radus_reject_passwd_error", File: "internal/radiusd/auth_plugin_runner.go"},
		{ID: "reject-tag-is-a-metric",
			Statement: "the same string is a metric name, so it is a contract other code can depend on",
			Token:     "MetricsRadiusRejectPasswdError", File: "internal/app/radius_metrics.go"},
		{ID: "profile-cache-error-exists",
			Statement: "a profile that cannot be resolved from the cache is logged with its own message",
			Token:     "failed to get profile from cache", File: "internal/domain/radius_user_getters.go"},
		{ID: "pipeline-stages-are-named",
			Statement: "the authentication pipeline names its stages, which is what makes a NAS-lookup failure addressable",
			Token:     "StageNasLookup", File: "internal/radiusd/auth_stages.go"},
		{ID: "rate-limit-stage-exists",
			Statement: "shedding happens in a stage of its own, so a rate-limit incident is distinguishable from an auth failure",
			Token:     "StageRateLimit", File: "internal/radiusd/auth_stages.go"},
		{ID: "mschap-validator-rejects",
			Statement: "the MS-CHAPv2 validators are a real source of the reject tag",
			Token:     "radus_reject_passwd_error", File: "internal/radiusd/plugins/auth/validators/mschap_validator.go"},
		{ID: "avp-truncation-is-a-fault",
			Statement: "a truncated AVP header is reported as an inner-protocol fault",
			Token:     "truncated AVP header", File: "internal/radiusd/plugins/eap/handlers/ttls_avp.go"},
		{ID: "ldap-bind-missing-user",
			Statement: "the inner tunnel distinguishes a missing directory record from a bad password",
			Token:     "missing user record for ldap bind", File: "internal/radiusd/plugins/eap/handlers/ttls_inner.go"},
		{ID: "tls-needs-a-trust-anchor",
			Statement: "EAP-TLS refuses to run without ClientCAs, which is a configuration class rather than a client class",
			Token:     "ClientCAs is required", File: "internal/radiusd/plugins/eap/tlsengine/engine.go"},
		{ID: "restore-gates-cross-tenant",
			Statement: "the restore path already detects a row that references another tenant",
			Token:     "from another tenant", File: "internal/adminapi/system_backup.go"},
		{ID: "backup-validates-first",
			Statement: "the backup endpoint runs a validation gate before doing anything",
			Token:     "Backup failed validation", File: "internal/adminapi/system_backup.go"},
		{ID: "whatsapp-device-store",
			Statement: "the notification channel loads a stored device, and its failure is a distinct incident class",
			Token:     "load WhatsApp device", File: "internal/notify/whatsapp.go"},
		{ID: "namespace-is-coarse",
			Statement: "the namespace field the export filters on is written with a handful of coarse values, so a subsystem-name classifier cannot separate the taxonomy",
			Token:     `zap.String("namespace", "radius")`, File: "internal/radiusd/auth_stages.go"},
	}
}

// Gaps are the absences this study does not close.
func Gaps() []Gap {
	return []Gap{
		{ID: "no-triage-in-product", Statement: "no product code triages a log line into a cause class",
			Dirs:    []string{"internal/adminapi", "internal/app", "internal/radiusd", "internal/domain", "pkg", "cmd"},
			Symbols: []string{"Triage", "triage"}},
		{ID: "no-labeled-incident-dataset", Statement: "the repository holds no labeled incident dataset to learn from",
			Dirs:    []string{"internal", "pkg", "cmd", "test"},
			Symbols: []string{"LabelDataset", "IncidentLabel", "TrainingSet"}},
		{ID: "no-model-client", Statement: "nothing in the tree calls a language model, so the AI half of this card has no baseline in production",
			Dirs:    []string{"internal", "pkg", "cmd"},
			Symbols: []string{"LLMClient", "openai", "anthropic"}},
		{ID: "no-auto-remediation", Statement: "no path from a log line to an action exists, which is what keeps this card's no-remediation half true",
			Dirs:    []string{"internal/adminapi", "internal/app", "internal/radiusd"},
			Symbols: []string{"AutoRemediate", "auto_remediate", "RemediateIncident"}},
		{ID: "no-persisted-incident", Statement: "incidents exist only as a window over the log file; nothing is stored, so a triage verdict has no record to attach to",
			Dirs:    []string{"internal", "pkg"},
			Symbols: []string{"IncidentRecord", "IncidentID"}},
		{ID: "no-namespace-on-the-supporting-packages",
			Statement: "the domain, notify and directory packages attach no namespace field to any log line, so part of the taxonomy reaches the export with an empty subsystem bucket",
			Dirs:      []string{"internal/domain", "internal/notify", "internal/ldapauth"},
			Symbols:   []string{`String("namespace"`}},
	}
}

func verifyClaims(root string) []ClaimStatus {
	out := make([]ClaimStatus, 0, len(Claims()))
	for _, c := range Claims() {
		st := ClaimStatus{Claim: c}
		path := filepath.Join(root, filepath.FromSlash(c.File))
		src, err := readFile(path)
		if err != nil {
			st.Broken = true
			out = append(out, st)
			continue
		}
		lines := strings.Split(blankComments(src), "\n")
		if n := locate(lines, c.Token); n > 0 {
			st.Confirmed = true
			st.Hits = []string{c.File + ":" + itoa(n)}
		}
		out = append(out, st)
	}
	return out
}

func verifyGaps(root string) []GapStatus {
	var out []GapStatus
	for _, g := range Gaps() {
		st := GapStatus{Gap: g, Open: true}
		for _, dir := range g.Dirs {
			base := filepath.Join(root, filepath.FromSlash(dir))
			_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					if filepath.Base(path) == ownPackage {
						return filepath.SkipDir
					}
					return nil
				}
				if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return nil
				}
				src, err := readFile(path)
				if err != nil {
					return nil
				}
				lines := strings.Split(blankComments(src), "\n")
				for _, sym := range g.Symbols {
					if n := locate(lines, sym); n > 0 {
						st.Hits = append(st.Hits, relToRoot(path)+":"+itoa(n))
					}
				}
				return nil
			})
		}
		sort.Strings(st.Hits)
		st.Hits = dedupe(st.Hits)
		if len(st.Hits) > 0 {
			st.Open = false
			st.Vacated = true
		}
		out = append(out, st)
	}
	return out
}

// logFieldLiterals counts, per literal, what product code attaches to one zap
// string field. The export lets an operator filter on namespace and job, so the
// number of distinct values each field actually carries is the ceiling on any
// classifier that reads only those fields. It is counted from the tree, not
// recalled.
func logFieldLiterals(root, field string) map[string]int {
	pattern := regexp.MustCompile(`String\("` + regexp.QuoteMeta(field) + `", "([^"]+)"`)
	out := map[string]int{}
	files, err := productFiles(root)
	if err != nil {
		return out
	}
	for _, path := range files {
		src, err := readFile(path)
		if err != nil {
			continue
		}
		for _, m := range pattern.FindAllStringSubmatch(blankComments(src), -1) {
			out[m[1]]++
		}
	}
	return out
}

func sumValues(counts map[string]int) int {
	n := 0
	for _, c := range counts {
		n += c
	}
	return n
}
