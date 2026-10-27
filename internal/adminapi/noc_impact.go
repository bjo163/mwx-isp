package adminapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// nocImpactLookback bounds the accounting history used to infer which
// subscribers depend on a NAS. Without a bound, every subscriber that ever
// connected to the device would be reported as impacted.
const nocImpactLookback = 24 * time.Hour

// nocImpactMinStaleAfter prevents short monitoring intervals from producing
// stale-state flapping between scheduled checks.
const nocImpactMinStaleAfter = 5 * time.Minute

const (
	nocImpactConfidenceConfirmed = "confirmed"
	nocImpactConfidenceInferred  = "inferred"
	nocImpactConfidenceUnknown   = "unknown"

	nocImpactStateOperational   = "operational"
	nocImpactStateOutage        = "confirmed_outage"
	nocImpactStateLastDownStale = "last_known_down_stale"
	nocImpactStateStale         = "stale"
	nocImpactStateNotMonitored  = "not_monitored"
	nocImpactStateUnknown       = "unknown"
)

type nocImpactIncident struct {
	ID        int64     `json:"id,string"`
	Summary   string    `json:"summary"`
	StartedAt time.Time `json:"started_at"`
}

type nocImpactMonitoring struct {
	Monitored         bool               `json:"monitored"`
	Enabled           bool               `json:"enabled"`
	Status            string             `json:"status"`
	Stale             bool               `json:"stale"`
	StaleAfterSeconds int                `json:"stale_after_seconds"`
	LastCheckedAt     *time.Time         `json:"last_checked_at"`
	IntervalSeconds   int                `json:"interval_seconds"`
	TargetID          string             `json:"target_id,omitempty"`
	TargetName        string             `json:"target_name,omitempty"`
	OpenIncident      *nocImpactIncident `json:"open_incident,omitempty"`
}

type nocImpactScope struct {
	Type     string `json:"type"`
	ID       int64  `json:"id,string"`
	Name     string `json:"name"`
	Code     string `json:"code,omitempty"`
	Zone     string `json:"zone,omitempty"`
	Address  string `json:"address,omitempty"`
	Kind     string `json:"kind,omitempty"`
	TargetID string `json:"target_id,omitempty"`
	NasID    string `json:"nas_id,omitempty"`
	NasName  string `json:"nas_name,omitempty"`
}

type nocImpactCounts struct {
	Customers      int64 `json:"customers"`
	Subscriptions  int64 `json:"subscriptions"`
	OnlineSessions int64 `json:"online_sessions"`
	OpenTickets    int64 `json:"open_tickets"`
}

type nocImpactSubscriber struct {
	CustomerID         string     `json:"customer_id"`
	CustomerNo         string     `json:"customer_no"`
	CustomerName       string     `json:"customer_name"`
	ODPCode            string     `json:"odp_code,omitempty"`
	ODPPort            int        `json:"odp_port,omitempty"`
	SubscriptionID     string     `json:"subscription_id"`
	SubscriptionNo     string     `json:"subscription_no"`
	SubscriptionStatus string     `json:"subscription_status"`
	Username           string     `json:"username,omitempty"`
	Online             bool       `json:"online"`
	LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
}

type nocImpactPagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type nocImpactResponse struct {
	Scope            nocImpactScope        `json:"scope"`
	State            string                `json:"state"`
	Confidence       string                `json:"confidence"`
	ConfidenceReason string                `json:"confidence_reason"`
	Monitoring       nocImpactMonitoring   `json:"monitoring"`
	Counts           nocImpactCounts       `json:"counts"`
	Impacted         []nocImpactSubscriber `json:"impacted"`
	Pagination       nocImpactPagination   `json:"pagination"`
	Warnings         []string              `json:"warnings"`
	GeneratedAt      time.Time             `json:"generated_at"`
}

type nocImpactSelection struct {
	where string
	args  []any
}

type nocImpactSubscriptionRow struct {
	SubscriptionID     int64
	SubscriptionNo     string
	SubscriptionStatus string
	CustomerID         int64
	CustomerNo         string
	CustomerName       string
	ODPCode            string
	ODPPort            int
	Username           string
}

type nocImpactSeen struct {
	Sessions int64
	LastSeen time.Time
}

