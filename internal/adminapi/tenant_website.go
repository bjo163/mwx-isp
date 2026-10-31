package adminapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"strconv"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/webserver"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const maxTenantWebsiteBytes = 96 << 10

var tenantWebsiteSections = map[string]bool{
	"header": true, "hero": true, "announcement": true, "about": true,
	"services": true, "coverage": true, "sla": true, "contact": true,
	"social": true, "footer": true,
}

type tenantWebsiteSection struct {
	Key string `json:"key"`
	Title string `json:"title"`
	Subtitle string `json:"subtitle"`
	Body string `json:"body"`
	ImageURL string `json:"image_url"`
	CTALabel string `json:"cta_label"`
	CTAURL string `json:"cta_url"`
	Visible bool `json:"visible"`
	Order int `json:"order"`
}

type tenantWebsiteContent struct {
	Sections []tenantWebsiteSection `json:"sections"`
}

type tenantWebsiteDraftResponse struct {
	Draft tenantWebsiteContent `json:"draft"`
	PublishedVersion int `json:"published_version"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type tenantWebsiteVersionResponse struct {
	Version int `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

func defaultTenantWebsiteContent() tenantWebsiteContent {
	keys := []string{"header","hero","announcement","about","services","coverage","sla","contact","social","footer"}
	labels := []string{"Header","Hero","Announcement","About us","Services","Coverage","SLA","Contact","Social media","Footer"}
	sections := make([]tenantWebsiteSection, 0, len(keys))
	for i, key := range keys {
		sections = append(sections, tenantWebsiteSection{Key:key, Title:labels[i], Visible:true, Order:i})
	}
	return tenantWebsiteContent{Sections:sections}
}

func registerTenantWebsiteRoutes() {
	webserver.ApiGET("/tenant-website/draft", getTenantWebsiteDraft, requireAdmin())
	webserver.ApiPUT("/tenant-website/draft", saveTenantWebsiteDraft, requireAdmin())
	webserver.ApiPOST("/tenant-website/publish", publishTenantWebsite, requireAdmin())
	webserver.ApiGET("/tenant-website/versions", listTenantWebsiteVersions, requireAdmin())
	webserver.ApiPOST("/tenant-website/versions/:version/rollback", rollbackTenantWebsiteVersion, requireAdmin())
	webserver.ApiGET("/tenant-website/public/:identifier", getPublicTenantWebsite)
}

func tenantWebsiteTenantID(c echo.Context) (int64, error) {
	id, ok := c.Get("tenant_id").(int64)
	if !ok || id <= 0 { return 0, fmt.Errorf("tenant context is required") }
	return id, nil
}

func decodeTenantWebsite(content string) (tenantWebsiteContent, error) {
	var result tenantWebsiteContent
	if len(content) > maxTenantWebsiteBytes { return result, errors.New("website content exceeds 96 KiB") }
	decoder := json.NewDecoder(bytes.NewBufferString(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil { return result, fmt.Errorf("invalid website content: %w", err) }
	if len(result.Sections) > len(tenantWebsiteSections) { return result, errors.New("too many website sections") }
	seen := make(map[string]bool, len(result.Sections))
	for _, section := range result.Sections {
		if !tenantWebsiteSections[section.Key] { return result, fmt.Errorf("unsupported website section %q", section.Key) }
		if seen[section.Key] { return result, fmt.Errorf("duplicate website section %q", section.Key) }
		seen[section.Key] = true
		if len(section.Title) > 120 || len(section.Subtitle) > 240 || len(section.Body) > 4000 ||
			len(section.ImageURL) > 500 || len(section.CTALabel) > 80 || len(section.CTAURL) > 500 {
			return result, fmt.Errorf("section %s exceeds a text limit", section.Key)
		}
		if section.Order < 0 || section.Order > 10000 { return result, fmt.Errorf("section %s has invalid order", section.Key) }
		if section.ImageURL != "" && !validTenantBrandingAssetURL(section.ImageURL) {
			return result, fmt.Errorf("section %s image must be same-origin relative path or HTTPS URL", section.Key)
		}
		if section.CTAURL != "" {
			if strings.HasPrefix(section.CTAURL, "/") && !strings.HasPrefix(section.CTAURL, "//") {
				continue
			}
			u, err := url.Parse(section.CTAURL)
			if err != nil || (u.Scheme != "https" && u.Scheme != "mailto" && u.Scheme != "tel") {
				return result, fmt.Errorf("section %s CTA must be HTTPS, mailto, tel or a relative path", section.Key)
			}
		}
	}
	return result, nil
}

func ensureTenantWebsite(db *gorm.DB, tenantID int64) (domain.TenantWebsite, error) {
	var row domain.TenantWebsite
	err := db.Where("tenant_id = ?", tenantID).First(&row).Error
	if err == nil { return row, nil }
	if !errors.Is(err, gorm.ErrRecordNotFound) { return row, err }
	raw, _ := json.Marshal(defaultTenantWebsiteContent())
	row = domain.TenantWebsite{TenantID:tenantID, DraftContent:string(raw)}
	if err := db.Create(&row).Error; err != nil { return row, err }
	return row, nil
}

func getTenantWebsiteDraft(c echo.Context) error {
	tenantID, err := tenantWebsiteTenantID(c)
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil) }
	row, err := ensureTenantWebsite(GetDB(c), tenantID)
	if err != nil { return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load website draft", err.Error()) }
	content, err := decodeTenantWebsite(row.DraftContent)
	if err != nil { content = defaultTenantWebsiteContent() }
	return ok(c, tenantWebsiteDraftResponse{Draft:content, PublishedVersion:row.PublishedVersion, PublishedAt:row.PublishedAt})
}

func saveTenantWebsiteDraft(c echo.Context) error {
	tenantID, err := tenantWebsiteTenantID(c)
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil) }
	var content tenantWebsiteContent
	decoder := json.NewDecoder(http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxTenantWebsiteBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&content); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_WEBSITE_CONTENT", "Invalid website draft", err.Error())
	}
	raw, err := json.Marshal(content)
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_WEBSITE_CONTENT", "Could not encode website draft", err.Error()) }
	validated, err := decodeTenantWebsite(string(raw))
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_WEBSITE_CONTENT", err.Error(), nil) }
	normalized, _ := json.Marshal(validated)
	row, err := ensureTenantWebsite(GetDB(c), tenantID)
	if err != nil { return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load website draft", err.Error()) }
	if err := GetDB(c).Model(&row).Where("tenant_id = ?", tenantID).Updates(map[string]any{"draft_content":string(normalized), "updated_at":time.Now().UTC()}).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save website draft", err.Error())
	}
	return ok(c, tenantWebsiteDraftResponse{Draft:validated, PublishedVersion:row.PublishedVersion, PublishedAt:row.PublishedAt})
}

