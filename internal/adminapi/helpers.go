package adminapi

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// List pagination bounds. The default list page is 20 with a 1000-row cap
// (system settings can legitimately return more than 20 items and bulk save
// broke when an oversized request was silently reset to 20 — see #583). Live
// sessions, accounting and profiles keep a smaller page and cap because their
// rows are the widest the API returns.
const (
	defaultPage            = 1
	defaultPageSize        = 20
	maxPageSize            = 1000
	compactListPageSize    = 10
	compactListMaxPageSize = 100
)

// parsePagination applies the shared grammar with the default list bounds.
func parsePagination(c echo.Context) (int, int) {
	return parsePaginationWith(c, defaultPageSize, maxPageSize)
}

// parsePaginationWith applies the shared pagination grammar with a per-endpoint
// default and cap. Every list handler must accept both `pageSize` and React
// Admin's `perPage` (pageSize wins when both are present), keep page >= 1, fall
// back to defaultSize for unparsable input, and clamp an oversized request to
// maxSize rather than silently returning a different small page.
func parsePaginationWith(c echo.Context, defaultSize, maxSize int) (int, int) {
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = defaultPage
	}

	rawSize := c.QueryParam("pageSize")
	if rawSize == "" {
		// Frontend dataProvider uses React Admin's `perPage` query key.
		rawSize = c.QueryParam("perPage")
	}
	pageSize, err := strconv.Atoi(rawSize)
	if err != nil || pageSize < 1 {
		pageSize = defaultSize
	} else if pageSize > maxSize {
		pageSize = maxSize
	}
	return page, pageSize
}

// parseSort extracts and validates the `sort` and `order` query parameters
// against an allowlist of sortable columns, returning values that are safe to
// interpolate into an ORDER BY clause. Unknown or empty inputs fall back to
// defaultField/defaultOrder. Centralizing this guard ensures every list
// endpoint applies the same SQL-injection-safe column allowlist instead of
// re-implementing (and potentially forgetting) it.
//
// The returned field is re-derived from the matching allowlist key (a
// compile-time constant) and the returned order is one of two string literals,
// so neither value carries the raw query string into the SQL ORDER BY clause.
func parseSort(c echo.Context, allowed map[string]bool, defaultField, defaultOrder string) (field, order string) {
	field = defaultField
	if requested := c.QueryParam("sort"); requested != "" {
		for col := range allowed {
			if col == requested {
				field = col
				break
			}
		}
	}

	order = "ASC"
	switch strings.ToUpper(strings.TrimSpace(c.QueryParam("order"))) {
	case "DESC":
		order = "DESC"
	case "ASC":
		order = "ASC"
	default:
		if strings.ToUpper(strings.TrimSpace(defaultOrder)) == "DESC" {
			order = "DESC"
		}
	}
	return field, order
}

func parseIDParam(c echo.Context, name string) (int64, error) {
	param := c.Param(name)
	if param == "" {
		param = c.Param("id")
	}
	if param == "" {
		return 0, errors.New("missing identifier")
	}
	id, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func parseTimeInput(value string, fallback time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	layouts := []string{
		time.RFC3339,          // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05", // ISO 8601 with seconds
		"2006-01-02T15:04",    // HTML5 datetime-local format (no seconds)
		"2006-01-02 15:04:05", // Common format with space
		"2006-01-02 15:04",    // Common format without seconds
		"2006-01-02",          // Date only
	}
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			// If only a date is provided, set it to the last second of that day
			if layout == "2006-01-02" {
				return ts.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
			}
			return ts, nil
		}
	}
	return time.Time{}, errors.New("invalid time format")
}
