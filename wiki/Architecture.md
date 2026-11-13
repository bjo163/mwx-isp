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


## FTTH inventory and provisioning boundary

The additive `NetworkAsset`, `FiberLink`, `SubscriberDeviceBinding`, and `ProvisioningJob` models form a tenant-scoped inventory/provisioning foundation. They complement existing `ODP`, `NetNas`, customer ODP fields, and IPAM data; no existing records are rewritten. Device credentials are represented only by a secret reference, never plaintext. The vendor-neutral adapter contract lives in `internal/provisioning`. It has no live vendor implementation yet: discovery/observe are the first integration stage, while apply requires an explicit non-dry-run plan and authenticated operator approval. No device changes are performed by this release.
