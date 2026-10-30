package adminapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/webserver"
)

func registerBrandingRoutes() {
	webserver.ApiGET("/public/branding/tenant/:identifier", getPublicTenantBrandingHandler)
	webserver.ApiGET("/branding", getTenantBrandingHandler)
	webserver.ApiPUT("/branding", updateTenantBrandingHandler)
}

type tenantBrandingResponse struct {
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

func getPublicTenantBrandingHandler(c echo.Context) error {
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

	return ok(c, tenantBrandingResponse{
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

func getTenantBrandingHandler(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(int64)
	if !ok {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", "Tenant context is required", nil)
	}

	var branding domain.TenantBranding
	err := GetDB(c).Where("tenant_id = ?", tenantID).First(&branding).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			branding = domain.TenantBranding{
				TenantID:       tenantID,
				PrimaryColor:   "#1976d2",
				SecondaryColor: "#dc004e",
				Theme:          "light",
			}
		} else {
			return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load branding", err.Error())
		}
	}

	return ok(c, branding)
}

type updateBrandingPayload struct {
	LogoURL        *string `json:"logo_url"`
	FaviconURL     *string `json:"favicon_url"`
	PrimaryColor   *string `json:"primary_color"`
	SecondaryColor *string `json:"secondary_color"`
	Theme          *string `json:"theme"`
	CompanyName    *string `json:"company_name"`
	Tagline        *string `json:"tagline"`
	WebsiteURL     *string `json:"website_url"`
	SupportEmail   *string `json:"support_email"`
	SupportPhone   *string `json:"support_phone"`
}

func updateTenantBrandingHandler(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(int64)
	if !ok {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", "Tenant context is required", nil)
	}

	var payload updateBrandingPayload
	if err := c.Bind(&payload); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid request body", err.Error())
	}

	var branding domain.TenantBranding
	err := GetDB(c).Where("tenant_id = ?", tenantID).First(&branding).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			branding = domain.TenantBranding{
				TenantID:       tenantID,
				PrimaryColor:   "#1976d2",
				SecondaryColor: "#dc004e",
				Theme:          "light",
			}
		} else {
			return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load branding", err.Error())
		}
	}

	if payload.LogoURL != nil {
		branding.LogoURL = *payload.LogoURL
	}
	if payload.FaviconURL != nil {
		branding.FaviconURL = *payload.FaviconURL
	}
	if payload.PrimaryColor != nil {
		branding.PrimaryColor = *payload.PrimaryColor
	}
	if payload.SecondaryColor != nil {
		branding.SecondaryColor = *payload.SecondaryColor
	}
	if payload.Theme != nil {
		branding.Theme = *payload.Theme
	}
	if payload.CompanyName != nil {
		branding.CompanyName = *payload.CompanyName
	}
	if payload.Tagline != nil {
		branding.Tagline = *payload.Tagline
	}
	if payload.WebsiteURL != nil {
		branding.WebsiteURL = *payload.WebsiteURL
	}
	if payload.SupportEmail != nil {
		branding.SupportEmail = *payload.SupportEmail
	}
	if payload.SupportPhone != nil {
		branding.SupportPhone = *payload.SupportPhone
	}

	db := GetDB(c)
	var saveErr error
	if branding.ID == 0 {
		saveErr = db.Create(&branding).Error
	} else {
		saveErr = db.Save(&branding).Error
	}
	if saveErr != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save branding", saveErr.Error())
	}

	return ok(c, branding)
}
