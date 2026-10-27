package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// subscriptionFixture owns one isolated database with a customer, a linked
// RADIUS account, and a subscription in the requested state. The linked account
// starts enabled only when the subscription starts active, mirroring
// createSubscription.
type subscriptionFixture struct {
	t *testing.T

	db           *gorm.DB
	e            *echo.Echo
	appCtx       app.AppContext
	customer     domain.Customer
	user         domain.RadiusUser
	sub          domain.Subscription
	initialUser  string
	initialState string
}

func newSubscriptionFixture(t *testing.T, status string) *subscriptionFixture {
	t.Helper()
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.Subscription{}, &domain.RadiusUser{}, &domain.BillingEvent{},
	))
	userStatus := "disabled"
	if status == domain.SubscriptionActive {
		userStatus = "enabled"
	}
	f := &subscriptionFixture{
		t: t, db: db, e: e, appCtx: appCtx,
		initialUser: userStatus, initialState: status,
	}
	f.customer = domain.Customer{
		TenantID: domain.DefaultTenantID, CustomerNo: "CUS-STATE",
		Name: "State Machine", Status: domain.CustomerActive,
	}
	require.NoError(t, db.Create(&f.customer).Error)
	f.user = domain.RadiusUser{
		TenantID: domain.DefaultTenantID, Username: "state-machine-user",
		Password: "secret", Status: userStatus,
	}
	require.NoError(t, db.Create(&f.user).Error)
	f.sub = domain.Subscription{
		TenantID: domain.DefaultTenantID, SubscriptionNo: "SUB-STATE-MACHINE",
		CustomerID: f.customer.ID, RadiusUserID: f.user.ID, Status: status,
	}
	require.NoError(t, db.Create(&f.sub).Error)
	return f
}

func (f *subscriptionFixture) action(action string) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/isp/subscriptions/%d/%s", f.sub.ID, action), nil)
	rec := httptest.NewRecorder()
	c := CreateTestContext(f.e, f.db, req, rec, f.appCtx)
	c.Set("tenant_id", domain.DefaultTenantID)
	c.SetParamNames("id", "action")
	c.SetParamValues(fmt.Sprint(f.sub.ID), action)
	require.NoError(f.t, subscriptionAction(c))
	return rec
}

// reload reads the subscription, its linked account, and every billing event
// for that subscription. A partial write shows up as a mismatch between them.
func (f *subscriptionFixture) reload() (domain.Subscription, domain.RadiusUser, []domain.BillingEvent) {
	f.t.Helper()
	var sub domain.Subscription
	require.NoError(f.t, f.db.First(&sub, f.sub.ID).Error)
	var user domain.RadiusUser
	require.NoError(f.t, f.db.First(&user, f.user.ID).Error)
	var events []domain.BillingEvent
	require.NoError(f.t, f.db.Where("subscription_id = ?", f.sub.ID).Order("id").Find(&events).Error)
	return sub, user, events
}

// TestSubscriptionStateMachineMatrix drives every state/action pair through the
// real handler. Accepted transitions must keep "active subscription <=> enabled
// account" and append exactly one billing event; rejected transitions must leave
// the subscription, the account, and the event log untouched.
func TestSubscriptionStateMachineMatrix(t *testing.T) {
	type expected struct {
		code   int
		status string
		reason string
		user   string
		event  string
	}
	valid := func(status, reason, user, event string) expected {
		return expected{code: http.StatusOK, status: status, reason: reason, user: user, event: event}
	}
	rejected := expected{code: http.StatusConflict}

	cases := []struct {
		name   string
		from   string
		action string
		want   expected
	}{
		{"pending activate", domain.SubscriptionPending, "activate", valid(domain.SubscriptionActive, "", "enabled", "subscription_activated")},
		{"pending suspend", domain.SubscriptionPending, "suspend", rejected},
		{"pending reactivate", domain.SubscriptionPending, "reactivate", rejected},
		{"pending terminate", domain.SubscriptionPending, "terminate", valid(domain.SubscriptionTerminated, "manual", "disabled", "subscription_terminated")},

		{"active activate", domain.SubscriptionActive, "activate", valid(domain.SubscriptionActive, "", "enabled", "subscription_activated")},
		{"active suspend", domain.SubscriptionActive, "suspend", valid(domain.SubscriptionSuspended, "manual", "disabled", "subscription_suspended")},
		{"active reactivate", domain.SubscriptionActive, "reactivate", valid(domain.SubscriptionActive, "", "enabled", "subscription_activated")},
		{"active terminate", domain.SubscriptionActive, "terminate", valid(domain.SubscriptionTerminated, "manual", "disabled", "subscription_terminated")},

		{"suspended activate", domain.SubscriptionSuspended, "activate", rejected},
		{"suspended suspend", domain.SubscriptionSuspended, "suspend", valid(domain.SubscriptionSuspended, "manual", "disabled", "subscription_suspended")},
		{"suspended reactivate", domain.SubscriptionSuspended, "reactivate", valid(domain.SubscriptionActive, "", "enabled", "subscription_activated")},
		{"suspended terminate", domain.SubscriptionSuspended, "terminate", valid(domain.SubscriptionTerminated, "manual", "disabled", "subscription_terminated")},

		{"terminated activate", domain.SubscriptionTerminated, "activate", rejected},
		{"terminated suspend", domain.SubscriptionTerminated, "suspend", rejected},
		{"terminated reactivate", domain.SubscriptionTerminated, "reactivate", rejected},
		{"terminated terminate", domain.SubscriptionTerminated, "terminate", rejected},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newSubscriptionFixture(t, tc.from)
			rec := f.action(tc.action)
			require.Equal(t, tc.want.code, rec.Code, rec.Body.String())

			sub, user, events := f.reload()
			if tc.want.code != http.StatusOK {
				var body ErrorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, "INVALID_SUBSCRIPTION_STATE", body.Error)
				assert.Equal(t, f.initialState, sub.Status, "a rejected request must not change the subscription")
				assert.Equal(t, f.initialUser, user.Status, "a rejected request must not change the account")
				assert.Empty(t, events, "a rejected request must not append a billing event")
				return
			}

			assert.Equal(t, tc.want.status, sub.Status)
			assert.Equal(t, tc.want.reason, sub.SuspensionReason)
			assert.Equal(t, tc.want.user, user.Status, "subscription and account state must stay consistent")
			require.Len(t, events, 1, "an accepted transition appends exactly one billing event")
			assert.Equal(t, tc.want.event, events[0].Type)
		})
	}
}

