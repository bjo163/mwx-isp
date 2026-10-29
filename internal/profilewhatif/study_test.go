package profilewhatif

import (
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}

var (
	once    sync.Once
	runOnce *Result
	runErr  error
)

func studyResult(t *testing.T) *Result {
	t.Helper()
	once.Do(func() { runOnce, runErr = Run(repoRoot(t)) })
	require.NoError(t, runErr)
	return runOnce
}

// TestTheCriterionHolds is the backlog line for FUT-007, written as one assertion:
// the output diff is reproducible, and running it leaves the cache and the profile
// rows exactly as it found them.
func TestTheCriterionHolds(t *testing.T) {
	root := repoRoot(t)
	first, err := Run(root)
	require.NoError(t, err)
	second, err := Run(root)
	require.NoError(t, err)
	assert.Equal(t, first.Digest, second.Digest, "the same fixtures over the same code must give the same digest")

	assert.Zero(t, first.Totals.BannedHits, "the ban list must stay empty for this package")
}

// TestTheDigestSurvivesATickingClock is the reproducibility claim against the one
// input nobody can freeze: Session-Timeout counts down in whole seconds, so two runs
// a second apart must still agree.
func TestTheDigestSurvivesATickingClock(t *testing.T) {
	root := repoRoot(t)
	first, err := Run(root)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	second, err := Run(root)
	require.NoError(t, err)
	assert.Equal(t, first.Digest, second.Digest,
		"the digest may not carry a value that moves with the wall clock")
}

// TestRunningTheStudyLeavesTheInputsUnmoved is the criterion's second half, run
// against tables the test owns: the study clones what it edits, so the caller's
// profiles and subscribers come back byte for byte as they went in.
func TestRunningTheStudyLeavesTheInputsUnmoved(t *testing.T) {
	users := Fixtures()
	profiles := Profiles()
	usersBefore := deepCopyUsers(users)
	profilesBefore := deepCopyProfiles(profiles)

	_, err := run(repoRoot(t), users, profiles)
	require.NoError(t, err)

	assert.Equal(t, usersBefore, users, "a run moved a subscriber row")
	assert.Equal(t, profilesBefore, profiles, "a run moved a profile row")
}

// TestTheNonMutationAssertionBites proves the comparison above is not a tautology:
// the same edit applied straight to the shared table does change it, and is caught.
func TestTheNonMutationAssertionBites(t *testing.T) {
	users := Fixtures()
	profiles := Profiles()
	usersBefore := deepCopyUsers(users)
	profilesBefore := deepCopyProfiles(profiles)

	applyEdit(profiles[1], Edit{Field: "UpRate", Value: "1"})
	dirty := &users[0]
	dirty.UpRate = 999

	assert.NotEqual(t, profilesBefore, profiles, "applyEdit on the shared table must be visible")
	assert.NotEqual(t, usersBefore, users, "an edit through a pointer must be visible")

	cleanUsers, cleanProfiles := Fixtures(), Profiles()
	_, err := run(repoRoot(t), cleanUsers, cleanProfiles)
	require.NoError(t, err)
	assert.Equal(t, usersBefore, cleanUsers)
	assert.Equal(t, profilesBefore, cleanProfiles)
}

// TestTheEnhancersThemselvesWriteNothing closes the hole the clone leaves: here the
// product's own enhancer chain is handed pointers straight into the tables, so an
// enhancer that wrote through the subscriber or the cache row it was shown would be
// caught by this test rather than absorbed by a copy.
func TestTheEnhancersThemselvesWriteNothing(t *testing.T) {
	s := newStudy(repoRoot(t))
	users := Fixtures()
	profiles := Profiles()
	usersBefore := deepCopyUsers(users)
	profilesBefore := deepCopyProfiles(profiles)
	view := newReadOnlyCache(profiles)

	for i := range users {
		for _, code := range vendorCodes() {
			s.packet(&users[i], code, view)
		}
	}

	assert.Equal(t, usersBefore, users, "an enhancer wrote through the subscriber pointer")
	assert.Equal(t, profilesBefore, profiles, "an enhancer wrote a profile row it was only shown")
}