func publishTenantWebsite(c echo.Context) error {
	tenantID, err := tenantWebsiteTenantID(c)
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil) }
	db := GetDB(c)
	row, err := ensureTenantWebsite(db, tenantID)
	if err != nil { return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load website draft", err.Error()) }
	content, err := decodeTenantWebsite(row.DraftContent)
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_WEBSITE_CONTENT", err.Error(), nil) }
	raw, _ := json.Marshal(content)
	now := time.Now().UTC()
	nextVersion := row.PublishedVersion + 1
	if err := db.Transaction(func(tx *gorm.DB) error {
		snapshot := domain.TenantWebsiteVersion{TenantID:tenantID, Version:nextVersion, ContentJSON:string(raw), CreatedAt:now}
		if err := tx.Create(&snapshot).Error; err != nil { return err }
		return tx.Model(&domain.TenantWebsite{}).Where("tenant_id = ?",tenantID).Updates(map[string]any{
			"published_content":string(raw), "published_version":nextVersion, "published_at":now, "updated_at":now,
		}).Error
	}); err != nil { return fail(c, http.StatusInternalServerError, "PUBLISH_FAILED", "Failed to publish website", err.Error()) }
	return ok(c, tenantWebsiteDraftResponse{Draft:content, PublishedVersion:nextVersion, PublishedAt:&now})
}

