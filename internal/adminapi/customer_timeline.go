package adminapi

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const (
	customerTimelineDefaultWindow = 90 * 24 * time.Hour
	customerTimelineMaxPageSize   = 100
)

var customerTimelineSourceOrder = []string{"billing", "invoice", "payment", "session", "ticket"}

// customerTimelineEvent is the normalized read-only projection of one persisted
// source row. It carries no credentials or private RADIUS attributes.
type customerTimelineEvent struct {
	EventID        string    `json:"event_id"`
	Source         string    `json:"source"`
	Type           string    `json:"type"`
	OccurredAt     time.Time `json:"occurred_at"`
	TenantID       int64     `json:"tenant_id,string"`
	CustomerID     int64     `json:"customer_id,string"`
	SubscriptionID int64     `json:"subscription_id,string,omitempty"`
	Summary        string    `json:"summary"`
	SourcePath     string    `json:"source_path"`
	Severity       string    `json:"severity"`
}

type customerTimelinePagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type customerTimelineResponse struct {
	CustomerID int64                      `json:"customer_id,string"`
	From       time.Time                  `json:"from"`
	To         time.Time                  `json:"to"`
	Sources    []string                   `json:"sources"`
	Events     []customerTimelineEvent    `json:"events"`
	Pagination customerTimelinePagination `json:"pagination"`
}

