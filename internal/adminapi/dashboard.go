package adminapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/internal/webserver"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DashboardStats is the aggregated snapshot returned by [GetDashboardStats]. It
// combines real-time counters, today's volumes, and several time series so the
// admin dashboard can render its entire overview from a single response.
type DashboardStats struct {
	TotalUsers          int64                     `json:"total_users"`          // Total number of users
	OnlineUsers         int64                     `json:"online_users"`         // Currently online users
	TodayAuthCount      int64                     `json:"today_auth_count"`     // Authentication count for today
	TodayAcctCount      int64                     `json:"today_acct_count"`     // Accounting record count for today
	TotalProfiles       int64                     `json:"total_profiles"`       // Total number of profiles
	DisabledUsers       int64                     `json:"disabled_users"`       // Disabled users
	ExpiredUsers        int64                     `json:"expired_users"`        // Expired users
	TodayInputGB        float64                   `json:"today_input_gb"`       // Today's upstream traffic (GB)
	TodayOutputGB       float64                   `json:"today_output_gb"`      // Today's downstream traffic (GB)
	AuthTrend           []DashboardAuthTrendPoint `json:"auth_trend"`           // Daily authentication trend (last 7 days)
	Traffic24h          []DashboardTrafficPoint   `json:"traffic_24h"`          // Hourly traffic statistics (last 24 hours)
	ProfileDistribution []DashboardProfileSlice   `json:"profile_distribution"` // Online users grouped by profile
	IPv6Stats           DashboardIPv6Stats        `json:"ipv6_stats"`           // IPv6 adoption among currently online sessions
}

// DashboardIPv6Stats summarizes IPv6 usage along two dimensions: live adoption
// across currently online sessions, and static provisioning across the user
// base. Together they let operators answer both "who is using IPv6 right now?"
// and "how many subscribers are provisioned for IPv6?" without inspecting the
// database manually.
type DashboardIPv6Stats struct {
	OnlineWithIPv6            int64   `json:"online_with_ipv6"`             // Online sessions carrying any IPv6 attribute
	OnlineWithIPv6Address     int64   `json:"online_with_ipv6_address"`     // Online sessions with a Framed-IPv6-Address
	OnlineWithFramedPrefix    int64   `json:"online_with_framed_prefix"`    // Online sessions with a Framed-IPv6-Prefix
	OnlineWithDelegatedPrefix int64   `json:"online_with_delegated_prefix"` // Online sessions with a Delegated-IPv6-Prefix
	UsersWithStaticAddress    int64   `json:"users_with_static_address"`    // Users provisioned with a static IPv6 address (RadiusUser.IpV6Addr)
	UsersWithDelegatedPrefix  int64   `json:"users_with_delegated_prefix"`  // Users provisioned with a static Delegated-IPv6-Prefix (RFC 4818)
	AdoptionRate              float64 `json:"adoption_rate"`                // Percentage of online sessions carrying any IPv6 attribute
}

// DashboardAuthTrendPoint is one day in the dashboard's 7-day authentication
// trend: the per-day count of accounting sessions started on that date.
type DashboardAuthTrendPoint struct {
	Date  string `json:"date"`  // Date label formatted as YYYY-MM-DD
	Count int64  `json:"count"` // Authentication count for the day
}

// DashboardTrafficPoint is one hour in the dashboard's 24-hour traffic series,
// holding the upstream and downstream totals (GB) for sessions started in that
// hour.
type DashboardTrafficPoint struct {
	Hour       string  `json:"hour"`        // Hour label formatted as YYYY-MM-DD HH:00
	UploadGB   float64 `json:"upload_gb"`   // Upload traffic in GB within the hour
	DownloadGB float64 `json:"download_gb"` // Download traffic in GB within the hour
}

// DashboardProfileSlice is one slice of the online-user-by-profile distribution:
// the profile identified by ProfileID/ProfileName and the number of currently
// online users assigned to it.
type DashboardProfileSlice struct {
	ProfileID   int64  `json:"profile_id"`   // Profile primary key
	ProfileName string `json:"profile_name"` // Profile display name
	Value       int64  `json:"value"`        // Online users assigned to the profile
}

const (
	dateKeyFormat = "2006-01-02"
	hourKeyFormat = "2006-01-02 15:00"
	bytesInGB     = float64(1024 * 1024 * 1024)
)