func parseNOCImpactID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func parseNOCImpactTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// getNetworkImpact returns a read-only, tenant-scoped estimate of which
// subscribers depend on one monitoring target, ODP enclosure, or NAS. It
// reports monitoring freshness separately from outage state so a stale "down"
// sample is never presented as a live outage.
func getNetworkImpact(c echo.Context) error {
	targetRaw := strings.TrimSpace(c.QueryParam("target_id"))
	odpRaw := strings.TrimSpace(c.QueryParam("odp_id"))
	nasRaw := strings.TrimSpace(c.QueryParam("nas_id"))

	selected := 0
	for _, raw := range []string{targetRaw, odpRaw, nasRaw} {
		if raw != "" {
			selected++
		}
	}
	if selected != 1 {
		return fail(c, http.StatusBadRequest, "INVALID_SELECTOR", "Provide exactly one of target_id, odp_id, or nas_id", nil)
	}

	page, size := parsePagination(c)
	db := GetDB(c)
	tenantID := domain.DefaultTenantID
	if scopedID, okTenant := tenancy.TenantID(c.Request().Context()); okTenant {
		tenantID = scopedID
	}

	switch {
	case targetRaw != "":
		id, err := parseNOCImpactID(targetRaw)
		if err != nil {
			return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid target ID", nil)
		}
		return impactForMonitorTarget(c, db, tenantID, id, page, size)
	case odpRaw != "":
		id, err := parseNOCImpactID(odpRaw)
		if err != nil {
			return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid ODP ID", nil)
		}
		return impactForODP(c, db, tenantID, id, page, size)
	default:
		id, err := parseNOCImpactID(nasRaw)
		if err != nil {
			return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid NAS ID", nil)
		}
		return impactForNAS(c, db, tenantID, id, page, size)
	}
}

func impactForMonitorTarget(c echo.Context, db *gorm.DB, tenantID, targetID int64, page, size int) error {
	var target domain.NetMonitorTarget
	if err := db.First(&target, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "NOT_FOUND", "Monitor target not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load monitor target", err)
	}

	now := time.Now().UTC()
	resp := newNOCImpactResponse()
	resp.GeneratedAt = now
	resp.Scope = nocImpactScope{
		Type:     "target",
		ID:       target.ID,
		Name:     target.Name,
		Address:  target.Address,
		Kind:     target.Kind,
		TargetID: strconv.FormatInt(target.ID, 10),
	}
	resp.Monitoring, resp.State, resp.Warnings = computeNOCMonitoring(db, &target, now)

	var nas domain.NetNas
	if err := db.Where("ipaddr = ?", target.Address).Order("id ASC").First(&nas).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to resolve NAS mapping", err)
		}
		resp.Confidence = nocImpactConfidenceUnknown
		resp.ConfidenceReason = "The target address does not match any NAS record, so no subscriber mapping can be derived."
		resp.Warnings = append(resp.Warnings, "Subscriber impact is unknown for this target: no NAS or ODP association exists.")
		resp.Pagination = nocImpactPagination{Page: page, PageSize: size}
		return ok(c, resp)
	}

	resp.Scope.NasID = strconv.FormatInt(nas.ID, 10)
	resp.Scope.NasName = nas.Name
	applyNOCImpactConfidence(&resp, "inferred", "Subscribers are inferred from RADIUS sessions and accounting observed on this NAS address.")

	sel := nocImpactSelection{where: nasUsernameSelection, args: []any{tenantID, nas.Ipaddr, tenantID, nas.Ipaddr, now.Add(-nocImpactLookback)}}
	if err := fillNOCImpact(db, tenantID, sel, page, size, now, &resp); err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to aggregate subscriber impact", err)
	}
	if resp.Pagination.Total == 0 {
		resp.Confidence = nocImpactConfidenceUnknown
		resp.ConfidenceReason = "No subscriber activity was observed on this NAS within the lookback window, so impact cannot be confirmed."
		resp.Warnings = append(resp.Warnings, fmt.Sprintf("No RADIUS sessions or accounting records were found for this NAS in the last %d hours.", int(nocImpactLookback.Hours())))
	}
	return ok(c, resp)
}