// getCustomerTimeline returns a read-only, tenant-scoped chronological view of
// billing, invoice, payment, RADIUS session and support-ticket events for one
// customer. Every event maps to an existing persisted row; the endpoint never
// fabricates an event or timestamp when a source has no data.
func getCustomerTimeline(c echo.Context) error {
	customerID, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, 400, "INVALID_ID", "Invalid customer ID", nil)
	}
	db := GetDB(c)
	var customer domain.Customer
	if err := db.First(&customer, customerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, 404, "NOT_FOUND", "Customer not found", nil)
		}
		return fail(c, 500, "DATABASE_ERROR", "Failed to query customer", err.Error())
	}

	page, size := parsePagination(c)
	if size > customerTimelineMaxPageSize {
		size = customerTimelineMaxPageSize
	}
	now := time.Now()
	to, err := parseCustomerTimelineTime(c.QueryParam("to"), now, true)
	if err != nil {
		return fail(c, 400, "INVALID_TIME_RANGE", "to must be a valid date or timestamp", nil)
	}
	from, err := parseCustomerTimelineTime(c.QueryParam("from"), to.Add(-customerTimelineDefaultWindow), false)
	if err != nil {
		return fail(c, 400, "INVALID_TIME_RANGE", "from must be a valid date or timestamp", nil)
	}
	if from.After(to) {
		return fail(c, 400, "INVALID_TIME_RANGE", "from must not be after to", nil)
	}
	sources, selectedSources, err := parseCustomerTimelineSources(c.QueryParam("sources"))
	if err != nil {
		return fail(c, 400, "INVALID_SOURCE", err.Error(), nil)
	}

	tenantID := domain.DefaultTenantID
	if scopedID, ok := tenancy.TenantID(c.Request().Context()); ok {
		tenantID = scopedID
	}

	offset := (page - 1) * size
	limit := offset + size
	events := make([]customerTimelineEvent, 0, limit)
	var total int64

	subsByUsername := map[string]int64{}
	if sources["session"] {
		var subscriptions []domain.Subscription
		if err := db.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).Order("id ASC").Find(&subscriptions).Error; err != nil {
			return fail(c, 500, "DATABASE_ERROR", "Failed to query customer subscriptions", err.Error())
		}
		userToSubscription := map[int64]int64{}
		userIDs := make([]int64, 0, len(subscriptions))
		for _, sub := range subscriptions {
			if sub.RadiusUserID > 0 {
				userIDs = append(userIDs, sub.RadiusUserID)
				userToSubscription[sub.RadiusUserID] = sub.ID
			}
		}
		if len(userIDs) > 0 {
			var users []domain.RadiusUser
			if err := db.Select("id, username").Where("tenant_id = ? AND id IN ?", tenantID, userIDs).Find(&users).Error; err != nil {
				return fail(c, 500, "DATABASE_ERROR", "Failed to query customer RADIUS users", err.Error())
			}
			for _, user := range users {
				if subID, ok := userToSubscription[user.ID]; ok && user.Username != "" {
					subsByUsername[user.Username] = subID
				}
			}
		}
	}

	if sources["billing"] {
		if err := appendBillingTimelineEvents(db, tenantID, customerID, from, to, limit, &events, &total); err != nil {
			return fail(c, 500, "DATABASE_ERROR", "Failed to query billing events", err.Error())
		}
	}
	if sources["invoice"] {
		if err := appendInvoiceTimelineEvents(db, tenantID, customerID, from, to, limit, &events, &total); err != nil {
			return fail(c, 500, "DATABASE_ERROR", "Failed to query invoices", err.Error())
		}
	}
	if sources["payment"] {
		if err := appendPaymentTimelineEvents(db, tenantID, customerID, from, to, limit, &events, &total); err != nil {
			return fail(c, 500, "DATABASE_ERROR", "Failed to query payments", err.Error())
		}
	}
	if sources["session"] {
		if err := appendSessionTimelineEvents(db, tenantID, customerID, from, to, limit, subsByUsername, &events, &total); err != nil {
			return fail(c, 500, "DATABASE_ERROR", "Failed to query sessions", err.Error())
		}
	}
	if sources["ticket"] {
		if err := appendTicketTimelineEvents(db, tenantID, customerID, from, to, &events, &total); err != nil {
			return fail(c, 500, "DATABASE_ERROR", "Failed to query trouble tickets", err.Error())
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].OccurredAt.Equal(events[j].OccurredAt) {
			return events[i].OccurredAt.After(events[j].OccurredAt)
		}
		if events[i].Source != events[j].Source {
			return events[i].Source < events[j].Source
		}
		return events[i].EventID < events[j].EventID
	})

	pageEvents := make([]customerTimelineEvent, 0)
	if offset < len(events) {
		end := offset + size
		if end > len(events) {
			end = len(events)
		}
		pageEvents = events[offset:end]
	}

	return paged(c, customerTimelineResponse{
		CustomerID: customerID,
		From:       from.UTC(),
		To:         to.UTC(),
		Sources:    selectedSources,
		Events:     pageEvents,
		Pagination: customerTimelinePagination{Page: page, PageSize: size, Total: total},
	}, total, page, size)
}

func appendBillingTimelineEvents(db *gorm.DB, tenantID, customerID int64, from, to time.Time, limit int, events *[]customerTimelineEvent, total *int64) error {
	query := func() *gorm.DB {
		return db.Model(&domain.BillingEvent{}).
			Where("tenant_id = ? AND customer_id = ? AND created_at >= ? AND created_at <= ?", tenantID, customerID, from, to)
	}
	var count int64
	if err := query().Count(&count).Error; err != nil {
		return err
	}
	*total += count
	var rows []domain.BillingEvent
	if err := query().Order("created_at DESC, id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		sourcePath := fmt.Sprintf("/isp/customers/%d/show", customerID)
		switch {
		case row.InvoiceID > 0:
			sourcePath = fmt.Sprintf("/isp/invoices/%d/show", row.InvoiceID)
		case row.SubscriptionID > 0:
			sourcePath = fmt.Sprintf("/isp/subscriptions/%d/show", row.SubscriptionID)
		}
		summary := strings.TrimSpace(row.Description)
		if summary == "" {
			summary = row.Type
		}
		*events = append(*events, customerTimelineEvent{
			EventID:        fmt.Sprintf("billing:%d", row.ID),
			Source:         "billing",
			Type:           row.Type,
			OccurredAt:     row.CreatedAt.UTC(),
			TenantID:       tenantID,
			CustomerID:     customerID,
			SubscriptionID: row.SubscriptionID,
			Summary:        summary,
			SourcePath:     sourcePath,
			Severity:       billingEventSeverity(row.Type),
		})
	}
	return nil
}

