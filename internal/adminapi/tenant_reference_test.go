package adminapi

import (
	"bytes"
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

const tenantIsolationSecondTenantID int64 = 2

type tenantIsolationFixture struct {
	Profile      domain.RadiusProfile
	User         domain.RadiusUser
	Customer     domain.Customer
	Package      domain.InternetPackage
	Subscription domain.Subscription
	ODP          domain.ODP
	Node         domain.NetNode
	NAS          domain.NetNas
}

func setupTenantIsolationDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{},
		&domain.InternetPackage{},
		&domain.Subscription{},
		&domain.TroubleTicket{},
		&domain.DocumentSequence{},
		&domain.Invoice{},
		&domain.ODP{},
		&domain.RadiusOnline{},
		&domain.RadiusAccounting{},
	))
	require.NoError(t, db.Create(&domain.Tenant{
		ID: tenantIsolationSecondTenantID, Name: "Second ISP", Slug: "second", Kind: "isp", Status: "active",
	}).Error)
}

// seedTenantIsolationFixture gives two tenants deliberately identical business
// identifiers so any query that forgets the tenant predicate returns two rows
// instead of one.
func seedTenantIsolationFixture(t *testing.T, db *gorm.DB, tenantID int64, suffix string) tenantIsolationFixture {
	t.Helper()
	profile := domain.RadiusProfile{TenantID: tenantID, Name: "Shared profile " + suffix, Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)
	user := domain.RadiusUser{
		TenantID: tenantID, ProfileId: profile.ID, Username: "shared-user",
		Password: "shared-secret", Status: "enabled", ExpireTime: time.Now().AddDate(1, 0, 0),
	}
	require.NoError(t, db.Create(&user).Error)
	odp := domain.ODP{TenantID: tenantID, Code: "ODP-SAME", Name: "Shared ODP " + suffix, TotalPorts: 8, Status: "active"}
	require.NoError(t, db.Create(&odp).Error)
	customer := domain.Customer{
		TenantID: tenantID, CustomerNo: "MWX-SAME-001", Name: "Shared Customer " + suffix,
		Status: domain.CustomerActive, ODPID: odp.ID, ODPCode: odp.Code,
	}
	require.NoError(t, db.Create(&customer).Error)
	pkg := domain.InternetPackage{
		TenantID: tenantID, Code: "PKG-SAME", Name: "Shared Package " + suffix,
		Price: 150000, RadiusProfileID: profile.ID, BillingCycle: "monthly", Status: "active",
	}
	require.NoError(t, db.Create(&pkg).Error)
	subscription := domain.Subscription{
		TenantID: tenantID, SubscriptionNo: "SUB-SAME", CustomerID: customer.ID,
		PackageID: pkg.ID, RadiusUserID: user.ID, Status: domain.SubscriptionActive,
		StartDate: time.Now().AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3,
	}
	require.NoError(t, db.Create(&subscription).Error)
	node := domain.NetNode{TenantID: tenantID, Name: "Shared Node " + suffix}
	require.NoError(t, db.Create(&node).Error)
	nas := domain.NetNas{ //nolint:gosec // G101: deterministic test fixture, not a real credential
		TenantID: tenantID, NodeId: node.ID, Name: "Shared NAS " + suffix,
		Identifier: "NAS-SAME", Ipaddr: fmt.Sprintf("198.51.100.%d", tenantID),
		Secret: "shared-nas-secret", Status: "enabled",
	}
	require.NoError(t, db.Create(&nas).Error)
	return tenantIsolationFixture{
		Profile: profile, User: user, Customer: customer, Package: pkg,
		Subscription: subscription, ODP: odp, Node: node, NAS: nas,
	}
}

func newTenantRequestContext(
	t *testing.T, e *echo.Echo, db *gorm.DB, appCtx app.AppContext,
	method, path, body string, tenantID int64, params map[string]string,
) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenancy.WithTenantID(req.Context(), tenantID))
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.Set("tenant_id", tenantID)
	if len(params) > 0 {
		names := make([]string, 0, len(params))
		values := make([]string, 0, len(params))
		for name, value := range params {
			names = append(names, name)
			values = append(values, value)
		}
		c.SetParamNames(names...)
		c.SetParamValues(values...)
	}
	return c, rec
}

