package adminapi

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/syslogd"
	"github.com/bjo163/mwx-isp/internal/webserver"
)

// TrafficPointDTO represents a single time-series rate measurement point.
type TrafficPointDTO struct {
	Timestamp time.Time `json:"timestamp"`
	InBps     int64     `json:"in_bps"`  // Inbound / Download rate (bps)
	OutBps    int64     `json:"out_bps"` // Outbound / Upload rate (bps)
}

// TrafficSeriesDTO returns aggregated MRTG telemetry metrics and time-series
// points. Counter resets and gaps are counted instead of plotted as zero, so a
// missing sample is never presented as idle traffic.
type TrafficSeriesDTO struct {
	EntityID            string            `json:"entity_id"`
	Title               string            `json:"title"`
	InterfaceName       string            `json:"interface_name,omitempty"`
	AvailableInterfaces []string          `json:"available_interfaces,omitempty"`
	CurrentInBps        int64             `json:"current_in_bps"`
	CurrentOutBps       int64             `json:"current_out_bps"`
	PeakInBps           int64             `json:"peak_in_bps"`
	PeakOutBps          int64             `json:"peak_out_bps"`
	AvgInBps            int64             `json:"avg_in_bps"`
	AvgOutBps           int64             `json:"avg_out_bps"`
	TotalInBytes        int64             `json:"total_in_bytes"`
	TotalOutBytes       int64             `json:"total_out_bytes"`
	Points              []TrafficPointDTO `json:"points"`
	Scope               string            `json:"scope"`
	InLabel             string            `json:"in_label"`
	OutLabel            string            `json:"out_label"`
	LastUpdated         *time.Time        `json:"last_updated,omitempty"`
	Stale               bool              `json:"stale"`
	StaleAfterSeconds   int               `json:"stale_after_seconds,omitempty"`
	SampleCount         int               `json:"sample_count"`
	DataAvailable       bool              `json:"data_available"`
	CounterResets       int               `json:"counter_resets,omitempty"`
	MissingIntervals    int               `json:"missing_intervals,omitempty"`
}

func registerTelemetryRoutes() {
	webserver.ApiGET("/network/monitor-targets/:id/traffic", getMonitorTargetTraffic)
	webserver.ApiGET("/network/syslog", listSyslogEvents)
	webserver.ApiPOST("/network/syslog/test", injectTestSyslog, requireAdmin())
	webserver.ApiGET("/radius/traffic/user/:username", getUserTraffic)
	webserver.ApiGET("/isp/subscriptions/:id/traffic", getSubscriptionTraffic)
}

func parseTelemetryTimeRange(r string) (time.Time, time.Duration) {
	now := time.Now()
	switch strings.ToLower(r) {
	case "1h":
		return now.Add(-1 * time.Hour), 1 * time.Minute
	case "6h":
		return now.Add(-6 * time.Hour), 5 * time.Minute
	case "7d":
		return now.Add(-7 * 24 * time.Hour), 1 * time.Hour
	case "30d":
		return now.Add(-30 * 24 * time.Hour), 4 * time.Hour
	default: // 24h
		return now.Add(-24 * time.Hour), 15 * time.Minute
	}
}

type snmpInterfaceMetric struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Operational string `json:"operational"`
	InOctets    string `json:"in_octets"`
	OutOctets   string `json:"out_octets"`
}