// GetDashboardStats handles GET /api/v1/dashboard/stats, returning a single
// [DashboardStats] snapshot that the admin dashboard renders in one request. It
// aggregates real-time counters (total, online, disabled, and expired users, and
// total profiles), today's volumes (authentication and accounting record counts
// plus today's upstream/downstream traffic in GB), and three time series — a
// 7-day daily authentication trend, a 24-hour hourly upload/download traffic
// series, and the distribution of online users across profiles — together with an
// IPv6 adoption summary ([DashboardIPv6Stats]).
//
// "Today" is measured from the local start of day in the configured system
// location (time.Local). The trend and traffic series are bucketed by explicit
// instant windows rather than by SQL DATE()/date_trunc()/strftime('localtime'),
// so the database session timezone cannot shift a point into a different column
// than the Go-generated label. Buckets are zero-filled so every day/hour is
// present even with no data, and traffic is reported in GB (bytes divided by
// 1024^3). TodayAuthCount is an estimate derived from sessions that started
// today, not a dedicated authentication counter. A failure in any individual
// section is logged and that section falls back to zero/empty rather than
// failing the whole response. Any authenticated operator may call it.
//
// @Summary get dashboard statistics
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} DashboardStats
// @Router /api/v1/dashboard/stats [get]
func GetDashboardStats(c echo.Context) error {
	db := GetDB(c).WithContext(c.Request().Context())
	now := time.Now()
	todayStart := startOfDay(now)

	stats := &DashboardStats{}

	// 1. Total users
	db.Model(&domain.RadiusUser{}).Count(&stats.TotalUsers)

	// 2. Online users
	db.Model(&domain.RadiusOnline{}).Count(&stats.OnlineUsers)

	// 3. Total profiles
	db.Model(&domain.RadiusProfile{}).Count(&stats.TotalProfiles)

	// 4. Disabled users
	db.Model(&domain.RadiusUser{}).Where("status = ?", "disabled").Count(&stats.DisabledUsers)

	// 5. Expired users
	db.Model(&domain.RadiusUser{}).Where("expire_time < ?", now).Count(&stats.ExpiredUsers)

	// 6. Today's authentication count (estimated from today's new online sessions)
	db.Model(&domain.RadiusOnline{}).Where("acct_start_time >= ?", todayStart).Count(&stats.TodayAuthCount)

	// 7. Today's accounting record count
	db.Model(&domain.RadiusAccounting{}).Where("acct_start_time >= ?", todayStart).Count(&stats.TodayAcctCount)

	// 8. Today's traffic statistics (bytes to GB)
	var flowStats struct {
		TotalInput  int64
		TotalOutput int64
	}
	db.Model(&domain.RadiusAccounting{}).
		Select("COALESCE(SUM(acct_input_total), 0) as total_input, COALESCE(SUM(acct_output_total), 0) as total_output").
		Where("acct_start_time >= ?", todayStart).
		Scan(&flowStats)

	stats.TodayInputGB = float64(flowStats.TotalInput) / bytesInGB
	stats.TodayOutputGB = float64(flowStats.TotalOutput) / bytesInGB
	stats.AuthTrend = fetchAuthTrend(db, now)
	stats.Traffic24h = fetchTrafficStats(db, now)
	stats.ProfileDistribution = fetchProfileDistribution(db)
	stats.IPv6Stats = fetchIPv6Stats(db, stats.OnlineUsers)

	return ok(c, stats)
}

// registerDashboardRoutes wires the dashboard endpoint, which is open to any
// authenticated operator.
func registerDashboardRoutes() {
	webserver.ApiGET("/dashboard/stats", GetDashboardStats)
}

// dashboardWindow is a half-open interval [start, end) covered by one point of
// a dashboard time series.
type dashboardWindow struct {
	start time.Time
	end   time.Time
}

// dayWindows returns consecutive calendar days in the time's location, ending
// with the day that contains now.
func dayWindows(now time.Time, days int) []dashboardWindow {
	windows := make([]dashboardWindow, days)
	for i := range windows {
		start := startOfDay(now).AddDate(0, 0, i-days+1)
		windows[i] = dashboardWindow{start: start, end: start.AddDate(0, 0, 1)}
	}
	return windows
}

