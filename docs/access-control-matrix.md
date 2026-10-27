# Access Control Matrix (H02)

This document is the source of truth for who may call each management mutation
and which UI controls mirror those rules. The server is the enforcement point;
UI gating only prevents dead-end actions.

## Roles

| Role | Identity | Scope |
| --- | --- | --- |
| `operator` | `SysOpr.Level == "operator"` | Read tenant data, create/update trouble tickets |
| `admin` | `SysOpr.Level == "admin"` | Everything an operator can do, plus tenant configuration, billing writes, network management and RADIUS user writes |
| `super` | `SysOpr.Level == "super"` | Everything an admin can do, plus operator creation, granting `super`, and editing another `super` |
| platform administrator | `SysOpr.Level == "super"` **and** `SysOpr.PlatformAdmin == true` | Cross-tenant platform surfaces (`/platform/*`, system config, certificates, backups, branding, WhatsApp pairing) |

Actor level is resolved from the authoritative `tenant_membership` row, not from
the raw JWT claim, so a role change applies on the next request and a revoked
membership loses access immediately. Every query runs through the tenant-scoped
DB handle, so a role matrix never widens the tenant boundary.

## Enforcement chain

| Layer | Implementation |
| --- | --- |
| Authentication | `internal/webserver/server.go` — JWT middleware on `/api/v1/*`; skip list is limited to `/ready`, `/realip`, `/auth/login`, `/auth/tenants`, `/auth/refresh`, `/public/branding` |
| Actor resolution | `resolveOperatorFromContext` → `tenant_membership` (`internal/tenancy`) |
| Role gate | `RequireLevel`, `requireAdmin` (super/admin), `requirePlatformAdmin` (super + platform flag) in `internal/adminapi/authz.go` / `platform_tenants.go` |
| Tenant scope | `tenancy.RegisterCallbacks` + `GetDB(c)` |

## Mutation matrix

`super` and `platform administrator` are separate columns because a tenant super
cannot call platform routes unless the platform flag is set.

| Surface | Routes | super | admin | operator | Notes |
| --- | --- | --- | --- | --- | --- |
| Customers | `POST/PUT/DELETE /isp/customers` | yes | yes | 403 | |
| Packages | `POST/PUT/DELETE /isp/packages` | yes | yes | 403 | |
| Subscriptions | `POST /isp/subscriptions`, `POST /isp/subscriptions/:id/:action` | yes | yes | 403 | Includes suspend, terminate, disconnect and reactivate actions |
| FUP | `POST /isp/subscriptions/:id/apply-fup`, `.../reset-fup` | yes | yes | 403 | |
| Payments | `POST /isp/payments` | yes | yes | 403 | Only writer of money state |
| Invoice/payment WhatsApp | `POST /isp/invoices/:id/send-whatsapp`, `POST /isp/payments/:id/send-whatsapp` | yes | yes | 403 | |
| Trouble tickets | `POST/PUT /isp/tickets` | yes | yes | yes | Operators run the field workflow |
| Ticket dispatch | `POST /isp/tickets/:id/dispatch` | yes | yes | 403 | Sends the customer-facing WhatsApp work order |
| RADIUS users | `POST /users`, `POST /users/import`, `PUT/DELETE /users/:id` | yes | yes | 403 | |
| Vouchers | `POST /isp/vouchers/generate`, `POST /isp/vouchers/:id/revoke`, `DELETE /isp/vouchers/batches/:id` | yes | yes | 403 | |
| RADIUS profiles | `POST/PUT/DELETE /radius-profiles` | yes | yes | 403 | |
| Sessions | `DELETE /sessions/:id`, `POST /sessions/:id/disconnect`, `POST /sessions/:id/coa`, `POST /session-actions/:id/retry` | yes | yes | 403 | |
| Network | nodes, NAS, ODP, IPAM pools, monitor targets, ping, syslog test | yes | yes | 403 | |
| Operators | `POST /system/operators` | yes | 403 | 403 | Only a super may create an account |
| Operators | `PUT/DELETE /system/operators/:id` | yes | yes | 403 | Admin cannot grant `super` or modify a `super` account |
| Own profile | `PUT /system/operators/me` | yes | yes | yes | Level and status are not editable here |
| Platform | `POST/PUT /platform/tenants*`, system config, certificates, backup/restore, branding writes, WhatsApp pairing | platform only | 403 | 403 | Requires `PlatformAdmin` |
| Public/portal reads | `/public/packages`, `/public/vouchers/check`, `/public/register`, `/portal/*` | — | — | — | Behind JWT: an anonymous call returns 401 (`TestLandingSurfaceEndpointsFailClosedAnonymously`). `/home` and `/portal` say a staff login is required instead of advertising customer self-service |

