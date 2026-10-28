// Command acctbench measures the query plans and latency of the accounting
// list and retention statements against a scratch PostgreSQL database seeded
// with synthetic session volume.
//
// The admin API lists radius_accounting rows with a tenant predicate, a
// descending acct_start_time order and a time-range filter (see
// internal/adminapi/accounting.go), while the daily retention job deletes
// terminated rows by acct_stop_time (see SchedClearExpireData in
// internal/app/jobs.go). Both depend on the indexes declared on
// domain.RadiusAccounting. This command reproduces that workload at a chosen
// scale, prints EXPLAIN (ANALYZE, BUFFERS) for every statement, and repeats the
// measurements with a candidate composite index so an index or partitioning
// decision is made from plans instead of intuition.
//
// The command creates, seeds and (unless -keep is set) truncates the
// radius_accounting table, so it refuses a database whose name does not look
// like a scratch database unless -force is passed. Never point it at
// production.
//
// Usage:
//
//	go run ./tools/acctbench -dsn "host=127.0.0.1 port=55432 user=... dbname=mwxbench" -rows 1000000
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/bjo163/mwx-isp/internal/domain"
)

const (
	// Postgres accepts at most 65535 bind parameters per statement, and a
	// RadiusAccounting row binds 22 of them.
	seedBatchSize       = 2000
	samplesPerIteration = 10
	warmupRuns          = 3
	retentionDays       = 90
	accountingHistory   = 90 * 24 * time.Hour
)

type benchQuery struct {
	name string
	sql  string
	args []interface{}
	// explainOnly statements are planned but never executed, so a DELETE can be
	// measured without removing the seeded rows.
	explainOnly bool
}

type result struct {
	name     string
	measured bool
	p50      time.Duration
	p95      time.Duration
	allocsM  float64
	plan     string
}

func main() {
	dsn := flag.String("dsn", "", "PostgreSQL DSN of a scratch database (required)")
	rows := flag.Int("rows", 1_000_000, "synthetic accounting rows to seed")
	tenants := flag.Int("tenants", 8, "tenant count the rows are spread over")
	iters := flag.Int("iters", 25, "measured iterations per query")
	keep := flag.Bool("keep", false, "keep the seeded rows and the candidate index")
	force := flag.Bool("force", false, "allow a database whose name is not obviously scratch")
	flag.Parse()

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "acctbench: -dsn is required")
		os.Exit(2)
	}
	if !*force && !looksScratch(*dsn) {
		fmt.Fprintln(os.Stderr, "acctbench: refusing to seed a database whose name is not bench/test; pass -force if this really is a scratch database")
		os.Exit(2)
	}
	if *rows < 1000 || *tenants < 1 || *iters < 1 {
		fmt.Fprintln(os.Stderr, "acctbench: -rows must be >= 1000, -tenants and -iters >= 1")
		os.Exit(2)
	}

	db := openDB(*dsn)
	if err := db.AutoMigrate(&domain.RadiusAccounting{}); err != nil {
		fatal("migrate", err)
	}

	fmt.Printf("seed: %d rows across %d tenants\n", *rows, *tenants)
	started := time.Now()
	if err := seed(db, *rows, *tenants); err != nil {
		fatal("seed", err)
	}
	fmt.Printf("seeded in %s\n\n", time.Since(started).Round(time.Millisecond))
	if err := db.Exec("ANALYZE radius_accounting").Error; err != nil {
		fatal("analyze", err)
	}
	printTableStats(db, "baseline")
	closeDB(db)

	queries := buildQueries()

	// Each phase opens its own pool: pgx caches one prepared plan per SQL text
	// per connection, so a pooled connection would keep executing the plan it
	// built before the candidate index existed.
	db = openDB(*dsn)
	fmt.Println("== baseline (existing indexes) ==")
	baseline := runAll(db, queries, *iters)
	printResults(baseline)
	closeDB(db)

	fmt.Println()
	fmt.Println("== candidate index (tenant_id, acct_start_time DESC) ==")
	const candidateIndex = "idx_radius_accounting_tenant_start"
	db = openDB(*dsn)
	if err := db.Exec("CREATE INDEX " + candidateIndex + " ON radius_accounting (tenant_id, acct_start_time DESC)").Error; err != nil {
		fatal("create index", err)
	}
	if err := db.Exec("ANALYZE radius_accounting").Error; err != nil {
		fatal("analyze", err)
	}
	printTableStats(db, "with candidate index")
	withIndex := runAll(db, queries, *iters)
	printResults(withIndex)
	closeDB(db)

	fmt.Println()
	printComparison("baseline vs candidate composite index", baseline, withIndex)

	fmt.Println()
	fmt.Println("== candidate: pg_trgm GIN on username ==")
	const trgmIndex = "idx_radius_accounting_username_trgm"
	trgmReady := false
	db = openDB(*dsn)
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		fmt.Println("skipped: pg_trgm extension unavailable:", err)
	} else if err := db.Exec("CREATE INDEX " + trgmIndex + " ON radius_accounting USING gin (username gin_trgm_ops)").Error; err != nil {
		fmt.Println("skipped: gin index creation failed:", err)
	} else {
		trgmReady = true
		if err := db.Exec("ANALYZE radius_accounting").Error; err != nil {
			fatal("analyze", err)
		}
		printTableStats(db, "with pg_trgm index")
		withTrgm := runAll(db, queries, *iters)
		printResults(withTrgm)
		fmt.Println()
		printComparison("baseline vs pg_trgm GIN", baseline, withTrgm)
	}
	closeDB(db)

	if !*keep {
		db = openDB(*dsn)
		if err := db.Exec("DROP INDEX IF EXISTS " + candidateIndex).Error; err != nil {
			fatal("drop index", err)
		}
		if trgmReady {
			if err := db.Exec("DROP INDEX IF EXISTS " + trgmIndex).Error; err != nil {
				fatal("drop index", err)
			}
		}
		if err := db.Exec("TRUNCATE radius_accounting").Error; err != nil {
			fatal("truncate", err)
		}
		closeDB(db)
		fmt.Println("\ncleaned up (candidate indexes dropped, table truncated)")
	}
}