func getMonitorTargetTraffic(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, 400, "INVALID_ID", "Invalid target ID", nil)
	}

	var target domain.NetMonitorTarget
	if err := GetDB(c).First(&target, id).Error; err != nil {
		return fail(c, 404, "NOT_FOUND", "Monitor target not found", nil)
	}

	timeRange := c.QueryParam("range")
	reqInterface := strings.TrimSpace(c.QueryParam("interface"))
	cutoff, _ := parseTelemetryTimeRange(timeRange)

	var samples []domain.NetMonitorSample
	if err := GetDB(c).Where("target_id = ? AND checked_at >= ?", id, cutoff).
		Order("checked_at ASC").
		Find(&samples).Error; err != nil {
		return fail(c, 500, "DATABASE_ERROR", "Failed to load telemetry samples", nil)
	}

	interfaceSet := make(map[string]bool)
	type ifSample struct {
		checkedAt time.Time
		inOctets  uint64
		outOctets uint64
	}
	seriesByInterface := make(map[string][]ifSample)

	for _, sample := range samples {
		if sample.InterfaceMetricsJSON == "" {
			continue
		}
		var metrics []snmpInterfaceMetric
		if err := json.Unmarshal([]byte(sample.InterfaceMetricsJSON), &metrics); err != nil {
			continue
		}
		for _, m := range metrics {
			if m.Name == "" {
				continue
			}
			interfaceSet[m.Name] = true
			inVal, _ := strconv.ParseUint(m.InOctets, 10, 64)
			outVal, _ := strconv.ParseUint(m.OutOctets, 10, 64)
			seriesByInterface[m.Name] = append(seriesByInterface[m.Name], ifSample{
				checkedAt: sample.CheckedAt,
				inOctets:  inVal,
				outOctets: outVal,
			})
		}
	}

	availableInterfaces := make([]string, 0, len(interfaceSet))
	for name := range interfaceSet {
		availableInterfaces = append(availableInterfaces, name)
	}
	sort.Strings(availableInterfaces)

	if reqInterface == "" {
		if len(availableInterfaces) > 0 {
			reqInterface = availableInterfaces[0]
		} else {
			reqInterface = "ether1"
		}
	}

	points := make([]TrafficPointDTO, 0)
	var peakIn, peakOut, sumIn, sumOut, totalInBytes, totalOutBytes int64
	var currentIn, currentOut int64
	var counterResets, missingIntervals int

	rawSeries := seriesByInterface[reqInterface]
	if len(rawSeries) >= 2 {
		for i := 1; i < len(rawSeries); i++ {
			prev := rawSeries[i-1]
			curr := rawSeries[i]
			deltaSec := curr.checkedAt.Sub(prev.checkedAt).Seconds()
			if deltaSec <= 0 {
				missingIntervals++
				continue
			}
			// A long gap means polling was interrupted, not that the link was
			// idle; averaging across it would invent a low-traffic interval.
			if deltaSec > 3600 {
				missingIntervals++
				continue
			}
			// A counter that moves backwards is a reboot or 32/64-bit wrap. Skip
			// the interval instead of plotting the unsigned difference or zero.
			if curr.inOctets < prev.inOctets || curr.outOctets < prev.outOctets {
				counterResets++
				continue
			}

			inDiff := curr.inOctets - prev.inOctets
			outDiff := curr.outOctets - prev.outOctets
			inBps := int64(float64(inDiff) * 8 / deltaSec)
			outBps := int64(float64(outDiff) * 8 / deltaSec)
			if inBps < 0 || outBps < 0 {
				counterResets++
				continue
			}

			points = append(points, TrafficPointDTO{
				Timestamp: curr.checkedAt,
				InBps:     inBps,
				OutBps:    outBps,
			})

			// A delta that does not fit int64 converts to a negative rate above
			// and never reaches this point, so the conversion is bounded.
			totalInBytes += int64(inDiff)   //nolint:gosec // G115: accepted interval's delta fit a non-negative int64 rate
			totalOutBytes += int64(outDiff) //nolint:gosec // G115: accepted interval's delta fit a non-negative int64 rate
			sumIn += inBps
			sumOut += outBps
			if inBps > peakIn {
				peakIn = inBps
			}
			if outBps > peakOut {
				peakOut = outBps
			}
			currentIn = inBps
			currentOut = outBps
		}
	}

	var avgIn, avgOut int64
	if len(points) > 0 {
		avgIn = sumIn / int64(len(points))
		avgOut = sumOut / int64(len(points))
	}

	if len(availableInterfaces) == 0 {
		availableInterfaces = []string{reqInterface}
	}

	dto := TrafficSeriesDTO{
		EntityID:            strconv.FormatInt(id, 10),
		Title:               target.Name + " (" + reqInterface + ")",
		InterfaceName:       reqInterface,
		AvailableInterfaces: availableInterfaces,
		CurrentInBps:        currentIn,
		CurrentOutBps:       currentOut,
		PeakInBps:           peakIn,
		PeakOutBps:          peakOut,
		AvgInBps:            avgIn,
		AvgOutBps:           avgOut,
		TotalInBytes:        totalInBytes,
		TotalOutBytes:       totalOutBytes,
		Points:              points,
		Scope:               "nas_interface",
		InLabel:             "Inbound (SNMP ifInOctets)",
		OutLabel:            "Outbound (SNMP ifOutOctets)",
		SampleCount:         len(rawSeries),
		DataAvailable:       len(points) > 0,
		CounterResets:       counterResets,
		MissingIntervals:    missingIntervals,
	}
	if len(rawSeries) > 0 {
		lastUpdated := rawSeries[len(rawSeries)-1].checkedAt
		dto.LastUpdated = &lastUpdated
		if target.IntervalSeconds > 0 {
			dto.StaleAfterSeconds = target.IntervalSeconds * 2
			dto.Stale = time.Since(lastUpdated) > time.Duration(dto.StaleAfterSeconds)*time.Second
		}
	}

	return ok(c, dto)
}

