package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bjo163/mwx-isp/internal/billing"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/pkg/metrics"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var cronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func (a *Application) initJob() {
	loc, _ := time.LoadLocation(a.appConfig.System.Location)
	a.sched = cron.New(cron.WithLocation(loc), cron.WithParser(cronParser))

	var err error
	_, err = a.sched.AddFunc("@every 30s", func() {
		go a.SchedSystemMonitorTask()
		go a.SchedProcessMonitorTask()
		go a.SchedNetworkMonitorTask()
		go a.SchedNotificationOutboxTask()
	})
	if err != nil {
		zap.S().Errorf("init job error %s", err.Error())
	}

	_, err = a.sched.AddFunc("@daily", func() {
		a.SchedClearExpireData()
	})

	if err != nil {
		zap.S().Errorf("init job error %s", err.Error())
	}

	_, err = a.sched.AddFunc("@daily", func() {
		a.SchedISPBillingTask()
	})
	if err != nil {
		zap.S().Errorf("init ISP billing job error %s", err.Error())
	}

	_, err = a.sched.AddFunc("@every 15m", func() {
		a.SchedFUPCheckTask()
	})
	if err != nil {
		zap.S().Errorf("init FUP check job error %s", err.Error())
	}

	_, err = a.sched.AddFunc("@every 5m", func() {
		a.SchedSessionActionRetryTask()
	})
	if err != nil {
		zap.S().Errorf("init session action retry job error %s", err.Error())
	}

	a.sched.Start()
	// Run one recovery pass at startup so pending actions left behind by a
	// restart are retried without waiting for the first cron tick.
	go a.SchedSessionActionRetryTask()
}

