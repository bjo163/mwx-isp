package revenue

import (
	"time"

	"gorm.io/gorm"
)

// Load reads the billing and access rows an audit needs and writes nothing.
// It reads every tenant in the handle it is given; Audit partitions rows by
// tenant before applying any rule, so a finding never quotes another tenant's
// data. A caller that must stay inside one tenant passes a tenant-scoped handle,
// exactly as the billing jobs do.
//
// The rows are read without ORDER BY because auditTenant sorts each slice before
// indexing it: the report cannot depend on how the database chose to return them.
func Load(db *gorm.DB) (Input, error) {
	var input Input
	if err := db.Find(&input.Subscriptions).Error; err != nil {
		return input, err
	}
	if err := db.Find(&input.Packages).Error; err != nil {
		return input, err
	}
	if err := db.Find(&input.Invoices).Error; err != nil {
		return input, err
	}
	if err := db.Find(&input.Payments).Error; err != nil {
		return input, err
	}
	if err := db.Find(&input.Users).Error; err != nil {
		return input, err
	}
	if err := db.Find(&input.Online).Error; err != nil {
		return input, err
	}
	return input, nil
}

// AuditDatabase loads one ledger from a database and audits it in one step.
func AuditDatabase(db *gorm.DB, now time.Time) (Report, error) {
	input, err := Load(db)
	if err != nil {
		return Report{}, err
	}
	return Audit(input, now), nil
}
