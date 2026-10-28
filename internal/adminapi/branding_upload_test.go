package adminapi

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type brandingUploadFixture struct {
	filename    string
	contentType string
	content     []byte
}

// performBrandingUpload sends one multipart branding save with the given logo
// part. The declared part content type and filename are the caller's: the
// handler is expected to decide on the bytes, not on those labels.
func performBrandingUpload(t *testing.T, db *gorm.DB, e *echo.Echo, appCtx app.AppContext, upload brandingUploadFixture) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{
		"product_name": "Branded ISP", "short_name": "BISP", "accent_color": "#28A86B",
	} {
		require.NoError(t, writer.WriteField(key, value))
	}
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="logo"; filename="` + upload.filename + `"`},
		"Content-Type":        {upload.contentType},
	})
	require.NoError(t, err)
	_, err = part.Write(upload.content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPut, "/api/v1/system/branding", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, saveBranding(c))
	return rec
}

func encodePNGFixture(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// pngHeaderOnly builds a PNG that stops after a well-formed IHDR, which is all
// png.DecodeConfig reads. It is the shape of a pixel bomb: a few dozen bytes
// that claim a canvas far larger than the policy allows.
func pngHeaderOnly(t *testing.T, width, height uint32) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'})
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], width)
	binary.BigEndian.PutUint32(ihdr[4:8], height)
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // truecolor with alpha
	buf.Write(pngChunk("IHDR", ihdr))
	return buf.Bytes()
}

func pngChunk(chunkType string, data []byte) []byte {
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data))) //nolint:gosec // G115: fixture chunks are a few bytes
	crc := crc32.NewIEEE()
	_, _ = crc.Write([]byte(chunkType))
	_, _ = crc.Write(data)
	checksum := make([]byte, 4)
	binary.BigEndian.PutUint32(checksum, crc.Sum32())
	out := append([]byte{}, length...)
	out = append(out, chunkType...)
	out = append(out, data...)
	return append(out, checksum...)
}

func brandingLogoFiles(t *testing.T, dataDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dataDir, "branding"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func requireBrandingRejected(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var payload ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, code, payload.Error, payload.Message)
}

func TestBrandingRejectsNonPNGContent(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	var gzipped bytes.Buffer
	zw := gzip.NewWriter(&gzipped)
	_, err := zw.Write(encodePNGFixture(t, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	cases := []struct {
		name   string
		upload brandingUploadFixture
	}{
		{
			name: "jpeg bytes named logo.png",
			upload: brandingUploadFixture{"logo.png", "image/png", []byte{
				0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01,
				0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
			}},
		},
		{
			name:   "svg declared as image/png",
			upload: brandingUploadFixture{"logo.png", "image/png", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		},
		{
			name:   "html declared as image/png",
			upload: brandingUploadFixture{"logo.png", "image/png", []byte("<html><body><script>alert(1)</script></body></html>")},
		},
		{
			name:   "gzipped png declared as image/png",
			upload: brandingUploadFixture{"logo.png", "image/png", gzipped.Bytes()},
		},
		{
			name:   "empty file",
			upload: brandingUploadFixture{"logo.png", "image/png", []byte{}},
		},
		{
			name:   "png signature only",
			upload: brandingUploadFixture{"logo.png", "image/png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performBrandingUpload(t, db, e, appCtx, tc.upload)
			requireBrandingRejected(t, rec, "INVALID_LOGO")
		})
	}

	var count int64
	require.NoError(t, db.Model(&domain.ProductBranding{}).Count(&count).Error)
	require.Zero(t, count)
	require.Empty(t, brandingLogoFiles(t, appCtx.Config().GetDataDir()))
}

func TestBrandingRejectsPixelBombHeaders(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	cases := []struct {
		name          string
		width, height uint32
	}{
		{name: "width over cap", width: 2049, height: 8},
		{name: "height over cap", width: 8, height: 2049},
		{name: "both over cap", width: 4096, height: 4096},
		{name: "absurd canvas", width: 100000, height: 100000},
		{name: "maximum uint32", width: 0xFFFFFFFF, height: 0xFFFFFFFF},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performBrandingUpload(t, db, e, appCtx, brandingUploadFixture{
				"bomb.png", "image/png", pngHeaderOnly(t, tc.width, tc.height),
			})
			requireBrandingRejected(t, rec, "INVALID_LOGO")
			// The refusal must come from the dimension policy, not from a parse
			// accident that a better-formed header could slip past.
			require.Contains(t, rec.Body.String(), "2048")
		})
	}

	var count int64
	require.NoError(t, db.Model(&domain.ProductBranding{}).Count(&count).Error)
	require.Zero(t, count)
	require.Empty(t, brandingLogoFiles(t, appCtx.Config().GetDataDir()))
}

func TestBrandingRejectsIncompletePNG(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	whole := encodePNGFixture(t, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	require.Greater(t, len(whole), 60)

	cases := []struct {
		name    string
		content []byte
	}{
		{name: "header only", content: pngHeaderOnly(t, 8, 8)},
		{name: "truncated image data", content: whole[:len(whole)-20]},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performBrandingUpload(t, db, e, appCtx, brandingUploadFixture{
				"logo.png", "image/png", tc.content,
			})
			requireBrandingRejected(t, rec, "INVALID_LOGO")
			require.Contains(t, rec.Body.String(), "complete")
		})
	}

	var count int64
	require.NoError(t, db.Model(&domain.ProductBranding{}).Count(&count).Error)
	require.Zero(t, count)
	require.Empty(t, brandingLogoFiles(t, appCtx.Config().GetDataDir()))
}

func TestBrandingRejectsOversizedLogoFile(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	// A decodable one-pixel PNG padded past the file cap: the size policy must
	// reject it even though the image itself is valid.
	padded := make([]byte, maxBrandLogoSize+1)
	copy(padded, encodePNGFixture(t, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	rec := performBrandingUpload(t, db, e, appCtx, brandingUploadFixture{"logo.png", "image/png", padded})
	requireBrandingRejected(t, rec, "INVALID_LOGO")
	require.Contains(t, rec.Body.String(), "1 MiB")

	// A body far above the request cap is refused before the logo reader runs.
	flood := make([]byte, 2<<20)
	copy(flood, pngHeaderOnly(t, 1, 1))
	rec = performBrandingUpload(t, db, e, appCtx, brandingUploadFixture{"logo.png", "image/png", flood})
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	var count int64
	require.NoError(t, db.Model(&domain.ProductBranding{}).Count(&count).Error)
	require.Zero(t, count)
	require.Empty(t, brandingLogoFiles(t, appCtx.Config().GetDataDir()))
}

func TestBrandingAcceptsEveryPNGEncoding(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	gray := image.NewGray(image.Rect(0, 0, 4, 4))
	for i := range gray.Pix {
		gray.Pix[i] = uint8(i * 16) //nolint:gosec // G115: bounded by the fixture's 16 pixels
	}
	paletted := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{
		color.RGBA{R: 0x16, G: 0xA3, B: 0x4A, A: 0xFF},
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	})
	for i := range paletted.Pix {
		paletted.Pix[i] = uint8(i % 2) //nolint:gosec // G115: palette index is 0 or 1
	}
	rgba := image.NewRGBA(image.Rect(0, 0, 4, 4))
	rgba.Set(1, 2, color.RGBA{R: 0xE6, G: 0xFF, B: 0x00, A: 0xFF})
	nrgba64 := image.NewNRGBA64(image.Rect(0, 0, 4, 4))
	nrgba64.Set(3, 3, color.NRGBA64{R: 0xFFFF, G: 0x8000, B: 0x0000, A: 0xFFFF})

	cases := []struct {
		name string
		img  image.Image
	}{
		{name: "grayscale", img: gray},
		{name: "paletted", img: paletted},
		{name: "rgba", img: rgba},
		{name: "nrgba64", img: nrgba64},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := encodePNGFixture(t, tc.img)
			rec := performBrandingUpload(t, db, e, appCtx, brandingUploadFixture{"logo.png", "image/png", content})
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var saved struct {
				Data brandingDTO `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))
			require.Contains(t, saved.Data.LogoURL, "/api/v1/public/branding/logo?v=")

			logoReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/branding/logo", nil)
			logoRec := httptest.NewRecorder()
			logoCtx := CreateTestContext(e, db, logoReq, logoRec, appCtx)
			require.NoError(t, getPublicBrandLogo(logoCtx))
			require.Equal(t, http.StatusOK, logoRec.Code)
			require.Equal(t, "image/png", logoRec.Header().Get(echo.HeaderContentType))
			require.Equal(t, "nosniff", logoRec.Header().Get(echo.HeaderXContentTypeOptions))
			require.Equal(t, content, logoRec.Body.Bytes())
		})
	}
}

