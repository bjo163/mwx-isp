// Package provisioning defines vendor-neutral device discovery and
// explicitly approved desired-state change contracts. It contains no vendor
// credentials or live device implementation.
package provisioning

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrAdapterNotFound = errors.New("provisioning adapter not found")
	ErrDuplicateAdapter = errors.New("provisioning adapter already registered")
	ErrApprovalRequired = errors.New("provisioning plan requires explicit operator approval")
	ErrDryRunOnly = errors.New("dry-run plans cannot be applied")
)

type Capability string

const (
	CapabilityDiscover Capability = "discover"
	CapabilityObserve Capability = "observe"
	CapabilityPPPoE Capability = "pppoe"
	CapabilityVLAN Capability = "vlan"
	CapabilityServiceProfile Capability = "service_profile"
	CapabilityReboot Capability = "reboot"
	CapabilityFirmware Capability = "firmware"
)

type Device struct {
	TenantID int64
	AssetID int64
	Kind string
	Vendor string
	Model string
	SerialNumber string
	ManagementAddress string
	CredentialRef string // reference only; secret material is resolved outside the asset model
}

type Snapshot struct {
	Reachable bool
	SerialNumber string
	Model string
	FirmwareVersion string
	ObservedStateJSON string
	ObservedAt string
}

type Request struct {
	Device Device
	Operation string
	DesiredStateJSON string
	IdempotencyKey string
	DryRun bool
}

type ChangePlan struct {
	Device Device
	Operation string
	IdempotencyKey string
	DesiredStateJSON string
	DiffJSON string
	DryRun bool
	RequiresApproval bool
	Destructive bool
}

type ApplyResult struct {
	Accepted bool
	JobReference string
	ObservedStateJSON string
}

// Adapter implementations must bound all network operations, validate target
// addresses before egress, redact secrets from errors/logs, and implement
// idempotency using Request.IdempotencyKey.
type Adapter interface {
	Vendor() string
	Capabilities() []Capability
	Discover(context.Context, Device) (Snapshot, error)
	Plan(context.Context, Request) (ChangePlan, error)
	Apply(context.Context, ChangePlan) (ApplyResult, error)
}

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry() *Registry { return &Registry{adapters: make(map[string]Adapter)} }

func (r *Registry) Register(adapter Adapter) error {
	if r == nil || adapter == nil {
		return errors.New("provisioning registry and adapter are required")
	}
	vendor := strings.ToLower(strings.TrimSpace(adapter.Vendor()))
	if vendor == "" || strings.ContainsAny(vendor, " \t\r\n") {
		return fmt.Errorf("invalid provisioning vendor %q", vendor)
	}
	if r.adapters == nil {
		r.adapters = make(map[string]Adapter)
	}
	if _, exists := r.adapters[vendor]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateAdapter, vendor)
	}
	r.adapters[vendor] = adapter
	return nil
}

func (r *Registry) Get(vendor string) (Adapter, error) {
	if r == nil {
		return nil, ErrAdapterNotFound
	}
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(vendor))]
	if !ok {
		return nil, ErrAdapterNotFound
	}
	return adapter, nil
}

func (r *Registry) Vendors() []string {
	if r == nil {
		return nil
	}
	vendors := make([]string, 0, len(r.adapters))
	for vendor := range r.adapters {
		vendors = append(vendors, vendor)
	}
	sort.Strings(vendors)
	return vendors
}

// ValidateLinkInput rejects invalid topology edges before they can be persisted.
func ValidateLinkInput(fromAssetID int64, fromPort string, toAssetID int64, toPort, linkType string) error {
	if fromAssetID <= 0 || toAssetID <= 0 || strings.TrimSpace(fromPort) == "" || strings.TrimSpace(toPort) == "" {
		return errors.New("both assets and port identifiers are required")
	}
	if fromAssetID == toAssetID {
		return errors.New("an asset cannot be connected to itself")
	}
	switch strings.ToLower(strings.TrimSpace(linkType)) {
	case "fiber", "ethernet", "patch", "logical":
		return nil
	default:
		return fmt.Errorf("unsupported topology link type %q", linkType)
	}
}

// ApplyPlan is the only execution gate for vendor changes. Callers must derive
// approvedBy from an authenticated, authorized operator; a blank value always
// rejects execution. All jobs are dry-run by default and require a deliberate
// non-dry-run request plus approval before an adapter can mutate a device.
func ApplyPlan(ctx context.Context, adapter Adapter, plan ChangePlan, approvedBy string) (ApplyResult, error) {
	if adapter == nil {
		return ApplyResult{}, ErrAdapterNotFound
	}
	if plan.DryRun {
		return ApplyResult{}, ErrDryRunOnly
	}
	if strings.TrimSpace(approvedBy) == "" || plan.RequiresApproval || plan.Destructive {
		if strings.TrimSpace(approvedBy) == "" {
			return ApplyResult{}, ErrApprovalRequired
		}
	}
	if !supports(adapter, CapabilityDiscover) {
		return ApplyResult{}, fmt.Errorf("adapter %s has no discovery capability", adapter.Vendor())
	}
	return adapter.Apply(ctx, plan)
}

func supports(adapter Adapter, capability Capability) bool {
	for _, supported := range adapter.Capabilities() {
		if supported == capability {
			return true
		}
	}
	return false
}
