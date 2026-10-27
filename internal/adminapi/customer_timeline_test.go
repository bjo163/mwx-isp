package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCustomerTimelineDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{},
		&domain.Subscription{},
		&domain.Invoice{},
		&domain.Payment{},
		&domain.BillingEvent{},
		&domain.TroubleTicket{},
	))
	return db
}

func newCustomerTimelineCtx(t *testing.T, db *gorm.DB, appCtx app.AppContext, customerID string, query string, tenantID int64) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/isp/customers/"+customerID+"/timeline"+query, nil)
	req = req.WithContext(tenancy.WithTenantID(req.Context(), tenantID))
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.Set("tenant_id", tenantID)
	c.SetParamNames("id")
	c.SetParamValues(customerID)
	return c, rec
}

type customerTimelineTestEnvelope struct {
	Data customerTimelineResponse `json:"data"`
	Meta Meta                     `json:"meta"`
}

func decodeCustomerTimeline(t *testing.T, rec *httptest.ResponseRecorder) customerTimelineTestEnvelope {
	t.Helper()
	var resp customerTimelineTestEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

type timelineSubject struct {
	Customer     domain.Customer
	Subscription domain.Subscription
	User         domain.RadiusUser
}

func seedTimelineSubject(t *testing.T, db *gorm.DB, tenantID, seed int64, username string) timelineSubject {
	t.Helper()
	customer := domain.Customer{
		TenantID: tenantID, CustomerNo: fmt.Sprintf("C-TL-%d-%d", tenantID, seed),
		Name: fmt.Sprintf("Timeline Customer %d", seed), Status: domain.CustomerActive,
	}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{TenantID: tenantID, Username: username, Password: "super-secret-credential", Status: "enabled"}
	require.NoError(t, db.Create(&user).Error)
	subscription := domain.Subscription{
		TenantID: tenantID, SubscriptionNo: fmt.Sprintf("SUB-TL-%d-%d", tenantID, seed),
		CustomerID: customer.ID, PackageID: 1, RadiusUserID: user.ID, Status: domain.SubscriptionActive,
		StartDate: time.Now().Add(-100 * 24 * time.Hour),
	}
	require.NoError(t, db.Create(&subscription).Error)
	return timelineSubject{Customer: customer, Subscription: subscription, User: user}
}

func timelineEventIDs(events []customerTimelineEvent) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.EventID)
	}
	return ids
}

