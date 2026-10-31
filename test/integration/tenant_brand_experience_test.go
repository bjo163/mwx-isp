//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tenantActor is one tenant plus a JWT scoped to its own admin membership, so
// every request below travels the production login, tenant-context and
// authorization middleware instead of calling handlers with a forged context.
type tenantActor struct {
	tenant  domain.Tenant
	client  *apiClient
	adminID int64
}

// provisionTenantActor creates a tenant, grants it a tenant-local admin through
// the platform API, and signs that admin in against the tenant slug.
func provisionTenantActor(t *testing.T, platform *apiClient, label string) tenantActor {
	t.Helper()
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	slug := fmt.Sprintf("it-%s-%s", label, suffix)
	tenant := domain.Tenant{
		Name: label + " tenant", Slug: slug, Subdomain: slug,
		Kind: "isp", Status: "active",
	}
	require.NoError(t, db.Create(&tenant).Error)

	// Platform usernames are capped at 30 chars, so keep only the tail of the
	// nanosecond suffix; it still distinguishes tenants within one run.
	username := "adm-" + suffix[len(suffix)-10:] + "-" + label
	password := "Strong-brand-2026"
	payload, err := json.Marshal(map[string]string{"username": username, "password": password, "level": "admin"})
	require.NoError(t, err)
	status, body := platform.post(t, "/api/v1/platform/tenants/"+fmt.Sprint(tenant.ID)+"/operators", payload)
	require.Equalf(t, http.StatusCreated, status, "membership grant for %s: %s", label, body)

	var operator domain.SysOpr
	unwrapData(t, body, &operator)
	require.NotZero(t, operator.ID, "granted operator must persist for %s", label)

	token, err := loginTenantToken(h.webBaseURL, tenant.Slug, username, password)
	require.NoErrorf(t, err, "tenant admin login for %s must succeed", label)
	return tenantActor{tenant: tenant, client: &apiClient{base: h.webBaseURL, http: &http.Client{}, token: token}, adminID: operator.ID}
}

// anonymousGet issues a request with no bearer token, optionally overriding the
// Host header so the subdomain tenant resolver can select a tenant the way a
// public browser would after DNS delegation.
func anonymousGet(t *testing.T, path, host string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, h.webBaseURL+path, nil)
	require.NoError(t, err)
	if host != "" {
		req.Host = host
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, data
}

// publicHost builds a managed-subdomain hostname for a tenant slug.
func publicHost(actor tenantActor) string {
	return actor.tenant.Subdomain + ".e-isp.id"
}

