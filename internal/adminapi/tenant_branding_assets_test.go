package adminapi

import (
	"bytes"
	"encoding/json"
	"image"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTenantAssetUploadContext(t *testing.T, db *gorm.DB, e *echo.Echo, appCtx app.AppContext, tenantID int64, filename string, content []byte) echo.Context {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("asset", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/branding/assets", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.Set("tenant_id", tenantID)
	return c
}

func TestTenantBrandingAssetUploadServeAndSafeDelete(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	appCtx.Config().System.Workdir = t.TempDir()
	require.NoError(t, db.AutoMigrate(
		&domain.Tenant{}, &domain.TenantBranding{}, &domain.TenantWebsite{},
		&domain.TenantWebsiteVersion{}, &domain.TenantPackagePresentation{},
	))
	var tenant domain.Tenant
	require.NoError(t, db.Where("id = ?", domain.DefaultTenantID).First(&tenant).Error)

	pngBytes := encodePNGFixture(t, image.NewRGBA(image.Rect(0, 0, 3, 3)))
	uploadCtx := newTenantAssetUploadContext(t, db, e, appCtx, tenant.ID, "brand.png", pngBytes)
	uploadRec := uploadCtx.Response().Writer.(*httptest.ResponseRecorder)
	require.NoError(t, uploadTenantBrandingAsset(uploadCtx))
	require.Equal(t, http.StatusCreated, uploadRec.Code, uploadRec.Body.String())

	var uploaded struct {
		Data struct {
			FileName string `json:"file_name"`
			URL      string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(uploadRec.Body.Bytes(), &uploaded))
	require.Regexp(t, tenantBrandingAssetPattern, uploaded.Data.FileName)
	require.Equal(t, "/api/v1/tenant-branding/assets/"+strconv.FormatInt(tenant.ID, 10)+"/"+uploaded.Data.FileName, uploaded.Data.URL)

	assetPath := filepath.Join(appCtx.Config().GetDataDir(), "tenant-branding", strconv.FormatInt(tenant.ID, 10), uploaded.Data.FileName)
	require.FileExists(t, assetPath)

	getReq := httptest.NewRequest(http.MethodGet, uploaded.Data.URL, nil)
	getRec := httptest.NewRecorder()
	getCtx := CreateTestContext(e, db, getReq, getRec, appCtx)
	getCtx.SetParamNames("tenant", "filename")
	getCtx.SetParamValues(strconv.FormatInt(tenant.ID, 10), uploaded.Data.FileName)
	require.NoError(t, getPublicTenantBrandingAsset(getCtx))
	require.Equal(t, http.StatusOK, getRec.Code)
	require.Equal(t, "image/png", getRec.Header().Get(echo.HeaderContentType))
	require.Equal(t, "nosniff", getRec.Header().Get(echo.HeaderXContentTypeOptions))
	require.Equal(t, pngBytes, getRec.Body.Bytes())

	brand := domain.TenantBranding{TenantID: tenant.ID, LogoURL: uploaded.Data.URL}
	require.NoError(t, db.Create(&brand).Error)
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/branding/assets/"+uploaded.Data.FileName, nil)
	deleteRec := httptest.NewRecorder()
	deleteCtx := CreateTestContext(e, db, deleteReq, deleteRec, appCtx)
	deleteCtx.Set("tenant_id", tenant.ID)
	deleteCtx.SetParamNames("filename")
	deleteCtx.SetParamValues(uploaded.Data.FileName)
	require.NoError(t, deleteTenantBrandingAsset(deleteCtx))
	require.Equal(t, http.StatusConflict, deleteRec.Code, deleteRec.Body.String())
	require.FileExists(t, assetPath)

	require.NoError(t, db.Model(&brand).Update("logo_url", "").Error)
	deleteReq = httptest.NewRequest(http.MethodDelete, "/api/v1/branding/assets/"+uploaded.Data.FileName, nil)
	deleteRec = httptest.NewRecorder()
	deleteCtx = CreateTestContext(e, db, deleteReq, deleteRec, appCtx)
	deleteCtx.Set("tenant_id", tenant.ID)
	deleteCtx.SetParamNames("filename")
	deleteCtx.SetParamValues(uploaded.Data.FileName)
	require.NoError(t, deleteTenantBrandingAsset(deleteCtx))
	require.Equal(t, http.StatusOK, deleteRec.Code, deleteRec.Body.String())
	_, statErr := os.Stat(assetPath)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestTenantBrandingAssetUploadRejectsInvalidPNG(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	appCtx.Config().System.Workdir = t.TempDir()
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}))
	uploadCtx := newTenantAssetUploadContext(t, db, e, appCtx, domain.DefaultTenantID,
		"script.png", []byte(`<svg><script>alert(1)</script></svg>`))
	rec := uploadCtx.Response().Writer.(*httptest.ResponseRecorder)
	require.NoError(t, uploadTenantBrandingAsset(uploadCtx))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assetDir := filepath.Join(appCtx.Config().GetDataDir(), "tenant-branding", strconv.FormatInt(domain.DefaultTenantID, 10))
	entries, err := os.ReadDir(assetDir)
	if !os.IsNotExist(err) {
		require.NoError(t, err)
		require.Empty(t, entries)
	}
}