func openDB(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fatal("connect", err)
	}
	return db
}

func closeDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func fatal(step string, err error) {
	fmt.Fprintf(os.Stderr, "acctbench: %s: %v\n", step, err)
	os.Exit(1)
}

func printComparison(title string, before, after []result) {
	fmt.Println("== " + title + " ==")
	fmt.Printf("%-22s %12s %12s %10s\n", "query", "p50 before", "p50 after", "speedup")
	for i := range before {
		if !before[i].measured || !after[i].measured || after[i].p50 == 0 {
			continue
		}
		delta := float64(before[i].p50) / float64(after[i].p50)
		fmt.Printf("%-22s %10.2fms %10.2fms %9.2fx\n",
			before[i].name,
			ms(before[i].p50), ms(after[i].p50), delta)
	}
}

func looksScratch(dsn string) bool {
	lower := strings.ToLower(dsn)
	for _, marker := range []string{"bench", "test", "scratch"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func seed(db *gorm.DB, rows, tenants int) error {
	if err := db.Exec("TRUNCATE radius_accounting RESTART IDENTITY").Error; err != nil {
		return err
	}
	now := time.Now().UTC()
	span := float64(accountingHistory)
	const usernamesPerTenant = 625
	batch := make([]domain.RadiusAccounting, 0, seedBatchSize)
	for i := 0; i < rows; i++ {
		tenant := int64(i%tenants) + 1
		start := now.Add(-time.Duration(span * float64(i) / float64(rows)))
		stop := start.Add(time.Duration(60+((i*37)%3600)) * time.Second)
		// One hour in 200 sessions is still open, so the epoch guard in the
		// retention delete (acct_stop_time > 1970) keeps having rows to exclude.
		if i%200 == 0 {
			stop = time.Time{}
		}
		batch = append(batch, domain.RadiusAccounting{
			TenantID:        tenant,
			Username:        fmt.Sprintf("user-%d-%06d", tenant, i%usernamesPerTenant),
			AcctSessionId:   fmt.Sprintf("sess-%09d", i),
			NasId:           fmt.Sprintf("nas-%02d", i%40),
			NasAddr:         fmt.Sprintf("10.%d.%d.1", 16+(i%40)/250, (i%40)%250),
			FramedIpaddr:    fmt.Sprintf("100.%d.%d.%d", 64+(i/254/254)%60, (i/254)%254, i%254+1),
			MacAddr:         fmt.Sprintf("02:00:%02x:%02x:%02x:%02x", (i/256)%256, i%256, (i/65536)%256, (i/16777216)%256),
			AcctSessionTime: int(stop.Sub(start).Seconds()),
			AcctInputTotal:  int64(i * 1024),
			AcctOutputTotal: int64(i * 4096),
			LastUpdate:      stop,
			AcctStartTime:   start,
			AcctStopTime:    stop,
		})
		if len(batch) == seedBatchSize {
			if err := db.CreateInBatches(batch, seedBatchSize).Error; err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return db.CreateInBatches(batch, len(batch)).Error
	}
	return nil
}

func printTableStats(db *gorm.DB, label string) {
	var total, sizeBytes int64
	_ = db.Raw("SELECT count(*) FROM radius_accounting").Row().Scan(&total)
	_ = db.Raw("SELECT pg_total_relation_size('radius_accounting')").Row().Scan(&sizeBytes)
	fmt.Printf("%s: %d rows, %.1f MB on disk\n\n", label, total, float64(sizeBytes)/(1024*1024))
}

func buildQueries() []benchQuery {
	now := time.Now().UTC()
	windowStart := now.Add(-7 * 24 * time.Hour)
	cutoff := now.Add(-accountingHistory)
	epoch := time.Unix(0, 0)
	const listColumns = "id, tenant_id, username, acct_session_id, nas_addr, framed_ipaddr, mac_addr, " +
		"acct_start_time, acct_stop_time, acct_input_total, acct_output_total, acct_session_time"
	return []benchQuery{
		{
			name: "list_default",
			sql:  "SELECT " + listColumns + " FROM radius_accounting WHERE tenant_id = $1 ORDER BY acct_start_time DESC LIMIT 10",
			args: []interface{}{int64(1)},
		},
		{
			name: "list_page_201",
			sql:  "SELECT " + listColumns + " FROM radius_accounting WHERE tenant_id = $1 ORDER BY acct_start_time DESC LIMIT 10 OFFSET 2000",
			args: []interface{}{int64(1)},
		},
		{
			name: "list_date_window",
			sql: "SELECT " + listColumns + " FROM radius_accounting WHERE tenant_id = $1 AND acct_start_time >= $2 AND acct_start_time <= $3 " +
				"ORDER BY acct_start_time DESC LIMIT 10",
			args: []interface{}{int64(1), windowStart, now},
		},
		{
			name: "list_username_like",
			sql: "SELECT " + listColumns + " FROM radius_accounting WHERE tenant_id = $1 AND username LIKE $2 " +
				"ORDER BY acct_start_time DESC LIMIT 10",
			args: []interface{}{int64(1), "%user-1-000123%"},
		},
		{
			name: "count_date_window",
			sql:  "SELECT count(*) FROM radius_accounting WHERE tenant_id = $1 AND acct_start_time >= $2 AND acct_start_time <= $3",
			args: []interface{}{int64(1), windowStart, now},
		},
		{
			name: "retention_count",
			sql:  "SELECT count(*) FROM radius_accounting WHERE tenant_id = $1 AND acct_stop_time > $2 AND acct_stop_time < $3",
			args: []interface{}{int64(1), epoch, cutoff},
		},
		{
			name: "retention_delete_plan",
			sql:  "DELETE FROM radius_accounting WHERE tenant_id = $1 AND acct_stop_time > $2 AND acct_stop_time < $3",
			args: []interface{}{int64(1), epoch, cutoff},

			explainOnly: true,
		},
	}
}

func runAll(db *gorm.DB, queries []benchQuery, iters int) []result {
	results := make([]result, 0, len(queries))
	for _, q := range queries {
		results = append(results, runOne(db, q, iters))
	}
	return results
}

func runOne(db *gorm.DB, q benchQuery, iters int) result {
	if q.explainOnly {
		return result{name: q.name, plan: explain(db, q, false)}
	}
	for i := 0; i < warmupRuns; i++ {
		for j := 0; j < samplesPerIteration; j++ {
			execute(db, q)
		}
	}
	durations := make([]time.Duration, 0, iters)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for i := 0; i < iters; i++ {
		// Time a short batch per sample: sub-millisecond statements are below the
		// timer granularity when timed one at a time.
		start := time.Now()
		for j := 0; j < samplesPerIteration; j++ {
			execute(db, q)
		}
		durations = append(durations, time.Since(start)/samplesPerIteration)
	}
	runtime.ReadMemStats(&after)
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	executed := float64(iters * samplesPerIteration)
	return result{
		name:     q.name,
		measured: true,
		p50:      durations[len(durations)/2],
		p95:      durations[(len(durations)*95)/100-1],
		allocsM:  float64(after.TotalAlloc-before.TotalAlloc) / executed / (1024 * 1024),
		plan:     explain(db, q, true),
	}
}

func execute(db *gorm.DB, q benchQuery) {
	rows, err := db.Raw(q.sql, q.args...).Rows()
	if err != nil {
		fmt.Fprintf(os.Stderr, "acctbench: %s: %v\n", q.name, err)
		return
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return
	}
	values := make([]interface{}, len(cols))
	pointers := make([]interface{}, len(cols))
	for i := range values {
		pointers[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return
		}
	}
}

func explain(db *gorm.DB, q benchQuery, analyze bool) string {
	prefix := "EXPLAIN (COSTS OFF) "
	if analyze {
		prefix = "EXPLAIN (ANALYZE, BUFFERS) "
	}
	rows, err := db.Raw(prefix+q.sql, q.args...).Rows()
	if err != nil {
		return "explain error: " + err.Error()
	}
	defer func() { _ = rows.Close() }()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			break
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func printResults(results []result) {
	for _, r := range results {
		if !r.measured {
			fmt.Printf("-- %s (plan only)\n%s\n\n", r.name, r.plan)
			continue
		}
		fmt.Printf("-- %s: p50 %.2fms  p95 %.2fms  go alloc %.1f MB/iter\n%s\n\n",
			r.name, ms(r.p50), ms(r.p95), r.allocsM, r.plan)
	}
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}