func listTenantWebsiteVersions(c echo.Context) error {
	tenantID, err := tenantWebsiteTenantID(c)
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil) }
	var versions []domain.TenantWebsiteVersion
	if err := GetDB(c).Where("tenant_id = ?",tenantID).Order("version DESC").Limit(50).Find(&versions).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list website versions", err.Error())
	}
	out := make([]tenantWebsiteVersionResponse,0,len(versions))
	for _, row := range versions { out=append(out,tenantWebsiteVersionResponse{Version:row.Version,CreatedAt:row.CreatedAt}) }
	return ok(c,out)
}

func rollbackTenantWebsiteVersion(c echo.Context) error {
	tenantID, err := tenantWebsiteTenantID(c)
	if err != nil { return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil) }
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version < 1 { return fail(c, http.StatusBadRequest, "INVALID_VERSION", "Invalid website version", nil) }
	var snapshot domain.TenantWebsiteVersion
	if err := GetDB(c).Where("tenant_id = ? AND version = ?",tenantID,version).First(&snapshot).Error; err != nil {
		if errors.Is(err,gorm.ErrRecordNotFound) { return fail(c,http.StatusNotFound,"VERSION_NOT_FOUND","Website version not found",nil) }
		return fail(c,http.StatusInternalServerError,"DATABASE_ERROR","Failed to load website version",err.Error())
	}
	if _, err := decodeTenantWebsite(snapshot.ContentJSON); err != nil { return fail(c,http.StatusInternalServerError,"INVALID_SNAPSHOT","Stored website version is invalid",err.Error()) }
	row, err := ensureTenantWebsite(GetDB(c),tenantID)
	if err != nil { return fail(c,http.StatusInternalServerError,"DATABASE_ERROR","Failed to load website",err.Error()) }
	if err := GetDB(c).Model(&row).Where("tenant_id = ?",tenantID).Update("draft_content",snapshot.ContentJSON).Error; err != nil {
		return fail(c,http.StatusInternalServerError,"DATABASE_ERROR","Failed to stage website rollback",err.Error())
	}
	c.Set("rollback_source_version",version)
	return publishTenantWebsite(c)
}

func getPublicTenantWebsite(c echo.Context) error {
	identifier := strings.TrimSpace(strings.ToLower(c.Param("identifier")))
	if identifier == "" { return fail(c,http.StatusBadRequest,"INVALID_IDENTIFIER","Tenant identifier is required",nil) }
	var tenant domain.Tenant
	if err := GetDB(c).Where("(slug = ? OR subdomain = ?) AND status = ?",identifier,identifier,"active").First(&tenant).Error; err != nil {
		if errors.Is(err,gorm.ErrRecordNotFound) { return fail(c,http.StatusNotFound,"TENANT_NOT_FOUND","Active tenant not found",nil) }
		return fail(c,http.StatusInternalServerError,"DATABASE_ERROR","Failed to resolve tenant",err.Error())
	}
	setRequestTenant(c, tenant.ID)
	var row domain.TenantWebsite
	if err := GetDB(c).Where("tenant_id = ?",tenant.ID).First(&row).Error; err != nil {
		if errors.Is(err,gorm.ErrRecordNotFound) { return fail(c,http.StatusNotFound,"WEBSITE_NOT_PUBLISHED","Tenant website is not published",nil) }
		return fail(c,http.StatusInternalServerError,"DATABASE_ERROR","Failed to load published website",err.Error())
	}
	if row.PublishedVersion <= 0 || strings.TrimSpace(row.PublishedContent) == "" {
		return fail(c, http.StatusNotFound, "WEBSITE_NOT_PUBLISHED", "Tenant website is not published", nil)
	}
	content, err := decodeTenantWebsite(row.PublishedContent)
	if err != nil { return fail(c,http.StatusInternalServerError,"INVALID_PUBLISHED_CONTENT","Published website content is invalid",nil) }
	return ok(c,content)
}
