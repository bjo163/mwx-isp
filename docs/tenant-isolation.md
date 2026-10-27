# Tenant isolation: join, raw SQL and foreign-reference audit

This document records the H01 audit (AUDIT backlog 2026-10-05, TR-F033/TR-F012):
which database paths bypass the model-aware tenant callbacks, how each is
scoped, and which cross-tenant foreign references are rejected. It is the
reference for future schema or query changes; update it when a new
`Table`/`Raw`/`Exec`/`Joins` call or tenant-owned foreign key is added.

## Mechanism

`internal/tenancy/scope.go` registers GORM callbacks that append
`tenant_id = <authenticated tenant>` to every model-aware query, row, create,
update and delete. Creates also overwrite or validate the record's `TenantID`,
and upserts carry the predicate into the conflict branch. Handlers obtain the
scoped handle through `GetDB(c)` in `internal/adminapi/context.go`.

The callbacks deliberately cannot see `Table(...)` queries, raw SQL, or joined
tables without a model, so those paths must add the predicate themselves.

## Non-model query inventory

Every production `Table`/`Raw`/`Exec`/`Joins`/aggregate-subquery call is listed
below with its tenant treatment or the reason it is excluded.

| Location | Path | Tenant handling |
| --- | --- | --- |
| `internal/adminapi/dashboard.go:240` | `Table("radius_online AS o")` joined to `radius_user` and `radius_profile` | `o.tenant_id = ?` plus `tenant_id` in both join predicates |
| `internal/adminapi/noc_impact.go:382` | `Table("isp_subscription AS s")` joined to customer and radius user | `s.tenant_id = ?` plus `c.tenant_id = s.tenant_id` / `u.tenant_id = s.tenant_id` |
| `internal/adminapi/noc_impact.go:435,446` | `Table("radius_accounting")`, `Table("radius_online")` aggregates | `tenant_id = ? AND username IN ?` |
| `internal/adminapi/noc_impact.go:496` | `Table("radius_online AS o")` over a subscription subquery | `o.tenant_id = ?` and the subquery is tenant-filtered |
| `internal/adminapi/noc_impact.go:506` | `Table("isp_trouble_ticket AS t")` over a customer subquery | `t.tenant_id = ?` and the subquery is tenant-filtered |
| `internal/adminapi/isp.go:156,205` | `MAX(id)`/`SUM(balance)` aggregate subqueries | Model-based (`Subscription`/`Invoice`), so callbacks append the filter |
| `internal/billing/service.go:263` | `DocumentSequence` upsert with `ON CONFLICT` + `RETURNING` | Model-based; callbacks scope both the insert and the conflict branch |
| `internal/app/app.go:475` | `setval(pg_get_serial_sequence('tenant', 'id'), ...)` | Excluded: platform bootstrap, not tenant-owned data |
| `internal/app/app.go:576,602` | DDL/index and startup dedup `Exec` | Excluded: platform migration, operates on whole tables by design |

Platform-only surfaces intentionally cross tenants and are protected by
`requirePlatformAdmin()` instead of the tenant filter: `/platform/*`
(`internal/adminapi/platform_tenants.go`) and `/system/backup|restore`
(`internal/adminapi/system_backup.go`). Background jobs and the RADIUS server
build an explicit tenant context per tenant/NAS with `tenancy.WithTenantID`
(`internal/app/jobs.go`, `internal/radiusd/radius.go`,
`internal/radiusd/radius_acct.go`), so they never run unscoped against
tenant-owned tables.

## Foreign-reference matrix

Every tenant-owned foreign key that is writable through the Admin API is
resolved through `GetDB(c)` before it is stored, so a foreign ID is invisible
and the request is rejected without writing anything.

| Reference | Validated in | Rejection |
| --- | --- | --- |
| `isp_customer.odp_id` | `resolveODPAssignment` (`isp.go`) | `400 INVALID_ODP` |
| `isp_package.radius_profile_id` | `savePackage` (`isp.go`) | `400 PROFILE_NOT_FOUND` |
| `isp_subscription.customer_id` / `package_id` | `createSubscription` (`isp.go`) | `400 CUSTOMER_NOT_FOUND` / `PACKAGE_NOT_FOUND` |
| `isp_subscription.radius_user_id` | `createSubscription` (`isp.go`) | `400`/rollback when the user is invisible or already linked |
| `radius_user.profile_id` | `createUser`/`updateUser` (`users.go`) | `400 PROFILE_NOT_FOUND` |
| `isp_payment.invoice_id` | `billing.RecordPayment` on the scoped handle | error, no payment row |
| `isp_trouble_ticket.customer_id` / `subscription_id` | `createTroubleTicket` (`isp_operations.go`) | `400 CUSTOMER_NOT_FOUND` / `SUBSCRIPTION_NOT_FOUND` / `SUBSCRIPTION_MISMATCH` |
| `net_nas.node_id` | `CreateNAS`/`UpdateNAS` (`nas.go`) | `400 NODE_NOT_FOUND` |
| `isp_hotspot_voucher.package_id` | `generateVouchers` (`isp_operations.go`) | `400` package/profile unavailable |
| ODP in-use delete | `deleteODP`, `deleteNode` (`isp_operations.go`, `nodes.go`) | `409` while customers/NAS still reference it |

`isp_customer.odp_code` is a denormalized display label, not a foreign key, so
it is excluded from the matrix. It is kept for legacy imports and never joins
to another tenant's row.

## Verification

- Unit: `go test ./internal/adminapi/ -run 'TestTenantIsolationRejectsCrossTenantForeignReferences|TestCreateNAS'`
  — two tenants share `MWX-SAME-001`, `shared-user`, `PKG-SAME` and
  `SUB-SAME`; the test proves list/detail/update isolation, rejects every
  foreign reference above, verifies no rejected mutation persisted a row, and
  shows username-based RADIUS audit does not leak the other tenant's session.
- Integration: `go test -tags=integration ./test/integration/ -run 'TestTenantAdminAPIRejectsCrossTenantIDsAndReferences'`
  — PostgreSQL suite validating cross-tenant IDs, updates, deletes, ticket
  references, NAS node references and unchanged rows after rejection.
- The integration suite is run manually with `docker-compose.test.yml`; it is
  the authoritative PostgreSQL evidence for this audit.

## Known limits

- The audit covers the current source tree; new raw SQL must extend the
  inventory above and add a two-tenant test.
- SQLite unit tests prove handler behavior; PostgreSQL remains the reference
  for constraint and concurrency behavior.
- Cross-tenant access through platform-only endpoints is by design and is
  governed by the platform-admin role matrix (`docs/access-control-matrix.md`).
