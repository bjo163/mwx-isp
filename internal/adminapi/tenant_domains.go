package adminapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/webserver"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const tenantDomainTXTPrefix = "mwx-isp-verification="

var (
	tenantHostnamePattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}$`)
	lookupTenantDomainTXT = net.LookupTXT
)

type tenantDomainInput struct {
	Hostname string `json:"hostname"`
}

type tenantDomainResponse struct {
	ID                int64      `json:"id,string"`
	Hostname          string     `json:"hostname"`
	VerificationTXT   string     `json:"verification_txt"`
	VerificationValue string     `json:"verification_value"`
	VerifiedAt        *time.Time `json:"verified_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

func registerTenantDomainRoutes() {
	webserver.ApiGET("/tenant-domains", listTenantDomains, requireAdmin())
	webserver.ApiPOST("/tenant-domains", createTenantDomain, requireAdmin())
	webserver.ApiPOST("/tenant-domains/:id/verify", verifyTenantDomain, requireAdmin())
	webserver.ApiDELETE("/tenant-domains/:id", deleteTenantDomain, requireAdmin())
}

func tenantDomainTenantID(c echo.Context) (int64, error) {
	id, ok := c.Get("tenant_id").(int64)
	if !ok || id <= 0 {
		return 0, fmt.Errorf("tenant context is required")
	}
	return id, nil
}

func normalizeTenantHostname(raw string) (string, error) {
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
	if host == "" || strings.ContainsAny(host, "/:@?#") || net.ParseIP(host) != nil || !tenantHostnamePattern.MatchString(host) {
		return "", fmt.Errorf("enter a valid DNS hostname without a scheme, path, port or IP address")
	}
	if host == "e-isp.id" || strings.HasSuffix(host, ".e-isp.id") {
		return "", fmt.Errorf("e-isp.id hostnames are managed by the platform and cannot be registered as custom domains")
	}
	return host, nil
}

func tenantDomainTXTName(host string) string {
	return "_mwx-isp-verification." + host
}

func tenantDomainTXTValue(token string) string {
	return tenantDomainTXTPrefix + token
}

func tenantDomainTXTMatches(records []string, token string) bool {
	want := tenantDomainTXTValue(token)
	for _, record := range records {
		if strings.TrimSpace(record) == want {
			return true
		}
	}
	return false
}

func toTenantDomainResponse(row domain.TenantDomain) tenantDomainResponse {
	return tenantDomainResponse{
		ID: row.ID, Hostname: row.Hostname,
		VerificationTXT: tenantDomainTXTName(row.Hostname),
		VerificationValue: tenantDomainTXTValue(row.VerificationToken),
		VerifiedAt: row.VerifiedAt, CreatedAt: row.CreatedAt,
	}
}

func listTenantDomains(c echo.Context) error {
	tenantID, err := tenantDomainTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	var rows []domain.TenantDomain
	if err := GetDB(c).Where("tenant_id = ?", tenantID).Order("hostname ASC").Find(&rows).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list tenant domains", err.Error())
	}
	result := make([]tenantDomainResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, toTenantDomainResponse(row))
	}
	return ok(c, result)
}

func createTenantDomain(c echo.Context) error {
	tenantID, err := tenantDomainTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	var input tenantDomainInput
	if err := c.Bind(&input); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_REQUEST", "Unable to parse custom domain", nil)
	}
	host, err := normalizeTenantHostname(input.Hostname)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_HOSTNAME", err.Error(), nil)
	}
	var existing domain.TenantDomain
	err = GetDB(c).Where("hostname = ?", host).First(&existing).Error
	if err == nil {
		return fail(c, http.StatusConflict, "HOSTNAME_CLAIMED", "This hostname is already registered", nil)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to check hostname ownership", err.Error())
	}
	var token [24]byte
	if _, err := rand.Read(token[:]); err != nil {
		return fail(c, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Could not create verification challenge", err.Error())
	}
	row := domain.TenantDomain{
		TenantID: tenantID, Hostname: host, VerificationToken: hex.EncodeToString(token[:]),
	}
	if err := GetDB(c).Create(&row).Error; err != nil {
		return fail(c, http.StatusConflict, "DOMAIN_SAVE_FAILED", "Could not register custom hostname", err.Error())
	}
	return c.JSON(http.StatusCreated, Response{Data: toTenantDomainResponse(row)})
}

func verifyTenantDomain(c echo.Context) error {
	tenantID, err := tenantDomainTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid custom domain ID", nil)
	}
	var row domain.TenantDomain
	if err := GetDB(c).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Custom domain not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load custom domain", err.Error())
	}
	records, err := lookupTenantDomainTXT(tenantDomainTXTName(row.Hostname))
	if err != nil {
		return fail(c, http.StatusServiceUnavailable, "DNS_LOOKUP_FAILED", "Could not resolve the domain verification TXT record yet", nil)
	}
	if !tenantDomainTXTMatches(records, row.VerificationToken) {
		return fail(c, http.StatusConflict, "DNS_VERIFICATION_PENDING", "The expected TXT value was not found. Add the exact record and retry.", nil)
	}
	now := time.Now().UTC()
	if err := GetDB(c).Model(&row).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(map[string]any{
		"verified_at": now, "updated_at": now,
	}).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "VERIFICATION_SAVE_FAILED", "DNS ownership was proven but could not be saved", err.Error())
	}
	row.VerifiedAt = &now
	return ok(c, toTenantDomainResponse(row))
}

func deleteTenantDomain(c echo.Context) error {
	tenantID, err := tenantDomainTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid custom domain ID", nil)
	}
	result := GetDB(c).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&domain.TenantDomain{})
	if result.Error != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to remove custom domain", result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fail(c, http.StatusNotFound, "DOMAIN_NOT_FOUND", "Custom domain not found", nil)
	}
	return ok(c, map[string]bool{"deleted": true})
}