func deepCopyUsers(in []User) []User {
	out := make([]User, len(in))
	copy(out, in)
	return out
}

func deepCopyProfiles(in map[int64]*domain.RadiusProfile) map[int64]*domain.RadiusProfile {
	out := make(map[int64]*domain.RadiusProfile, len(in))
	for id, p := range in {
		c := *p
		out[id] = &c
	}
	return out
}

// TestTheOnlyCacheShapeTheGettersAccept is the read-only guarantee turned inside
// out: the product's getters ask for an interface with one method, and that is the
// whole surface the prototype can be asked to implement.
func TestTheOnlyCacheShapeTheGettersAccept(t *testing.T) {
	var getterType = reflect.TypeOf((*domain.ProfileCacheGetter)(nil)).Elem()
	require.Equal(t, 1, getterType.NumMethod(), "the product's contract is one read method")
	assert.Equal(t, "Get", getterType.Method(0).Name)

	c := newReadOnlyCache(Profiles())
	impl := reflect.TypeOf(c)
	assert.Equal(t, 1, impl.NumMethod(), "the injected cache exposes exactly the contract")
	assert.Implements(t, (*domain.ProfileCacheGetter)(nil), c)

	p, err := c.Get(1)
	require.NoError(t, err)
	assert.Equal(t, *Profiles()[1], *p, "the getter must be handed the row it asked for")
	assert.Equal(t, 1, c.gets)

	_, err = c.Get(4242)
	assert.Error(t, err, "an unknown profile id must fail the way a database miss does")
}

// TestStaticModeAsksTheCacheNothing separates the two link modes by their measured
// cost instead of by their definition.
func TestStaticModeAsksTheCacheNothing(t *testing.T) {
	root := repoRoot(t)
	profiles := Profiles()
	s := newStudy(root)

	dyn := findUser(Fixtures(), "dyn-full")
	require.NotNil(t, dyn)
	dynamicRun := s.packet(cloneUser(dyn), vendorCodes()[0], newReadOnlyCache(profiles))
	assert.Positive(t, dynamicRun.lookups, "a dynamically linked subscriber resolves through the cache")

	stat := findUser(Fixtures(), "static-snapshot")
	require.NotNil(t, stat)
	staticRun := s.packet(cloneUser(stat), vendorCodes()[0], newReadOnlyCache(profiles))
	assert.Zero(t, staticRun.lookups, "a snapshot subscriber never asks the cache a question")
}

func newStudy(root string) *study {
	s := &study{root: root, roster: enhancerRoster(root)}
	s.dict = vendorDict(root)
	s.std = stdNames()
	s.names = map[string]string{}
	for k, v := range s.dict {
		s.names[k] = v
	}
	for k, v := range s.std {
		s.names[k] = v
	}
	s.unstable = map[string]bool{}
	return s
}

// TestThePrototypeCannotSendAnything is the boundary claim, measured on the package
// itself: no database, no HTTP stack, no management API, no accounting plugin, and
// not even the package that declares the CoA and Disconnect clients.
func TestThePrototypeCannotSendAnything(t *testing.T) {
	res := studyResult(t)
	assert.Zero(t, res.Totals.BannedHits, "imports: %v", res.Imports)
	assert.Zero(t, res.Totals.SendCalls, "the wire library may build packets but must never be asked to send one")

	for _, imp := range res.Imports {
		if strings.HasPrefix(imp, "github.com/bjo163/mwx-isp/internal/") {
			assert.NotContains(t, imp, "/adminapi")
			assert.NotContains(t, imp, "/app")
			assert.NotEqual(t, "github.com/bjo163/mwx-isp/internal/radiusd", imp,
				"the CoA-declaring package must stay out of the import set")
		}
	}
	assert.NotEmpty(t, res.Totals.CoAPackage)
	assert.Contains(t, res.Totals.CoAPackage, "radiusd")
}

