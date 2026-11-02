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
| `TOUGHRADIUS_RADIUS_ENABLED` | true | Enable RADIUS |
| `TOUGHRADIUS_RADIUS_HOST` | 0.0.0.0 | RADIUS host |
| `TOUGHRADIUS_RADIUS_AUTHPORT` | 1812 | Auth port |
| `TOUGHRADIUS_RADIUS_ACCTPORT` | 1813 | Accounting port |
| `TOUGHRADIUS_RADIUS_RADSEC_PORT` | 2083 | RADSEC port |
| `TOUGHRADIUS_METRICS_TOKEN` | (empty) | Metrics token |
| `TOUGHRADIUS_LOGGER_MODE` | production | Logger mode |

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