## Immediate revocation (TR-F033)

A token embeds `membership_version`, which is compared against the live
`TenantMembership.TokenVersion` on every authenticated request
(`resolveOperatorFromContext`, `internal/adminapi/auth.go:280`). `SysOpr.AfterUpdate`
(`internal/domain/system.go:106`) bumps that version whenever an operator's
level or status changes, so a downgrade, disable, or delete takes effect on the
**next** request -- no waiting for JWT expiry. `deleteOperator` removes the
membership grant in the same transaction as the account row.

Proven by `TestPrivilegeDowngradeInvalidatesLiveToken` in
`internal/adminapi/access_control_test.go`: a downgrade to `operator`, a
disable, and a delete all reject the operator's previously-issued token on a
real mutation endpoint (401), and a cross-tenant token cannot read, update, or
write into another tenant's rows (404, no persisted side effect).

## UI gating

The shared hook `web/src/hooks/useIsAdmin.ts` mirrors `requireAdmin`
(`super` and `admin`). Platform-only surfaces check the `platform_admin`
permission issued by the login response.

| UI surface | Gated for operator | File |
| --- | --- | --- |
| Customer/package/subscription/payment list create buttons and empty-state Create link | hidden | `web/src/resources/isp.tsx` |
| Customer edit and add-subscription | hidden | `web/src/resources/isp.tsx` |
| Package row click-to-edit | disabled | `web/src/resources/isp.tsx` |
| Subscription lifecycle and FUP buttons | hidden (invoice link remains) | `web/src/resources/isp.tsx` |
| Invoice record-payment and server WhatsApp send | hidden | `web/src/resources/isp.tsx` |
| Payment server WhatsApp send | hidden | `web/src/resources/isp.tsx` |
| RADIUS user create/import, edit save/delete, empty-state create hint | hidden | `web/src/resources/radiusUsers.tsx` |
| Ticket dispatch and flapping auto-ticket | hidden (ticket create/update remain) | `web/src/pages/TroubleTicketsPage.tsx` |
| Voucher generate/revoke/delete batch | hidden (print/filter remain) | `web/src/pages/HotspotVouchersPage.tsx` |
| Branding/website editor | hidden unless `platform_admin` | `web/src/components/CustomMenu.tsx`, `web/src/pages/BrandingPage.tsx` |

## Alternate payment paths

`POST /portal/invoices/:id/simulate-pay` always returns `410 SIMULATED_PAYMENT_DISABLED`.
`POST /portal/payments/webhook` returns `503 PAYMENT_PROVIDER_UNCONFIGURED` until a
verified provider is configured. The only write path is `POST /isp/payments`,
which is admin-only. The role-matrix test asserts both closures.

## Verification

| Check | Command |
| --- | --- |
| Route matrix (in-process router, real JWT claims) | `go test ./internal/adminapi/ -run TestMutationRouteRoleMatrix -count=1` |
| Operator privilege boundaries | `go test ./internal/adminapi/ -run TestUpdateOperatorPrivilegeBoundaries -count=1` |
| Full admin API unit suite | `go test ./internal/adminapi/ ./internal/webserver/ -count=1` |
| Browser role matrix (menu, create/import/print/dispatch controls) | `cd web && npx playwright test e2e/role-matrix.spec.ts --project=chromium --reporter=line` |
| Frontend type check | `cd web && npm run type-check` |

The matrix test seeds one `super`, one `admin` and one `operator`, calls the real
router with signed JWTs, and asserts 2xx/403 per row. It also proves that an
operator cannot record a payment through the portal simulation or webhook routes.

The Playwright spec is the UI half of the same matrix: it seeds operator, admin,
super, and `platform_admin` sessions into `localStorage` and asserts which menu
entries and action controls each role can see. It does not replace the server
test -- hiding a button is never treated as enforcement. The read-only API
fixtures in the spec return empty lists and canned rows; mutations are not part
of the browser suite because the Go matrix already covers them against the real
router.