func appendInvoiceTimelineEvents(db *gorm.DB, tenantID, customerID int64, from, to time.Time, limit int, events *[]customerTimelineEvent, total *int64) error {
	// Issuance is the invoice date, not the row insertion time: backdated
	// invoices imported into the system would otherwise appear as if they
	// were issued today.
	query := func() *gorm.DB {
		return db.Model(&domain.Invoice{}).
			Where("tenant_id = ? AND customer_id = ? AND invoice_date >= ? AND invoice_date <= ?", tenantID, customerID, from, to)
	}
	var count int64
	if err := query().Count(&count).Error; err != nil {
		return err
	}
	*total += count
	var rows []domain.Invoice
	if err := query().Order("invoice_date DESC, id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		severity := "info"
		switch row.Status {
		case domain.InvoicePaid:
			severity = "success"
		case domain.InvoiceOverdue:
			severity = "warning"
		}
		*events = append(*events, customerTimelineEvent{
			EventID:        fmt.Sprintf("invoice:%d:issued", row.ID),
			Source:         "invoice",
			Type:           "invoice_issued",
			OccurredAt:     row.InvoiceDate.UTC(),
			TenantID:       tenantID,
			CustomerID:     customerID,
			SubscriptionID: row.SubscriptionID,
			Summary:        fmt.Sprintf("Invoice %s issued: Rp %s (status %s)", row.InvoiceNo, formatIDR(row.Total), row.Status),
			SourcePath:     fmt.Sprintf("/isp/invoices/%d/show", row.ID),
			Severity:       severity,
		})
	}
	return nil
}

func appendPaymentTimelineEvents(db *gorm.DB, tenantID, customerID int64, from, to time.Time, limit int, events *[]customerTimelineEvent, total *int64) error {
	query := func() *gorm.DB {
		return db.Model(&domain.Payment{}).
			Where("tenant_id = ? AND customer_id = ? AND paid_at >= ? AND paid_at <= ?", tenantID, customerID, from, to)
	}
	var count int64
	if err := query().Count(&count).Error; err != nil {
		return err
	}
	*total += count
	var rows []domain.Payment
	if err := query().Order("paid_at DESC, id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		summary := fmt.Sprintf("Payment %s received: Rp %s via %s", row.PaymentNo, formatIDR(row.Amount), row.Method)
		if strings.TrimSpace(row.Reference) != "" {
			summary += fmt.Sprintf(" (ref %s)", row.Reference)
		}
		severity := "success"
		if row.Status == domain.PaymentReversed {
			summary += " (reversed)"
			severity = "warning"
		}
		*events = append(*events, customerTimelineEvent{
			EventID:    fmt.Sprintf("payment:%d:received", row.ID),
			Source:     "payment",
			Type:       "payment_received",
			OccurredAt: row.PaidAt.UTC(),
			TenantID:   tenantID,
			CustomerID: customerID,
			Summary:    summary,
			SourcePath: fmt.Sprintf("/isp/payments/%d/show", row.ID),
			Severity:   severity,
		})
	}
	return nil
}