func getUserTraffic(c echo.Context) error {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		return fail(c, 400, "INVALID_USERNAME", "Username is required", nil)
	}

	timeRange := c.QueryParam("range")
	cutoff, _ := parseTelemetryTimeRange(timeRange)

	var samples []domain.RadiusTrafficSample
	_ = GetDB(c).Where("username = ? AND recorded_at >= ?", username, cutoff).
		Order("recorded_at ASC").
		Find(&samples)

	var online domain.RadiusOnline
	isOnline := GetDB(c).Where("username = ?", username).First(&online).Error == nil

	points := make([]TrafficPointDTO, 0, len(samples))
	var peakIn, peakOut, sumIn, sumOut, totalInBytes, totalOutBytes int64
	var currentIn, currentOut int64

	for _, s := range samples {
		points = append(points, TrafficPointDTO{
			Timestamp: s.RecordedAt,
			InBps:     s.InRateBps,
			OutBps:    s.OutRateBps,
		})
		if s.InRateBps > peakIn {
			peakIn = s.InRateBps
		}
		if s.OutRateBps > peakOut {
			peakOut = s.OutRateBps
		}
		sumIn += s.InRateBps
		sumOut += s.OutRateBps
		currentIn = s.InRateBps
		currentOut = s.OutRateBps
	}

	if isOnline {
		if online.AcctInputTotal > totalInBytes {
			totalInBytes = online.AcctInputTotal
		}
		if online.AcctOutputTotal > totalOutBytes {
			totalOutBytes = online.AcctOutputTotal
		}
	}

	var avgIn, avgOut int64
	if len(points) > 0 {
		avgIn = sumIn / int64(len(points))
		avgOut = sumOut / int64(len(points))
	}

	// RadiusTrafficSample stores the rate between two accounting updates. A
	// missing sample is reported as missing rather than smoothed into a fake
	// baseline, so an offline subscriber never looks like idle traffic.
	dto := TrafficSeriesDTO{
		EntityID:      username,
		Title:         "PPPoE Subscriber: " + username,
		CurrentInBps:  currentIn,
		CurrentOutBps: currentOut,
		PeakInBps:     peakIn,
		PeakOutBps:    peakOut,
		AvgInBps:      avgIn,
		AvgOutBps:     avgOut,
		TotalInBytes:  totalInBytes,
		TotalOutBytes: totalOutBytes,
		Points:        points,
		Scope:         "subscriber_session",
		InLabel:       "Upload (Acct-Input-Octets)",
		OutLabel:      "Download (Acct-Output-Octets)",
		SampleCount:   len(samples),
		DataAvailable: len(points) > 0,
	}
	if isOnline {
		dto.InLabel = "Upload (Acct-Input-Octets, session online)"
	}
	if len(samples) > 0 {
		lastUpdated := samples[len(samples)-1].RecordedAt
		dto.LastUpdated = &lastUpdated
		dto.StaleAfterSeconds = 300
		dto.Stale = time.Since(lastUpdated) > 5*time.Minute
	}

	return ok(c, dto)
}

