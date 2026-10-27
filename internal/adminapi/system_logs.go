package adminapi

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/bjo163/mwx-isp/internal/webserver"
	webresp "github.com/bjo163/mwx-isp/pkg/web"
)

type systemLogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Namespace string                 `json:"namespace"`
	Job       string                 `json:"job"`
	Message   string                 `json:"message"`
	Line      string                 `json:"line"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

type systemLogQueryResult struct {
	Entries []systemLogEntry `json:"entries"`
	Total   int              `json:"total"`
	Limit   int              `json:"limit"`
}

func querySystemLogs(c echo.Context) error {
	now := time.Now()
	startTime, err := parseTimeInput(c.QueryParam("starttime"), now.Add(-1*time.Hour))
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TIME", "Invalid starttime format", nil)
	}
	endTime, err := parseTimeInput(c.QueryParam("endtime"), now)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TIME", "Invalid endtime format", nil)
	}

	limit := parseSystemLogLimit(c.QueryParam("limit"))
	var keyRegex *regexp.Regexp
	if keyreg := strings.TrimSpace(c.QueryParam("keyreg")); keyreg != "" {
		keyRegex, err = regexp.Compile(keyreg)
		if err != nil {
			return fail(c, http.StatusBadRequest, "INVALID_REGEX", "Invalid keyreg format", nil)
		}
	}

	entries, err := readSystemLogEntries(c, startTime, endTime, limit, keyRegex)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "LOG_READ_ERROR", "Failed to read system logs", err.Error())
	}

	return c.JSON(http.StatusOK, webresp.RestResult(systemLogQueryResult{
		Entries: entries,
		Total:   len(entries),
		Limit:   limit,
	}))
}

func parseSystemLogLimit(value string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || limit < 1 {
		return 100
	}
	if limit > 5000 {
		return 5000
	}
	return limit
}

func readSystemLogEntries(c echo.Context, startTime, endTime time.Time, limit int, keyRegex *regexp.Regexp) ([]systemLogEntry, error) {
	logfile := strings.TrimSpace(GetAppContext(c).Config().Logger.Filename)
	if logfile == "" {
		return []systemLogEntry{}, nil
	}

	file, err := os.Open(logfile) //nolint:gosec // G304: logfile path comes from trusted application configuration.
	if os.IsNotExist(err) {
		return []systemLogEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	filter := systemLogFilter{
		startTime: startTime,
		endTime:   endTime,
		job:       strings.TrimSpace(c.QueryParam("job")),
		namespace: strings.TrimSpace(c.QueryParam("namespace")),
		level:     strings.TrimSpace(c.QueryParam("level")),
		keyword:   strings.TrimSpace(c.QueryParam("keyword")),
		keyRegex:  keyRegex,
		defaultJob: strings.ToLower(strings.TrimSpace(
			GetAppContext(c).Config().System.Appid,
		)),
	}

	entries := make([]systemLogEntry, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if len(entries) >= limit {
			break
		}
		entry, ok := parseSystemLogLine(scanner.Text())
		if !ok || !filter.match(entry) {
			continue
		}
		entries = append(entries, redactSystemLogEntry(entry))
	}
	return entries, scanner.Err()
}

// Log lines are filtered on their raw content (operators still search by the
// identifier they already know) but only the redacted form is exported. The
// rules mirror a support bundle: secrets disappear, customer identifiers become
// a stable pseudonym, and addresses keep only their network prefix. Correlation
// survives because the pseudonym and the prefix are deterministic.
const systemLogRedactedMarker = "[REDACTED]"

var (
	secretLogKeyFragments = []string{"password", "passwd", "secret", "token", "authorization", "apikey", "credential", "community", "privatekey"}
	piiLogKeyFragments    = []string{"phone", "msisdn", "email", "address", "fullname", "customername", "contact"}

	secretAssignmentPattern = regexp.MustCompile(`(?i)\b(password|passwd|secret|token|authorization|api[_-]?key)\b(\s*[=:]\s*)("[^"]*"|'[^']*'|[^\s,;}]+)`)
	ipv4Pattern             = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3})\.\d{1,3}\b`)
	emailPattern            = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
)

type systemLogFieldClass int

const (
	logClassPlain systemLogFieldClass = iota
	logClassSecret
	logClassPII
	logClassIP
)

func classifySystemLogKey(key string) systemLogFieldClass {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key))
	if containsAnyFragment(normalized, secretLogKeyFragments) {
		return logClassSecret
	}
	if normalized == "ip" || strings.HasSuffix(normalized, "ip") || strings.HasSuffix(normalized, "ipaddress") || strings.HasPrefix(normalized, "ipaddr") {
		return logClassIP
	}
	if normalized == "nik" || containsAnyFragment(normalized, piiLogKeyFragments) {
		return logClassPII
	}
	return logClassPlain
}

func containsAnyFragment(value string, fragments []string) bool {
	for _, fragment := range fragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func redactSystemLogEntry(entry systemLogEntry) systemLogEntry {
	if len(entry.Fields) > 0 {
		redacted := make(map[string]interface{}, len(entry.Fields))
		for key, value := range entry.Fields {
			redacted[key] = redactSystemLogValue(key, value)
		}
		entry.Fields = redacted
		if encoded, err := json.Marshal(redacted); err == nil {
			entry.Line = redactSystemLogText(string(encoded))
		}
	} else {
		entry.Line = redactSystemLogText(entry.Line)
	}
	entry.Message = redactSystemLogText(entry.Message)
	return entry
}

func redactSystemLogValue(key string, value interface{}) interface{} {
	switch classifySystemLogKey(key) {
	case logClassSecret:
		return systemLogRedactedMarker
	case logClassPII:
		return pseudonymizeSystemLogValue(toSystemLogString(value))
	case logClassIP:
		return maskSystemLogAddress(toSystemLogString(value))
	default:
		if text, ok := value.(string); ok {
			return redactSystemLogText(text)
		}
		return value
	}
}

// pseudonymizeSystemLogValue keeps a stable, non-reversible identifier so two
// entries about the same customer can still be tied together.
func pseudonymizeSystemLogValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// maskSystemLogAddress keeps the IPv4 /24 or the IPv6 /48 prefix. The host part
// is what identifies a subscriber, while the prefix is enough to correlate.
func maskSystemLogAddress(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return value
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.x", v4[0], v4[1], v4[2])
	}
	groups := strings.Split(ip.String(), ":")
	if len(groups) < 3 {
		return value
	}
	return strings.Join(groups[:3], ":") + "::x"
}

func redactSystemLogText(text string) string {
	text = secretAssignmentPattern.ReplaceAllString(text, "$1$2"+systemLogRedactedMarker)
	text = emailPattern.ReplaceAllStringFunc(text, pseudonymizeSystemLogValue)
	return ipv4Pattern.ReplaceAllString(text, "$1.x")
}

type systemLogFilter struct {
	startTime  time.Time
	endTime    time.Time
	job        string
	namespace  string
	level      string
	keyword    string
	keyRegex   *regexp.Regexp
	defaultJob string
}

func (f systemLogFilter) match(entry systemLogEntry) bool {
	entryTime, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err == nil && (entryTime.Before(f.startTime) || entryTime.After(f.endTime)) {
		return false
	}
	if !matchOptionalString(f.level, entry.Level) {
		return false
	}
	if !matchOptionalString(f.namespace, entry.Namespace) {
		return false
	}
	if f.job != "" {
		entryJob := entry.Job
		if entryJob == "" {
			entryJob = f.defaultJob
		}
		if !strings.EqualFold(f.job, entryJob) {
			return false
		}
	}
	if f.keyword != "" && !strings.Contains(strings.ToLower(entry.Line), strings.ToLower(f.keyword)) {
		return false
	}
	if f.keyRegex != nil && !f.keyRegex.MatchString(entry.Line) {
		return false
	}
	return true
}

func matchOptionalString(filter, value string) bool {
	return filter == "" || strings.EqualFold(filter, value)
}

func parseSystemLogLine(line string) (systemLogEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return systemLogEntry{}, false
	}

	fields := make(map[string]interface{})
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return systemLogEntry{Line: line}, true
	}

	entry := systemLogEntry{
		Timestamp: extractSystemLogTimestamp(fields),
		Level:     extractSystemLogString(fields, "level", "Level", "L"),
		Namespace: extractSystemLogString(fields, "namespace", "Namespace"),
		Job:       extractSystemLogString(fields, "job", "Job"),
		Message:   extractSystemLogString(fields, "msg", "message", "Message", "M"),
		Line:      line,
		Fields:    fields,
	}
	return entry, true
}

func extractSystemLogString(fields map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			return strings.TrimSpace(toSystemLogString(value))
		}
	}
	return ""
}

func extractSystemLogTimestamp(fields map[string]interface{}) string {
	if ts, ok := fields["ts"].(float64); ok {
		return time.Unix(0, int64(ts*float64(time.Second))).Format(time.RFC3339)
	}
	if ts := extractSystemLogString(fields, "timestamp", "time", "Time", "T"); ts != "" {
		if parsed, err := parseTimeInput(ts, time.Time{}); err == nil {
			return parsed.Format(time.RFC3339)
		}
		return ts
	}
	return ""
}

func toSystemLogString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(v), "\n", " "), "\r", " "))
	}
}

func registerSystemLogRoutes() {
	webserver.GET("/admin/loki/query", querySystemLogs, requirePlatformAdmin())
}