func appendSessionTimelineEvents(db *gorm.DB, tenantID, customerID int64, from, to time.Time, limit int, subscriptionsByUsername map[string]int64, events *[]customerTimelineEvent, total *int64) error {
	if len(subscriptionsByUsername) == 0 {
		return nil
	}
	usernames := make([]string, 0, len(subscriptionsByUsername))
	for username := range subscriptionsByUsername {
		usernames = append(usernames, username)
	}
	sort.Strings(usernames)

	onlineQuery := func() *gorm.DB {
		return db.Model(&domain.RadiusOnline{}).
			Where("tenant_id = ? AND username IN ? AND acct_start_time >= ? AND acct_start_time <= ?", tenantID, usernames, from, to)
	}
	var onlineCount int64
	if err := onlineQuery().Count(&onlineCount).Error; err != nil {
		return err
	}
	*total += onlineCount
	var onlineRows []domain.RadiusOnline
	if err := onlineQuery().Order("acct_start_time DESC, id ASC").Limit(limit).Find(&onlineRows).Error; err != nil {
		return err
	}
	for _, row := range onlineRows {
		summary := fmt.Sprintf("Session connected on NAS %s", row.NasAddr)
		if row.FramedIpaddr != "" {
			summary += fmt.Sprintf(" (IP %s)", row.FramedIpaddr)
		}
		*events = append(*events, customerTimelineEvent{
			EventID:        fmt.Sprintf("session:online:%d", row.ID),
			Source:         "session",
			Type:           "session_connected",
			OccurredAt:     row.AcctStartTime.UTC(),
			TenantID:       tenantID,
			CustomerID:     customerID,
			SubscriptionID: subscriptionsByUsername[row.Username],
			Summary:        summary,
			SourcePath:     fmt.Sprintf("/radius/online/%d/show", row.ID),
			Severity:       "info",
		})
	}

	accountingQuery := func() *gorm.DB {
		return db.Model(&domain.RadiusAccounting{}).
			Where("tenant_id = ? AND username IN ? AND acct_stop_time >= ? AND acct_stop_time <= ?", tenantID, usernames, from, to)
	}
	var accountingCount int64
	if err := accountingQuery().Count(&accountingCount).Error; err != nil {
		return err
	}
	*total += accountingCount
	var accountingRows []domain.RadiusAccounting
	if err := accountingQuery().Order("acct_stop_time DESC, id ASC").Limit(limit).Find(&accountingRows).Error; err != nil {
		return err
	}
	for _, row := range accountingRows {
		summary := fmt.Sprintf("Session disconnected after %s (%s in / %s out)",
			(time.Duration(row.AcctSessionTime) * time.Second).String(),
			timelineBytes(row.AcctInputTotal), timelineBytes(row.AcctOutputTotal))
		*events = append(*events, customerTimelineEvent{
			EventID:        fmt.Sprintf("session:accounting:%d", row.ID),
			Source:         "session",
			Type:           "session_disconnected",
			OccurredAt:     row.AcctStopTime.UTC(),
			TenantID:       tenantID,
			CustomerID:     customerID,
			SubscriptionID: subscriptionsByUsername[row.Username],
			Summary:        summary,
			SourcePath:     fmt.Sprintf("/radius/accounting/%d/show", row.ID),
			Severity:       "info",
		})
	}
	return nil
}

