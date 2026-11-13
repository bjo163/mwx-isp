package provisioning

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeAdapter struct {
	applied int
	vendor string
}
func (f *fakeAdapter) Vendor() string { return f.vendor }
func (f *fakeAdapter) Capabilities() []Capability { return []Capability{CapabilityDiscover, CapabilityObserve, CapabilityVLAN} }
func (f *fakeAdapter) Discover(context.Context, Device) (Snapshot, error) { return Snapshot{Reachable:true}, nil }
func (f *fakeAdapter) Plan(_ context.Context, in Request) (ChangePlan, error) { return ChangePlan{Device:in.Device,Operation:in.Operation,IdempotencyKey:in.IdempotencyKey,DryRun:in.DryRun,RequiresApproval:true},nil }
func (f *fakeAdapter) Apply(context.Context, ChangePlan) (ApplyResult, error) { f.applied++; return ApplyResult{Accepted:true,JobReference:"fake-1"},nil }

func TestRegistryNormalizesVendorAndRejectsDuplicate(t *testing.T) {
	r := NewRegistry()
	a := &fakeAdapter{vendor:"  MiKrOtIk "}
	require.NoError(t,r.Register(a))
	got,err := r.Get("mikrotik")
	require.NoError(t,err)
	require.Same(t,a,got)
	require.Equal(t,[]string{"mikrotik"},r.Vendors())
	require.ErrorIs(t,r.Register(&fakeAdapter{vendor:"mikrotik"}),ErrDuplicateAdapter)
}

func TestApplyPlanRequiresApprovedNonDryRun(t *testing.T) {
	a := &fakeAdapter{vendor:"test"}
	_,err := ApplyPlan(context.Background(),a,ChangePlan{DryRun:true},"admin")
	require.ErrorIs(t,err,ErrDryRunOnly)
	_,err = ApplyPlan(context.Background(),a,ChangePlan{DryRun:false,RequiresApproval:true},"")
	require.ErrorIs(t,err,ErrApprovalRequired)
	require.Equal(t,0,a.applied)
	res,err := ApplyPlan(context.Background(),a,ChangePlan{DryRun:false,RequiresApproval:true},"operator-1")
	require.NoError(t,err)
	require.True(t,res.Accepted)
	require.Equal(t,1,a.applied)
}

func TestValidateLinkInput(t *testing.T) {
	require.NoError(t,ValidateLinkInput(1,"pon-1",2,"in-1","fiber"))
	require.Error(t,ValidateLinkInput(1,"",2,"in-1","fiber"))
	require.Error(t,ValidateLinkInput(1,"p1",1,"p2","fiber"))
	require.Error(t,ValidateLinkInput(1,"p1",2,"p2","unknown"))
	require.True(t,errors.Is(ErrApprovalRequired,ErrApprovalRequired))
}
