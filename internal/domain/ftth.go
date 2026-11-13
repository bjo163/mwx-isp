package domain

import "time"

// NetworkAsset is a tenant-owned physical/logical asset in the FTTH and access
// network inventory. It complements ODP and NetNas legacy records; it does not
// silently replace them.
type NetworkAsset struct {
	ID                int64      `json:"id,string" gorm:"primaryKey"`
	TenantID          int64      `json:"-" gorm:"not null;default:1;index;uniqueIndex:udx_network_asset_tenant_code,priority:1;uniqueIndex:udx_network_asset_tenant_serial,priority:1,where:serial_number <> ''"`
	Code              string     `json:"code" gorm:"size:48;not null;uniqueIndex:udx_network_asset_tenant_code,priority:2"`
	Kind              string     `json:"kind" gorm:"size:24;not null;index"`
	Name              string     `json:"name" gorm:"size:150;not null;index"`
	ParentID          int64      `json:"parent_id,string,omitempty" gorm:"index"`
	Vendor            string     `json:"vendor" gorm:"size:64;index"`
	Model             string     `json:"model" gorm:"size:96"`
	SerialNumber      string     `json:"serial_number" gorm:"size:128;uniqueIndex:udx_network_asset_tenant_serial,priority:2"`
	LOID              string     `json:"loid,omitempty" gorm:"size:128;index"`
	MACAddress        string     `json:"mac_address,omitempty" gorm:"size:32;index"`
	ManagementAddress string    `json:"management_address,omitempty" gorm:"size:128"`
	FirmwareVersion   string     `json:"firmware_version,omitempty" gorm:"size:96"`
	LifecycleStatus   string     `json:"lifecycle_status" gorm:"size:24;not null;default:planned;index"`
	TotalPorts        int        `json:"total_ports"`
	UsedPorts         int        `json:"used_ports"`
	OpticalRxPowerDBm *float64   `json:"optical_rx_power_dbm,omitempty"`
	Latitude          float64    `json:"latitude"`
	Longitude         float64    `json:"longitude"`
	LegacyODPID       int64      `json:"legacy_odp_id,string,omitempty" gorm:"index"`
	CredentialRef     string     `json:"-" gorm:"size:255"`
	MetadataJSON      string     `json:"metadata_json,omitempty" gorm:"type:text"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty" gorm:"index"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TableName returns the FTTH network inventory table.
func (NetworkAsset) TableName() string { return "network_asset" }

// FiberLink records a physical or logical connection between two asset ports.
// The explicit endpoints allow the topology graph to be reconstructed and
// audited without inferring fiber routes from free-form ODP labels.
type FiberLink struct {
	ID              int64     `json:"id,string" gorm:"primaryKey"`
	TenantID        int64     `json:"-" gorm:"not null;default:1;index;uniqueIndex:udx_fiber_link_source,priority:1,where:status = 'active';uniqueIndex:udx_fiber_link_destination,priority:1,where:status = 'active'"`
	FromAssetID     int64     `json:"from_asset_id,string" gorm:"not null;index;uniqueIndex:udx_fiber_link_source,priority:2,where:status = 'active'"`
	FromPort        string    `json:"from_port" gorm:"size:64;not null;uniqueIndex:udx_fiber_link_source,priority:3,where:status = 'active'"`
	ToAssetID       int64     `json:"to_asset_id,string" gorm:"not null;index;uniqueIndex:udx_fiber_link_destination,priority:2,where:status = 'active'"`
	ToPort          string    `json:"to_port" gorm:"size:64;not null;uniqueIndex:udx_fiber_link_destination,priority:3,where:status = 'active'"`
	LinkType        string    `json:"link_type" gorm:"size:24;not null;default:fiber"`
	Status          string    `json:"status" gorm:"size:16;not null;default:active;index"`
	LengthMeters    float64   `json:"length_meters"`
	LossDB          float64   `json:"loss_db"`
	Notes           string    `json:"notes,omitempty" gorm:"size:500"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName returns the fiber topology edge table.
func (FiberLink) TableName() string { return "fiber_link" }

// SubscriberDeviceBinding connects an ISP service to its terminal network
// equipment. Existing Customer.ODPID/ODPPort fields remain authoritative until
// a migration explicitly creates a matching binding.
type SubscriberDeviceBinding struct {
	ID             int64      `json:"id,string" gorm:"primaryKey"`
	TenantID       int64      `json:"-" gorm:"not null;default:1;index;uniqueIndex:udx_subscriber_device_binding,priority:1,where:status = 'active'"`
	CustomerID     int64      `json:"customer_id,string" gorm:"not null;index;uniqueIndex:udx_subscriber_device_binding,priority:2,where:status = 'active'"`
	SubscriptionID int64      `json:"subscription_id,string" gorm:"not null;index"`
	AssetID        int64      `json:"asset_id,string" gorm:"not null;index"`
	Port           string     `json:"port" gorm:"size:64"`
	InstalledAt    *time.Time `json:"installed_at,omitempty"`
	RemovedAt      *time.Time `json:"removed_at,omitempty"`
	Status         string     `json:"status" gorm:"size:16;not null;default:active;index"`
	Notes          string     `json:"notes,omitempty" gorm:"size:500"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName returns the subscriber-to-equipment relationship table.
func (SubscriberDeviceBinding) TableName() string { return "subscriber_device_binding" }

// ProvisioningJob is an auditable desired-state request. New jobs default to
// dry-run and cannot be applied until a privileged operator approves them.
type ProvisioningJob struct {
	ID                int64      `json:"id,string" gorm:"primaryKey"`
	TenantID          int64      `json:"-" gorm:"not null;default:1;index;uniqueIndex:udx_provisioning_job_tenant_idem,priority:1"`
	AssetID           int64      `json:"asset_id,string" gorm:"not null;index"`
	CustomerID        int64      `json:"customer_id,string,omitempty" gorm:"index"`
	SubscriptionID    int64      `json:"subscription_id,string,omitempty" gorm:"index"`
	Operation         string     `json:"operation" gorm:"size:48;not null;index"`
	IdempotencyKey    string     `json:"idempotency_key" gorm:"size:96;not null;uniqueIndex:udx_provisioning_job_tenant_idem,priority:2"`
	DesiredStateJSON  string     `json:"desired_state_json" gorm:"type:text;not null"`
	ObservedStateJSON string     `json:"observed_state_json,omitempty" gorm:"type:text"`
	DryRun            bool       `json:"dry_run" gorm:"not null;default:true;index"`
	Status            string     `json:"status" gorm:"size:24;not null;default:planned;index"`
	AttemptCount      int        `json:"attempt_count"`
	ApprovedBy        string     `json:"approved_by,omitempty" gorm:"size:64"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	LastErrorCode     string     `json:"last_error_code,omitempty" gorm:"size:48"`
	LastError         string     `json:"last_error,omitempty" gorm:"size:500"`
	CreatedBy         string     `json:"created_by" gorm:"size:64"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TableName returns the idempotent provisioning job table.
func (ProvisioningJob) TableName() string { return "provisioning_job" }