// TestBrandingTrustsContentNotDeclaredType pins the deliberate behavior for a
// content-type mismatch in the other direction: a real PNG labeled image/svg+xml
// is stored, and the public response is always image/png with nosniff, so the
// client-declared type never becomes the served type.
func TestBrandingTrustsContentNotDeclaredType(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	content := encodePNGFixture(t, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	rec := performBrandingUpload(t, db, e, appCtx, brandingUploadFixture{"logo.svg", "image/svg+xml", content})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	logoReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/branding/logo", nil)
	logoRec := httptest.NewRecorder()
	logoCtx := CreateTestContext(e, db, logoReq, logoRec, appCtx)
	require.NoError(t, getPublicBrandLogo(logoCtx))
	require.Equal(t, http.StatusOK, logoRec.Code)
	require.Equal(t, "image/png", logoRec.Header().Get(echo.HeaderContentType))
	require.Equal(t, content, logoRec.Body.Bytes())
}

// TestBrandingRecordIsGlobalAndPlatformOwned pins the ownership model: the
// branding row is a single installation-wide record, a tenant-scoped request
// context still writes that one row, every tenant's public read sees it, and a
// tenant administrator cannot change it.
func TestBrandingRecordIsGlobalAndPlatformOwned(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	content := encodePNGFixture(t, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("product_name", "Branded ISP"))
	require.NoError(t, writer.WriteField("short_name", "BISP"))
	require.NoError(t, writer.WriteField("accent_color", "#28A86B"))
	part, err := writer.CreateFormFile("logo", "logo.png")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPut, "/api/v1/system/branding", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	tenantCtx := CreateTestContext(e, db, req, rec, appCtx)
	tenantCtx.Set("tenant_id", int64(7))
	require.NoError(t, saveBranding(tenantCtx))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var rows []domain.ProductBranding
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.EqualValues(t, brandingRecordID, rows[0].ID)

	// A public read without any tenant context still sees the same identity.
	publicRec := httptest.NewRecorder()
	publicCtx := CreateTestContext(e, db, httptest.NewRequest(http.MethodGet, "/api/v1/public/branding", nil), publicRec, appCtx)
	require.NoError(t, getPublicBranding(publicCtx))
	var public struct {
		Data brandingDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(publicRec.Body.Bytes(), &public))
	require.Equal(t, "Branded ISP", public.Data.ProductName)
	require.NotEmpty(t, public.Data.LogoURL)

	// A tenant administrator is not a platform administrator.
	deniedReq := httptest.NewRequest(http.MethodPut, "/api/v1/system/branding", bytes.NewBufferString(
		`{"product_name":"Tenant Override","short_name":"TNT","accent_color":"#000000"}`))
	deniedReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	deniedRec := httptest.NewRecorder()
	deniedCtx := CreateTestContext(e, db, deniedReq, deniedRec, appCtx)
	deniedCtx.Set("current_operator", &domain.SysOpr{ID: 2, Level: LevelAdmin, Status: "enabled"})
	require.NoError(t, requirePlatformAdmin()(saveBranding)(deniedCtx))
	require.Equal(t, http.StatusForbidden, deniedRec.Code)
	var denied ErrorResponse
	require.NoError(t, json.Unmarshal(deniedRec.Body.Bytes(), &denied))
	require.Equal(t, "PLATFORM_ACCESS_REQUIRED", denied.Error)

	var stored domain.ProductBranding
	require.NoError(t, db.First(&stored, brandingRecordID).Error)
	require.Equal(t, "Branded ISP", stored.ProductName)
}
