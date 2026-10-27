//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type itCustomerTimeline struct {
	CustomerID string   `json:"customer_id"`
	Sources    []string `json:"sources"`
	Events     []struct {
		EventID    string    `json:"event_id"`
		Source     string    `json:"source"`
		Type       string    `json:"type"`
		OccurredAt time.Time `json:"occurred_at"`
		CustomerID string    `json:"customer_id"`
		Summary    string    `json:"summary"`
		SourcePath string    `json:"source_path"`
		Severity   string    `json:"severity"`
	} `json:"events"`
	Pagination struct {
		Total    int64 `json:"total"`
		Page     int   `json:"page"`
		PageSize int   `json:"page_size"`
	} `json:"pagination"`
}

func TestPostgresCustomerTimelineAggregatesPersistedRows(t *testing.T) {
	client := newAPIClient(t)
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	now := time.Now().UTC().Truncate(time.Second)
	profileID := seedProfile(t, "it-timeline-profile-"+suffix)

	customer := domain.Customer{
		ID: common.UUIDint64(), CustomerNo: "C-TL-" + suffix, Name: "Timeline " + suffix, Status: "active",
	}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{
		ID: common.UUIDint64(), ProfileId: profileID, Username: "timeline-" + suffix,
		Password: "integration-secret", Status: common.ENABLED, ExpireTime: now.AddDate(1, 0, 0),
	}
	require.NoError(t, db.Create(&user).Error)
	subscription := domain.Subscription{
		ID: common.UUIDint64(), SubscriptionNo: "SUB-TL-" + suffix, CustomerID: customer.ID,
		PackageID: 1, RadiusUserID: user.ID, Status: "active", StartDate: now.AddDate(-1, 0, 0),
	}
	require.NoError(t, db.Create(&subscription).Error)
	invoice := domain.Invoice{
		TenantID: domain.DefaultTenantID, InvoiceNo: "INV-TL-" + suffix, CustomerID: customer.ID,
		SubscriptionID: subscription.ID, InvoiceDate: now.Add(-3 * time.Hour), DueDate: now.Add(72 * time.Hour),
		PeriodStart: now.Add(-3 * time.Hour), PeriodEnd: now.Add(30 * 24 * time.Hour),
		Subtotal: 250000, Total: 250000, Balance: 250000, Status: domain.InvoiceOverdue,
	}
	require.NoError(t, db.Create(&invoice).Error)
	require.NoError(t, db.Create(&domain.Payment{
		TenantID: domain.DefaultTenantID, PaymentNo: "PAY-TL-" + suffix, CustomerID: customer.ID,
		InvoiceID: invoice.ID, Amount: 100000, Method: "cash", PaidAt: now.Add(-2 * time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.BillingEvent{
		TenantID: domain.DefaultTenantID, CustomerID: customer.ID, SubscriptionID: subscription.ID,
		Type: "subscription_suspended", Description: "integration suspension", CreatedAt: now.Add(-4 * time.Hour),
	}).Error)
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: domain.DefaultTenantID, Username: user.Username, NasAddr: "203.0.113.10",
		FramedIpaddr: "100.64.7.10", AcctSessionId: "tl-session-" + suffix,
		AcctStartTime: now.Add(-time.Hour), LastUpdate: now.Add(-time.Minute),
	}).Error)
	resolvedAt := now.Add(-30 * time.Minute)
	require.NoError(t, db.Create(&domain.TroubleTicket{
		TenantID: domain.DefaultTenantID, TicketNo: "TCK-TL-" + suffix, CustomerID: customer.ID,
		SubscriptionID: subscription.ID, Subject: "integration ticket", Category: "no_internet",
		Priority: "high", Status: "resolved", CreatedAt: now.Add(-5 * time.Hour),
		UpdatedAt: resolvedAt, ResolvedAt: &resolvedAt,
	}).Error)

	status, body := client.get(t, "/api/v1/isp/customers/"+strconv.FormatInt(customer.ID, 10)+"/timeline?pageSize=50")
	require.Equalf(t, http.StatusOK, status, "timeline body: %s", body)
	var response struct {
		Data itCustomerTimeline `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &response))
	assert.Equal(t, strconv.FormatInt(customer.ID, 10), response.Data.CustomerID)
	assert.Equal(t, []string{"billing", "invoice", "payment", "session", "ticket"}, response.Data.Sources)
	assert.Equal(t, int64(6), response.Data.Pagination.Total)
	require.Len(t, response.Data.Events, 6)
	var persisted domain.TroubleTicket
	require.NoError(t, db.Where("ticket_no = ?", "TCK-TL-"+suffix).First(&persisted).Error)
	assert.WithinDuration(t, now.Add(-5*time.Hour), persisted.CreatedAt, 2*time.Second)
	require.NotNil(t, persisted.ResolvedAt)
	assert.WithinDuration(t, resolvedAt, *persisted.ResolvedAt, 2*time.Second)
	actualTypes := make([]string, 0, len(response.Data.Events))
	for _, event := range response.Data.Events {
		actualTypes = append(actualTypes, event.Type)
		assert.NotEmpty(t, event.SourcePath)
		assert.Equal(t, time.UTC, event.OccurredAt.Location())
	}
	assert.Equal(t, []string{"ticket_resolved", "session_connected", "payment_received", "invoice_issued", "subscription_suspended", "ticket_created"}, actualTypes)
	assert.NotContains(t, string(body), "integration-secret")
	assert.NotContains(t, string(body), `"password"`)

	status, body = client.get(t, "/api/v1/isp/customers/"+strconv.FormatInt(customer.ID, 10)+"/timeline?sources=payment&pageSize=10")
	require.Equalf(t, http.StatusOK, status, "payment timeline body: %s", body)
	require.NoError(t, json.Unmarshal(body, &response))
	assert.Equal(t, int64(1), response.Data.Pagination.Total)
	require.Len(t, response.Data.Events, 1)
	assert.Equal(t, "payment_received", response.Data.Events[0].Type)

	from := now.Add(-90 * time.Minute).Format(time.RFC3339)
	status, body = client.get(t, "/api/v1/isp/customers/"+strconv.FormatInt(customer.ID, 10)+"/timeline?sources=payment&from="+from+"&pageSize=10")
	require.Equalf(t, http.StatusOK, status, "filtered timeline body: %s", body)
	require.NoError(t, json.Unmarshal(body, &response))
	assert.Equal(t, int64(0), response.Data.Pagination.Total)
}