func TestCustomerTimelineAggregatesSourcesInStableOrder(t *testing.T) {
	db := setupCustomerTimelineDB(t)
	appCtx := setupTestApp(t, db)
	subject := seedTimelineSubject(t, db, 1, 1, "timeline-alice")
	now := time.Now().UTC().Truncate(time.Second)
	resolvedAt := now.Add(-30 * time.Minute)

	invoice := domain.Invoice{
		TenantID: 1, InvoiceNo: "INV-TL-1", CustomerID: subject.Customer.ID,
		SubscriptionID: subject.Subscription.ID, InvoiceDate: now.Add(-4 * time.Hour),
		DueDate: now.Add(72 * time.Hour), PeriodStart: now.Add(-4 * time.Hour), PeriodEnd: now.Add(26 * 24 * time.Hour),
		Subtotal: 150000, Total: 150000, Balance: 150000, Status: domain.InvoiceOverdue,
		CreatedAt: now.Add(-4 * time.Hour),
	}
	require.NoError(t, db.Create(&invoice).Error)
	require.NoError(t, db.Create(&domain.Payment{
		TenantID: 1, PaymentNo: "PAY-TL-1", CustomerID: subject.Customer.ID, InvoiceID: invoice.ID,
		Amount: 50000, Method: "cash", Reference: "RCPT-9", PaidAt: now.Add(-3 * time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.BillingEvent{
		TenantID: 1, CustomerID: subject.Customer.ID, SubscriptionID: subject.Subscription.ID,
		Type: "subscription_activated", Description: "Activated after survey", CreatedAt: now.Add(-5 * time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: 1, Username: subject.User.Username, NasAddr: "192.0.2.9", FramedIpaddr: "100.64.0.10",
		AcctSessionId: "timeline-online-1", AcctStartTime: now.Add(-2 * time.Hour), LastUpdate: now.Add(-time.Minute),
	}).Error)
	require.NoError(t, db.Create(&domain.RadiusAccounting{
		TenantID: 1, Username: subject.User.Username, NasAddr: "192.0.2.9", FramedIpaddr: "100.64.0.11",
		AcctSessionId: "timeline-accounting-1", AcctStartTime: now.Add(-6 * time.Hour), AcctStopTime: now.Add(-time.Hour),
		AcctSessionTime: 5 * 3600, AcctInputTotal: 1024 * 1024 * 5, AcctOutputTotal: 1024 * 1024 * 50,
		LastUpdate: now.Add(-time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.TroubleTicket{
		TenantID: 1, TicketNo: "TCK-TL-1", CustomerID: subject.Customer.ID, SubscriptionID: subject.Subscription.ID,
		Subject: "No internet access", Category: "no_internet", Priority: "high", Status: "resolved",
		CreatedAt: now.Add(-6 * time.Hour), UpdatedAt: now.Add(-30 * time.Minute), ResolvedAt: &resolvedAt,
	}).Error)

	c, firstRec := newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(subject.Customer.ID, 10), "?pageSize=3", 1)
	require.NoError(t, getCustomerTimeline(c))
	require.Equal(t, http.StatusOK, firstRec.Code)
	first := decodeCustomerTimeline(t, firstRec)
	require.Equal(t, int64(7), first.Meta.Total)
	require.Equal(t, int64(7), first.Data.Pagination.Total)
	require.Equal(t, 1, first.Data.Pagination.Page)
	require.Equal(t, 3, first.Data.Pagination.PageSize)
	assert.Equal(t, []string{
		"ticket:1:resolved",
		"session:accounting:1",
		"session:online:1",
	}, timelineEventIDs(first.Data.Events))
	assert.Equal(t, subject.Customer.ID, first.Data.CustomerID)
	assert.Equal(t, []string{"billing", "invoice", "payment", "session", "ticket"}, first.Data.Sources)

	seen := map[string]bool{}
	all := make([]customerTimelineEvent, 0, 7)
	pages := []int64{}
	for page := 1; page <= 3; page++ {
		c, rec := newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(subject.Customer.ID, 10), fmt.Sprintf("?pageSize=3&page=%d", page), 1)
		require.NoError(t, getCustomerTimeline(c))
		require.Equal(t, http.StatusOK, rec.Code)
		resp := decodeCustomerTimeline(t, rec)
		pages = append(pages, resp.Meta.Total)
		for _, event := range resp.Data.Events {
			require.Falsef(t, seen[event.EventID], "duplicate event %s across pages", event.EventID)
			seen[event.EventID] = true
			all = append(all, event)
		}
	}
	require.Len(t, all, 7)
	assert.Equal(t, []int64{7, 7, 7}, pages)
	assert.Equal(t, []string{
		"ticket:1:resolved", "session:accounting:1", "session:online:1",
		"payment:1:received", "invoice:1:issued", "billing:1", "ticket:1:created",
	}, timelineEventIDs(all))
	assert.Equal(t, "warning", all[4].Severity)
	assert.Equal(t, "/isp/invoices/1/show", all[4].SourcePath)
	for _, event := range all {
		assert.Equal(t, time.UTC, event.OccurredAt.Location())
		assert.Equal(t, int64(1), event.TenantID)
	}
	assert.NotContains(t, firstRec.Body.String(), "super-secret-credential")
	assert.NotContains(t, firstRec.Body.String(), `"password"`)
}

func TestCustomerTimelineDateRangeAndSourceFilters(t *testing.T) {
	db := setupCustomerTimelineDB(t)
	appCtx := setupTestApp(t, db)
	subject := seedTimelineSubject(t, db, 1, 2, "timeline-bob")
	feb10 := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)

	require.NoError(t, db.Create(&domain.Invoice{
		TenantID: 1, InvoiceNo: "INV-TL-2", CustomerID: subject.Customer.ID, SubscriptionID: subject.Subscription.ID,
		InvoiceDate: feb10, DueDate: feb10.Add(72 * time.Hour), PeriodStart: feb10, PeriodEnd: feb10.Add(30 * 24 * time.Hour),
		Total: 200000, Balance: 200000, Status: domain.InvoiceIssued, CreatedAt: feb10,
	}).Error)
	require.NoError(t, db.Create(&domain.Payment{
		TenantID: 1, PaymentNo: "PAY-TL-2", CustomerID: subject.Customer.ID, InvoiceID: 1,
		Amount: 200000, Method: "bank_transfer", PaidAt: feb10.Add(24 * time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.BillingEvent{
		TenantID: 1, CustomerID: subject.Customer.ID, SubscriptionID: subject.Subscription.ID,
		Type: "subscription_suspended", Description: "Overdue", CreatedAt: feb10.Add(10 * 24 * time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: 1, Username: subject.User.Username, NasAddr: "192.0.2.10",
		AcctSessionId: "timeline-online-2", AcctStartTime: feb10.Add(48 * time.Hour), LastUpdate: feb10.Add(48 * time.Hour),
	}).Error)
	oldBilling := domain.BillingEvent{
		TenantID: 1, CustomerID: subject.Customer.ID, SubscriptionID: subject.Subscription.ID,
		Type: "subscription_created", Description: "Created long ago", CreatedAt: time.Now().Add(-120 * 24 * time.Hour),
	}
	require.NoError(t, db.Create(&oldBilling).Error)

	query := "?from=2026-02-01&to=2026-02-28&sources=invoice,payment&pageSize=20"
	c, rec := newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(subject.Customer.ID, 10), query, 1)
	require.NoError(t, getCustomerTimeline(c))
	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeCustomerTimeline(t, rec)
	assert.Equal(t, int64(2), resp.Meta.Total)
	assert.Equal(t, []string{"invoice", "payment"}, resp.Data.Sources)
	for _, event := range resp.Data.Events {
		assert.Contains(t, []string{"invoice", "payment"}, event.Source)
	}

	c, rec = newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(subject.Customer.ID, 10), "?from=2026-03-01&to=2026-03-02", 1)
	require.NoError(t, getCustomerTimeline(c))
	resp = decodeCustomerTimeline(t, rec)
	assert.Equal(t, int64(0), resp.Meta.Total)
	require.Empty(t, resp.Data.Events)

	// The default window is 90 days, so the February and 120-day-old events are
	// both outside it until the caller supplies an explicit lower bound.
	c, rec = newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(subject.Customer.ID, 10), "?sources=billing", 1)
	require.NoError(t, getCustomerTimeline(c))
	resp = decodeCustomerTimeline(t, rec)
	assert.Equal(t, int64(0), resp.Meta.Total)

	c, rec = newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(subject.Customer.ID, 10), "?from=2025-01-01&sources=billing", 1)
	require.NoError(t, getCustomerTimeline(c))
	resp = decodeCustomerTimeline(t, rec)
	assert.Equal(t, int64(2), resp.Meta.Total)
	assert.Equal(t, "subscription_created", resp.Data.Events[0].Type)
	assert.Equal(t, "info", resp.Data.Events[0].Severity)
	assert.Equal(t, "subscription_suspended", resp.Data.Events[1].Type)
	assert.Equal(t, "warning", resp.Data.Events[1].Severity)
}

func TestCustomerTimelineIsTenantScoped(t *testing.T) {
	db := setupCustomerTimelineDB(t)
	appCtx := setupTestApp(t, db)
	require.NoError(t, db.Create(&domain.Tenant{ID: 2, Name: "Tenant Two", Slug: "tenant-two", Kind: "isp", Status: "active"}).Error)

	first := seedTimelineSubject(t, db, 1, 3, "timeline-shared")
	second := seedTimelineSubject(t, db, 2, 4, "timeline-shared")
	now := time.Now().UTC()

	require.NoError(t, db.Create(&domain.BillingEvent{
		TenantID: 1, CustomerID: first.Customer.ID, SubscriptionID: first.Subscription.ID,
		Type: "subscription_activated", Description: "tenant one event", CreatedAt: now.Add(-time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.BillingEvent{
		TenantID: 2, CustomerID: second.Customer.ID, SubscriptionID: second.Subscription.ID,
		Type: "subscription_activated", Description: "tenant two event", CreatedAt: now.Add(-time.Hour),
	}).Error)
	// A foreign tenant must not leak sessions even when the username matches.
	require.NoError(t, db.Create(&domain.RadiusAccounting{
		TenantID: 2, Username: first.User.Username, NasAddr: "198.51.100.20",
		AcctSessionId: "timeline-foreign-session", AcctStartTime: now.Add(-2 * time.Hour), AcctStopTime: now.Add(-time.Hour),
		AcctSessionTime: 3600, LastUpdate: now.Add(-time.Hour),
	}).Error)

	c, rec := newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(first.Customer.ID, 10), "?pageSize=20", 1)
	require.NoError(t, getCustomerTimeline(c))
	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeCustomerTimeline(t, rec)
	require.Len(t, resp.Data.Events, 1)
	assert.Equal(t, "billing:1", resp.Data.Events[0].EventID)

	// Cross-tenant lookup must behave like a missing record.
	c, rec = newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(second.Customer.ID, 10), "", 1)
	require.NoError(t, getCustomerTimeline(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	c, rec = newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(second.Customer.ID, 10), "?pageSize=20", 2)
	require.NoError(t, getCustomerTimeline(c))
	require.Equal(t, http.StatusOK, rec.Code)
	resp = decodeCustomerTimeline(t, rec)
	// Tenant two sees its own billing event and its own session, never tenant one's.
	require.Len(t, resp.Data.Events, 2)
	assert.Equal(t, "billing:2", resp.Data.Events[0].EventID)
	assert.Equal(t, "session:accounting:1", resp.Data.Events[1].EventID)
	for _, event := range resp.Data.Events {
		assert.Equal(t, int64(2), event.TenantID)
	}
}

func TestCustomerTimelinePaginationIsDeterministic(t *testing.T) {
	db := setupCustomerTimelineDB(t)
	appCtx := setupTestApp(t, db)
	subject := seedTimelineSubject(t, db, 1, 5, "timeline-carol")
	sameTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	for i := 0; i < 5; i++ {
		require.NoError(t, db.Create(&domain.BillingEvent{
			TenantID: 1, CustomerID: subject.Customer.ID, SubscriptionID: subject.Subscription.ID,
			Type: "subscription_updated", Description: fmt.Sprintf("event %d", i), CreatedAt: sameTime,
		}).Error)
	}
	require.NoError(t, db.Create(&domain.Invoice{
		TenantID: 1, InvoiceNo: "INV-TL-5", CustomerID: subject.Customer.ID, SubscriptionID: subject.Subscription.ID,
		InvoiceDate: sameTime, DueDate: sameTime.Add(24 * time.Hour), PeriodStart: sameTime, PeriodEnd: sameTime.Add(30 * 24 * time.Hour),
		Total: 100000, Balance: 100000, Status: domain.InvoiceIssued, CreatedAt: sameTime,
	}).Error)

	collect := func() []string {
		ids := make([]string, 0, 6)
		for page := 1; page <= 3; page++ {
			c, rec := newCustomerTimelineCtx(t, db, appCtx, strconv.FormatInt(subject.Customer.ID, 10), fmt.Sprintf("?pageSize=2&page=%d", page), 1)
			require.NoError(t, getCustomerTimeline(c))
			require.Equal(t, http.StatusOK, rec.Code)
			resp := decodeCustomerTimeline(t, rec)
			ids = append(ids, timelineEventIDs(resp.Data.Events)...)
		}
		return ids
	}

	firstRun := collect()
	secondRun := collect()
	assert.Equal(t, firstRun, secondRun)
	require.Len(t, firstRun, 6)
	assert.Equal(t, []string{
		"billing:1", "billing:2", "billing:3", "billing:4", "billing:5", "invoice:1:issued",
	}, firstRun)
	seen := map[string]bool{}
	for _, id := range firstRun {
		assert.Falsef(t, seen[id], "duplicate event %s", id)
		seen[id] = true
	}
}

func TestCustomerTimelineValidatesInput(t *testing.T) {
	db := setupCustomerTimelineDB(t)
	appCtx := setupTestApp(t, db)
	subject := seedTimelineSubject(t, db, 1, 6, "timeline-dave")
	customerID := strconv.FormatInt(subject.Customer.ID, 10)

	cases := []struct {
		name       string
		customerID string
		query      string
		status     int
	}{
		{name: "invalid id", customerID: "abc", status: http.StatusBadRequest},
		{name: "missing customer", customerID: "999999", status: http.StatusNotFound},
		{name: "invalid from", customerID: customerID, query: "?from=not-a-date", status: http.StatusBadRequest},
		{name: "invalid to", customerID: customerID, query: "?to=2026-13-40", status: http.StatusBadRequest},
		{name: "reversed range", customerID: customerID, query: "?from=2026-02-10&to=2026-02-01", status: http.StatusBadRequest},
		{name: "unknown source", customerID: customerID, query: "?sources=billing,calendar", status: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newCustomerTimelineCtx(t, db, appCtx, tc.customerID, tc.query, 1)
			require.NoError(t, getCustomerTimeline(c))
			assert.Equal(t, tc.status, rec.Code)
		})
	}
}
