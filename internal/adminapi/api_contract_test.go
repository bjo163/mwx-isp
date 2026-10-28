package adminapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// contractRows is how many fixtures each list endpoint is seeded with.
const contractRows = 5

// contractListEndpoint describes one collection endpoint that the React Admin
// dataProvider consumes. The table is the API contract: every resource mapped in
// web/src/providers/dataProvider.ts must have an entry here, otherwise the UI
// can ship a page whose endpoint breaks the payload grammar without any test
// noticing (see TestDataProviderResourcesHaveContractCoverage).
type contractListEndpoint struct {
	path        string
	handler     echo.HandlerFunc
	defaultSize int
	maxSize     int
	filter      url.Values
	filterTotal int64
	rowKey      string
	firstRow    string
}

func pagedListEndpoints() []contractListEndpoint {
	return []contractListEndpoint{
		{
			path: "/users", handler: listRadiusUsers,
			defaultSize: defaultPageSize, maxSize: maxPageSize,
			filter: url.Values{"status": {"disabled"}}, filterTotal: 1,
			rowKey: "username", firstRow: "user-1",
		},
		{
			path: "/sessions", handler: ListOnlineSessions,
			defaultSize: compactListPageSize, maxSize: compactListMaxPageSize,
			filter: url.Values{"nas_addr": {"10.9.0.2"}}, filterTotal: 2,
			rowKey: "username", firstRow: "user-5",
		},
		{
			path: "/accounting", handler: ListAccounting,
			defaultSize: compactListPageSize, maxSize: compactListMaxPageSize,
			filter: url.Values{"username": {"user-4"}}, filterTotal: 1,
			rowKey: "username", firstRow: "user-5",
		},
		{
			path: "/radius-profiles", handler: ListProfiles,
			defaultSize: compactListPageSize, maxSize: compactListMaxPageSize,
			filter: url.Values{"name": {"profile-2"}}, filterTotal: 1,
			rowKey: "name", firstRow: "profile-5",
		},
		{
			path: "/isp/customers", handler: listCustomers,
			defaultSize: defaultPageSize, maxSize: maxPageSize,
			filter: url.Values{"status": {"suspended"}}, filterTotal: 1,
			rowKey: "name", firstRow: "Customer 5",
		},
		{
			path: "/isp/packages", handler: listPackages,
			defaultSize: defaultPageSize, maxSize: maxPageSize,
			filter: url.Values{"status": {"inactive"}}, filterTotal: 1,
			rowKey: "name", firstRow: "Package 5",
		},
		{
			path: "/isp/subscriptions", handler: listSubscriptions,
			defaultSize: defaultPageSize, maxSize: maxPageSize,
			filter: url.Values{"customer_id": {"4"}}, filterTotal: 1,
			rowKey: "subscription_no", firstRow: "SUB-005",
		},
		{
			path: "/isp/invoices", handler: listInvoices,
			defaultSize: defaultPageSize, maxSize: maxPageSize,
			filter: url.Values{"invoice_no": {"INV-005"}}, filterTotal: 1,
			rowKey: "invoice_no", firstRow: "INV-005",
		},
		{
			path: "/isp/payments", handler: listPayments,
			defaultSize: defaultPageSize, maxSize: maxPageSize,
			filter: url.Values{"invoice_id": {"2"}}, filterTotal: 1,
			rowKey: "payment_no", firstRow: "PAY-005",
		},
	}
}

func unpagedListEndpoints() []contractListEndpoint {
	return []contractListEndpoint{
		{path: "/system/config/schemas", handler: getConfigSchemas},
	}
}

func seedContractFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{},
		&domain.InternetPackage{},
		&domain.Subscription{},
		&domain.Invoice{},
		&domain.Payment{},
	))

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= contractRows; i++ {
		userStatus := "enabled"
		if i == 3 {
			userStatus = "disabled"
		}
		require.NoError(t, db.Create(&domain.RadiusUser{
			Username: fmt.Sprintf("user-%d", i),
			Password: "secret",
			Email:    fmt.Sprintf("user-%d@example.com", i),
			Status:   userStatus,
		}).Error)
		require.NoError(t, db.Create(&domain.RadiusProfile{
			Name:   fmt.Sprintf("profile-%d", i),
			Status: "enabled",
		}).Error)

		nasAddr := "10.9.0.1"
		if i%2 == 0 {
			nasAddr = "10.9.0.2"
		}
		require.NoError(t, db.Create(&domain.RadiusOnline{
			Username:      fmt.Sprintf("user-%d", i),
			AcctSessionId: fmt.Sprintf("sess-%d", i),
			NasAddr:       nasAddr,
			AcctStartTime: base.Add(time.Duration(i) * time.Hour),
		}).Error)
		require.NoError(t, db.Create(&domain.RadiusAccounting{
			Username:      fmt.Sprintf("user-%d", i),
			AcctSessionId: fmt.Sprintf("acct-%d", i),
			NasAddr:       nasAddr,
			AcctStartTime: base.Add(time.Duration(i) * time.Hour),
			AcctStopTime:  base.Add(time.Duration(i)*time.Hour + 30*time.Minute),
		}).Error)

		customerStatus := "active"
		if i == 2 {
			customerStatus = "suspended"
		}
		require.NoError(t, db.Create(&domain.Customer{
			CustomerNo: fmt.Sprintf("CUST-%03d", i),
			Name:       fmt.Sprintf("Customer %d", i),
			Status:     customerStatus,
		}).Error)

		packageStatus := "active"
		if i == 3 {
			packageStatus = "inactive"
		}
		require.NoError(t, db.Create(&domain.InternetPackage{
			Code:            fmt.Sprintf("PKG-%03d", i),
			Name:            fmt.Sprintf("Package %d", i),
			Price:           150000,
			RadiusProfileID: int64(i),
			Status:          packageStatus,
		}).Error)

		subscriptionStatus := "active"
		if i == 4 {
			subscriptionStatus = "suspended"
		}
		require.NoError(t, db.Create(&domain.Subscription{
			SubscriptionNo: fmt.Sprintf("SUB-%03d", i),
			CustomerID:     int64(i),
			PackageID:      int64(i),
			Status:         subscriptionStatus,
			StartDate:      base,
		}).Error)

		require.NoError(t, db.Create(&domain.Invoice{
			InvoiceNo:      fmt.Sprintf("INV-%03d", i),
			CustomerID:     int64(i),
			SubscriptionID: int64(i),
			InvoiceDate:    base,
			DueDate:        base.AddDate(0, 0, 30),
			PeriodStart:    base,
			PeriodEnd:      base.AddDate(0, 1, 0),
			Subtotal:       150000,
			Total:          150000,
			Balance:        150000,
			Status:         "issued",
		}).Error)
		require.NoError(t, db.Create(&domain.Payment{
			PaymentNo:  fmt.Sprintf("PAY-%03d", i),
			CustomerID: int64(i),
			InvoiceID:  int64(i),
			Amount:     150000,
			Method:     "cash",
			PaidAt:     base.Add(time.Duration(i) * time.Hour),
			Status:     "received",
		}).Error)
	}
}

// contractPage is a decoded list response whose payload has already been
// validated against the shared success envelope.
type contractPage struct {
	rec  *httptest.ResponseRecorder
	data json.RawMessage
	meta *Meta
	rows []map[string]any
}

