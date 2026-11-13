# Deployment

## Docker Compose (Coolify)

Lihat `docker-compose.coolify.yml` di repo.

- Service `db`: postgres:16-alpine3.24, volume `postgres_data`
- Service `app`: `ghcr.io/bjo163/mwx-isp:latest`, `pull_policy: always`
- Ports: 1816 (web), 1812/1813 (RADIUS UDP), 2083 (RADSEC TCP)
- Healthcheck: `curl http://127.0.0.1:1816/admin/`

## Environment Variables

| Var | Default | Keterangan |
|-----|---------|-----------|
| `TOUGHRADIUS_WEB_HOST` | 0.0.0.0 | Web host |
| `TOUGHRADIUS_WEB_PORT` | 1816 | Web port |
| `TOUGHRADIUS_WEB_TLS_ENABLED` | false | TLS |
| `TOUGHRADIUS_WEB_SECRET` | required | Web secret |
| `TOUGHRADIUS_ADMIN_PASSWORD` | required | Admin password |
| `TOUGHRADIUS_DB_TYPE` | postgres | Database type |
| `TOUGHRADIUS_DB_HOST` | db | Database host |
| `TOUGHRADIUS_DB_PORT` | 5432 | Database port |
| `TOUGHRADIUS_DB_NAME` | mwxisp | Database name |
| `TOUGHRADIUS_DB_USER` | mwxisp | Database user |
| `TOUGHRADIUS_DB_PWD` | required | Database password |
| `TOUGHRADIUS_DB_SSLMODE` | require | PostgreSQL TLS mode; use `verify-full` for external production DB |
| `TOUGHRADIUS_DB_SSLROOTCERT` | empty | CA bundle path for PostgreSQL TLS verification |
| `TOUGHRADIUS_DB_SSLCERT` / `TOUGHRADIUS_DB_SSLKEY` | empty | Optional client certificate pair |
| `TOUGHRADIUS_DB_CONNECT_TIMEOUT` | 5 | Connection timeout in seconds (1–60) |
| `TOUGHRADIUS_RADIUS_ENABLED` | true | Enable RADIUS |
| `TOUGHRADIUS_RADIUS_HOST` | 0.0.0.0 | RADIUS host |
| `TOUGHRADIUS_RADIUS_AUTHPORT` | 1812 | Auth port |
| `TOUGHRADIUS_RADIUS_ACCTPORT` | 1813 | Accounting port |
| `TOUGHRADIUS_RADIUS_RADSEC_PORT` | 2083 | RADSEC port |
| `TOUGHRADIUS_METRICS_TOKEN` | (empty) | Metrics token |
| `TOUGHRADIUS_LOGGER_MODE` | production | Logger mode |

## Online payments (optional)

To enable Midtrans Snap, set `TOUGHRADIUS_PAYMENT_PROVIDER=midtrans`, choose `TOUGHRADIUS_MIDTRANS_ENVIRONMENT=sandbox` for testing (or `production` for live transactions), and provide the Server Key and Client Key through the deployment secret manager. The webhook endpoint is `POST /api/v1/portal/payments/webhook`; configure that exact HTTPS URL in the Midtrans dashboard. The endpoint is excluded from staff JWT middleware but validates the provider's SHA-512 signature, order ID, amount, and currency before posting to the billing ledger. Duplicate callbacks are idempotent. Partial refunds/chargebacks and mismatches require operator reconciliation. Never commit real keys to Git.

## Database safety

PostgreSQL is fail-closed: if the configured host, credentials, TLS handshake, or initial health check fails, the process exits before migrations, schedulers, or listeners start. It never falls back to a new local SQLite file. SQLite is used only when `TOUGHRADIUS_DB_TYPE=sqlite` is explicitly selected.

The bundled Compose Postgres service is reached over the private Docker bridge and uses `TOUGHRADIUS_DB_SSLMODE=disable` by default. For an external PostgreSQL service, set `TOUGHRADIUS_DB_SSLMODE=verify-full` and provide the CA bundle through `TOUGHRADIUS_DB_SSLROOTCERT`. Never use `disable` over an untrusted network.

## Auto-Deploy

- Coolify GitHub App otomatis deploy dari `main`
- `docker-compose.coolify.yml` menarik `:latest` dengan `pull_policy: always`
- Merge ke `main` → container restart dengan image rilis terbaru
- Image baru ada setelah tag `v*` menyelesaikan `release.yml`

## Production

- Domain: `e-isp.id`
- Admin: `http://e-isp.id/admin/`

---

*Last updated: 2026-11-02*