func impactForODP(c echo.Context, db *gorm.DB, tenantID, odpID int64, page, size int) error {
	var odp domain.ODP
	if err := db.First(&odp, odpID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "NOT_FOUND", "ODP enclosure not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load ODP", err)
	}

	now := time.Now().UTC()
	resp := newNOCImpactResponse()
	resp.GeneratedAt = now
	resp.Scope = nocImpactScope{
		Type: "odp",
		ID:   odp.ID,
		Name: odp.Name,
		Code: odp.Code,
		Zone: odp.Zone,
	}
	resp.State = nocImpactStateNotMonitored
	resp.Monitoring = nocImpactMonitoring{Status: "unknown", StaleAfterSeconds: int(nocImpactMinStaleAfter / time.Second)}
	resp.Warnings = append(resp.Warnings, "ODP enclosures have no direct probe; this view maps exposure but does not confirm an outage. Online status comes from live RADIUS sessions.")

	sel := nocImpactSelection{where: "c.odp_id = ?", args: []any{odp.ID}}
	if err := fillNOCImpact(db, tenantID, sel, page, size, now, &resp); err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to aggregate subscriber impact", err)
	}

	switch {
	case resp.Pagination.Total > 0:
		applyNOCImpactConfidence(&resp, nocImpactConfidenceConfirmed, "Customers are mapped by an explicit ODP assignment.")
	case odp.Code != "":
		fallback := nocImpactSelection{where: "c.odp_id = 0 AND c.odp_code = ?", args: []any{odp.Code}}
		var fallbackTotal int64
		if err := nocImpactSubscriptionQuery(db, tenantID, fallback).Count(&fallbackTotal).Error; err != nil {
			return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to aggregate subscriber impact", err)
		}
		if fallbackTotal == 0 {
			resp.Confidence = nocImpactConfidenceUnknown
			resp.ConfidenceReason = "No customer, subscription, or session references this ODP."
			resp.Warnings = append(resp.Warnings, "No subscribers are assigned to this ODP by ID or cached code; the impact list is empty, not confirmed clear.")
		} else {
			if err := fillNOCImpact(db, tenantID, fallback, page, size, now, &resp); err != nil {
				return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to aggregate subscriber impact", err)
			}
			applyNOCImpactConfidence(&resp, nocImpactConfidenceInferred, "No customer stores this ODP ID; customers are matched by the cached ODP code instead.")
			resp.Warnings = append(resp.Warnings, "Impact was inferred from the cached ODP code because customers do not store the ODP ID.")
		}
	default:
		resp.Confidence = nocImpactConfidenceUnknown
		resp.ConfidenceReason = "No customer, subscription, or session references this ODP."
		resp.Warnings = append(resp.Warnings, "No subscribers are assigned to this ODP; the impact list is empty, not confirmed clear.")
	}
	return ok(c, resp)
}

func impactForNAS(c echo.Context, db *gorm.DB, tenantID, nasID int64, page, size int) error {
	var nas domain.NetNas
	if err := db.First(&nas, nasID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "NOT_FOUND", "NAS not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load NAS", err)
	}

	now := time.Now().UTC()
	resp := newNOCImpactResponse()
	resp.GeneratedAt = now
	resp.Scope = nocImpactScope{
		Type:    "nas",
		ID:      nas.ID,
		Name:    nas.Name,
		Address: nas.Ipaddr,
	}

	target, err := pickNOCMonitoringTarget(db, nas.Ipaddr)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load monitoring state", err)
	}
	if target != nil {
		resp.Scope.TargetID = strconv.FormatInt(target.ID, 10)
		resp.Monitoring, resp.State, resp.Warnings = computeNOCMonitoring(db, target, now)
	} else {
		resp.State = nocImpactStateUnknown
		resp.Monitoring = nocImpactMonitoring{Status: "unknown", StaleAfterSeconds: int(nocImpactMinStaleAfter / time.Second)}
		resp.Warnings = append(resp.Warnings, "No monitor target is configured for this NAS address, so freshness and outage state are unknown.")
	}

	sel := nocImpactSelection{where: nasUsernameSelection, args: []any{tenantID, nas.Ipaddr, tenantID, nas.Ipaddr, now.Add(-nocImpactLookback)}}
	if err := fillNOCImpact(db, tenantID, sel, page, size, now, &resp); err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to aggregate subscriber impact", err)
	}
	if resp.Pagination.Total == 0 {
		resp.Confidence = nocImpactConfidenceUnknown
		resp.ConfidenceReason = "No subscriber activity was observed on this NAS within the lookback window, so impact cannot be confirmed."
		resp.Warnings = append(resp.Warnings, fmt.Sprintf("No RADIUS sessions or accounting records were found for this NAS in the last %d hours.", int(nocImpactLookback.Hours())))
	} else {
		applyNOCImpactConfidence(&resp, nocImpactConfidenceInferred, "Subscribers are inferred from RADIUS sessions and accounting observed on this NAS address.")
	}
	return ok(c, resp)
}

// nasUsernameSelection maps a NAS address to subscriber usernames through
// current sessions and recent accounting. The UNION collapses duplicate
// usernames so joined subscription rows are never multiplied.
const nasUsernameSelection = "u.username <> '' AND u.username IN (" +
	"SELECT username FROM radius_online WHERE tenant_id = ? AND nas_addr = ? " +
	"UNION SELECT username FROM radius_accounting WHERE tenant_id = ? AND nas_addr = ? AND last_update >= ?)"