// TestNothingImportsThePrototype keeps the prototype out of the product.
func TestNothingImportsThePrototype(t *testing.T) {
	root := repoRoot(t)
	hits := countSubstring(corpus(productFiles(root)), ownPackage)
	assert.Zero(t, hits, "a research package the product imports stops being research")
}

// TestTheDiffCarriesNoAction is the structural half of "this is a preview, not a
// change": nothing in the diff types is shaped like an instruction to a NAS.
func TestTheDiffCarriesNoAction(t *testing.T) {
	actiony := []string{"Action", "Command", "Apply", "Enforce", "Revalidate", "Reauthorize",
		"Disconnect", "CoA", "Kick", "Suspend", "Delete", "Disable", "Send"}
	for _, typ := range []any{Change{}, Outcome{}, FieldEffect{}, Attr{}, Edit{}, Scenario{}} {
		rt := reflect.TypeOf(typ)
		for i := 0; i < rt.NumField(); i++ {
			name := rt.Field(i).Name
			for _, verb := range actiony {
				assert.NotContains(t, name, verb,
					"%s.%s reads like an instruction to a NAS", rt.Name(), name)
			}
		}
	}
}

// TestEveryVendorAttributeIsNamedByTheProduct keeps the naming honest: an attribute
// only gets a name if one of the product's own generated dictionaries says so.
func TestEveryVendorAttributeIsNamedByTheProduct(t *testing.T) {
	res := studyResult(t)
	names := res.Names
	require.NotEmpty(t, names)
	total, named := 0, 0
	for _, o := range res.Outcomes {
		for _, a := range append(append([]Attr{}, o.Before...), o.After...) {
			if a.Vendor == 0 {
				continue
			}
			total++
			if a.Name != "" {
				named++
			}
		}
	}
	assert.Positive(t, total)
	assert.Equal(t, total, named, "a vendor attribute reached the ledger with no dictionary name")
}

// TestTheLedgerDiffusesNothingClockDriven protects the reproducibility claim: an
// attribute whose value comes from the expiry column or from tenant config is never
// part of a diff.
func TestTheLedgerDiffusesNothingClockDriven(t *testing.T) {
	res := studyResult(t)
	require.NotEmpty(t, res.HeldOut)
	held := map[string]bool{}
	for _, a := range res.HeldOut {
		held[a.Key] = true
	}
	require.True(t, held["type-27"],
		"moving only the expiry column must flag Session-Timeout, or this test guards nothing")

	for _, o := range res.Outcomes {
		for _, a := range append(append([]Attr{}, o.Before...), o.After...) {
			assert.True(t, a.Stable, "%s is held out of the ledger and must not appear", a.Key)
			assert.False(t, held[a.Key], "%s is held out and still reached the ledger", a.Key)
		}
		for _, c := range o.Changes {
			assert.False(t, held[c.Key], "%s moves with the clock or the tenant config, not the profile", c.Key)
		}
		for _, a := range o.HeldOut {
			assert.True(t, held[a.Key], "%s left the study-wide held-out set", a.Key)
		}
	}
	for _, c := range res.ConfigReads {
		if c.Matched {
			assert.True(t, held[c.Key],
				"%s emitted the source's own fallback, so a tenant config could change it", c.Key)
		}
	}
}

// TestCheckerOnlyColumnsMoveNothing is the finding that bounds what a preview can
// promise: three of the ten columns change acceptance, not the accept.
func TestCheckerOnlyColumnsMoveNothing(t *testing.T) {
	root := repoRoot(t)
	only := checkerOnlyFields(root)
	assert.Equal(t, []string{"ActiveNum", "BindMac", "BindVlan"}, only)

	res := studyResult(t)
	for _, e := range res.Effects {
		assert.NotContains(t, only, e.Field, "%s moved an attribute, so the scan is wrong", e.Field)
	}
	for _, o := range res.Outcomes {
		if o.Scenario == "S16" || o.Scenario == "S17" || o.Scenario == "S18" {
			assert.Empty(t, o.Changes, "%s must be an empty diff", o.Scenario)
		}
	}
}