// TestPostgresTenantBrandingAndWebsiteStayIsolated drives the tenant brand and
// website surfaces end to end against PostgreSQL with two real tenant admins.
// It proves that a write by one tenant never surfaces in the other tenant's
// public projection, that drafts do not leak before publish, and that rollback
// republishes history without rewriting the neighbour's site.
func TestPostgresTenantBrandingAndWebsiteStayIsolated(t *testing.T) {
	platform := newAPIClient(t)
	alpha := provisionTenantActor(t, platform, "alpha")
	beta := provisionTenantActor(t, platform, "beta")

	// --- Branding: each tenant owns exactly its own identity record. ---
	alphaBrand, err := json.Marshal(map[string]string{
		"brand_name": "Alpha Fiber", "short_name": "ALFA", "primary_color": "#16A34A",
		"secondary_color": "#0F172A", "tagline": "Internet kampung cepat",
	})
	require.NoError(t, err)
	status, body := alpha.client.put(t, "/api/v1/branding", alphaBrand)
	require.Equalf(t, http.StatusOK, status, "alpha branding update: %s", body)

	betaBrand, err := json.Marshal(map[string]string{
		"brand_name": "Beta Net", "primary_color": "#2563EB", "tagline": "Stabil sepanjang hari",
	})
	require.NoError(t, err)
	status, body = beta.client.put(t, "/api/v1/branding", betaBrand)
	require.Equalf(t, http.StatusOK, status, "beta branding update: %s", body)

	for _, tc := range []struct {
		actor tenantActor
		own   string
		other string
	}{{alpha, "Alpha Fiber", "Beta Net"}, {beta, "Beta Net", "Alpha Fiber"}} {
		status, body = tc.actor.client.get(t, "/api/v1/branding")
		require.Equalf(t, http.StatusOK, status, "branding read for %s: %s", tc.actor.tenant.Slug, body)
		require.Contains(t, string(body), tc.own)
		require.NotContains(t, string(body), tc.other, "tenant branding must never expose another tenant's brand")

		status, body = anonymousGet(t, "/api/v1/tenant-branding/"+tc.actor.tenant.Slug, "")
		require.Equalf(t, http.StatusOK, status, "public branding for %s: %s", tc.actor.tenant.Slug, body)
		require.Contains(t, string(body), tc.own)
		require.NotContains(t, string(body), tc.other)
	}

	// An invalid brand write is rejected and leaves the stored brand untouched.
	invalid, err := json.Marshal(map[string]string{"primary_color": "green"})
	require.NoError(t, err)
	status, body = alpha.client.put(t, "/api/v1/branding", invalid)
	require.Equalf(t, http.StatusBadRequest, status, "invalid branding must be refused: %s", body)
	require.Contains(t, string(body), "INVALID_BRANDING")
	status, body = anonymousGet(t, "/api/v1/tenant-branding/"+alpha.tenant.Slug, "")
	require.Equalf(t, http.StatusOK, status, "alpha public branding after rejected write: %s", body)
	require.Contains(t, string(body), "Alpha Fiber")

	// --- Website: draft, publish, isolation and rollback. ---
	writeDraft := func(actor tenantActor, title string) {
		t.Helper()
		payload, err := json.Marshal(map[string]any{
			"sections": []map[string]any{{
				"key": "hero", "title": title, "subtitle": "Gagal konek? Hubungi kami.",
				"visible": true, "order": 0,
			}},
		})
		require.NoError(t, err)
		status, body := actor.client.put(t, "/api/v1/tenant-website/draft", payload)
		require.Equalf(t, http.StatusOK, status, "draft save for %s: %s", actor.tenant.Slug, body)
	}
	publish := func(actor tenantActor) int {
		t.Helper()
		status, body := actor.client.post(t, "/api/v1/tenant-website/publish", nil)
		require.Equalf(t, http.StatusOK, status, "publish for %s: %s", actor.tenant.Slug, body)
		var draft struct {
			PublishedVersion int `json:"published_version"`
		}
		unwrapData(t, body, &draft)
		return draft.PublishedVersion
	}

	writeDraft(alpha, "Alpha Internet Cepat")
	require.Equal(t, 1, publish(alpha))
	writeDraft(beta, "Beta Internet Stabil")
	require.Equal(t, 1, publish(beta))

	for _, tc := range []struct {
		actor tenantActor
		own   string
		other string
	}{{alpha, "Alpha Internet Cepat", "Beta Internet Stabil"}, {beta, "Beta Internet Stabil", "Alpha Internet Cepat"}} {
		status, body = anonymousGet(t, "/api/v1/tenant-website/public/"+tc.actor.tenant.Slug, "")
		require.Equalf(t, http.StatusOK, status, "public website for %s: %s", tc.actor.tenant.Slug, body)
		require.Contains(t, string(body), tc.own)
		require.NotContains(t, string(body), tc.other)
	}

	// A saved draft stays private until publish; the public site keeps the old
	// version, and the neighbour's site is untouched by alpha's edit.
	writeDraft(alpha, "Alpha Internet Cepat v2")
	status, body = anonymousGet(t, "/api/v1/tenant-website/public/"+alpha.tenant.Slug, "")
	require.Equalf(t, http.StatusOK, status, "public alpha site after draft save: %s", body)
	require.Contains(t, string(body), "Alpha Internet Cepat")
	require.NotContains(t, string(body), "v2", "an unpublished draft must not reach the public projection")
	status, body = anonymousGet(t, "/api/v1/tenant-website/public/"+beta.tenant.Slug, "")
	require.Equalf(t, http.StatusOK, status, "public beta site after alpha draft: %s", body)
	require.Contains(t, string(body), "Beta Internet Stabil")

	require.Equal(t, 2, publish(alpha))
	status, body = anonymousGet(t, "/api/v1/tenant-website/public/"+alpha.tenant.Slug, "")
	require.Equalf(t, http.StatusOK, status, "public alpha site after publish: %s", body)
	require.Contains(t, string(body), "Alpha Internet Cepat v2")

	var alphaVersions, betaVersions []struct {
		Version int `json:"version"`
	}
	status, body = alpha.client.get(t, "/api/v1/tenant-website/versions")
	require.Equalf(t, http.StatusOK, status, "alpha versions: %s", body)
	unwrapData(t, body, &alphaVersions)
	require.Len(t, alphaVersions, 2, "alpha must keep both published snapshots")
	status, body = beta.client.get(t, "/api/v1/tenant-website/versions")
	require.Equalf(t, http.StatusOK, status, "beta versions: %s", body)
	unwrapData(t, body, &betaVersions)
	require.Len(t, betaVersions, 1, "alpha's publishes must not appear in beta's history")

	status, body = alpha.client.post(t, "/api/v1/tenant-website/versions/1/rollback", nil)
	require.Equalf(t, http.StatusOK, status, "alpha rollback to version 1: %s", body)
	status, body = anonymousGet(t, "/api/v1/tenant-website/public/"+alpha.tenant.Slug, "")
	require.Equalf(t, http.StatusOK, status, "public alpha site after rollback: %s", body)
	require.Contains(t, string(body), "Alpha Internet Cepat")
	require.NotContains(t, string(body), "v2")
	status, body = anonymousGet(t, "/api/v1/tenant-website/public/"+beta.tenant.Slug, "")
	require.Equalf(t, http.StatusOK, status, "public beta site after alpha rollback: %s", body)
	require.Contains(t, string(body), "Beta Internet Stabil")
	require.NotContains(t, string(body), "Alpha Internet")

	// Rollback history is append-only: the restored content is published as a
	// new version rather than deleting the superseded snapshot.
	status, body = alpha.client.get(t, "/api/v1/tenant-website/versions")
	require.Equalf(t, http.StatusOK, status, "alpha versions after rollback: %s", body)
	unwrapData(t, body, &alphaVersions)
	require.Len(t, alphaVersions, 3, "rollback must add a version, not overwrite history")
}