func newNOCImpactResponse() nocImpactResponse {
	return nocImpactResponse{
		Counts:     nocImpactCounts{},
		Impacted:   make([]nocImpactSubscriber, 0),
		Warnings:   make([]string, 0),
		Pagination: nocImpactPagination{},
	}
}

func applyNOCImpactConfidence(resp *nocImpactResponse, level, reason string) {
	resp.Confidence = level
	resp.ConfidenceReason = reason
}

func nocImpactSubscriptionQuery(db *gorm.DB, tenantID int64, sel nocImpactSelection) *gorm.DB {
	return db.Table("isp_subscription AS s").
		Joins("JOIN isp_customer AS c ON c.id = s.customer_id AND c.tenant_id = s.tenant_id").
		Joins("LEFT JOIN radius_user AS u ON u.id = s.radius_user_id AND u.tenant_id = s.tenant_id").
		Where("s.tenant_id = ?", tenantID).
		Where(sel.where, sel.args...)
}

func fillNOCImpact(db *gorm.DB, tenantID int64, sel nocImpactSelection, page, size int, now time.Time, resp *nocImpactResponse) error {
	resp.Pagination = nocImpactPagination{Page: page, PageSize: size}

	var total int64
	if err := nocImpactSubscriptionQuery(db, tenantID, sel).Count(&total).Error; err != nil {
		return err
	}
	resp.Pagination.Total = total
	resp.Counts.Subscriptions = total

	var customerCount int64
	if err := nocImpactSubscriptionQuery(db, tenantID, sel).Distinct("c.id").Count(&customerCount).Error; err != nil {
		return err
	}
	resp.Counts.Customers = customerCount

	var rows []nocImpactSubscriptionRow
	err := nocImpactSubscriptionQuery(db, tenantID, sel).
		Select("s.id AS subscription_id, s.subscription_no, s.status AS subscription_status, " +
			"c.id AS customer_id, c.customer_no, c.name AS customer_name, c.odp_code, c.odp_port, " +
			"COALESCE(u.username, '') AS username").
		Order("c.name ASC, s.id ASC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&rows).Error
	if err != nil {
		return err
	}

	usernames := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Username != "" {
			usernames = append(usernames, row.Username)
		}
	}

	seen := map[string]nocImpactSeen{}
	if len(usernames) > 0 {
		type seenRow struct {
			Username string
			Sessions int64
			// Aggregates are scanned as text because SQLite returns MAX(time)
			// as a string while PostgreSQL returns a timestamp.
			LastUpdate string
		}
		var accountingRows []seenRow
		if err := db.Table("radius_accounting").
			Select("username, MAX(last_update) AS last_update").
			Where("tenant_id = ? AND username IN ?", tenantID, usernames).
			Group("username").
			Scan(&accountingRows).Error; err != nil {
			return err
		}
		for _, row := range accountingRows {
			seen[row.Username] = nocImpactSeen{LastSeen: parseNOCImpactTime(row.LastUpdate)}
		}
		var onlineRows []seenRow
		if err := db.Table("radius_online").
			Select("username, COUNT(*) AS sessions, MAX(last_update) AS last_update").
			Where("tenant_id = ? AND username IN ?", tenantID, usernames).
			Group("username").
			Scan(&onlineRows).Error; err != nil {
			return err
		}
		for _, row := range onlineRows {
			current := seen[row.Username]
			current.Sessions = row.Sessions
			if lastUpdate := parseNOCImpactTime(row.LastUpdate); lastUpdate.After(current.LastSeen) {
				current.LastSeen = lastUpdate
			}
			seen[row.Username] = current
		}
	}

	for _, row := range rows {
		item := nocImpactSubscriber{
			CustomerID:         strconv.FormatInt(row.CustomerID, 10),
			CustomerNo:         row.CustomerNo,
			CustomerName:       row.CustomerName,
			ODPCode:            row.ODPCode,
			ODPPort:            row.ODPPort,
			SubscriptionID:     strconv.FormatInt(row.SubscriptionID, 10),
			SubscriptionNo:     row.SubscriptionNo,
			SubscriptionStatus: row.SubscriptionStatus,
			Username:           row.Username,
		}
		if observed, okSeen := seen[row.Username]; okSeen && row.Username != "" {
			item.Online = observed.Sessions > 0
			if !observed.LastSeen.IsZero() {
				lastSeen := observed.LastSeen
				item.LastSeenAt = &lastSeen
			}
		}
		resp.Impacted = append(resp.Impacted, item)
	}

	if err := fillNOCImpactCounts(db, tenantID, sel, resp); err != nil {
		return err
	}
	return nil
}

