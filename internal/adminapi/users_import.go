package adminapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/webserver"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// ImportUserError describes one failed input row during a batch user import.
//
// Row is 1-based within the parsed payload and Username may be empty when the
// failure occurs before a username can be resolved (for example missing column).
type ImportUserError struct {
	Row      int    `json:"row"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// ImportUserResult summarizes the outcome of a batch user import request.
//
// Success and Failed are per-row counts and Errors carries only failed rows so
// callers can present actionable feedback without re-reading the source file.
// The import is all-or-nothing, so a nonzero Failed always means Success is 0
// and nothing was written.
type ImportUserResult struct {
	Total   int               `json:"total"`
	Success int               `json:"success"`
	Failed  int               `json:"failed"`
	Errors  []ImportUserError `json:"errors"`
}

// importMapString reads a trimmed string value from the row map using the first
// candidate key that yields a non-empty value.
func importMapString(item map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := item[k]; ok {
			s := strings.TrimSpace(cast.ToString(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

// importRadiusUsers handles batch import of RADIUS users from an uploaded file.
// Supported formats: Excel (.xlsx), CSV (.csv) and line-delimited JSON (.json).
//
// The import is atomic: every row is validated before the first write, and the
// accepted rows are inserted in one transaction. A payload with any invalid row
// is rejected with per-row errors and leaves the user table untouched, so a bad
// file cannot leave a half-imported batch behind.
func importRadiusUsers(c echo.Context) error {
	items, err := webserver.ImportData(c, "Sheet1")
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_FILE", "Unable to read import file", err.Error())
	}

	db := GetDB(c)
	users, rowErrors := planUserImport(db, items)
	result := ImportUserResult{Total: len(items), Failed: len(rowErrors), Errors: rowErrors}
	if len(rowErrors) > 0 {
		return ok(c, result)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, user := range users {
			if err := tx.Create(user).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fail(c, http.StatusInternalServerError, "IMPORT_FAILED",
			"Import failed and was rolled back; no users were created", err.Error())
	}

	result.Success = len(users)
	return ok(c, result)
}

// planUserImport validates every row and builds the users to insert without
// touching the database. It returns one error entry per rejected row; callers
// must not write anything when that slice is non-empty.
func planUserImport(db *gorm.DB, items []map[string]interface{}) ([]*domain.RadiusUser, []ImportUserError) {
	users := make([]*domain.RadiusUser, 0, len(items))
	errs := make([]ImportUserError, 0)
	seen := make(map[string]int, len(items))
	now := time.Now()
	defaultExpire := now.AddDate(1, 0, 0)

	for idx, item := range items {
		row := idx + 1
		reject := func(username, message string) {
			errs = append(errs, ImportUserError{Row: row, Username: username, Message: message})
		}

		username := importMapString(item, "username", "Username", "USERNAME")
		if username == "" {
			reject("", "Username is required")
			continue
		}
		if first, duplicate := seen[username]; duplicate {
			reject(username, fmt.Sprintf("Username duplicates row %d", first))
			continue
		}
		seen[username] = row

		password := importMapString(item, "password", "Password", "PASSWORD")
		if password == "" {
			reject(username, "Password is required")
			continue
		}

		profileRaw := importMapString(item, "profile_id", "profileid", "ProfileId", "profile")
		if profileRaw == "" {
			reject(username, "Billing profile is required")
			continue
		}
		profileID, convErr := strconv.ParseInt(profileRaw, 10, 64)
		if convErr != nil || profileID <= 0 {
			reject(username, "Billing profile id must be a positive integer")
			continue
		}

		var profile domain.RadiusProfile
		if err := db.Where("id = ?", profileID).First(&profile).Error; err != nil {
			reject(username, "Associated billing profile not found")
			continue
		}

		var exists int64
		if err := db.Model(&domain.RadiusUser{}).Where("username = ?", username).Count(&exists).Error; err != nil {
			reject(username, "Unable to check for an existing username")
			continue
		}
		if exists > 0 {
			reject(username, "Username already exists")
			continue
		}

		expire, err := parseTimeInput(importMapString(item, "expire_time", "ExpireTime"), defaultExpire)
		if err != nil {
			reject(username, "Invalid expire time format")
			continue
		}

		status := strings.ToLower(importMapString(item, "status", "Status"))
		if status == "" {
			status = common.ENABLED
		}
		// The auth checker only rejects common.DISABLED, so any other unknown
		// value would silently import an enabled user.
		if status != common.ENABLED && status != common.DISABLED {
			reject(username, "Status must be enabled or disabled")
			continue
		}

		nodeID := int64(0)
		if nodeRaw := importMapString(item, "node_id", "NodeId"); nodeRaw != "" {
			parsed, convErr := strconv.ParseInt(nodeRaw, 10, 64)
			if convErr != nil || parsed <= 0 {
				reject(username, "Network node id must be a positive integer")
				continue
			}
			nodeID = parsed
			var node domain.NetNode
			if err := db.First(&node, nodeID).Error; err != nil {
				reject(username, "Associated network node not found")
				continue
			}
		}

		addrPool := importMapString(item, "addr_pool", "AddrPool")
		if addrPool == "" {
			addrPool = profile.AddrPool
		}
		userDomain := importMapString(item, "domain", "Domain")
		if userDomain == "" {
			userDomain = profile.Domain
		}
		radiusClass := importMapString(item, "radius_class", "RadiusClass", "Class")
		if radiusClass == "" {
			radiusClass = profile.RadiusClass
		}

		users = append(users, &domain.RadiusUser{
			ID:              common.UUIDint64(),
			NodeId:          nodeID,
			ProfileId:       profileID,
			Realname:        importMapString(item, "realname", "Realname"),
			Email:           importMapString(item, "email", "Email"),
			Mobile:          importMapString(item, "mobile", "Mobile"),
			Address:         importMapString(item, "address", "Address"),
			Username:        username,
			Password:        password,
			AddrPool:        addrPool,
			RadiusClass:     radiusClass,
			ActiveNum:       profile.ActiveNum,
			UpRate:          profile.UpRate,
			DownRate:        profile.DownRate,
			Vlanid1:         cast.ToInt(importMapString(item, "vlanid1", "Vlanid1")),
			Vlanid2:         cast.ToInt(importMapString(item, "vlanid2", "Vlanid2")),
			IpAddr:          importMapString(item, "ip_addr", "IpAddr"),
			MacAddr:         importMapString(item, "mac_addr", "MacAddr"),
			Domain:          userDomain,
			IPv6PrefixPool:  profile.IPv6PrefixPool,
			BindVlan:        profile.BindVlan,
			BindMac:         profile.BindMac,
			ProfileLinkMode: domain.ProfileLinkModeStatic,
			ExpireTime:      expire,
			Status:          status,
			Remark:          importMapString(item, "remark", "Remark"),
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	return users, errs
}