func TestSubscriptionUnknownActionIsRejectedWithoutWrites(t *testing.T) {
	f := newSubscriptionFixture(t, domain.SubscriptionActive)
	rec := f.action("explode")
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	var body ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "UNKNOWN_ACTION", body.Error)

	sub, user, events := f.reload()
	assert.Equal(t, domain.SubscriptionActive, sub.Status)
	assert.Equal(t, "enabled", user.Status)
	assert.Empty(t, events)
}

// TestSubscriptionTransitionRollsBackOnDatabaseFailure injects a write failure
// inside the transition transaction. The subscription update runs first, the
// linked account second, and the billing event last, so each case checks that
// the earlier writes are rolled back rather than left half-applied.
func TestSubscriptionTransitionRollsBackOnDatabaseFailure(t *testing.T) {
	triggers := []struct {
		name    string
		trigger string
	}{
		{
			"account update fails after the subscription update",
			`CREATE TRIGGER inject_radius_user_failure BEFORE UPDATE ON radius_user
			 BEGIN SELECT RAISE(ABORT, 'injected account failure'); END;`,
		},
		{
			"billing event insert fails after both updates",
			`CREATE TRIGGER inject_billing_event_failure BEFORE INSERT ON isp_billing_event
			 BEGIN SELECT RAISE(ABORT, 'injected billing event failure'); END;`,
		},
	}

	for _, tc := range triggers {
		t.Run(tc.name, func(t *testing.T) {
			f := newSubscriptionFixture(t, domain.SubscriptionActive)
			require.NoError(t, f.db.Exec(tc.trigger).Error)

			rec := f.action("suspend")
			require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())

			var body ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "DATABASE_ERROR", body.Error)

			sub, user, events := f.reload()
			assert.Equal(t, domain.SubscriptionActive, sub.Status, "the subscription update must roll back")
			assert.Empty(t, sub.SuspensionReason)
			assert.Equal(t, "enabled", user.Status, "the account update must roll back too")
			assert.Empty(t, events, "no billing event may survive a failed transaction")
		})
	}
}

// createSubscriptionFixture seeds the rows createSubscription needs: a customer,
// a RADIUS profile, and a package that points at that profile.
type createSubscriptionFixture struct {
	t *testing.T

	db       *gorm.DB
	e        *echo.Echo
	appCtx   app.AppContext
	customer domain.Customer
	pkg      domain.InternetPackage
}

func newCreateSubscriptionFixture(t *testing.T) *createSubscriptionFixture {
	t.Helper()
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.Subscription{}, &domain.RadiusUser{}, &domain.InternetPackage{},
		&domain.RadiusProfile{}, &domain.BillingEvent{},
	))
	f := &createSubscriptionFixture{t: t, db: db, e: e, appCtx: appCtx}
	f.customer = domain.Customer{
		TenantID: domain.DefaultTenantID, CustomerNo: "CUS-RELINK",
		Name: "Relink", Status: domain.CustomerActive,
	}
	require.NoError(t, db.Create(&f.customer).Error)
	profile := domain.RadiusProfile{
		TenantID: domain.DefaultTenantID, Name: "state-profile", Status: "enabled",
	}
	require.NoError(t, db.Create(&profile).Error)
	f.pkg = domain.InternetPackage{
		TenantID: domain.DefaultTenantID, Code: "STATE", Name: "State Package",
		RadiusProfileID: profile.ID, Status: "active",
	}
	require.NoError(t, db.Create(&f.pkg).Error)
	return f
}

