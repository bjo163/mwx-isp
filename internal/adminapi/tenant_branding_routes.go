package adminapi

import (
	"net/http"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/webserver"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func registerTenantBrandingRoutes() {
	webserver.ApiGET("/tenant-branding/:identifier", getTenantBrandingPublicHandler)
}

type tenantBrandingPublicResponse struct {
	LogoURL        string `json:"logo_url"`
	FaviconURL     string `json:"favicon_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	Theme          string `json:"theme"`
	CompanyName    string `json:"company_name"`
	Tagline        string `json:"tagline"`
	WebsiteURL     string `json:"website_url"`
	SupportEmail   string `json:"support_email"`
	SupportPhone   string `json:"support_phone"`
}

func getTenantBrandingPublicHandler(c echo.Context) error {
	identifier := c.Param("identifier")
	if identifier == "" {
		return fail(c, http.StatusBadRequest, "INVALID_IDENTIFIER", "Tenant identifier is required", nil)
	}

	var branding domain.TenantBranding
	err := GetDB(c).
		Joins("JOIN tenant ON tenant.id = tenant_branding.tenant_id").
		Where("(tenant.slug = ? OR tenant.subdomain = ?) AND tenant.status = ?", identifier, identifier, "active").
		First(&branding).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fail(c, http.StatusNotFound, "BRANDING_NOT_FOUND", "Branding not found for tenant", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load branding", err.Error())
	}

	return ok(c, tenantBrandingPublicResponse{
		LogoURL:        branding.LogoURL,
		FaviconURL:     branding.FaviconURL,
		PrimaryColor:   branding.PrimaryColor,
		SecondaryColor: branding.SecondaryColor,
		Theme:          branding.Theme,
		CompanyName:    branding.CompanyName,
		Tagline:        branding.Tagline,
		WebsiteURL:     branding.WebsiteURL,
		SupportEmail:   branding.SupportEmail,
		SupportPhone:   branding.SupportPhone,
	})
}