// SchedISPBillingTask creates due monthly invoices, processes overdue invoices
// when automatic suspension is enabled, and disconnects sessions for accounts
// suspended by billing enforcement. It logs failures and returns without
// stopping the application scheduler; the next scheduled run retries unfinished
// work. Database errors are not returned to the cron runner.
func (a *Application) SchedISPBillingTask() {
	defer func() {
		if err := recover(); err != nil {
			zap.L().Error("ISP billing scheduler panic", zap.Any("error", err))
			a.enqueueSchedulerFailure("isp-billing", "scheduler recovered from a task error")
		}
	}()
	now := time.Now()
	dueDays := int(a.GetSettingsInt64Value("isp", "DefaultDueDays"))
	if dueDays < 0 || dueDays > 90 {
		dueDays = 10
	}
	// Settings are installation-wide today; each active tenant gets the same
	// billing policy while all database reads/writes remain tenant scoped.
	autoSuspend := a.GetSettingsBoolValue("isp", "AutoSuspend")
	var tenants []domain.Tenant
	if err := a.gormDB.Where("status = ?", "active").Order("id ASC").Find(&tenants).Error; err != nil {
		zap.S().Error("active tenant query failed for ISP billing", zap.Error(err))
		a.enqueueSchedulerFailure("isp-billing-tenants", "active tenant query failed")
		return
	}
	for _, tenant := range tenants {
		if err := a.runBillingForTenant(tenant.ID, now, dueDays, autoSuspend); err != nil {
			zap.S().Error("tenant ISP billing run failed", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
			a.enqueueSchedulerFailure("isp-billing", fmt.Sprintf("billing failed for tenant %d", tenant.ID))
		}
	}
}

func (a *Application) runBillingForTenant(tenantID int64, now time.Time, dueDays int, autoSuspend bool) error {
	if tenantID <= 0 {
		return fmt.Errorf("invalid tenant ID %d", tenantID)
	}
	ctx := tenancy.WithTenantID(context.Background(), tenantID)
	db := a.gormDB.WithContext(ctx)
	if _, err := billing.GenerateMonthlyInvoices(db, now, dueDays); err != nil {
		return fmt.Errorf("generate monthly invoices: %w", err)
	}
	if !autoSuspend {
		return nil
	}
	if err := billing.ProcessOverdueInvoices(db, now); err != nil {
		return fmt.Errorf("process overdue invoices: %w", err)
	}
	if err := billing.SuspendOverdueSubscriptions(db, now); err != nil {
		return fmt.Errorf("suspend overdue subscriptions: %w", err)
	}
	var subs []domain.Subscription
	if err := db.Where("status = ? AND suspension_reason = ?", domain.SubscriptionSuspended, domain.SuspensionBillingOverdue).Find(&subs).Error; err != nil {
		return fmt.Errorf("load billing-suspended subscriptions: %w", err)
	}
	// Queue a durable Disconnect-Request per online session instead of a
	// best-effort inline call: if the NAS is unreachable the action stays pending
	// and the retry scheduler converges it once the NAS is back.
	for _, sub := range subs {
		var user domain.RadiusUser
		if sub.RadiusUserID == 0 || db.First(&user, sub.RadiusUserID).Error != nil {
			continue
		}
		var sessions []domain.RadiusOnline
		if err := db.Where("username = ?", user.Username).Find(&sessions).Error; err != nil {
			return fmt.Errorf("load online sessions for %q: %w", user.Username, err)
		}
		for _, session := range sessions {
			if _, err := sessionqueue.EnqueueDisconnect(db, session, sessionqueue.ReasonBillingSuspension, now); err != nil {
				return fmt.Errorf("queue disconnect for %q: %w", user.Username, err)
			}
		}
	}
	summary, err := sessionqueue.ProcessDue(context.Background(), a.gormDB, tenantID, now, a.sessionActionSender(), 200)
	if err != nil {
		zap.L().Warn("billing session action pass failed", zap.Int64("tenant_id", tenantID), zap.Error(err))
	}
	if summary.Attempted > 0 {
		zap.L().Info("billing session actions processed", zap.Int64("tenant_id", tenantID),
			zap.Int("attempted", summary.Attempted), zap.Int("succeeded", summary.Succeeded),
			zap.Int("retried", summary.Retried), zap.Int("failed", summary.Failed),
			zap.Int("canceled", summary.Canceled), zap.Int("audit_errors", summary.AuditErrors))
	}
	return nil
}

// SchedSessionActionRetryTask retries pending session actions whose backoff has
// elapsed. It runs every five minutes and is safe to overlap: claims are fenced
// with a token, so two passes cannot execute the same action. A process-local
// guard additionally avoids stacking passes within one instance.
func (a *Application) SchedSessionActionRetryTask() {
	defer func() {
		if err := recover(); err != nil {
			zap.L().Error("session action retry scheduler panic", zap.Any("error", err))
			a.enqueueSchedulerFailure("session-actions", "scheduler recovered from a task error")
		}
	}()
	if a.gormDB == nil {
		return
	}
	if !a.sessionActionBusy.CompareAndSwap(false, true) {
		return
	}
	defer a.sessionActionBusy.Store(false)

	now := time.Now()
	var tenants []domain.Tenant
	if err := a.gormDB.Where("status = ?", "active").Order("id ASC").Find(&tenants).Error; err != nil {
		zap.L().Error("active tenant query failed for session actions", zap.Error(err))
		a.enqueueSchedulerFailure("session-actions-tenants", "active tenant query failed")
		return
	}
	send := a.sessionActionSender()
	for _, tenant := range tenants {
		summary, err := sessionqueue.ProcessDue(context.Background(), a.gormDB, tenant.ID, now, send, 100)
		if err != nil {
			zap.L().Warn("tenant session action retry pass failed", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
			a.enqueueSchedulerFailure("session-actions", fmt.Sprintf("session action retry failed for tenant %d", tenant.ID))
			continue
		}
		if summary.Attempted > 0 || summary.AuditErrors > 0 {
			zap.L().Info("session actions retried", zap.Int64("tenant_id", tenant.ID),
				zap.Int("attempted", summary.Attempted), zap.Int("succeeded", summary.Succeeded),
				zap.Int("retried", summary.Retried), zap.Int("failed", summary.Failed),
				zap.Int("canceled", summary.Canceled), zap.Int("audit_errors", summary.AuditErrors))
		}
		if summary.Failed > 0 {
			a.enqueueSchedulerFailure("session-actions-dead-letter",
				fmt.Sprintf("%d session action(s) for tenant %d exhausted the retry budget", summary.Failed, tenant.ID))
		}
	}
}

func (a *Application) enqueueSchedulerFailure(job, summary string) {
	if a.notificationOutbox == nil {
		return
	}
	now := time.Now()
	key := fmt.Sprintf("scheduler:%s:%s", job, now.Format("200601021504"))
	if err := a.notificationOutbox.Enqueue("scheduler.failure", key, "MWX-ISP: "+summary+". Review the scheduler and application logs."); err != nil {
		zap.L().Warn("enqueue scheduler failure notification failed", zap.Error(err))
	}
}

// SchedNetworkMonitorTask polls explicitly configured network targets.
func (a *Application) SchedNetworkMonitorTask() {
	if a.networkMonitor == nil || a.gormDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	var tenants []domain.Tenant
	if err := a.gormDB.WithContext(ctx).Where("status = ?", "active").Order("id ASC").Find(&tenants).Error; err != nil {
		zap.L().Warn("active tenant query failed for network monitor", zap.Error(err))
		a.enqueueSchedulerFailure("network-monitor-tenants", "active tenant query failed")
		return
	}
	for _, tenant := range tenants {
		tenantCtx := tenancy.WithTenantID(ctx, tenant.ID)
		if err := a.networkMonitor.PollDue(tenantCtx, time.Now()); err != nil {
			zap.L().Warn("tenant network monitor poll failed", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
			a.enqueueSchedulerFailure("network-monitor", fmt.Sprintf("network monitor polling failed for tenant %d", tenant.ID))
		}
	}
}

// SchedNotificationOutboxTask enqueues recent billing changes and delivers a
// bounded batch of opt-in operator notifications.
func (a *Application) SchedNotificationOutboxTask() {
	if a.notificationOutbox == nil || a.gormDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var tenants []domain.Tenant
	if err := a.gormDB.WithContext(ctx).Where("status = ?", "active").Order("id ASC").Find(&tenants).Error; err != nil {
		zap.L().Warn("active tenant query failed for notification dispatcher", zap.Error(err))
		return
	}
	for _, tenant := range tenants {
		tenantCtx := tenancy.WithTenantID(ctx, tenant.ID)
		if err := a.NotifyBillingEvents(tenantCtx, time.Now().Add(-2*time.Minute)); err != nil {
			zap.L().Warn("tenant billing notification enqueue failed", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
		}
		if err := a.notificationOutbox.ProcessOnce(tenantCtx, time.Now()); err != nil {
			zap.L().Warn("tenant notification outbox processing failed", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
		}
	}
}

// SchedSystemMonitorTask system monitor
func (a *Application) SchedSystemMonitorTask() {
	defer func() {
		if err := recover(); err != nil {
			zap.S().Error(err)
		}
	}()

	// Collect CPU usage
	_cpuuse, err := cpu.Percent(0, false)
	if err == nil && len(_cpuuse) > 0 {
		metrics.SetGauge("system_cpuuse", int64(_cpuuse[0]*100)) // Store as percentage * 100
	}

	// Collect memory usage
	_meminfo, err := mem.VirtualMemory()
	if err == nil {
		metrics.SetGauge("system_memuse", int64(_meminfo.Used/1024/1024)) //nolint:gosec // G115: memory MB value fits in int64
	}
}

// SchedProcessMonitorTask app process monitor
func (a *Application) SchedProcessMonitorTask() {
	defer func() {
		if err := recover(); err != nil {
			zap.S().Error(err)
		}
	}()

	p, err := process.NewProcess(int32(os.Getpid())) //nolint:gosec // G115: PID is always within int32 range
	if err != nil {
		return
	}

	// Collect process CPU usage
	cpuuse, err := p.CPUPercent()
	if err == nil {
		metrics.SetGauge("mwx_isp_cpuuse", int64(cpuuse*100)) // Store as percentage * 100
	}

	// Collect process memory usage
	meminfo, err := p.MemoryInfo()
	if err == nil {
		metrics.SetGauge("mwx_isp_memuse", int64(meminfo.RSS/1024/1024)) //nolint:gosec // G115: memory MB value fits in int64
	}
}

// SchedClearExpireData purges stale operational data and is registered as a
// @daily cron job by initJob.
//
// It performs tenant-scoped cleanup for every organization:
//   - radius_online: deletes rows that have not refreshed for at least three
//     radius.AcctInterimInterval periods, reclaiming sessions left dangling by a
//     missed Accounting-Stop. A live session updates every interim interval, so
//     requiring several missed updates avoids dropping active sessions (which
//     would also under-count the per-user concurrency limit).
//   - radius_accounting: deletes terminated records whose AcctStopTime predates
//     the radius.AccountingHistoryDays retention window. The window defaults to
//     90 days (config seed); a value of 0 disables accounting cleanup entirely.
//     Active sessions carry a zero AcctStopTime (stamped only at Accounting-Stop)
//     and are always excluded, so an online session never loses its billing row.
//
// Operator audit history is retained per tenant with the same one-year window.
// Any panic is recovered and logged so a cleanup failure never crashes the
// scheduler goroutine.
func (a *Application) SchedClearExpireData() {
	defer func() {
		if err := recover(); err != nil {
			zap.S().Error(err)
		}
	}()

	var tenants []domain.Tenant
	if err := a.gormDB.Find(&tenants).Error; err != nil {
		zap.L().Error("load tenants for session retention", zap.Error(err))
		return
	}
	if len(tenants) == 0 {
		// Keep cleanup working on legacy/test installations before tenant rows
		// have been initialized; the migration assigns their data to tenant 1.
		tenants = []domain.Tenant{{ID: domain.DefaultTenantID}}
	}

	// Reclaim dangling sessions and accounting history within each tenant.
	interim := a.ConfigMgr().GetInt("radius", "AcctInterimInterval")
	if interim <= 0 {
		interim = 300
	}
	onlineStaleWindow := time.Duration(interim*3) * time.Second
	idays := a.ConfigMgr().GetInt("radius", "AccountingHistoryDays")
	for _, tenant := range tenants {
		db := a.gormDB.WithContext(tenancy.WithTenantID(context.Background(), tenant.ID))
		if err := db.Where("opt_time < ?", time.Now().Add(-time.Hour*24*365)).Delete(&domain.SysOprLog{}).Error; err != nil {
			zap.L().Error("prune tenant operator audit logs", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
		}
		if err := db.Where("last_update <= ?", time.Now().Add(-onlineStaleWindow)).Delete(&domain.RadiusOnline{}).Error; err != nil {
			zap.L().Error("prune stale tenant sessions", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
		}
		// A zero retention setting disables accounting cleanup. The epoch guard
		// excludes active sessions with a zero AcctStopTime.
		if idays > 0 {
			cutoff := time.Now().Add(-time.Hour * 24 * time.Duration(idays))
			if err := db.Where("acct_stop_time > ? AND acct_stop_time < ?", time.Unix(0, 0), cutoff).Delete(&domain.RadiusAccounting{}).Error; err != nil {
				zap.L().Error("prune tenant accounting history", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
			}
		}
	}

	// Prune network monitoring health samples older than 30 days
	a.gormDB.Where("checked_at < ?", time.Now().AddDate(0, 0, -30)).Delete(&domain.NetMonitorSample{})

	// Prune processed notification outbox records older than 30 days
	a.gormDB.Where("created_at < ?", time.Now().AddDate(0, 0, -30)).Delete(&domain.NotificationOutbox{})

	// Prune historical billing lifecycle audit events older than 180 days
	a.gormDB.Where("created_at < ?", time.Now().AddDate(0, 0, -180)).Delete(&domain.BillingEvent{})

	// Prune RADIUS bandwidth traffic samples older than 14 days
	a.gormDB.Where("recorded_at < ?", time.Now().AddDate(0, 0, -14)).Delete(&domain.RadiusTrafficSample{})

	// Prune Syslog event history older than 7 days
	a.gormDB.Where("created_at < ?", time.Now().AddDate(0, 0, -7)).Delete(&domain.SyslogEvent{})
}

type fupTrafficTotals struct {
	Input  int64
	Output int64
}

// SchedFUPCheckTask evaluates subscriber traffic usage against package FUP quotas,
// automatically throttling accounts exceeding thresholds and resetting restored quotas.
//
// Each active tenant is processed with its own database context so duplicate
// usernames in different organizations cannot share traffic totals, throttled
// rates, audit events or disconnect actions.
func (a *Application) SchedFUPCheckTask() {
	defer func() {
		if err := recover(); err != nil {
			zap.L().Error("FUP check scheduler panic", zap.Any("error", err))
			a.enqueueSchedulerFailure("isp-fup", "scheduler recovered from a task error")
		}
	}()
	if a.gormDB == nil {
		return
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var tenants []domain.Tenant
	if err := a.gormDB.Where("status = ?", "active").Order("id ASC").Find(&tenants).Error; err != nil {
		zap.S().Error("active tenant query failed for FUP", zap.Error(err))
		a.enqueueSchedulerFailure("isp-fup-tenants", "active tenant query failed")
		return
	}
	for _, tenant := range tenants {
		fupTenantAttempts.Add(1)
		if err := a.runFUPCheckForTenant(tenant.ID, now, monthStart); err != nil {
			fupTenantFailures.Add(1)
			zap.S().Error("tenant FUP check failed", zap.Int64("tenant_id", tenant.ID), zap.Error(err))
			a.enqueueSchedulerFailure("isp-fup", fmt.Sprintf("FUP failed for tenant %d", tenant.ID))
		}
	}
}

func (a *Application) runFUPCheckForTenant(tenantID int64, now, monthStart time.Time) error {
	if tenantID <= 0 {
		return fmt.Errorf("invalid tenant ID %d", tenantID)
	}
	ctx := tenancy.WithTenantID(context.Background(), tenantID)
	db := a.gormDB.WithContext(ctx)

	var activeSubs []domain.Subscription
	if err := db.Where("status = ?", domain.SubscriptionActive).Find(&activeSubs).Error; err != nil {
		return fmt.Errorf("load active subscriptions: %w", err)
	}

	var firstErr error
	for i := range activeSubs {
		sub := &activeSubs[i]
		var pkg domain.InternetPackage
		if err := db.First(&pkg, sub.PackageID).Error; err != nil || pkg.FupLimitGB <= 0 {
			continue
		}
		var user domain.RadiusUser
		if sub.RadiusUserID <= 0 || db.First(&user, sub.RadiusUserID).Error != nil {
			continue
		}

		totalTraffic, err := a.fupMonthlyTraffic(db, user.Username, monthStart)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("aggregate FUP traffic for %q: %w", user.Username, err)
			}
			continue
		}
		limitBytes := pkg.FupLimitGB * 1024 * 1024 * 1024

		if totalTraffic >= limitBytes && !sub.FupTriggered && pkg.FupRateDown > 0 {
			changed, err := a.throttleSubscriptionFUP(db, sub, &pkg, &user, now)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("apply FUP throttle for subscription %d: %w", sub.ID, err)
				}
				continue
			}
			if changed {
				a.disconnectTenantSessions(ctx, user.Username)
			}
			continue
		}

		if totalTraffic < limitBytes && sub.FupTriggered {
			// A throttle recorded in the current period must survive temporary
			// traffic drops (for example accounting retention pruning) until the
			// next period. The persisted event is the period marker because the
			// subscription row only stores the boolean state.
			throttled, err := a.fupThrottledThisMonth(db, sub.ID, monthStart)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("load FUP period marker for subscription %d: %w", sub.ID, err)
				}
				continue
			}
			if throttled {
				continue
			}
			changed, err := a.restoreSubscriptionFUP(db, sub, &pkg, &user, now)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("restore FUP for subscription %d: %w", sub.ID, err)
				}
				continue
			}
			if changed {
				a.disconnectTenantSessions(ctx, user.Username)
			}
		}
	}
	return firstErr
}

// fupMonthlyTraffic sums current online sessions plus closed accounting records
// for the current period. Open accounting rows mirror online sessions and are
// excluded to avoid counting an active session twice.
func (a *Application) fupMonthlyTraffic(db *gorm.DB, username string, monthStart time.Time) (int64, error) {
	var live, hist fupTrafficTotals
	if err := db.Model(&domain.RadiusOnline{}).
		Where("username = ?", username).
		Select("COALESCE(SUM(acct_input_total), 0) as input, COALESCE(SUM(acct_output_total), 0) as output").
		Scan(&live).Error; err != nil {
		return 0, err
	}
	if err := db.Model(&domain.RadiusAccounting{}).
		Where("username = ? AND acct_start_time >= ? AND acct_stop_time > ?", username, monthStart, time.Unix(0, 0)).
		Select("COALESCE(SUM(acct_input_total), 0) as input, COALESCE(SUM(acct_output_total), 0) as output").
		Scan(&hist).Error; err != nil {
		return 0, err
	}
	return live.Input + live.Output + hist.Input + hist.Output, nil
}

func (a *Application) fupThrottledThisMonth(db *gorm.DB, subscriptionID int64, monthStart time.Time) (bool, error) {
	var count int64
	if err := db.Model(&domain.BillingEvent{}).
		Where("subscription_id = ? AND type = ? AND created_at >= ?", subscriptionID, "fup_throttled", monthStart).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *Application) throttleSubscriptionFUP(db *gorm.DB, sub *domain.Subscription, pkg *domain.InternetPackage, user *domain.RadiusUser, now time.Time) (bool, error) {
	changed := false
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&domain.Subscription{}).
			Where("id = ? AND fup_triggered = ?", sub.ID, false).
			Updates(map[string]interface{}{"fup_triggered": true, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if pkg.FupRateUp > 0 && pkg.FupRateDown > 0 {
			if err := tx.Model(&domain.RadiusUser{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
				"up_rate":    pkg.FupRateUp,
				"down_rate":  pkg.FupRateDown,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&domain.BillingEvent{
			CustomerID:     sub.CustomerID,
			SubscriptionID: sub.ID,
			Type:           "fup_throttled",
			Description:    fmt.Sprintf("Auto FUP quota (%d GB) reached: speed throttled", pkg.FupLimitGB),
			CreatedAt:      now,
		}).Error; err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed, err
}

func (a *Application) restoreSubscriptionFUP(db *gorm.DB, sub *domain.Subscription, pkg *domain.InternetPackage, user *domain.RadiusUser, now time.Time) (bool, error) {
	var profile domain.RadiusProfile
	if err := db.First(&profile, pkg.RadiusProfileID).Error; err != nil {
		return false, fmt.Errorf("load package profile: %w", err)
	}
	changed := false
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&domain.Subscription{}).
			Where("id = ? AND fup_triggered = ?", sub.ID, true).
			Updates(map[string]interface{}{"fup_triggered": false, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&domain.RadiusUser{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
			"up_rate":    profile.UpRate,
			"down_rate":  profile.DownRate,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.BillingEvent{
			CustomerID:     sub.CustomerID,
			SubscriptionID: sub.ID,
			Type:           "fup_restored",
			Description:    "Monthly cycle start: FUP quota reset and normal speed restored",
			CreatedAt:      now,
		}).Error; err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed, err
}

func (a *Application) disconnectTenantSessions(ctx context.Context, username string) {
	handler := a.sessionDisconnectHandler()
	if handler == nil {
		return
	}
	var sessions []domain.RadiusOnline
	if err := a.gormDB.WithContext(ctx).Where("username = ?", username).Find(&sessions).Error; err != nil {
		zap.L().Warn("load FUP sessions for disconnect failed", zap.String("username", username), zap.Error(err))
		return
	}
	for _, session := range sessions {
		if err := handler(ctx, session); err != nil {
			zap.L().Warn("FUP session disconnect failed", zap.String("username", username), zap.Error(err))
		}
	}
}