func (f *createSubscriptionFixture) create(body string) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/isp/subscriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := CreateTestContext(f.e, f.db, req, rec, f.appCtx)
	c.Set("tenant_id", domain.DefaultTenantID)
	require.NoError(f.t, createSubscription(c))
	return rec
}

func (f *createSubscriptionFixture) linkBody(radiusUserID int64) string {
	return fmt.Sprintf(`{"customer_id":"%d","package_id":"%d","radius_user_id":"%d"}`,
		f.customer.ID, f.pkg.ID, radiusUserID)
}

type subscriptionCounts struct {
	subscriptions int64
	accounts      int64
	events        int64
}

func (f *createSubscriptionFixture) counts() subscriptionCounts {
	f.t.Helper()
	var c subscriptionCounts
	require.NoError(f.t, f.db.Model(&domain.Subscription{}).Count(&c.subscriptions).Error)
	require.NoError(f.t, f.db.Model(&domain.RadiusUser{}).Count(&c.accounts).Error)
	require.NoError(f.t, f.db.Model(&domain.BillingEvent{}).Count(&c.events).Error)
	return c
}

// TestSubscriptionRelinkRejectsInvalidWithoutPartialWrites covers attaching an
// existing account and creating a new one. Rejected links must not create a
// subscription, an account, or an event; a failed insert must undo the account
// created earlier in the same transaction.
func TestSubscriptionRelinkRejectsInvalidWithoutPartialWrites(t *testing.T) {
	t.Run("account already linked", func(t *testing.T) {
		f := newCreateSubscriptionFixture(t)
		user := domain.RadiusUser{
			TenantID: domain.DefaultTenantID, Username: "already-linked",
			Password: "secret", Status: "enabled",
		}
		require.NoError(t, f.db.Create(&user).Error)
		require.NoError(t, f.db.Create(&domain.Subscription{
			TenantID: domain.DefaultTenantID, SubscriptionNo: "SUB-ALREADY-LINKED",
			CustomerID: f.customer.ID, RadiusUserID: user.ID,
			Status: domain.SubscriptionActive, StartDate: time.Now(),
		}).Error)
		before := f.counts()

		rec := f.create(f.linkBody(user.ID))
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

		var body ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "SUBSCRIPTION_CREATE_FAILED", body.Error)
		assert.Equal(t, before, f.counts())
	})

	t.Run("unknown account", func(t *testing.T) {
		f := newCreateSubscriptionFixture(t)
		before := f.counts()

		rec := f.create(f.linkBody(999999))
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		assert.Equal(t, before, f.counts())
	})

	t.Run("non-numeric account ID", func(t *testing.T) {
		f := newCreateSubscriptionFixture(t)
		before := f.counts()

		rec := f.create(fmt.Sprintf(`{"customer_id":"%d","package_id":"%d","radius_user_id":"not-a-number"}`,
			f.customer.ID, f.pkg.ID))
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

		var body ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "INVALID_RADIUS_USER_ID", body.Error)
		assert.Equal(t, before, f.counts())
	})

	t.Run("subscription insert failure rolls back the new account", func(t *testing.T) {
		f := newCreateSubscriptionFixture(t)
		require.NoError(t, f.db.Exec(`CREATE TRIGGER inject_subscription_failure
			BEFORE INSERT ON isp_subscription
			BEGIN SELECT RAISE(ABORT, 'injected subscription failure'); END;`).Error)

		rec := f.create(fmt.Sprintf(`{"customer_id":"%d","package_id":"%d","username":"state-new-user","password":"secret1"}`,
			f.customer.ID, f.pkg.ID))
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

		var body ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "SUBSCRIPTION_CREATE_FAILED", body.Error)

		var created int64
		require.NoError(t, f.db.Model(&domain.RadiusUser{}).
			Where("username = ?", "state-new-user").Count(&created).Error)
		assert.Zero(t, created, "the account created before the failed insert must roll back")
		assert.Equal(t, subscriptionCounts{}, f.counts())
	})

	t.Run("unlinked account can be attached", func(t *testing.T) {
		f := newCreateSubscriptionFixture(t)
		user := domain.RadiusUser{
			TenantID: domain.DefaultTenantID, Username: "unlinked",
			Password: "secret", Status: "disabled",
		}
		require.NoError(t, f.db.Create(&user).Error)

		rec := f.create(f.linkBody(user.ID))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		var sub domain.Subscription
		require.NoError(t, f.db.Where("radius_user_id = ?", user.ID).First(&sub).Error)
		assert.Equal(t, domain.SubscriptionPending, sub.Status)
		assert.NotEmpty(t, sub.SubscriptionNo)

		var events []domain.BillingEvent
		require.NoError(t, f.db.Where("subscription_id = ?", sub.ID).Find(&events).Error)
		require.Len(t, events, 1)
		assert.Equal(t, "subscription_created", events[0].Type)
	})
}
