# MWX-ISP

RADIUS server with billing, LDAP, EAP, and vendor enhancers.

## Overview

- **Core**: RADIUS authentication/authorization/accounting
- **Billing**: Invoice generation, overdue enforcement, payment processing
- **Auth**: EAP-TLS, EAP-TTLS, PEAP, TEAP, PAP, CHAP, MS-CHAP, MS-CHAPv2
- **VSA**: Vendor-specific attributes (Alcatel, Juniper, Microsoft, F5, Hillstone, pfSense, etc.)
- **LDAP/AD**: Bind authentication backend for PAP-family methods
- **Metrics**: Prometheus text exposition at `/metrics` (token-gated)
- **Multi-arch**: Builds for amd64, arm64, arm/v7, loong64, riscv64

## Quick Start

```bash
# Build
go build ./...

# Run unit tests
go test -v -short ./...

# Run integration tests (requires Docker)
TEST_DATABASE_HOST=127.0.0.1 TEST_DATABASE_PORT=15432 \
TEST_LDAP_URL=ldap://127.0.0.1:1389 INTEGRATION_REQUIRED=1 \
go test -tags=integration -v ./test/integration/...

# Build multi-arch (Docker)
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t mwx-isp:latest .
```

## Configuration

Environment variables (see `config/` directory):
- `TOUGHRADIUS_METRICS_TOKEN` - Prometheus metrics token
- `DATABASE_URL` - PostgreSQL connection
- `LDAP_URL` - LDAP server URL
- `RADIUS_SECRET` - RADIUS shared secret

## Project Structure

```
internal/     - Core packages (billing, auth, LDAP, etc.)
cmd/          - Command-line tools (radtest, config-tool, etc.)
pkg/          - Shared utilities
test/         - Integration tests (Docker required)
web/          - Frontend assets (embedded via go:embed)
```

## Release

Automated via GitHub Actions:
- Push to `main` → bump version → publish GHCR image + GitHub Release
- Tag `vX.Y.Z` → release assets published

## Single Work Queue

All work lives in [`Todo.md`](Todo.md). Mark items as completed there.