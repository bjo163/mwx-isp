package app

import (
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestEnsureCustomerODPPortIndex verifies the partial unique index closes the
// check-then-insert race for ODP ports while still letting terminated customers
// release their port and exempting the "no numbered port" value.
func TestEnsureCustomerODPPortIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Customer{}))

	a := &Application{gormDB: db}
	a.ensureCustomerODPPortIndex()

	customer := func(suffix string, tenantID, odpID int64, port int, status string) *domain.Customer {
		return &domain.Customer{
			TenantID: tenantID, CustomerNo: "C-" + suffix, Name: "Customer " + suffix,
			ODPID: odpID, ODPPort: port, Status: status,
		}
	}
	require.NoError(t, db.Create(customer("live", 1, 7, 3, domain.CustomerActive)).Error)

	err = db.Create(customer("race", 1, 7, 3, domain.CustomerActive)).Error
	require.Error(t, err, "a second live customer on the same ODP port must be rejected")

	require.NoError(t, db.Create(customer("released", 1, 7, 3, domain.CustomerTerminated)).Error,
		"terminated customers release their port")
	require.NoError(t, db.Create(customer("unported", 1, 7, 0, domain.CustomerActive)).Error,
		"port zero means no numbered port and is exempt")
	require.NoError(t, db.Create(customer("other-tenant", 2, 7, 3, domain.CustomerActive)).Error,
		"each tenant has its own ODP capacity")

	var indexCount int64
	require.NoError(t, db.Raw(
		"SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND name = 'udx_isp_customer_odp_port'",
	).Scan(&indexCount).Error)
	require.Equal(t, int64(1), indexCount)
}
