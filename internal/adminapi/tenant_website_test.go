package adminapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantWebsiteDraftPublishVersionAndRollbackAreTenantScoped(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}, &domain.TenantWebsite{}, &domain.TenantWebsiteVersion{}))
	require.NoError(t, db.Create(&domain.Tenant{ID: 21, Name: "Website One", Slug: "website-one", Kind: "isp", Status: "active"}).Error)
	require.NoError(t, db.Create(&domain.Tenant{ID: 22, Name: "Website Two", Slug: "website-two", Kind: "isp", Status: "active"}).Error)

	firstDraft := `{"sections":[{"key":"hero","title":"Website One","subtitle":"Welcome","body":"Tenant one public content","visible":true,"order":1,"cta_label":"Contact","cta_url":"https://one.example.test"}]}`
	c, recorder := newTenantRequestContext(t, e, db, appCtx, http.MethodPut,
		"/api/v1/tenant-website/draft", firstDraft, 21, nil)
	require.NoError(t, saveTenantWebsiteDraft(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())

	// Drafts never appear in public output before an explicit publish.
	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/tenant-website/public/website-one", "", 21, map[string]string{"identifier":"website-one"})
	require.NoError(t, getPublicTenantWebsite(c))
	require.Equal(t, http.StatusNotFound, recorder.Code)

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPost,
		"/api/v1/tenant-website/publish", "", 21, nil)
	require.NoError(t, publishTenantWebsite(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/tenant-website/public/website-one", "", 21, map[string]string{"identifier":"website-one"})
	require.NoError(t, getPublicTenantWebsite(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	var public struct { Data tenantWebsiteContent `json:"data"` }
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &public))
	require.Len(t, public.Data.Sections, 1)
	assert.Equal(t, "Tenant one public content", public.Data.Sections[0].Body)

	secondDraft := `{"sections":[{"key":"hero","title":"Changed","body":"Tenant one second version","visible":true,"order":1}]}`
	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPut,
		"/api/v1/tenant-website/draft", secondDraft, 21, nil)
	require.NoError(t, saveTenantWebsiteDraft(c))
	require.Equal(t, http.StatusOK, recorder.Code)
	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPost,
		"/api/v1/tenant-website/publish", "", 21, nil)
	require.NoError(t, publishTenantWebsite(c))
	require.Equal(t, http.StatusOK, recorder.Code)

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPost,
		"/api/v1/tenant-website/versions/1/rollback", "", 21, map[string]string{"version":strconv.Itoa(1)})
	require.NoError(t, rollbackTenantWebsiteVersion(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/tenant-website/public/website-one", "", 21, map[string]string{"identifier":"website-one"})
	require.NoError(t, getPublicTenantWebsite(c))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &public))
	assert.Equal(t, "Tenant one public content", public.Data.Sections[0].Body, "rollback republishes the selected immutable version")

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/tenant-website/versions", "", 22, nil)
	require.NoError(t, listTenantWebsiteVersions(c))
	require.Equal(t, http.StatusOK, recorder.Code)
	var otherVersions struct { Data []tenantWebsiteVersionResponse `json:"data"` }
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &otherVersions))
	assert.Empty(t, otherVersions.Data, "another tenant cannot see website version history")
}

func TestTenantWebsiteRejectsUnsafeLinksAndUnknownSections(t *testing.T) {
	_, err := decodeTenantWebsite(`{"sections":[{"key":"hero","title":"x","cta_url":"javascript:alert(1)"}]}`)
	require.Error(t, err)
	_, err = decodeTenantWebsite(`{"sections":[{"key":"custom-script","body":"no"}]}`)
	require.Error(t, err)
	_, err = decodeTenantWebsite(`{"sections":[{"key":"hero","body":"ok"},{"key":"hero","body":"duplicate"}]}`)
	require.Error(t, err)
}