func invokeContractList(t *testing.T, db *gorm.DB, e *echo.Echo, appCtx app.AppContext, ep contractListEndpoint, query url.Values) contractPage {
	t.Helper()

	target := ep.path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, ep.handler(c))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	require.Contains(t, rec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), "body: %s", rec.Body.String())
	for key := range envelope {
		require.Contains(t, []string{"data", "meta"}, key, "unexpected success envelope key %q", key)
	}

	var body struct {
		Data json.RawMessage `json:"data"`
		Meta *Meta           `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEqual(t, "null", strings.TrimSpace(string(body.Data)),
		"a list payload must serialize as an array, never null")

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(body.Data, &rows), "data must be a JSON array: %s", body.Data)

	return contractPage{rec: rec, data: body.Data, meta: body.Meta, rows: rows}
}

// TestContractListPaginationGrammar pins the pagination contract shared by every
// collection endpoint: both `pageSize` and React Admin's `perPage` are accepted,
// pageSize wins when both are present, defaults and caps differ per endpoint, and
// an empty page is still a JSON array with a full meta block.
func TestContractListPaginationGrammar(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	seedContractFixtures(t, db)

	for _, ep := range pagedListEndpoints() {
		t.Run(ep.path, func(t *testing.T) {
			cases := []struct {
				name      string
				query     url.Values
				wantRows  int
				wantPage  int
				wantSize  int
				wantTotal int64
				checkSort bool
			}{
				{"defaults", nil, contractRows, 1, ep.defaultSize, contractRows, true},
				{"pageSize honored", url.Values{"pageSize": {"2"}}, 2, 1, 2, contractRows, true},
				{"perPage honored", url.Values{"perPage": {"2"}}, 2, 1, 2, contractRows, true},
				{"pageSize wins over perPage", url.Values{"pageSize": {"3"}, "perPage": {"1"}}, 3, 1, 3, contractRows, true},
				{"third page", url.Values{"page": {"3"}, "pageSize": {"2"}}, 1, 3, 2, contractRows, false},
				{"page past the end", url.Values{"page": {"9"}, "pageSize": {"2"}}, 0, 9, 2, contractRows, false},
				{"garbage falls back", url.Values{"page": {"abc"}, "pageSize": {"xyz"}, "perPage": {"nope"}}, contractRows, 1, ep.defaultSize, contractRows, true},
				{"non-positive falls back", url.Values{"page": {"0"}, "pageSize": {"-3"}}, contractRows, 1, ep.defaultSize, contractRows, true},
				{"oversized clamped", url.Values{"pageSize": {fmt.Sprint(ep.maxSize + 500)}}, contractRows, 1, ep.maxSize, contractRows, true},
				{"documented filter narrows", ep.filter, int(ep.filterTotal), 1, ep.defaultSize, ep.filterTotal, false},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					page := invokeContractList(t, db, e, appCtx, ep, tc.query)

					assert.Len(t, page.rows, tc.wantRows)
					require.NotNil(t, page.meta, "list responses must carry meta")
					assert.Equal(t, tc.wantTotal, page.meta.Total)
					assert.Equal(t, tc.wantPage, page.meta.Page)
					assert.Equal(t, tc.wantSize, page.meta.PageSize)
					if tc.checkSort {
						require.NotEmpty(t, page.rows)
						assert.Equal(t, ep.firstRow, page.rows[0][ep.rowKey],
							"the endpoint must keep its documented default order")
					}
				})
			}
		})
	}
}

// TestContractListSortFallback pins that sort/order never reach the ORDER BY
// clause raw: handlers that support sorting validate against an allowlist and
// fall back to their default, handlers that do not support sorting ignore the
// parameter. Either way the request must stay 200 and the data intact.
func TestContractListSortFallback(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	seedContractFixtures(t, db)

	injection := url.Values{
		"sort":  {"password; DROP TABLE radius_user; --"},
		"order": {"sideways"},
	}

	for _, ep := range pagedListEndpoints() {
		t.Run(ep.path, func(t *testing.T) {
			page := invokeContractList(t, db, e, appCtx, ep, injection)

			assert.Len(t, page.rows, contractRows)
			require.NotNil(t, page.meta)
			assert.Equal(t, int64(contractRows), page.meta.Total)
			assert.Equal(t, ep.firstRow, page.rows[0][ep.rowKey])

			var users int64
			require.NoError(t, db.Model(&domain.RadiusUser{}).Count(&users).Error)
			assert.Equal(t, int64(contractRows), users, "a sort injection attempt must not alter the schema")
		})
	}

	t.Run("user list never returns the stored password", func(t *testing.T) {
		page := invokeContractList(t, db, e, appCtx, pagedListEndpoints()[0], nil)
		require.NotEmpty(t, page.rows)
		for _, row := range page.rows {
			assert.Empty(t, row["password"])
		}
	})
}

// TestContractUnpaginatedConfigSchemas documents the one collection the UI reads
// without pagination: a plain array with no meta block (the dataProvider's
// extractTotal falls back to the array length).
func TestContractUnpaginatedConfigSchemas(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)

	ep := unpagedListEndpoints()[0]
	page := invokeContractList(t, db, e, appCtx, ep, nil)

	assert.Nil(t, page.meta, "the config schema list is not paginated")
	require.NotEmpty(t, page.rows)
	for _, row := range page.rows {
		assert.NotEmpty(t, row["key"])
		assert.NotEmpty(t, row["type"])
	}
}

func assertErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) ErrorResponse {
	t.Helper()

	assert.Contains(t, rec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), "body: %s", rec.Body.String())
	for key := range envelope {
		assert.Contains(t, []string{"error", "message", "details"}, key,
			"unexpected error envelope key %q", key)
	}

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, wantCode, resp.Error)
	assert.NotEmpty(t, resp.Message)
	return resp
}

// TestErrorContractEnvelope pins the failure envelope the UI branches on:
// stable machine codes, a human message, and no extra top-level keys.
func TestErrorContractEnvelope(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)

	noOperatorContext := func(req *http.Request, rec *httptest.ResponseRecorder) echo.Context {
		c := e.NewContext(req, rec)
		c.Set("appCtx", appCtx)
		c.Set("db", db)
		return c
	}

	t.Run("401 without an operator", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		handler := RequireLevel(LevelSuper)(func(c echo.Context) error {
			return c.NoContent(http.StatusNoContent)
		})

		require.NoError(t, handler(noOperatorContext(req, rec)))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assertErrorEnvelope(t, rec, "UNAUTHORIZED")
	})

	t.Run("403 for an insufficient level", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		handler := requireAdmin()(func(c echo.Context) error {
			return c.NoContent(http.StatusNoContent)
		})

		c := noOperatorContext(req, rec)
		c.Set("current_operator", &domain.SysOpr{
			ID:       2,
			Username: "operator",
			Level:    LevelOperator,
			Status:   "enabled",
		})

		require.NoError(t, handler(c))
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assertErrorEnvelope(t, rec, "PERMISSION_DENIED")
	})

	t.Run("400 for a non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/abc", nil)
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		require.NoError(t, getRadiusUser(c))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assertErrorEnvelope(t, rec, "INVALID_ID")
	})

	t.Run("404 for a missing record", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/424242", nil)
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		c.SetParamNames("id")
		c.SetParamValues("424242")

		require.NoError(t, getRadiusUser(c))
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assertErrorEnvelope(t, rec, "USER_NOT_FOUND")
	})

	t.Run("validation failures keep code message and details", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]interface{}{
			"error":   "INVALID_FIELD",
			"message": "tenant_id is required",
			"details": map[string]interface{}{"field": "tenant_id"},
		})
		require.NoError(t, handleValidationError(c, err))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		resp := assertErrorEnvelope(t, rec, "INVALID_FIELD")
		assert.Equal(t, "tenant_id is required", resp.Message)
		assert.Equal(t, map[string]interface{}{"field": "tenant_id"}, resp.Details)
	})

	t.Run("plain validation errors use the generic code", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		require.NoError(t, handleValidationError(c, errors.New("invalid request body")))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		resp := assertErrorEnvelope(t, rec, "VALIDATION_ERROR")
		assert.Equal(t, "invalid request body", resp.Message)
		assert.Nil(t, resp.Details)
	})
}

var dataProviderPairPattern = regexp.MustCompile(`'([a-zA-Z0-9_/-]+)'\s*:\s*'([a-zA-Z0-9_/-]+)'`)

func readDataProviderResourceMap() (map[string]string, error) {
	path := filepath.Join("..", "..", "web", "src", "providers", "dataProvider.ts")
	content, err := os.ReadFile(path) //nolint:gosec // reads a tracked source file at a fixed path
	if err != nil {
		return nil, err
	}

	source := string(content)
	start := strings.Index(source, "const resourcePathMap")
	if start < 0 {
		return nil, errors.New("resourcePathMap not found in dataProvider.ts")
	}
	block := source[start:]
	if end := strings.Index(block, "};"); end >= 0 {
		block = block[:end]
	}

	mapped := make(map[string]string)
	for _, match := range dataProviderPairPattern.FindAllStringSubmatch(block, -1) {
		mapped[match[1]] = match[2]
	}
	return mapped, nil
}

// TestDataProviderResourcesHaveContractCoverage is the contract lint: every
// resource the React Admin dataProvider exposes must be backed by an endpoint
// case in this file, so adding a UI resource without pinning its payload fails
// the build instead of shipping an unchecked breaking change.
func TestDataProviderResourcesHaveContractCoverage(t *testing.T) {
	mapped, err := readDataProviderResourceMap()
	require.NoError(t, err, "resourcePathMap in web/src/providers/dataProvider.ts must stay parseable")
	require.NotEmpty(t, mapped)

	covered := make(map[string]bool)
	for _, ep := range append(pagedListEndpoints(), unpagedListEndpoints()...) {
		covered[ep.path] = true
	}

	for resource, path := range mapped {
		target := "/" + strings.TrimPrefix(path, "/")
		assert.True(t, covered[target],
			"resource %q maps to %q but no API contract case covers it; add one to api_contract_test.go",
			resource, target)
	}
}
