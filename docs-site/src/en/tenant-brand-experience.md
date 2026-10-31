# Tenant Brand Experience

> Bahasa Indonesia: [Pengalaman Brand Tenant](../id/tenant-brand-experience.md)

MWX-ISP serves several ISPs and RT/RW Net organisations from one installation.
Each tenant presents its own brand, website and package catalogue without
gaining access to another tenant's data. This chapter covers the three studios
an operator uses, the public surfaces they feed, and the boundaries that keep
them apart.

The domain wording lives in
[`docs/tenant-brand-experience.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/tenant-brand-experience.md);
this page is the operator handbook.

## Roles

| Role | Brand Studio | Website Studio | Package Studio | Public site |
| --- | --- | --- | --- | --- |
| Tenant `admin` / `super` | Read and write | Read, save, publish, roll back | Read and write presentation | Read |
| Tenant `operator` | Read only | Read only | Read only | Read |
| Platform `platform_admin` | Not applicable | Not applicable | Not applicable | Read |

Every write is checked on the server. Hiding a control in the browser never
grants or removes a permission; the endpoint itself rejects the request.

## Brand Studio

Route: `#/tenant/branding`.

Brand Studio stores the marketing identity of the signed-in tenant:

- brand name, short name, tagline and legacy company display name;
- logo variants (primary, compact, light, dark), favicon and social image;
- a fixed palette (primary, secondary, accent, background, surface, text, link);
- theme (`light`, `dark`, `system`) and typography preset;
- support email, phone, WhatsApp number, website and public address.

Rules enforced by the API:

- Colours must be six-digit hexadecimal values; arbitrary CSS is never stored.
- Image references must be a same-origin relative path or an absolute HTTPS URL;
  protocol-relative URLs are rejected.
- Uploaded assets must be PNG, are re-encoded server-side, and are stored under
  tenant-scoped names. Delete and replacement refuse to remove a file that a
  published website version, package presentation or invoice snapshot still
  references.
- The legal issuer identity (company name, tax ID, billing address) stays on the
  `Tenant` record and is never edited here.

## Website Studio

Route: `#/tenant/website`.

The website is structured section content, not free HTML. The ten sections are
`header`, `hero`, `announcement`, `about`, `services`, `coverage`, `sla`,
`contact`, `social` and `footer`; each carries title, subtitle, body, image,
call-to-action label and URL, visibility and order.

Workflow:

1. Edit sections in the **Section editor**. The live draft preview shows the
   selected section as text.
2. **Save draft** stores the draft only. The public site keeps the published
   version unchanged.
3. **Publish website** snapshots the draft into an immutable version and updates
   the public projection for this tenant alone.
4. **Restore** republishes an earlier version as a new version. History is
   append-only; restoring never deletes a snapshot.

Draft saves are limited to 96 KiB, unknown section keys and duplicate keys are
refused, and call-to-action URLs accept HTTPS, `mailto:`, `tel:` or a
same-origin relative path only.

## Verified hostnames

The public site and catalogue resolve the tenant from the request hostname, never
from a client-supplied tenant identifier.

- A managed subdomain (`<tenant>.e-isp.id`) resolves against an active tenant.
- A custom domain must pass a DNS ownership challenge: MWX-ISP writes a random
  token, the tenant publishes `_mwx-isp-verification.<hostname>` with
  `mwx-isp-verification=<token>`, and only then does the mapping become active.
- Unverified, duplicate and inactive mappings are rejected. Hostnames under
  `e-isp.id` cannot be registered as custom domains.
- Forwarded headers are trusted only when the reverse proxy is declared
  trusted.

## Package Studio

Route: `#/isp/package-studio`.

Package Studio edits the *presentation* of an operational package. It can add a
display name, short description, category, icon, image, badge, highlight colour,
highlights, CTA label, terms, display order, featured flag and public visibility.

It cannot change the operational truth:

| Field | Source of truth |
| --- | --- |
| Price | `InternetPackage.Price` |
| Billing cycle | `InternetPackage.BillingCycle` |
| Status | `InternetPackage.Status` |
| Speed | `RadiusProfile.UpRate` / `DownRate` of the referenced profile |
| FUP | `InternetPackage.FupLimitGB` |

The studio shows the operational price as read-only text and states its source
above the editor. A payload that tries to send `price` is ignored: the server
writes presentation fields only. Every package write proves the package belongs
to the authenticated tenant, so another tenant's package answers `404`.

## Public surfaces

- Website: `/#/site/<slug-or-verified-host>` renders the published, visible
  sections in order. A tenant without a published version sees a notice; no other
  tenant's content is substituted.
- Branding: `GET /api/v1/tenant-branding/<identifier>` returns the safe display
  fields only.
- Catalogue: `GET /api/v1/public/packages` returns active packages of the resolved
  tenant, honours `public_visible`, and exposes `price`, derived speed and FUP as
  read-only operational data.

All three are read-only. Customer lookup, voucher check, registration, payment
and self-service endpoints remain fail-closed and are not opened by this feature.

## API surface

All administrative routes require a tenant-scoped JWT and reuse the standard
`{"data": …}` success envelope and `{"error", "message"}` failure envelope
described in [API contract](https://github.com/bjo163/mwx-isp/blob/main/docs/api-contract.md).

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` / `PUT` | `/api/v1/branding` | Read or update the tenant brand |
| `POST` | `/api/v1/branding/assets` | Upload a PNG brand asset |
| `DELETE` | `/api/v1/branding/assets/:filename` | Delete an unreferenced asset |
| `GET` / `PUT` | `/api/v1/tenant-website/draft` | Read or save the website draft |
| `POST` | `/api/v1/tenant-website/publish` | Publish the draft as a new version |
| `GET` | `/api/v1/tenant-website/versions` | List published versions |
| `POST` | `/api/v1/tenant-website/versions/:version/rollback` | Restore a version |
| `GET` / `POST` | `/api/v1/tenant-domains` | List or request a custom domain |
| `POST` | `/api/v1/tenant-domains/:id/verify` | Verify the DNS challenge |
| `DELETE` | `/api/v1/tenant-domains/:id` | Remove a custom domain |
| `GET` | `/api/v1/isp/package-presentations` | List presentations of this tenant |
| `GET` / `PUT` | `/api/v1/isp/packages/:id/presentation` | Read or update one presentation |
| `GET` | `/api/v1/public/packages` | Public catalogue of the resolved tenant |
| `GET` | `/api/v1/tenant-website/public/:identifier` | Public published website |
| `GET` | `/api/v1/tenant-branding/:identifier` | Public display branding |

## Verification evidence

- `test/integration/tenant_brand_experience_test.go` drives two real tenant
  admins over HTTP against PostgreSQL: branding isolation, unpublished drafts,
  publish/rollback history, hostname-resolved catalogue, hidden packages,
  cross-tenant presentation writes and price integrity.
- `web/e2e/tenant-brand-experience.spec.ts` drives both tenants through Brand
  Studio, Website Studio, Package Studio and the public site in a browser.
- Unit suites `branding_test.go`, `tenant_branding_assets_test.go`,
  `branding_editor_test.go`, `tenant_website_test.go`, `tenant_domains_test.go`
  and `package_presentation_test.go` pin the handler-level validation rules.
