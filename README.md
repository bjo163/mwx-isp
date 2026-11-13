# MWX-ISP

RADIUS server plus ISP billing: authentication, authorization, accounting, vouchers, invoices and a React admin console embedded in the binary.

| | |
| --- | --- |
| RADIUS | PAP, CHAP, MS-CHAP, MS-CHAPv2, EAP-MD5, EAP-TLS, PEAP, EAP-TTLS, EAP-TEAP (RFC 9930), CoA/Disconnect, RADSEC (TLS) |
| Billing | vouchers, packages, invoices/receipts, overdue enforcement, FUP quotas per tenant |
| Backends | PostgreSQL (production) or SQLite, LDAP/AD bind auth for PAP-family methods |
| Multi-tenancy | every query is tenant-scoped; per-tenant limits, branding, metrics |
| Vendors | VSAs for Alcatel, Aruba, Cisco, F5, H3C, Hillstone, Huawei, iKuai, Juniper, Microsoft, MikroTik, pfSense, Radback, ZTE |
| Observability | Prometheus text endpoint at `/metrics` (bearer token), online-session stats |
| Artefacts | 9 static binaries + multi-arch image `linux/amd64`, `arm64`, `arm/v7` |

## Requirements

Go 1.26, Node.js 20, and Docker only if you want the integration suite or the container image.

## Build and run