func fillNOCImpactCounts(db *gorm.DB, tenantID int64, sel nocImpactSelection, resp *nocImpactResponse) error {
	onlineSubquery := nocImpactSubscriptionQuery(db, tenantID, sel).
		Select("u.username").
		Where("u.username <> ''")
	var onlineSessions int64
	if err := db.Table("radius_online AS o").
		Where("o.tenant_id = ?", tenantID).
		Where("o.username IN (?)", onlineSubquery).
		Count(&onlineSessions).Error; err != nil {
		return err
	}
	resp.Counts.OnlineSessions = onlineSessions

	customerSubquery := nocImpactSubscriptionQuery(db, tenantID, sel).Select("c.id")
	var openTickets int64
	if err := db.Table("isp_trouble_ticket AS t").
		Where("t.tenant_id = ? AND t.status NOT IN ?", tenantID, []string{"resolved", "closed"}).
		Where("t.customer_id IN (?)", customerSubquery).
		Count(&openTickets).Error; err != nil {
		return err
	}
	resp.Counts.OpenTickets = openTickets
	return nil
}

func pickNOCMonitoringTarget(db *gorm.DB, address string) (*domain.NetMonitorTarget, error) {
	var targets []domain.NetMonitorTarget
	if err := db.Where("address = ?", address).
		Order("enabled DESC, last_checked_at DESC, id ASC").
		Limit(10).
		Find(&targets).Error; err != nil {
		return nil, err
	}
	for i := range targets {
		if targets[i].Enabled {
			return &targets[i], nil
		}
	}
	if len(targets) > 0 {
		return &targets[0], nil
	}
	return nil, nil
}

func computeNOCMonitoring(db *gorm.DB, target *domain.NetMonitorTarget, now time.Time) (nocImpactMonitoring, string, []string) {
	staleAfter := time.Duration(target.IntervalSeconds) * 3 * time.Second
	if staleAfter < nocImpactMinStaleAfter {
		staleAfter = nocImpactMinStaleAfter
	}
	monitoring := nocImpactMonitoring{
		Monitored:         target.Enabled,
		Enabled:           target.Enabled,
		Status:            target.LastStatus,
		IntervalSeconds:   target.IntervalSeconds,
		StaleAfterSeconds: int(staleAfter / time.Second),
		LastCheckedAt:     target.LastCheckedAt,
		TargetID:          strconv.FormatInt(target.ID, 10),
		TargetName:        target.Name,
	}
	warnings := make([]string, 0)

	var incident domain.NetMonitorIncident
	if err := db.Where("target_id = ? AND resolved_at IS NULL", target.ID).
		Order("started_at DESC").
		First(&incident).Error; err == nil {
		monitoring.OpenIncident = &nocImpactIncident{ID: incident.ID, Summary: incident.Summary, StartedAt: incident.StartedAt}
	}

	if !target.Enabled {
		return monitoring, nocImpactStateNotMonitored, append(warnings, "Monitoring is disabled for this target; the current outage state is unknown.")
	}
	if target.LastCheckedAt == nil {
		return monitoring, nocImpactStateUnknown, append(warnings, "This target has never been checked; the outage state is unknown.")
	}

	monitoring.Stale = now.Sub(*target.LastCheckedAt) > staleAfter
	checkedAt := target.LastCheckedAt.UTC().Format(time.RFC3339)

	switch {
	case monitoring.Stale && target.LastStatus == "down":
		warnings = append(warnings, fmt.Sprintf("The last probe failed at %s, older than the %d-second freshness window. Treat this as last-known state, not a confirmed live outage.", checkedAt, monitoring.StaleAfterSeconds))
		return monitoring, nocImpactStateLastDownStale, warnings
	case monitoring.Stale:
		warnings = append(warnings, fmt.Sprintf("The last probe completed at %s, older than the %d-second freshness window; the outage state cannot be confirmed.", checkedAt, monitoring.StaleAfterSeconds))
		return monitoring, nocImpactStateStale, warnings
	case target.LastStatus == "down":
		return monitoring, nocImpactStateOutage, warnings
	case target.LastStatus == "up":
		return monitoring, nocImpactStateOperational, warnings
	default:
		return monitoring, nocImpactStateUnknown, warnings
	}
}