// TestPostgresTenantPackageCatalogIsolatesPresentation proves the public package
// catalog resolves its tenant from the verified hostname, honours per-tenant
// visibility, and that presentation edits never move price, speed or FUP.
func TestPostgresTenantPackageCatalogIsolatesPresentation(t *testing.T) {
	db := h.appCtx.DB()
	platform := newAPIClient(t)
	alpha := provisionTenantActor(t, platform, "catalog-alpha")
	beta := provisionTenantActor(t, platform, "catalog-beta")

	seedProfileForTenant := func(actor tenantActor, name string, up, down int) int64 {
		t.Helper()
		tenantDB := db.WithContext(tenancy.WithTenantID(t.Context(), actor.tenant.ID))
		profile := domain.RadiusProfile{
			ID: common.UUIDint64(), Name: name, Status: common.ENABLED,
			UpRate: up, DownRate: down,
		}
		require.NoError(t, tenantDB.Create(&profile).Error)
		return profile.ID
	}
	alphaProfile := seedProfileForTenant(alpha, "Alpha 100M", 20000, 100000)
	betaProfile := seedProfileForTenant(beta, "Beta 50M", 10000, 50000)

	createPackage := func(actor tenantActor, name string, profileID, price int64) int64 {
		t.Helper()
		payload, err := json.Marshal(map[string]any{
			"name": name, "price": price, "status": "active",
			"radius_profile_id": fmt.Sprint(profileID),
			"description":       "paket uji acceptance",
		})
		require.NoError(t, err)
		status, body := actor.client.post(t, "/api/v1/isp/packages", payload)
		require.Equalf(t, http.StatusCreated, status, "package create for %s: %s", actor.tenant.Slug, body)
		var pkg domain.InternetPackage
		unwrapData(t, body, &pkg)
		require.NotZero(t, pkg.ID)
		return pkg.ID
	}
	alphaPackage := createPackage(alpha, "Alpha Paket 100 Mbps", alphaProfile, 250000)
	betaPackage := createPackage(beta, "Beta Paket 50 Mbps", betaProfile, 150000)

	writePresentation := func(actor tenantActor, packageID int64, payload map[string]any) {
		t.Helper()
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		status, body := actor.client.put(t, fmt.Sprintf("/api/v1/isp/packages/%d/presentation", packageID), raw)
		require.Equalf(t, http.StatusOK, status, "presentation save for %s: %s", actor.tenant.Slug, body)
	}
	writePresentation(alpha, alphaPackage, map[string]any{
		"display_name": "Alpha Gold 100", "category": "Home Broadband",
		"badge": "Best Seller", "display_order": 5,
		"highlights": "Tanpa FUP\nSupport 24 jam", "public_visible": true,
	})
	writePresentation(beta, betaPackage, map[string]any{
		"display_name": "Beta Hidden 50", "public_visible": false,
	})

	// The public catalog resolves the tenant from the managed subdomain and only
	// emits the active, public-visible packages of that tenant.
	status, body := anonymousGet(t, "/api/v1/public/packages", publicHost(alpha))
	require.Equalf(t, http.StatusOK, status, "public alpha catalog: %s", body)
	var alphaCatalog []struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Price        int64    `json:"price"`
		DownRateKbps int      `json:"down_rate_kbps"`
		SpeedDisplay string   `json:"speed_display"`
		Category     string   `json:"category"`
		Highlights   []string `json:"highlights"`
		FupLimitGB   int64    `json:"fup_limit_gb"`
		BillingCycle string   `json:"billing_cycle"`
	}
	unwrapData(t, body, &alphaCatalog)
	require.Len(t, alphaCatalog, 1, "alpha catalog must expose exactly its own public package: %s", body)
	assert.Equal(t, fmt.Sprint(alphaPackage), alphaCatalog[0].ID)
	assert.Equal(t, "Alpha Gold 100", alphaCatalog[0].Name)
	assert.EqualValues(t, 250000, alphaCatalog[0].Price, "presentation must not move the operational price")
	assert.Equal(t, 100000, alphaCatalog[0].DownRateKbps, "public speed must come from the tenant profile")
	assert.Equal(t, "100 Mbps", alphaCatalog[0].SpeedDisplay)
	assert.Equal(t, "Home Broadband", alphaCatalog[0].Category)
	assert.Equal(t, []string{"Tanpa FUP", "Support 24 jam"}, alphaCatalog[0].Highlights)
	assert.NotContains(t, string(body), "Beta")

	status, body = anonymousGet(t, "/api/v1/public/packages", publicHost(beta))
	require.Equalf(t, http.StatusOK, status, "public beta catalog: %s", body)
	var betaCatalog []map[string]any
	unwrapData(t, body, &betaCatalog)
	assert.Empty(t, betaCatalog, "a package hidden from the public catalog must not be emitted: %s", body)
	assert.NotContains(t, string(body), "Alpha")

	// A presentation write may not reach another tenant's package.
	raw, err := json.Marshal(map[string]any{"display_name": "Cross tenant overwrite"})
	require.NoError(t, err)
	status, body = alpha.client.put(t, fmt.Sprintf("/api/v1/isp/packages/%d/presentation", betaPackage), raw)
	require.Equalf(t, http.StatusNotFound, status, "cross-tenant presentation write must be hidden: %s", body)
	var betaPresentation domain.TenantPackagePresentation
	require.NoError(t, db.Where("tenant_id = ? AND package_id = ?", beta.tenant.ID, betaPackage).First(&betaPresentation).Error)
	assert.Equal(t, "Beta Hidden 50", betaPresentation.DisplayName, "cross-tenant write must leave the row unchanged")

	// Presentation payloads carry marketing fields only; an attempt to send a
	// price must be ignored and the operational truth must stay put.
	writePresentation(alpha, alphaPackage, map[string]any{"display_name": "Alpha Gold 100", "price": 1})
	status, body = alpha.client.get(t, fmt.Sprintf("/api/v1/isp/packages/%d", alphaPackage))
	require.Equalf(t, http.StatusOK, status, "package read after presentation write: %s", body)
	var operational domain.InternetPackage
	unwrapData(t, body, &operational)
	assert.EqualValues(t, 250000, operational.Price, "presentation must never mutate the package price")
	assert.EqualValues(t, alphaProfile, operational.RadiusProfileID, "presentation must never reassign the RADIUS profile")
	status, body = anonymousGet(t, "/api/v1/public/packages", publicHost(alpha))
	require.Equalf(t, http.StatusOK, status, "public alpha catalog after price attempt: %s", body)
	unwrapData(t, body, &alphaCatalog)
	require.Len(t, alphaCatalog, 1)
	assert.EqualValues(t, 250000, alphaCatalog[0].Price)

	// Presentation listings are tenant-scoped rather than joined by package id.
	status, body = alpha.client.get(t, "/api/v1/isp/package-presentations")
	require.Equalf(t, http.StatusOK, status, "alpha presentation list: %s", body)
	var alphaPresentations []domain.TenantPackagePresentation
	unwrapData(t, body, &alphaPresentations)
	require.Len(t, alphaPresentations, 1)
	assert.Equal(t, alpha.tenant.ID, alphaPresentations[0].TenantID)
	assert.Equal(t, alphaPackage, alphaPresentations[0].PackageID)
}