// TestPrecedenceScenariosBehave checks the three rules an operator is most likely to
// be surprised by, each against the product's own answer.
func TestPrecedenceScenariosBehave(t *testing.T) {
	res := studyResult(t)
	byID := map[string]Outcome{}
	for _, o := range res.Outcomes {
		byID[o.Scenario] = o
	}

	override, ok := byID["S07"]
	require.True(t, ok)
	assert.Empty(t, override.Changes, "a user that sets its own rate must ignore the profile edit")

	detached, ok := byID["S14"]
	require.True(t, ok)
	assert.Empty(t, detached.Changes, "a snapshot must freeze the answer at the moment of detaching")

	lost, ok := byID["S15"]
	require.True(t, ok)
	assert.NotEmpty(t, lost.Changes, "an unresolvable profile must fall back to the stale user row")

	inherits, ok := byID["S08"]
	require.True(t, ok)
	assert.NotEmpty(t, inherits.Changes, "the NA sentinel means unset, so the subscriber follows the profile")

	assert.Len(t, res.Outcomes, len(Scenarios()), "every scenario must resolve to a fixture user")
}

// TestTheLedgerShowsWhatASuccessfulEditDoes is the smallest useful end-to-end check:
// one upload rate edit on a Mikrotik NAS moves exactly the rate attribute, and the
// value is the one the product's own arithmetic produces.
func TestTheLedgerShowsWhatASuccessfulEditDoes(t *testing.T) {
	res := studyResult(t)
	var o *Outcome
	for i := range res.Outcomes {
		if res.Outcomes[i].Scenario == "S01" {
			o = &res.Outcomes[i]
		}
	}
	require.NotNil(t, o)
	require.Len(t, o.Changes, 1)
	assert.Equal(t, "14988-8", o.Changes[0].Key)
	assert.Equal(t, "33333k/5120k", o.Changes[0].After)
	assert.Equal(t, "10240k/5120k", o.Changes[0].Before)
}

// TestEveryStatementIsPinnedToCode refuses a claim that no longer lands.
func TestEveryStatementIsPinnedToCode(t *testing.T) {
	res := studyResult(t)
	for _, c := range res.Claims {
		assert.Equal(t, "confirmed", c.Status(), "%s: %s (%s:%d)", c.ID, c.Statement, c.File, c.Line)
	}
	assert.Equal(t, len(res.Claims), res.Totals.ClaimsConfirmed,
		"the report's claim count must equal the claims it could pin")
	assert.NotEmpty(t, res.Gaps)
}

// TestTheGapsSayWhatThisDoesNotClose keeps the gap register honest: each one carries
// a measurement, not an apology.
func TestTheGapsSayWhatThisDoesNotClose(t *testing.T) {
	res := studyResult(t)
	open := map[string]bool{}
	for _, g := range res.Gaps {
		if g.Open {
			open[g.ID] = true
		}
		assert.NotEmpty(t, g.Evidence, "%s has no measurement behind it", g.ID)
	}
	assert.True(t, open["G01"], "the management API preview is still missing")
	assert.True(t, open["G02"], "the untested inheritance path is still untested")
	assert.True(t, open["G05"], "no NAS is reachable from CI")
}

// TestLookupsScaleWithInheritedColumns measures the cost the criterion never
// mentions: how many cache reads one Access-Accept actually performs.
func TestLookupsScaleWithInheritedColumns(t *testing.T) {
	res := studyResult(t)
	// The two scans meet: one accept resolves every inherited column except the ones
	// only the checkers read, so the measured cost cross-checks both counts.
	assert.Equal(t, len(Fields())-len(checkerOnlyFields(repoRoot(t))), res.Totals.MaxLookups,
		"a fully inherited subscriber resolves each wire-bound column once through the cache")
	assert.Greater(t, res.Totals.Lookups, res.Totals.MaxLookups)
}