// hourWindows returns consecutive clock hours, ending with the hour that
// contains now.
func hourWindows(now time.Time, hours int) []dashboardWindow {
	windows := make([]dashboardWindow, hours)
	next := startOfHour(now).Add(time.Hour)
	for i := range windows {
		start := next.Add(time.Duration(i-hours) * time.Hour)
		windows[i] = dashboardWindow{start: start, end: start.Add(time.Hour)}
	}
	return windows
}

// countSessionsPerWindow counts accounting sessions that started inside each
// window in a single statement. The per-window CASE keeps the boundaries as
// instant comparisons, so the result does not depend on the database session
// timezone the way DATE() or date_trunc() would.
func countSessionsPerWindow(db *gorm.DB, windows []dashboardWindow) []int64 {
	counts := make([]int64, len(windows))
	selects := make([]string, len(windows))
	args := make([]any, 0, len(windows)*2)
	for i, w := range windows {
		selects[i] = fmt.Sprintf("COUNT(CASE WHEN acct_start_time >= ? AND acct_start_time < ? THEN 1 END) AS bucket_%d", i)
		args = append(args, w.start, w.end)
	}
	row := db.Model(&domain.RadiusAccounting{}).Select(strings.Join(selects, ", "), args...).Row()
	dests := make([]any, len(counts))
	for i := range counts {
		dests[i] = &counts[i]
	}
	if err := row.Scan(dests...); err != nil {
		logDashboardQueryError("count sessions per window", err)
	}
	return counts
}

// sumTrafficPerWindow returns the uploaded and downloaded bytes of sessions
// that started inside each window, using the same instant boundaries as
// [countSessionsPerWindow].
func sumTrafficPerWindow(db *gorm.DB, windows []dashboardWindow) (upload, download []float64) {
	upload = make([]float64, len(windows))
	download = make([]float64, len(windows))
	selects := make([]string, 0, len(windows)*2)
	args := make([]any, 0, len(windows)*4)
	for i, w := range windows {
		selects = append(selects, fmt.Sprintf("COALESCE(SUM(CASE WHEN acct_start_time >= ? AND acct_start_time < ? THEN acct_input_total END), 0) AS upload_%d", i))
		args = append(args, w.start, w.end)
	}
	for i, w := range windows {
		selects = append(selects, fmt.Sprintf("COALESCE(SUM(CASE WHEN acct_start_time >= ? AND acct_start_time < ? THEN acct_output_total END), 0) AS download_%d", i))
		args = append(args, w.start, w.end)
	}
	row := db.Model(&domain.RadiusAccounting{}).Select(strings.Join(selects, ", "), args...).Row()
	dests := make([]any, 0, len(windows)*2)
	for i := range upload {
		dests = append(dests, &upload[i])
	}
	for i := range download {
		dests = append(dests, &download[i])
	}
	if err := row.Scan(dests...); err != nil {
		logDashboardQueryError("sum traffic per window", err)
	}
	return upload, download
}

func fetchAuthTrend(db *gorm.DB, now time.Time) []DashboardAuthTrendPoint {
	windows := dayWindows(now, 7)
	counts := countSessionsPerWindow(db, windows)
	result := make([]DashboardAuthTrendPoint, len(windows))
	for i, w := range windows {
		result[i] = DashboardAuthTrendPoint{
			Date:  w.start.Format(dateKeyFormat),
			Count: counts[i],
		}
	}
	return result
}

func fetchTrafficStats(db *gorm.DB, now time.Time) []DashboardTrafficPoint {
	windows := hourWindows(now, 24)
	upload, download := sumTrafficPerWindow(db, windows)
	result := make([]DashboardTrafficPoint, len(windows))
	for i, w := range windows {
		result[i] = DashboardTrafficPoint{
			Hour:       w.start.Format(hourKeyFormat),
			UploadGB:   upload[i] / bytesInGB,
			DownloadGB: download[i] / bytesInGB,
		}
	}
	return result
}