func getSubscriptionTraffic(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, 400, "INVALID_ID", "Invalid subscription ID", nil)
	}

	var sub domain.Subscription
	if err := GetDB(c).First(&sub, id).Error; err != nil {
		return fail(c, 404, "NOT_FOUND", "Subscription not found", nil)
	}

	username := sub.SubscriptionNo
	if sub.RadiusUserID > 0 {
		var user domain.RadiusUser
		if err := GetDB(c).First(&user, sub.RadiusUserID).Error; err == nil && user.Username != "" {
			username = user.Username
		}
	}

	c.SetParamNames("username")
	c.SetParamValues(username)
	return getUserTraffic(c)
}

func listSyslogEvents(c echo.Context) error {
	page, size := parsePagination(c)
	q := GetDB(c).Model(&domain.SyslogEvent{})

	// Test/injected events are hidden by default so the operational log reflects
	// what real devices sent; ?include_synthetic=true reveals them.
	if c.QueryParam("include_synthetic") != "true" {
		q = q.Where("synthetic = ?", false)
	}
	if nasIP := strings.TrimSpace(c.QueryParam("nas_ip")); nasIP != "" {
		q = q.Where("nas_ip = ?", nasIP)
	}
	if sev := c.QueryParam("severity"); sev != "" {
		if sevVal, err := strconv.Atoi(sev); err == nil {
			q = q.Where("severity = ?", sevVal)
		}
	}
	if tag := strings.TrimSpace(c.QueryParam("tag")); tag != "" {
		q = q.Where("tag LIKE ?", "%"+tag+"%")
	}
	if search := strings.TrimSpace(c.QueryParam("q")); search != "" {
		q = q.Where("message LIKE ? OR tag LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return fail(c, 500, "DATABASE_ERROR", "Failed to count syslog events", nil)
	}

	var rows []domain.SyslogEvent
	if err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return fail(c, 500, "DATABASE_ERROR", "Failed to query syslog events", nil)
	}

	return paged(c, rows, total, page, size)
}

func injectTestSyslog(c echo.Context) error {
	var input struct {
		NasIP    string `json:"nas_ip"`
		Severity int    `json:"severity"`
		Tag      string `json:"tag"`
		Message  string `json:"message"`
	}
	if err := c.Bind(&input); err != nil {
		return fail(c, 400, "INVALID_INPUT", "Invalid syslog payload", nil)
	}
	if input.NasIP == "" {
		input.NasIP = "192.168.88.1"
	}
	if input.Tag == "" {
		input.Tag = "pppoe,info"
	}
	if input.Message == "" {
		input.Message = "PPPoE session connected"
	}
	event := domain.SyslogEvent{
		NasIP:        input.NasIP,
		Facility:     1,
		Severity:     input.Severity,
		SeverityName: syslogd.SeverityName(input.Severity),
		Tag:          input.Tag,
		Message:      input.Message,
		Synthetic:    true,
		CreatedAt:    time.Now(),
	}
	if err := GetDB(c).Create(&event).Error; err != nil {
		return fail(c, 500, "DATABASE_ERROR", "Failed to record test syslog", nil)
	}
	return ok(c, event)
}
