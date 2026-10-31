package adminapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type packagePresentationInput struct {
	DisplayName      *string `json:"display_name"`
	ShortDescription *string `json:"short_description"`
	Category         *string `json:"category"`
	Icon             *string `json:"icon"`
	ImageURL         *string `json:"image_url"`
	Badge            *string `json:"badge"`
	Featured         *bool   `json:"featured"`
	HighlightColor   *string `json:"highlight_color"`
	Highlights       *string `json:"highlights"`
	CTALabel         *string `json:"cta_label"`
	Terms            *string `json:"terms"`
	DisplayOrder     *int    `json:"display_order"`
	PublicVisible    *bool   `json:"public_visible"`
}

func packagePresentationTenantID(c echo.Context) (int64, error) {
	id, ok := c.Get("tenant_id").(int64)
	if !ok || id <= 0 {
		return 0, fmt.Errorf("tenant context is required")
	}
	return id, nil
}

func listPackagePresentations(c echo.Context) error {
	tenantID, err := packagePresentationTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	var rows []domain.TenantPackagePresentation
	if err := GetDB(c).Where("tenant_id = ?", tenantID).Order("display_order ASC, package_id ASC").Find(&rows).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list package presentation", err.Error())
	}
	return ok(c, rows)
}

func getPackagePresentation(c echo.Context) error {
	tenantID, err := packagePresentationTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	packageID, err := parseIDParam(c, "id")
	if err != nil || packageID <= 0 {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid package ID", nil)
	}
	var pkg domain.InternetPackage
	if err := GetDB(c).Where("tenant_id = ? AND id = ?", tenantID, packageID).First(&pkg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fail(c, http.StatusNotFound, "PACKAGE_NOT_FOUND", "Package not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load package", err.Error())
	}
	var row domain.TenantPackagePresentation
	err = GetDB(c).Where("tenant_id = ? AND package_id = ?", tenantID, packageID).First(&row).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load package presentation", err.Error())
	}
	if err == gorm.ErrRecordNotFound {
		row = domain.TenantPackagePresentation{TenantID: tenantID, PackageID: packageID, PublicVisible: true}
	}
	return ok(c, row)
}

func validatePackagePresentation(input packagePresentationInput) error {
	fields := []struct {
		name  string
		value *string
		max   int
	}{
		{"display_name", input.DisplayName, 150}, {"short_description", input.ShortDescription, 500},
		{"category", input.Category, 80}, {"icon", input.Icon, 48}, {"image_url", input.ImageURL, 500},
		{"badge", input.Badge, 40}, {"highlight_color", input.HighlightColor, 7},
		{"highlights", input.Highlights, 2000}, {"cta_label", input.CTALabel, 64}, {"terms", input.Terms, 1000},
	}
	for _, field := range fields {
		if field.value != nil && len(strings.TrimSpace(*field.value)) > field.max {
			return fmt.Errorf("%s must not exceed %d characters", field.name, field.max)
		}
	}
	if input.ImageURL != nil && !validTenantBrandingAssetURL(*input.ImageURL) {
		return fmt.Errorf("image_url must be a relative path or an absolute HTTPS URL")
	}
	if input.HighlightColor != nil && strings.TrimSpace(*input.HighlightColor) != "" && !brandingAccentPattern.MatchString(strings.TrimSpace(*input.HighlightColor)) {
		return fmt.Errorf("highlight_color must be a six-digit hexadecimal value")
	}
	if input.DisplayOrder != nil && (*input.DisplayOrder < 0 || *input.DisplayOrder > 100000) {
		return fmt.Errorf("display_order must be between 0 and 100000")
	}
	return nil
}

func updatePackagePresentation(c echo.Context) error {
	tenantID, err := packagePresentationTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	packageID, err := parseIDParam(c, "id")
	if err != nil || packageID <= 0 {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid package ID", nil)
	}
	var pkg domain.InternetPackage
	if err := GetDB(c).Where("tenant_id = ? AND id = ?", tenantID, packageID).First(&pkg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fail(c, http.StatusNotFound, "PACKAGE_NOT_FOUND", "Package not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load package", err.Error())
	}
	var input packagePresentationInput
	if err := c.Bind(&input); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_REQUEST", "Unable to parse package presentation", nil)
	}
	if err := validatePackagePresentation(input); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_PACKAGE_PRESENTATION", err.Error(), nil)
	}
	db := GetDB(c)
	var row domain.TenantPackagePresentation
	err = db.Where("tenant_id = ? AND package_id = ?", tenantID, packageID).First(&row).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load package presentation", err.Error())
	}
	if err == gorm.ErrRecordNotFound {
		row = domain.TenantPackagePresentation{TenantID: tenantID, PackageID: packageID, PublicVisible: true}
	}
	set := func(target *string, value *string) {
		if value != nil {
			*target = strings.TrimSpace(*value)
		}
	}
	set(&row.DisplayName, input.DisplayName)
	set(&row.ShortDescription, input.ShortDescription)
	set(&row.Category, input.Category)
	set(&row.Icon, input.Icon)
	set(&row.ImageURL, input.ImageURL)
	set(&row.Badge, input.Badge)
	set(&row.HighlightColor, input.HighlightColor)
	set(&row.Highlights, input.Highlights)
	set(&row.CTALabel, input.CTALabel)
	set(&row.Terms, input.Terms)
	if input.Featured != nil {
		row.Featured = *input.Featured
	}
	if input.DisplayOrder != nil {
		row.DisplayOrder = *input.DisplayOrder
	}
	if input.PublicVisible != nil {
		row.PublicVisible = *input.PublicVisible
	}
	row.UpdatedAt = time.Now().UTC()
	if row.ID == 0 {
		if err := db.Create(&row).Error; err != nil {
			return fail(c, http.StatusConflict, "PRESENTATION_SAVE_FAILED", "Failed to create package presentation", err.Error())
		}
	} else if err := db.Save(&row).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "PRESENTATION_SAVE_FAILED", "Failed to update package presentation", err.Error())
	}
	return ok(c, row)
}
