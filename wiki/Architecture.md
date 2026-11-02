# Arsitektur MWX-ISP

## Komponen Utama

- **`cmd/`** — entrypoint binary (radtest, demo-seed, isp-simulator, dll)
- **`internal/radiusd/`** — RADIUS server core (auth, accounting, CoA, RADSEC)
- **`internal/radiusd/plugins/`** — EAP handlers (MD5, TLS, PEAP, TTLS, TEAP), vendor parsers, accounting handlers
- **`internal/adminapi/`** — HTTP admin API (Echo)
- **`internal/webserver/`** — web server (metrics, TLS)
- **`internal/billing/`** — billing service
- **`internal/ldapauth/`** — LDAP backend
- **`internal/vendors/`** — 14 vendor VSA packages (alcatel, aruba, cisco, dll)
- **`web/`** — frontend (embedded via `//go:embed dist/*`)

## Data Flow

```
NAS → RADIUS (1812/1813) → radiusd → plugins → backend (Postgres/LDAP)
                              ↓
                          RADSEC (2083) → TLS → peer
                              ↓
                          Web Admin (1816) ← adminapi
```

## Teknologi

- Go 1.26
- Echo (HTTP)
- GORM (Postgres default)
- RADSEC (RADIUS over TLS)
- EAP: MD5, TLS, PEAP, TTLS, TEAP (RFC 9930, 9427)

## Multi-tenancy

- Tenant branding (invoice, domain, subdomain)
- Per-tenant database schema isolation

---

*Last updated: 2026-11-02*
