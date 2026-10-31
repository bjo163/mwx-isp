# Tenant Brand Experience — TR-F034 / TR-F035

## Purpose

Allow every ISP or RT/RW Net tenant in one MWX-ISP installation to present its own brand, website and package catalog while preserving the operational, security and billing boundaries already established by tenant isolation.

## Domain boundaries

1. **Platform Product Branding** (`ProductBranding`, TR-F029) is deployment-wide and identifies MWX-ISP. It never becomes tenant state.
2. **Tenant Legal Identity** (`Tenant`) is the legal/billing issuer identity: company name, tax ID, billing address and legal contact. It is not a marketing display name.
3. **Tenant Brand** (`TenantBranding`) controls visual identity and support-facing display details. Use fixed palette tokens and approved typography presets; never arbitrary CSS, HTML or JavaScript.
4. **Tenant Website** is structured section content with draft and published versions. Text is inert, links are allowlisted, and public output is a published allowlist only.
5. **Package Presentation** is a separate tenant-owned record keyed by tenant and `InternetPackage`. It must not mutate price, billing cycle, status, RADIUS profile, speed or FUP.
6. **Invoice/Receipt Presentation** (TR-F035) uses immutable issuer and template snapshots captured at document issuance.

## Package truth rules

- Public price is always `InternetPackage.Price`.
- Public speed is derived from the tenant-owned `RadiusProfile` referenced by the package.
- FUP/quota/rates are derived from operational package/profile fields.
- Presentation may add display name, category, badge, image, icon, highlights, order, visibility and CTA only.
- Presentation references must prove that the package belongs to the authenticated tenant.

## Tenant and domain security

- Resolve tenant only from trusted server-side tenant context or a verified hostname mapping; never accept a client tenant ID as authority.
- A configured subdomain is resolved against an active tenant record.
- A custom domain is not active until ownership verification is complete; reject duplicate, unverified and inactive mappings.
- Use trusted proxy forwarding headers only when the proxy is configured as trusted.
- All asset reads, updates and deletes are tenant-isolated. Remote assets must use HTTPS or a same-origin relative path; no protocol-relative URLs or arbitrary fetch targets.
- Tenant brand management is tenant-admin only. Operators are read-only.
- Public API responses expose only safe published fields and active public-visible packages.
- Public package catalog is read-only. Customer lookup, voucher checks, registration, payment and self-service endpoints remain fail-closed.

## Website publishing

- Content is section-based: header, hero, announcement, about, services, coverage, SLA, contact, social and footer.
- Each section supports visibility and ordering.
- Draft saves never alter the current public version.
- Publish creates an immutable version and invalidates only that tenant's cached projection.
- Rollback republishes a prior known version; do not delete version history.
- Content is plain text plus validated structured links; do not render arbitrary HTML.

## Brand assets

- Store files under generated tenant-scoped identifiers.
- Validate actual file bytes, decoded format, dimensions and size; ignore supplied filenames/content-type claims.
- Do not accept executable formats or untrusted SVG.
- Delete/replacement must not remove assets still referenced by published website versions, package presentations or invoice snapshots.
- Public image responses set explicit content type, `nosniff` and suitable cache policy.

## API surface

All routes live under `/api/v1`. Admin routes require a tenant JWT; public routes
resolve the tenant from a verified hostname or the identifier in the path.

| Route | Access | Purpose |
| --- | --- | --- |
| `GET/PUT /branding` | tenant admin (operators read-only) | Tenant brand settings |
| `POST /branding/assets`, `DELETE /branding/assets/:filename` | tenant admin | Tenant-scoped logo/social assets |
| `GET /tenant-branding/:identifier` | public | Allowlisted brand projection for an active tenant |
| `GET /tenant-branding/assets/:tenant/:filename` | public | Tenant brand asset bytes |
| `GET/PUT /tenant-website/draft` | tenant admin | Draft website content |
| `POST /tenant-website/publish` | tenant admin | Publish a new immutable version |
| `GET /tenant-website/versions` | tenant admin | Version history |
| `POST /tenant-website/versions/:version/rollback` | tenant admin | Restage and republish a prior version |
| `GET /tenant-website/public/:identifier` | public | Published projection only |
| `GET /tenant-domains`, `POST /tenant-domains`, `POST /tenant-domains/:id/verify`, `DELETE /tenant-domains/:id` | tenant admin | Verified custom hostnames |
| `GET /isp/package-presentations`, `GET/PUT /isp/packages/:id/presentation` | tenant admin | Package presentation metadata |
| `GET /public/packages` | public, rate-limited | Read-only public-visible catalog |

Do not make existing customer lookup, voucher, registration or payment APIs
anonymous.

## Operator workflow

1. Platform operator creates the tenant with a subdomain; the tenant admin is
   provisioned against that tenant.
2. Tenant admin sets brand tokens, uploads PNG assets and saves the website
   draft in **Brand Studio** and **Website Studio**.
3. Tenant admin publishes the website; only then does the public site and its
   cached projection change.
4. Tenant admin curates packages in **Package Studio**: display name, category,
   badge, image, highlights and order. Price, billing cycle, speed and FUP stay
   on the package and RADIUS profile.
5. Customers reach `https://<subdomain>.e-isp.id` (or a verified custom host)
   and see the published brand, website and public-visible catalog.
6. To change a live site, edit the draft and publish again; use rollback to
   restore a prior version without deleting history.
7. To change the public hostname, add the domain, publish the DNS TXT challenge
   and verify it; unverified hosts never resolve to the tenant.

## Verification evidence

- Unit: `internal/adminapi/branding_test.go`, `branding_editor_test.go`,
  `branding_upload_test.go`, `tenant_branding_assets_test.go`,
  `tenant_website_test.go`, `tenant_domains_test.go`,
  `package_presentation_test.go`, `public_surface_test.go`.
- PostgreSQL, two tenants: `test/integration/tenant_brand_experience_test.go`
  (`TestPostgresTenantBrandingAndWebsiteStayIsolated`,
  `TestPostgresTenantPackageCatalogIsolatesPresentation`) covers branding,
  website draft/publish/rollback history, cross-tenant writes, host-resolved
  public catalog, hidden packages and price integrity.
- Browser: `web/e2e/tenant-brand-experience.spec.ts` drives Brand Studio,
  Website Studio, Package Studio and both public sites with mocked APIs and
  asserts tenant isolation and that no `price` field is sent.

## TR-F035 document invariants

- Copy legal issuer details into an immutable invoice snapshot at issuance.
- Render invoice and receipt from that snapshot, not the tenant's current profile.
- Snapshot template/brand version for historical document fidelity.
- No template setting may change line items, totals, tax or balance, payment state, numbering or the ledger.
- Bank instructions and QR are display-only unless sourced from a verified actual payment integration.
- No fabricated account numbers or payment QR payloads.

## Acceptance

- Tenant A and B can independently manage and preview brand, website and package presentation.
- Tenant A cannot read or mutate tenant B's configuration, content, package references or assets.
- Tenant without explicit settings renders safe defaults without changing the global MWX brand.
- Package price, speed, FUP and lifecycle values remain derived from operational records.
- Draft/publish/rollback and per-tenant cache behavior are deterministic.
- Historical invoices remain visually and legally stable after later profile changes.
- PostgreSQL, API role/tenant-isolation, browser E2E and CI checks pass.

## Explicit non-goals

No arbitrary CMS, user-defined code/CSS, CRM, captive portal, public customer self-service, new order/payment system, reseller marketplace, tenant hierarchy or database-per-tenant architecture.