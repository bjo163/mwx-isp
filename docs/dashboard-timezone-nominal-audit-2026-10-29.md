# Dashboard chart timezone & nominal consistency audit — 2026-10-29

Feature: TR-F015 (dashboard statistics) · backlog item ENG-057.

The dashboard aggregates a 7-day authentication trend and a 24-hour traffic
series, and renders both with client-side labels and nominal values. This audit
checks one invariant end to end: **the bucket a row is counted in must be the
bucket whose label is drawn, and every number must be formatted with the
administration's locale rather than the browser's**.

## Method

- Read the whole query path: `GetDashboardStats` → `fetchAuthTrend` /
  `fetchTrafficStats` → response structs → `web/src/pages/Dashboard.tsx`.
- Compared every place a date boundary or a number is produced: SQL expressions,
  Go `time.Local` handling, the Postgres DSN, and the browser formatters.
- Wrote deterministic tests with a fixed `time.Local` so the assertions do not
  depend on the machine's OS timezone (the previous test explicitly tolerated a
  mismatch — that tolerance was the defect).

## Findings

| # | Severity | Finding | Evidence (before) | Decision |
| --- | --- | --- | --- | --- |
| F1 | High | Series buckets were built by SQL `DATE()` / `TO_CHAR(date_trunc('hour', …))` on Postgres and `strftime(…, 'localtime')` on SQLite, i.e. in the **database session timezone / host OS timezone**, while the response labels and the "today" boundary were computed from Go `time.Local`. A deployment whose DB session TZ differs from `system.location` filed early-morning sessions under the previous day and silently dropped the point (the lookup map missed and the bucket was zero-filled). | removed `dateBucketExpression` (`DATE(field)`, `date_trunc`, `strftime(…,'localtime')`) | Replaced with explicit half-open **instant windows** generated in Go and aggregated in one statement (`dayWindows`, `hourWindows`, `countSessionsPerWindow`, `sumTrafficPerWindow`). |
| F2 | Medium | The Postgres DSN hardcodes `TimeZone=Asia/Jakarta` independently of `system.location`, so even a "correct" SQL grouping would disagree with the labels after a location change. | `internal/app/database.go:100` | No longer load-bearing for the dashboard: the series compares `acct_start_time` against bound instants, which is timezone-neutral on `timestamptz`. The DSN value is left as-is (out of scope; it only affects ad-hoc SQL and logs). |
| F3 | Medium | `time.LoadLocation(cfg.System.Location)` failure is logged and then ignored, so `time.Local` silently stays at the container default (UTC). Every "today"/day-window boundary then shifts by the offset. | `internal/app/app.go:191` | Recorded as a deployment limit (see below). The shipped image installs `tzdata` (`Dockerfile:36`) and sets `TZ`, so the supported deployment is correct; a distroless/scratch deployment must either embed `time/tzdata` or mount zoneinfo. No code change here to keep the audit scoped to the dashboard. |
| F4 | Medium | Frontend formatters were constructed without a locale (`Intl.DateTimeFormat(undefined, …)`, `Intl.NumberFormat()`), so the admin UI ignored the selected `en-US` / `id-ID` locale and followed the browser language — decimal separators, month names and digits could differ from the rest of the page. | `Dashboard.tsx` formatters | All formatters are now built from React-Admin's `useLocale()` and rebuilt when the locale changes. |
| F5 | Medium | Hour labels used `hour12: false`, which yields `24:00` for midnight in some ICU locales instead of `00:00`. | `hourFormatter` | Switched to `hourCycle: 'h23'`. |
| F6 | Low | Today's traffic used `toFixed(2)`, always with a dot; the outstanding amount was hardcoded to `en-US`; percentages used `toFixed(1)`. In `id-ID` these disagree with the localized counters on the same page. | today-traffic card, outstanding tile, hero ratio, IPv6 chip | Replaced by `decimalFormatter`, `currencyFormatter`, `percentFormatter` bound to the admin locale. |
| F7 | Low | Chart tooltips printed raw float values (`0.30000000000000004`) and unit-less counts. | ECharts `tooltip` in both series | Added `valueFormatter`: counts through `numberFormatter`, traffic through `decimalFormatter` + ` GB`. |

