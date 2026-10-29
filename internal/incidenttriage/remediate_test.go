package incidenttriage

import "testing"

// TestVerdictCarriesNoActionField is the structural half of "no automatic
// remediate": the type has nowhere to put an instruction, so no engine can
// issue one and no caller can be handed one by accident.
func TestVerdictCarriesNoActionField(t *testing.T) {
	if got := actionableFields(); len(got) != 0 {
		t.Errorf("Verdict grew fields a remediation could ride on: %v", got)
	}
}

// TestTheVerbListIsMeasured checks the other half against real vocabulary: a
// verdict may not name an action the console itself accepts.
func TestTheVerbListIsMeasuredNotEmpty(t *testing.T) {
	r := testResult(t)
	if len(r.Actions) == 0 {
		t.Fatal("the admin API yielded no mutating verbs, so the no-action check would be vacuous")
	}
}

func TestNoVerdictNamesAMutatingVerb(t *testing.T) {
	r := testResult(t)
	for _, s := range r.Scores {
		for i, v := range r.Verdicts[s.Engine] {
			hits := namedActions(v.Cause+" "+v.Reason+" "+evidenceTokens(v), r.Actions)
			if len(hits) > 0 {
				t.Errorf("%s verdict %d names mutating verb %v", s.Engine, i, hits)
			}
		}
	}
	if n := actionNamesIn(r); n != 0 {
		t.Errorf("%d verdicts named an action", n)
	}
}

func TestMeasuredVerbsAreActionWords(t *testing.T) {
	for _, v := range actionSegmentTests {
		if got := actionSegment(v.path); got != v.want {
			t.Errorf("actionSegment(%q) = %q, want %q", v.path, got, v.want)
		}
	}
}

// actionSegmentTests pin the extractor to routes the admin API really
// registers, so a rename cannot silently empty the verb list. The negative rows
// are the shapes that must stay out: collections, plural tails, and a tail that
// is itself a parameter.
var actionSegmentTests = []struct {
	path string
	want string
}{
	{"/sessions/:id/disconnect", "disconnect"},
	{"/isp/payments/:id/reverse", "reverse"},
	{"/network/monitor-targets/:id/toggle", "toggle"},
	{"/network/monitor-targets/:id/samples", ""},
	{"/platform/tenants/:id/operators", ""},
	{"/isp/subscriptions/:id/:action", ""},
	{"/system/branding/reset", ""},
	{"/isp/packages", ""},
}