func TestTenantIsolationRejectsCrossTenantForeignReferences(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	setupTenantIsolationDB(t, db)

	first := seedTenantIsolationFixture(t, db, domain.DefaultTenantID, "tenant one")
	second := seedTenantIsolationFixture(t, db, tenantIsolationSecondTenantID, "tenant two")

	// A second subscription inside the first tenant gives the mismatch case a
	// reference that exists but belongs to a different customer.
	otherCustomer := domain.Customer{
		TenantID: domain.DefaultTenantID, CustomerNo: "MWX-SAME-002",
		Name: "Other Customer", Status: domain.CustomerActive,
	}
	require.NoError(t, db.Create(&otherCustomer).Error)
	otherSubscription := domain.Subscription{
		TenantID: domain.DefaultTenantID, SubscriptionNo: "SUB-SAME-2",
		CustomerID: otherCustomer.ID, PackageID: first.Package.ID, Status: domain.SubscriptionActive,
		StartDate: time.Now().AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3,
	}
	require.NoError(t, db.Create(&otherSubscription).Error)

	// Identical customer numbers, usernames and subscription numbers must be
	// legal once per tenant and must never collapse into one row.
	var sharedCustomers int64
	require.NoError(t, db.Model(&domain.Customer{}).Where("customer_no = ?", "MWX-SAME-001").Count(&sharedCustomers).Error)
	assert.EqualValues(t, 2, sharedCustomers)
	var sharedUsers int64
	require.NoError(t, db.Model(&domain.RadiusUser{}).Where("username = ?", "shared-user").Count(&sharedUsers).Error)
	assert.EqualValues(t, 2, sharedUsers)

	// Listing resolves only the caller's tenant.
	c, rec := newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/isp/customers?q=MWX-SAME-001", "", domain.DefaultTenantID, nil)
	require.NoError(t, listCustomers(c))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var listed struct {
		Data []domain.Customer `json:"data"`
		Meta Meta              `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1)
	assert.Equal(t, first.Customer.ID, listed.Data[0].ID)
	assert.EqualValues(t, 1, listed.Meta.Total)

	// A foreign customer ID must not resolve for detail or update.
	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/isp/customers/"+strconv.FormatInt(second.Customer.ID, 10), "", domain.DefaultTenantID,
		map[string]string{"id": strconv.FormatInt(second.Customer.ID, 10)})
	require.NoError(t, getCustomer(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodPut,
		"/api/v1/isp/customers/"+strconv.FormatInt(second.Customer.ID, 10),
		`{"name":"Cross tenant overwrite","status":"active"}`, domain.DefaultTenantID,
		map[string]string{"id": strconv.FormatInt(second.Customer.ID, 10)})
	require.NoError(t, updateCustomer(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	var persistedCustomer domain.Customer
	require.NoError(t, db.First(&persistedCustomer, second.Customer.ID).Error)
	assert.Equal(t, second.Customer.Name, persistedCustomer.Name, "cross-tenant update must not modify the row")

	// Tickets: customer and subscription references must stay inside the tenant.
	ticketBodies := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{
			"foreign customer",
			fmt.Sprintf(`{"subject":"cross tenant","customer_id":"%d"}`, second.Customer.ID),
			http.StatusBadRequest, "CUSTOMER_NOT_FOUND",
		},
		{
			"foreign subscription",
			fmt.Sprintf(`{"subject":"cross tenant","subscription_id":"%d"}`, second.Subscription.ID),
			http.StatusBadRequest, "SUBSCRIPTION_NOT_FOUND",
		},
		{
			"mismatched pair",
			fmt.Sprintf(`{"subject":"cross tenant","customer_id":"%d","subscription_id":"%d"}`, first.Customer.ID, otherSubscription.ID),
			http.StatusBadRequest, "SUBSCRIPTION_MISMATCH",
		},
	}
	for _, tc := range ticketBodies {
		t.Run("ticket_"+tc.code, func(t *testing.T) {
			c, rec := newTenantRequestContext(t, e, db, appCtx, http.MethodPost, "/api/v1/isp/tickets", tc.body, domain.DefaultTenantID, nil)
			require.NoError(t, createTroubleTicket(c))
			assert.Equal(t, tc.status, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), tc.code)
		})
	}
	var ticketsInFirstTenant int64
	require.NoError(t, db.WithContext(tenancy.WithTenantID(t.Context(), domain.DefaultTenantID)).
		Model(&domain.TroubleTicket{}).Count(&ticketsInFirstTenant).Error)
	assert.Zero(t, ticketsInFirstTenant, "rejected references must not persist a ticket")

	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodPost, "/api/v1/isp/tickets",
		fmt.Sprintf(`{"subject":"valid ticket","customer_id":"%d","subscription_id":"%d"}`, first.Customer.ID, first.Subscription.ID),
		domain.DefaultTenantID, nil)
	require.NoError(t, createTroubleTicket(c))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var createdTicket struct {
		Data domain.TroubleTicket `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createdTicket))
	assert.Equal(t, first.Customer.ID, createdTicket.Data.CustomerID)
	var storedTicket domain.TroubleTicket
	require.NoError(t, db.Where("ticket_no = ?", createdTicket.Data.TicketNo).First(&storedTicket).Error)
	assert.Equal(t, domain.DefaultTenantID, storedTicket.TenantID)
	assert.Equal(t, first.Customer.ID, storedTicket.CustomerID)

	// NAS: node references must stay inside the tenant for create and update.
	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodPost, "/api/v1/network/nas",
		fmt.Sprintf(`{"node_id":"%d","name":"Cross NAS","ipaddr":"203.0.113.44","secret":"cross-secret"}`, second.Node.ID),
		domain.DefaultTenantID, nil)
	require.NoError(t, CreateNAS(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "NODE_NOT_FOUND")
	var crossNAS domain.NetNas
	assert.ErrorIs(t, db.Where("name = ?", "Cross NAS").First(&crossNAS).Error, gorm.ErrRecordNotFound,
		"rejected node reference must not persist a NAS")

	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodPut,
		"/api/v1/network/nas/"+strconv.FormatInt(first.NAS.ID, 10),
		fmt.Sprintf(`{"node_id":"%d"}`, second.Node.ID), domain.DefaultTenantID,
		map[string]string{"id": strconv.FormatInt(first.NAS.ID, 10)})
	require.NoError(t, UpdateNAS(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "NODE_NOT_FOUND")
	var persistedNAS domain.NetNas
	require.NoError(t, db.First(&persistedNAS, first.NAS.ID).Error)
	assert.Equal(t, first.Node.ID, persistedNAS.NodeId, "rejected node reference must not rewrite the NAS")

	// Other foreign references keep the same guarantee.
	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodPost, "/api/v1/isp/packages",
		fmt.Sprintf(`{"name":"Cross package","price":100000,"radius_profile_id":"%d"}`, second.Profile.ID),
		domain.DefaultTenantID, nil)
	require.NoError(t, savePackage(c, 0))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "PROFILE_NOT_FOUND")

	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodPost, "/api/v1/isp/subscriptions",
		fmt.Sprintf(`{"customer_id":"%d","package_id":"%d","username":"cross-sub","password":"secret123"}`, first.Customer.ID, second.Package.ID),
		domain.DefaultTenantID, nil)
	require.NoError(t, createSubscription(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "PACKAGE_NOT_FOUND")

	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodPost, "/api/v1/isp/customers",
		fmt.Sprintf(`{"name":"Cross ODP","odp_id":"%d"}`, second.ODP.ID), domain.DefaultTenantID, nil)
	require.NoError(t, createCustomer(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "INVALID_ODP")

	// Username-based RADIUS lookups must also respect the tenant boundary even
	// when both tenants use the same username.
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: domain.DefaultTenantID, Username: "shared-user",
		AcctSessionId: "session-tenant-one", FramedIpaddr: "192.0.2.77",
		AcctStartTime: time.Now().Add(-2 * time.Hour), LastUpdate: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: tenantIsolationSecondTenantID, Username: "shared-user",
		AcctSessionId: "session-tenant-two", FramedIpaddr: "192.0.2.77",
		AcctStartTime: time.Now().Add(-2 * time.Hour), LastUpdate: time.Now(),
	}).Error)

	c, rec = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/network/ipam/audit?ip=192.0.2.77", "", domain.DefaultTenantID, nil)
	require.NoError(t, auditIPHistory(c))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var audit struct {
		Data struct {
			Results []struct {
				AcctSessionID string `json:"acct_session_id"`
			} `json:"results"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &audit))
	require.NotEmpty(t, audit.Data.Results)
	for _, result := range audit.Data.Results {
		assert.Equal(t, "session-tenant-one", result.AcctSessionID,
			"an identical username in another tenant must not leak its sessions")
	}
}