Not findings, verified deliberately:

- `acct_start_time` is a `time.Time` on both dialects (`internal/domain/radius.go:123`),
  so on Postgres the column is `timestamptz` and `<`/`>=` comparisons are
  instant-based regardless of the session `TimeZone`; on SQLite driver values and
  stored values go through the same conversion, so the comparison stays monotonic
  while the process offset is constant (it is set once at startup).
- The "today" counters (`TodayAuthCount`, `TodayAcctCount`, today's traffic)
  already used a Go-computed `todayStart`; they were consistent and are unchanged.
- Tenant scoping still applies: the aggregation uses `db.Model(...).Row()`, and
  `internal/tenancy` registers a `Row` callback, so the single-row query keeps the
  tenant predicate.
- Labels are rendered **not** converted into the viewer's timezone: the server
  sends wall-clock keys in the configured system location, and the client only
  reformats them in the admin locale. An operator in another timezone therefore
  sees the server's day/hour, which is the intended NOC reading.

## Changes

Backend — `internal/adminapi/dashboard.go`:

- `dayWindows` (line 170) and `hourWindows` (line 181) produce half-open
  `[start, end)` windows in `time.Local`, ending with the current day/hour.
- `countSessionsPerWindow` (line 195) and `sumTrafficPerWindow` (line 217) fold
  every window into one `SELECT` of `COUNT/SUM(CASE WHEN acct_start_time >= ?
  AND acct_start_time < ? …)` columns. The boundaries travel as parameters, so
  the result cannot depend on the database session timezone; query count stays at
  one per series.
- `fetchAuthTrend` (line 244) and `fetchTrafficStats` (line 257) map window *i* to
  label *i* directly — no string-key lookup that could silently miss.
- `dateBucketExpression` deleted.

Frontend — `web/src/pages/Dashboard.tsx`:

- `dateFormatter` (line 133), `hourFormatter` (136, `hourCycle: 'h23'`),
  `decimalFormatter` (142), `percentFormatter` (146) and `currencyFormatter` (150)
  are locale-driven memos.
- Tooltips on the auth-trend and traffic charts format their values (F7).

## Verification

```bash
# deterministic bucket placement, independent of the host OS timezone
go test ./internal/adminapi/ -count=1
# frontend gate used by CI
cd web && npm run type-check && npm run build
```

Results on 2026-10-29 (`dev`): `go test ./internal/adminapi/` → `ok … 45.5s`;
`tsc --noEmit` clean; `vite build` succeeded.

New/strengthened tests in `internal/adminapi/dashboard_test.go`:

- `TestDayWindowsUseLocalCalendarDays` — pins the three day windows to local
  calendar days in a fixed `UTC+7` zone (labels `2026-03-08..10`, midnight starts,
  exact 24-hour ends).
- `TestCountSessionsPerWindowBucketsByInstant` — a 23:30 and a 00:30 session land
  in the first and last window respectively; under the old SQL grouping the
  00:30 record would have fallen into the previous (UTC) day.
- `TestGetDashboardStats` — the fixture now starts "today" and the assertions
  require the point in the **last** day bucket, exactly one non-zero 24-hour
  traffic bucket, and the 2 GB total. The old "timezone differences" allowance is
  gone.

## Limits

- Timezone correctness is verified in SQLite unit tests only; the Postgres path
  relies on `timestamptz` comparison semantics and is covered by CI's integration
  job for the wider dashboard queries, not by a dedicated DB-session-TZ matrix.
- A non-Docker deployment without zoneinfo and without embedded `time/tzdata`
  will log a `timezone config error` and run in UTC (F3); the shipped image is
  unaffected.
- Fallback behaviour is unchanged by design: if an aggregation statement fails,
  the section logs and returns zeroes. A failed query is indistinguishable from a
  genuinely empty day in the UI.
- The audit covers the dashboard page only. Other admin pages may still contain
  browser-locale or `toFixed` formatting; that is a separate item.
