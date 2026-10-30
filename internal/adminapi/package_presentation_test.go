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

func TestPackagePresentationIsTenantScopedAndOperationalValuesStayAuthoritative(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Tenant{}, &domain.RadiusProfile{}, &domain.InternetPackage{}, &domain.TenantPackagePresentation{},
	))
	require.NoError(t, db.Create(&domain.Tenant{ID: domain.DefaultTenantID, Name: "Tenant One", Slug: "tenant-one", Kind: "isp", Status: "active"}).Error)
	require.NoError(t, db.Create(&domain.Tenant{ID: 2, Name: "Tenant Two", Slug: "tenant-two", Kind: "isp", Status: "active"}).Error)

	profileOne := domain.RadiusProfile{TenantID: domain.DefaultTenantID, Name: "profile-one", UpRate: 100000, DownRate: 100000, Status: "enabled"}
	profileTwo := domain.RadiusProfile{TenantID: 2, Name: "profile-two", UpRate: 200000, DownRate: 200000, Status: "enabled"}
	require.NoError(t, db.Create(&profileOne).Error)
	require.NoError(t, db.Create(&profileTwo).Error)
	packageOne := domain.InternetPackage{TenantID: domain.DefaultTenantID, Code: "PKG-ONE", Name: "Operational One", Price: 150000, RadiusProfileID: profileOne.ID, FupLimitGB: 100, Status: "active"}
	packageTwo := domain.InternetPackage{TenantID: 2, Code: "PKG-TWO", Name: "Operational Two", Price: 250000, RadiusProfileID: profileTwo.ID, FupLimitGB: 200, Status: "active"}
	require.NoError(t, db.Create(&packageOne).Error)
	require.NoError(t, db.Create(&packageTwo).Error)

	oneID := strconv.FormatInt(packageOne.ID, 10)
	c, recorder := newTenantRequestContext(t, e, db, appCtx, http.MethodPut,
		"/api/v1/isp/packages/"+oneID+"/presentation",
		`{"display_name":"Family Fibre 100","category":"Home","badge":"Popular","featured":true,"public_visible":false,"price":1}`,
		domain.DefaultTenantID, map[string]string{"id": oneID})
	require.NoError(t, updatePackagePresentation(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())

	var stored domain.InternetPackage
	require.NoError(t, db.Where("tenant_id = ? AND id = ?", domain.DefaultTenantID, packageOne.ID).First(&stored).Error)
	assert.EqualValues(t, 150000, stored.Price, "presentation metadata cannot alter operational price")
	assert.Equal(t, profileOne.ID, stored.RadiusProfileID, "presentation cannot change RADIUS profile")
	assert.EqualValues(t, 100, stored.FupLimitGB, "presentation cannot change FUP")

	twoID := strconv.FormatInt(packageTwo.ID, 10)
	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPut,
		"/api/v1/isp/packages/"+twoID+"/presentation",
		`{"display_name":"Cross Tenant Attempt","public_visible":true}`,
		domain.DefaultTenantID, map[string]string{"id": twoID})
	require.NoError(t, updatePackagePresentation(c))
	require.Equalf(t, http.StatusNotFound, recorder.Code, "%s", recorder.Body.String())

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/public/packages", "", domain.DefaultTenantID, nil)
	require.NoError(t, listPublicPackages(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	var publicOne struct { Data []publicPackageDTO `json:"data"` }
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &publicOne))
	assert.Empty(t, publicOne.Data, "a package hidden by its tenant must not appear in the public catalog")

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet,
		"/api/v1/public/packages", "", 2, nil)
	require.NoError(t, listPublicPackages(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	var publicTwo struct { Data []publicPackageDTO `json:"data"` }
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &publicTwo))
	require.Len(t, publicTwo.Data, 1)
	assert.Equal(t, "Operational Two", publicTwo.Data[0].Name)
	assert.EqualValues(t, 250000, publicTwo.Data[0].Price)
}
