package adminapi

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/domain"
)

// diagnosticTarget is a validated destination for a reachability probe.
type diagnosticTarget struct {
	Address string
	Port    string
}

// diagnosticTargetError is a validation refusal that maps to one API error code.
type diagnosticTargetError struct {
	Code    string
	Message string
}

func (e *diagnosticTargetError) Error() string { return e.Message }

func rejectTarget(code, message string) error {
	return &diagnosticTargetError{Code: code, Message: message}
}

var sharedAddressSpace = mustCIDR("100.64.0.0/10")

func mustCIDR(value string) *net.IPNet {
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		panic(err)
	}
	return network
}

// parseProbeTarget accepts only an explicit IP literal with an optional port.
// A hostname is refused instead of resolved: the name could resolve to a
// different address than the one checked here (DNS rebinding), and a
// reachability probe is not a name resolver.
func parseProbeTarget(raw string) (diagnosticTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return diagnosticTarget{}, rejectTarget("HOST_REQUIRED", "Target host or IP address is required")
	}
	host, port := raw, "80"
	if splitHost, splitPort, err := net.SplitHostPort(raw); err == nil {
		host, port = splitHost, splitPort
	} else if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		host = strings.Trim(raw, "[]")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return diagnosticTarget{}, rejectTarget("TARGET_NOT_IP",
			"Target must be an explicit IPv4 or IPv6 address; hostnames are not resolved")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return diagnosticTarget{}, rejectTarget("INVALID_PORT", "Target port must be a number between 1 and 65535")
	}
	if err := rejectSpecialUseAddress(ip); err != nil {
		return diagnosticTarget{}, err
	}
	return diagnosticTarget{Address: ip.String(), Port: strconv.Itoa(portNumber)}, nil
}

// rejectSpecialUseAddress refuses destinations that are never a reachable
// subscriber or device, and that are the usual target of an SSRF attempt
// (link-local metadata, loopback services, broadcast/multicast).
func rejectSpecialUseAddress(ip net.IP) error {
	switch {
	case ip.IsUnspecified():
		return rejectTarget("TARGET_NOT_ALLOWED", "The unspecified address cannot be probed")
	case ip.IsLoopback():
		return rejectTarget("TARGET_NOT_ALLOWED", "Loopback addresses cannot be probed")
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast(), ip.IsInterfaceLocalMulticast():
		return rejectTarget("TARGET_NOT_ALLOWED", "Link-local addresses cannot be probed")
	case ip.IsMulticast():
		return rejectTarget("TARGET_NOT_ALLOWED", "Multicast addresses cannot be probed")
	}
	return nil
}

// isPublicUnicast reports whether ip is globally routable, i.e. not one of the
// private or shared (CGNAT) ranges an ISP reserves for its own network.
func isPublicUnicast(ip net.IP) bool {
	return !ip.IsPrivate() && !sharedAddressSpace.Contains(ip)
}

// registeredProbeTarget reports whether ip belongs to the tenant's inventory:
// an IPAM pool, a NAS address, or a registered monitor target. The caller's
// database handle carries the tenant scope.
func registeredProbeTarget(db *gorm.DB, ip net.IP) (bool, error) {
	var pools []domain.IPAMPool
	if err := db.Find(&pools).Error; err != nil {
		return false, err
	}
	for _, pool := range pools {
		if _, network, err := net.ParseCIDR(strings.TrimSpace(pool.CIDR)); err == nil && network.Contains(ip) {
			return true, nil
		}
	}
	var routers []domain.NetNas
	if err := db.Find(&routers).Error; err != nil {
		return false, err
	}
	for _, router := range routers {
		if sameAddress(router.Ipaddr, ip) {
			return true, nil
		}
	}
	var monitorTargets []domain.NetMonitorTarget
	if err := db.Find(&monitorTargets).Error; err != nil {
		return false, err
	}
	for _, target := range monitorTargets {
		if sameAddress(target.Address, ip) {
			return true, nil
		}
	}
	return false, nil
}

func sameAddress(candidate string, ip net.IP) bool {
	parsed := net.ParseIP(strings.TrimSpace(candidate))
	return parsed != nil && parsed.Equal(ip)
}

// authorizeProbeTarget enforces the diagnostics policy: public unicast targets
// may be probed directly, while private, CGNAT and unique-local addresses must
// be part of the tenant's registered inventory so the endpoint cannot be used
// to sweep an unrelated network.
func authorizeProbeTarget(db *gorm.DB, ip net.IP) error {
	if isPublicUnicast(ip) {
		return nil
	}
	registered, err := registeredProbeTarget(db, ip)
	if err != nil {
		return fmt.Errorf("check registered probe target: %w", err)
	}
	if !registered {
		return rejectTarget("TARGET_NOT_REGISTERED",
			"Private targets must belong to an IPAM pool, a NAS record, or a registered monitor target")
	}
	return nil
}

// failDiagnosticTarget maps a validation refusal to an API response.
func failDiagnosticTarget(c echo.Context, err error) error {
	var targetErr *diagnosticTargetError
	if errors.As(err, &targetErr) {
		status := 400
		if targetErr.Code == "TARGET_NOT_REGISTERED" {
			status = 403
		}
		return fail(c, status, targetErr.Code, targetErr.Message, nil)
	}
	return fail(c, 500, "DATABASE_ERROR", "Failed to validate the probe target", nil)
}
