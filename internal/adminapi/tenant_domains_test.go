package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func tenantDomainContext(t *testing.T, db *gorm.DB, e *echo.Echo, appCtx app.AppContext, method, path, body string, params map[string]string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.Set("tenant_id", domain.DefaultTenantID)
	if len(params) > 0 {
		c.SetParamNames("id")
		c.SetParamValues(params["id"])
	}
	return c, rec
}

func TestNormalizeTenantHostname(t *testing.T) {
	for _, test := range []struct {
		input string
		want string
		bad bool
	}{
		{input: "WWW.Example.COM.", want: "www.example.com"},
		{input: "isp.example.co.id", want: "isp.example.co.id"},
		{input: "localhost", bad: true},
		{input: "127.0.0.1", bad: true},
		{input: "https://isp.example.com", bad: true},
		{input: "isp.example.com/path", bad: true},
		{input: "isp.e-isp.id", bad: true},
		{input: "*.example.com", bad: true},
	} {
		got, err := normalizeTenantHostname(test.input)
		if test.bad {
			require.Error(t, err, test.input)
		} else {
			require.NoError(t, err, test.input)
			require.Equal(t, test.want, got)
		}
	}
}

func TestTenantDomainDNSChallengeLifecycle(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.TenantDomain{}))
	createCtx, createRec := tenantDomainContext(t, db, e, appCtx, http.MethodPost,
		"/api/v1/tenant-domains", `{"hostname":"isp.example.com"}`, nil)
	require.NoError(t, createTenantDomain(createCtx))
	require.Equal(t, http.StatusCreated, createRec.Code, createRec.Body.String())
	var created struct { Data tenantDomainResponse `json:"data"` }
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	require.Equal(t, "isp.example.com", created.Data.Hostname)
	require.Equal(t, "_mwx-isp-verification.isp.example.com", created.Data.VerificationTXT)
	require.True(t, strings.HasPrefix(created.Data.VerificationValue, tenantDomainTXTPrefix))
	require.Nil(t, created.Data.VerifiedAt)

	secondCtx, secondRec := tenantDomainContext(t, db, e, appCtx, http.MethodPost,
		"/api/v1/tenant-domains", `{"hostname":"portal.example.com"}`, nil)
	require.NoError(t, createTenantDomain(secondCtx))
	require.Equal(t, http.StatusCreated, secondRec.Code, secondRec.Body.String())

	oldLookup := lookupTenantDomainTXT
	lookupTenantDomainTXT = func(name string) ([]string, error) {
		require.Equal(t, created.Data.VerificationTXT, name)
		return []string{"some-other-value", created.Data.VerificationValue}, nil
	}
	defer func() { lookupTenantDomainTXT = oldLookup }()

	id := strconv.FormatInt(created.Data.ID, 10)
	verifyCtx, verifyRec := tenantDomainContext(t, db, e, appCtx, http.MethodPost,
		"/api/v1/tenant-domains/"+id+"/verify", "", map[string]string{"id": id})
	require.NoError(t, verifyTenantDomain(verifyCtx))
	require.Equal(t, http.StatusOK, verifyRec.Code, verifyRec.Body.String())
	var verified struct { Data tenantDomainResponse `json:"data"` }
	require.NoError(t, json.Unmarshal(verifyRec.Body.Bytes(), &verified))
	require.NotNil(t, verified.Data.VerifiedAt)

	var stored domain.TenantDomain
	require.NoError(t, db.Where("id = ? AND tenant_id = ?", created.Data.ID, domain.DefaultTenantID).First(&stored).Error)
	require.NotNil(t, stored.VerifiedAt)

	deleteCtx, deleteRec := tenantDomainContext(t, db, e, appCtx, http.MethodDelete,
		"/api/v1/tenant-domains/"+id, "", map[string]string{"id": id})
	require.NoError(t, deleteTenantDomain(deleteCtx))
	require.Equal(t, http.StatusOK, deleteRec.Code, deleteRec.Body.String())
}

func TestTenantDomainDoesNotVerifyWithoutExactTXTValue(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.TenantDomain{}))
	row := domain.TenantDomain{TenantID: domain.DefaultTenantID, Hostname: "pending.example.com", VerificationToken: "correct-token"}
	require.NoError(t, db.Create(&row).Error)
	oldLookup := lookupTenantDomainTXT
	lookupTenantDomainTXT = func(string) ([]string, error) { return []string{tenantDomainTXTValue("wrong-token")}, nil }
	defer func() { lookupTenantDomainTXT = oldLookup }()

	id := strconv.FormatInt(row.ID, 10)
	ctx, rec := tenantDomainContext(t, db, e, appCtx, http.MethodPost,
		"/api/v1/tenant-domains/"+id+"/verify", "", map[string]string{"id": id})
	require.NoError(t, verifyTenantDomain(ctx))
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	var stored domain.TenantDomain
	require.NoError(t, db.First(&stored, row.ID).Error)
	require.Nil(t, stored.VerifiedAt)
}