func fetchProfileDistribution(db *gorm.DB) []DashboardProfileSlice {
	var rows []struct {
		ProfileID   int64
		ProfileName string
		Count       int64
	}
	onlineTable := domain.RadiusOnline{}.TableName()
	userTable := domain.RadiusUser{}.TableName()
	profileTable := domain.RadiusProfile{}.TableName()
	query := db.Table(fmt.Sprintf("%s AS o", onlineTable)).
		Select("COALESCE(u.profile_id, 0) AS profile_id, COALESCE(p.name, '') AS profile_name, COUNT(*) AS count").
		Joins(fmt.Sprintf("LEFT JOIN %s AS u ON u.username = o.username AND u.tenant_id = o.tenant_id", userTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s AS p ON p.id = u.profile_id AND p.tenant_id = u.tenant_id", profileTable))
	if tenantID, ok := tenancy.TenantID(db.Statement.Context); ok {
		query = query.Where("o.tenant_id = ?", tenantID)
	}
	if err := query.
		Group("profile_id, profile_name").
		Order("count DESC").
		Scan(&rows).Error; err != nil {
		logDashboardQueryError("fetch profile distribution", err)
	}
	result := make([]DashboardProfileSlice, 0, len(rows))
	for _, row := range rows {
		result = append(result, DashboardProfileSlice{
			ProfileID:   row.ProfileID,
			ProfileName: row.ProfileName,
			Value:       row.Count,
		})
	}
	return result
}

// fetchIPv6Stats aggregates IPv6 usage along two dimensions. The "online"
// counters reflect live adoption across currently online sessions; the "users"
// counters reflect static provisioning across the RadiusUser base. A field is
// considered "present" when it is non-null and not an empty/"NA" placeholder.
// onlineTotal is the total online session count used to compute the adoption rate.
func fetchIPv6Stats(db *gorm.DB, onlineTotal int64) DashboardIPv6Stats {
	stats := DashboardIPv6Stats{}
	notEmpty := func(col string) string {
		return fmt.Sprintf("%s IS NOT NULL AND %s <> '' AND %s <> 'NA'", col, col, col)
	}
	model := func() *gorm.DB { return db.Model(&domain.RadiusOnline{}) }

	if err := model().Where(notEmpty("framed_ipv6_address")).Count(&stats.OnlineWithIPv6Address).Error; err != nil {
		logDashboardQueryError("fetch ipv6 address count", err)
	}
	if err := model().Where(notEmpty("framed_ipv6_prefix")).Count(&stats.OnlineWithFramedPrefix).Error; err != nil {
		logDashboardQueryError("fetch ipv6 framed prefix count", err)
	}
	if err := model().Where(notEmpty("delegated_ipv6_prefix")).Count(&stats.OnlineWithDelegatedPrefix).Error; err != nil {
		logDashboardQueryError("fetch ipv6 delegated prefix count", err)
	}

	anyCond := fmt.Sprintf("(%s) OR (%s) OR (%s)",
		notEmpty("framed_ipv6_address"), notEmpty("framed_ipv6_prefix"), notEmpty("delegated_ipv6_prefix"))
	if err := model().Where(anyCond).Count(&stats.OnlineWithIPv6).Error; err != nil {
		logDashboardQueryError("fetch ipv6 any count", err)
	}

	// User-base provisioning dimension: how many subscribers carry a static IPv6
	// identity (RadiusUser.IpV6Addr -> column ip_v6_addr; RadiusUser.DelegatedIpv6Prefix).
	userModel := func() *gorm.DB { return db.Model(&domain.RadiusUser{}) }
	if err := userModel().Where(notEmpty("ip_v6_addr")).Count(&stats.UsersWithStaticAddress).Error; err != nil {
		logDashboardQueryError("fetch ipv6 user static address count", err)
	}
	if err := userModel().Where(notEmpty("delegated_ipv6_prefix")).Count(&stats.UsersWithDelegatedPrefix).Error; err != nil {
		logDashboardQueryError("fetch ipv6 user delegated prefix count", err)
	}

	if onlineTotal > 0 {
		stats.AdoptionRate = float64(stats.OnlineWithIPv6) / float64(onlineTotal) * 100
	}
	return stats
}

func startOfDay(t time.Time) time.Time {
	loc := t.Location()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func startOfHour(t time.Time) time.Time {
	loc := t.Location()
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
}

func logDashboardQueryError(action string, err error) {
	if err == nil {
		return
	}
	zap.L().Warn("dashboard query failed",
		zap.Error(err),
		zap.String("action", action),
		zap.String("namespace", "adminapi"))
}
