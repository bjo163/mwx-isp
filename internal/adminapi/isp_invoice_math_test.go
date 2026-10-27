package adminapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSavePackagePriceBoundaries pins the integer-IDR input contract: negative
// prices and numbers outside int64 are rejected, zero stays valid for free
// packages, and a maximum int64 price is stored without truncation so invoice
// totals derived from it remain exact.
func TestSavePackagePriceBoundaries(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.InternetPackage{}, &domain.RadiusProfile{}))
	profile := domain.RadiusProfile{TenantID: domain.DefaultTenantID, Name: "price-boundary-profile", Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)

	save := func(price string) *httptest.ResponseRecorder {
		body := fmt.Sprintf(`{"name":"Price Boundary","price":%s,"radius_profile_id":"%d"}`, price, profile.ID)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/isp/packages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		c.Set("tenant_id", domain.DefaultTenantID)
		// savePackage returns Echo's HTTPError for validator failures instead of
		// writing a response; record the status the default error handler would use.
		if err := savePackage(c, 0); err != nil {
			he, ok := err.(*echo.HTTPError)
			require.True(t, ok, "unexpected handler error: %v", err)
			rec.Code = he.Code
		}
		return rec
	}

	for _, tc := range []struct {
		name  string
		price string
		code  int
	}{
		{name: "negative is rejected", price: "-1", code: http.StatusBadRequest},
		{name: "above int64 is rejected", price: "9223372036854775808", code: http.StatusBadRequest},
		{name: "zero stays valid for free packages", price: "0", code: http.StatusCreated},
		{name: "max int64 is stored exactly", price: "9223372036854775807", code: http.StatusCreated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := save(tc.price)
			require.Equal(t, tc.code, rec.Code, rec.Body.String())
		})
	}

	var maxPackage domain.InternetPackage
	require.NoError(t, db.Where("price = ?", int64(math.MaxInt64)).First(&maxPackage).Error)
	assert.Equal(t, int64(math.MaxInt64), maxPackage.Price)

	var storedNegative int64
	require.NoError(t, db.Model(&domain.InternetPackage{}).Where("price < 0").Count(&storedNegative).Error)
	assert.Zero(t, storedNegative, "a negative price must never be persisted")

	var envelope struct {
		Data domain.InternetPackage `json:"data"`
	}
	rec := save("12345")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	assert.Equal(t, int64(12345), envelope.Data.Price)
}