The admin console is embedded with `//go:embed dist/*` ([web/static.go:12](web/static.go#L12)), so the frontend has to exist before the Go build.

```bash
npm --prefix web ci
npm --prefix web run build     # writes web/dist/admin
go build -o mwx-isp main.go
./mwx-isp -v                   # version stamp; -h for flags, -initdb to create schemas
```

Then open `http://localhost:1816/`. The first boot prints the generated admin password unless `TOUGHRADIUS_ADMIN_PASSWORD` is set.

## Ports

| Port | Protocol | Purpose |
| --- | --- | --- |
| 1816 | tcp | admin API + embedded console |
| 1817 | tcp | same, TLS (only when `web.tls_enabled` is on) |
| 1812 | udp | RADIUS authentication |
| 1813 | udp | RADIUS accounting |
| 2083 | tcp | RADSEC (TLS), needs the certs under the work dir |

## Configuration

Config is a YAML file, resolved in this order (see [config/config.go:534](config/config.go#L534)): explicit `-c` path, `./mwx-isp.yml`, `./toughradius.yml`, `/etc/mwx-isp.yml`, `/etc/toughradius.yml`, then built-in defaults.

Any value can be overridden from the environment with the `TOUGHRADIUS_` prefix. PostgreSQL settings include `TOUGHRADIUS_DB_SSLMODE` (`require` by default; use `verify-full` with a CA for external production databases), `TOUGHRADIUS_DB_SSLROOTCERT`, `TOUGHRADIUS_DB_SSLCERT`, `TOUGHRADIUS_DB_SSLKEY`, and `TOUGHRADIUS_DB_CONNECT_TIMEOUT` (5 seconds by default). The included Compose stacks explicitly use `disable` only for their private Docker bridge to the bundled PostgreSQL service. A failed PostgreSQL connection now aborts startup; MWX-ISP never creates a hidden SQLite fallback.

Beyond the YAML overrides there are three standalone variables:

| Variable | Effect |
| --- | --- |
| `TOUGHRADIUS_ADMIN_PASSWORD` | bootstrap admin password during `-initdb` |
| `TOUGHRADIUS_METRICS_TOKEN` | bearer token required by `GET /metrics`; unset means metrics stay off |
| `MWX_PLATFORM_ADMIN_USERNAME` / `MWX_PLATFORM_ADMIN_PASSWORD` | platform superuser for tenant provisioning |

`env.coolify` is the annotated template for container deployments.

## Tests

```bash
go test -short ./...                       # unit suite, no external services
golangci-lint run ./...                    # uses .golangci.yml
```

The integration suite talks to a real PostgreSQL and OpenLDAP. CI starts both with `docker run`; locally:

```bash
docker run -d --name postgres -e POSTGRES_USER=mwxisp_test -e POSTGRES_PASSWORD=test-only-password \
  -e POSTGRES_DB=postgres -p 15432:5432 postgres:16-alpine
docker run -d --name openldap -e LDAP_ROOT=dc=example,dc=org -e LDAP_ADMIN_USERNAME=admin \
  -e LDAP_ADMIN_PASSWORD=adminpassword -e LDAP_USERS=alice -e LDAP_PASSWORDS=alice-password \
  -p 1389:1389 bitnamilegacy/openldap:2.6
TEST_DATABASE_HOST=127.0.0.1 TEST_DATABASE_PORT=15432 TEST_DATABASE_USER=mwxisp_test \
TEST_DATABASE_PASSWORD=test-only-password TEST_DATABASE_NAME=postgres \
TEST_LDAP_URL=ldap://127.0.0.1:1389 TEST_LDAP_BASE_DN=dc=example,dc=org \
TEST_LDAP_ADMIN_DN=cn=admin,dc=example,dc=org TEST_LDAP_ADMIN_PASSWORD=adminpassword \
go test -tags=integration -count=1 -v ./test/integration/...
```

Without `INTEGRATION_REQUIRED=1` the suite skips quietly when those services are missing.

## Containers

`docker build -t mwx-isp:develop .` cross-compiles from the multi-stage `Dockerfile` (frontend stage, Go stage, Alpine runtime as non-root uid 10001).

```bash
docker compose up -d          # db + app on localhost, see docker-compose.yml
```

On Coolify, point *Docker Compose Location* at `docker-compose.coolify.yml`: it ships `db` + `app` only, because the platform's Traefik already owns 80/443 and terminates TLS for the app. Custom domains need hand-written Traefik labels in that file — Coolify does not generate them for compose apps — and no custom networks, or Traefik loses the route and returns 503.

## Release

Tag-driven, no bot:

```bash
# 1. edit VERSION and prepend a CHANGELOG.md block
git commit -am "chore(release): v0.23.0" && git push origin dev
# 2. open the dev -> main PR, merge once CI is green
git tag v0.23.0 && git push origin v0.23.0
```

Pushing a `v*` tag runs `.github/workflows/release.yml`, which builds the frontend, cross-compiles the 9 binaries with `checksums.txt`, publishes `ghcr.io/bjo163/mwx-isp:<version>` plus `:latest`, and creates the GitHub Release. Run the same workflow by hand (`Run workflow` → tag) to publish a **draft** release when you want to prove the pipeline without shipping.

`.github/workflows/ci.yml` is the gate on every push and PR to `main`/`dev`: frontend build, unit tests, integration tests, lint, govulncheck, cross-compile.

## Layout

```
main.go          entry point, flag parsing, server wiring
config/          YAML + TOUGHRADIUS_* environment configuration
internal/
  adminapi/      Echo HTTP handlers for the admin console
  webserver/     Echo server, JWT auth, static asset serving, /metrics
  radiusd/       RADIUS server: auth validators, EAP methods, vendors, CoA
  app/           database init, config manager, runtime settings
  billing/ invoicedoc/ revenue/ domain/   plans, invoices, money, domain types
  tenancy/       tenant scoping shared by API and RADIUS paths
  ldapauth/ networkmonitor/ notify/ syslogd/   LDAP bind, NAS health, alerts
cmd/             radtest, certgen, config-tool, reset-password, rotate-secrets,
                 demo-seed, isp-simulator, benchmark, support-bundle
pkg/             certgen, common, excel, metrics, timeutil, validator, validutil, web
test/integration PostgreSQL + OpenLDAP acceptance suites, build tag `integration`
web/             React admin console (Vite), embedded into the binary
docs/rfcs/       the two specifications the TEAP tests check themselves against
```

Work queue: [`Todo.md`](Todo.md).