func appendTicketTimelineEvents(db *gorm.DB, tenantID, customerID int64, from, to time.Time, events *[]customerTimelineEvent, total *int64) error {
	var rows []domain.TroubleTicket
	if err := db.Model(&domain.TroubleTicket{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Where("(created_at >= ? AND created_at <= ?) OR (resolved_at IS NOT NULL AND resolved_at >= ? AND resolved_at <= ?) OR (resolved_at IS NULL AND updated_at > created_at AND updated_at >= ? AND updated_at <= ?)",
			from, to, from, to, from, to).
		Order("updated_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return err
	}
	generated := 0
	for _, row := range rows {
		basePath := "/isp/tickets"
		if withinCustomerTimelineRange(row.CreatedAt, from, to) {
			generated++
			*events = append(*events, customerTimelineEvent{
				EventID:        fmt.Sprintf("ticket:%d:created", row.ID),
				Source:         "ticket",
				Type:           "ticket_created",
				OccurredAt:     row.CreatedAt.UTC(),
				TenantID:       tenantID,
				CustomerID:     customerID,
				SubscriptionID: row.SubscriptionID,
				Summary:        fmt.Sprintf("Ticket %s opened: %s", row.TicketNo, row.Subject),
				SourcePath:     basePath,
				Severity:       ticketEventSeverity(row.Priority),
			})
		}
		if row.ResolvedAt != nil && withinCustomerTimelineRange(*row.ResolvedAt, from, to) {
			generated++
			*events = append(*events, customerTimelineEvent{
				EventID:        fmt.Sprintf("ticket:%d:resolved", row.ID),
				Source:         "ticket",
				Type:           "ticket_resolved",
				OccurredAt:     row.ResolvedAt.UTC(),
				TenantID:       tenantID,
				CustomerID:     customerID,
				SubscriptionID: row.SubscriptionID,
				Summary:        fmt.Sprintf("Ticket %s resolved", row.TicketNo),
				SourcePath:     basePath,
				Severity:       "success",
			})
			continue
		}
		if row.ResolvedAt == nil && row.UpdatedAt.After(row.CreatedAt) && withinCustomerTimelineRange(row.UpdatedAt, from, to) {
			generated++
			*events = append(*events, customerTimelineEvent{
				EventID:        fmt.Sprintf("ticket:%d:updated", row.ID),
				Source:         "ticket",
				Type:           "ticket_updated",
				OccurredAt:     row.UpdatedAt.UTC(),
				TenantID:       tenantID,
				CustomerID:     customerID,
				SubscriptionID: row.SubscriptionID,
				Summary:        fmt.Sprintf("Ticket %s updated: %s (status %s)", row.TicketNo, row.Subject, row.Status),
				SourcePath:     basePath,
				Severity:       ticketEventSeverity(row.Priority),
			})
		}
	}
	*total += int64(generated)
	return nil
}

func parseCustomerTimelineSources(raw string) (map[string]bool, []string, error) {
	allowed := make(map[string]bool, len(customerTimelineSourceOrder))
	for _, source := range customerTimelineSourceOrder {
		allowed[source] = true
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return allowed, append([]string(nil), customerTimelineSourceOrder...), nil
	}
	selected := make(map[string]bool, len(customerTimelineSourceOrder))
	for _, part := range strings.Split(raw, ",") {
		source := strings.ToLower(strings.TrimSpace(part))
		if source == "" {
			continue
		}
		if !allowed[source] {
			return nil, nil, fmt.Errorf("unknown timeline source %q; allowed: %s", source, strings.Join(customerTimelineSourceOrder, ", "))
		}
		selected[source] = true
	}
	if len(selected) == 0 {
		return nil, nil, fmt.Errorf("at least one source is required; allowed: %s", strings.Join(customerTimelineSourceOrder, ", "))
	}
	ordered := make([]string, 0, len(selected))
	for _, source := range customerTimelineSourceOrder {
		if selected[source] {
			ordered = append(ordered, source)
		}
	}
	return selected, ordered, nil
}

func parseCustomerTimelineTime(raw string, fallback time.Time, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, raw, time.Local)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			return parsed.Add(24*time.Hour - time.Nanosecond), nil
		}
		return parsed, nil
	}
	return time.Time{}, errors.New("invalid time format")
}

func withinCustomerTimelineRange(ts time.Time, from, to time.Time) bool {
	return !ts.IsZero() && !ts.Before(from) && !ts.After(to)
}

func billingEventSeverity(eventType string) string {
	value := strings.ToLower(eventType)
	switch {
	case strings.Contains(value, "suspend"), strings.Contains(value, "terminat"), strings.Contains(value, "overdue"):
		return "warning"
	case strings.Contains(value, "restored"), strings.Contains(value, "reactivat"), strings.Contains(value, "paid"), strings.Contains(value, "activated"):
		return "success"
	default:
		return "info"
	}
}

func ticketEventSeverity(priority string) string {
	switch strings.ToLower(priority) {
	case "urgent", "high":
		return "warning"
	default:
		return "info"
	}
}

func timelineBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for value := n / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
