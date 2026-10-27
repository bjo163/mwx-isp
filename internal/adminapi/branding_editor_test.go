package adminapi

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/domain"
)

// TestBrandingRouteGuardAllowsOnlyPlatformAdmin drives the registered routes so
// the assertions cover the real middleware chain: anonymous reads of the
// public branding stay open, every mutation is refused for operators, plain
// admins, and super operators without the platform flag, and a platform admin
// can save, read the live state, and reset.
func TestBrandingRouteGuardAllowsOnlyPlatformAdmin(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	operator := seedMatrixOperator(t, db, 9201, LevelOperator, "branding-operator")
	admin := seedMatrixOperator(t, db, 9202, LevelAdmin, "branding-admin")
	super := seedMatrixOperator(t, db, 9203, LevelSuper, "branding-super")
	platform := seedMatrixOperator(t, db, 9204, LevelSuper, "branding-platform")
	require.NoError(t, db.Model(&domain.SysOpr{}).Where("id = ?", platform.ID).Update("platform_admin", true).Error)
	platform.PlatformAdmin = true

	payload := map[string]any{
		"product_name": "Guard ISP", "short_name": "GUARD", "tagline": "Route guard",
		"accent_color": "#112233", "hero_headline": "Guard headline",
	}

	for _, tc := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "anonymous", token: "", want: http.StatusUnauthorized},
		{name: "operator", token: matrixToken(t, appCtx, operator), want: http.StatusForbidden},
		{name: "admin", token: matrixToken(t, appCtx, admin), want: http.StatusForbidden},
		{name: "super without platform flag", token: matrixToken(t, appCtx, super), want: http.StatusForbidden},
		{name: "super platform admin", token: matrixToken(t, appCtx, platform), want: http.StatusOK},
	} {
		t.Run(tc.name+" cannot publish", func(t *testing.T) {
			status, body := matrixRequest(t, handler, http.MethodPut, "/api/v1/system/branding", tc.token, payload)
			assert.Equalf(t, tc.want, status, "%s", body)
		})
	}

	var stored int64
	require.NoError(t, db.Model(&domain.ProductBranding{}).Count(&stored).Error)
	require.Equal(t, int64(1), stored, "only the platform admin save may persist a row")

	// The admin read is the read-only preview of the live record; it must be
	// closed to everyone except a platform admin.
	for _, tc := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "operator", token: matrixToken(t, appCtx, operator), want: http.StatusForbidden},
		{name: "admin", token: matrixToken(t, appCtx, admin), want: http.StatusForbidden},
		{name: "platform admin", token: matrixToken(t, appCtx, platform), want: http.StatusOK},
	} {
		t.Run(tc.name+" preview", func(t *testing.T) {
			status, body := matrixRequest(t, handler, http.MethodGet, "/api/v1/system/branding", tc.token, nil)
			assert.Equalf(t, tc.want, status, "%s", body)
		})
	}

	// Anonymous visitors keep the public read, and it is the same live record.
	status, body := anonymousRequest(t, handler, http.MethodGet, "/api/v1/public/branding", nil)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	var publicDTO struct {
		Data brandingDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &publicDTO))
	assert.Equal(t, "Guard ISP", publicDTO.Data.ProductName)

	// Reset is the only rollback path and is platform-admin only.
	for _, tc := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "operator", token: matrixToken(t, appCtx, operator), want: http.StatusForbidden},
		{name: "admin", token: matrixToken(t, appCtx, admin), want: http.StatusForbidden},
	} {
		t.Run(tc.name+" cannot reset", func(t *testing.T) {
			status, body := matrixRequest(t, handler, http.MethodPost, "/api/v1/system/branding/reset", tc.token, nil)
			assert.Equalf(t, tc.want, status, "%s", body)
		})
	}
	status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/system/branding/reset", matrixToken(t, appCtx, platform), nil)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	var resetDTO struct {
		Data brandingDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resetDTO))
	assert.Equal(t, defaultBranding().ProductName, resetDTO.Data.ProductName)
	require.NoError(t, db.Model(&domain.ProductBranding{}).Count(&stored).Error)
	assert.Zero(t, stored, "reset removes the override so the compiled defaults return")
}

// TestBrandingPublishKeepsMarkupInert proves a published field cannot smuggle
// executable markup to the landing page: the value is stored as plain text,
// the public response is JSON with "<" escaped, and the decoded value matches
// the operator's text exactly. Reset then rolls the installation back to the
// compiled defaults.
func TestBrandingPublishKeepsMarkupInert(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)
	require.NoError(t, db.AutoMigrate(&domain.ProductBranding{}))
	appCtx.Config().System.Workdir = t.TempDir()

	platform := seedMatrixOperator(t, db, 9205, LevelSuper, "branding-script")
	require.NoError(t, db.Model(&domain.SysOpr{}).Where("id = ?", platform.ID).Update("platform_admin", true).Error)
	platform.PlatformAdmin = true
	token := matrixToken(t, appCtx, platform)

	hostile := `<script>alert("x")</script><img src=x onerror=alert(1)>`
	payload := map[string]any{
		"product_name": "Inert ISP", "short_name": "INERT", "tagline": "No markup",
		"accent_color": "#445566", "hero_headline": hostile,
		"hero_subtitle": hostile, "ticker_text": hostile,
	}
	status, body := matrixRequest(t, handler, http.MethodPut, "/api/v1/system/branding", token, payload)
	require.Equalf(t, http.StatusOK, status, "%s", body)

	var stored domain.ProductBranding
	require.NoError(t, db.First(&stored, brandingRecordID).Error)
	assert.Equal(t, hostile, stored.HeroHeadline, "the editor stores text, not filtered or rendered HTML")

	status, body = anonymousRequest(t, handler, http.MethodGet, "/api/v1/public/branding", nil)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	assert.NotContains(t, string(body), "<script", "the public payload must not contain a raw script element")
	var publicDTO struct {
		Data brandingDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &publicDTO))
	assert.Equal(t, hostile, publicDTO.Data.HeroHeadline, "text round-trips unchanged and stays inert")
	assert.Equal(t, hostile, publicDTO.Data.HeroSubtitle)
	assert.Equal(t, hostile, publicDTO.Data.TickerText)

	status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/system/branding/reset", token, nil)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	status, body = anonymousRequest(t, handler, http.MethodGet, "/api/v1/public/branding", nil)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	require.NoError(t, json.Unmarshal(body, &publicDTO))
	assert.Equal(t, defaultBranding().HeroHeadline, publicDTO.Data.HeroHeadline, "reset restores the compiled default")
	assert.False(t, strings.Contains(string(body), "onerror=alert"), "rollback leaves no hostile text behind")
}

// TestBrandingEditorSourceRendersTextOnly keeps the frontend half of the
// publish guarantee: branding fields are rendered as React text children, so
// any future dangerouslySetInnerHTML use would turn stored text into markup.
func TestBrandingEditorSourceRendersTextOnly(t *testing.T) {
	root := "../../web/src"
	var offenders []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".tsx") {
			return nil
		}
		content, err := os.ReadFile(path) //nolint:gosec // fixed test root inside the repository.
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "dangerouslySetInnerHTML") {
			offenders = append(offenders, path)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, offenders, "branding text must never be injected as HTML")
}
