# Admin API response contract

The management frontend (React Admin over `web/src/providers/dataProvider.ts`)
depends on a small, stable set of shapes. This document is the reference those
handlers share; [`internal/adminapi/api_contract_test.go`](../internal/adminapi/api_contract_test.go)
enforces it so a handler cannot drift without a failing test.

## Success envelope

Every JSON success response uses the same wrapper (`internal/adminapi/responses.go`):

```json
{ "data": <object or array> }
```

A list response adds pagination metadata:

```json
{ "data": [ ... ], "meta": { "total": 42, "page": 2, "pageSize": 20 } }
```

Rules:

- `data` for a list is always an array. A query that matches nothing, or a page
  past the end, returns `"data": []` — never `null`. `paged()` normalizes nil
  slices, so handlers do not need to pre-allocate their result.
- `meta.total` is the number of rows matching the current filters (not the page
  length), `meta.page` and `meta.pageSize` echo the values actually applied
  after defaults and clamping.
- Single-object endpoints omit `meta`.
- `/api/v1/system/config/schemas` is the one collection read without
  pagination: it returns a plain array with no `meta` (the dataProvider falls
  back to the array length for `total`). The schema list is sorted by key.

## Pagination grammar

Query parameters: `page`, and either `pageSize` or React Admin's `perPage`.

- `page` defaults to `1`; a value that is missing, unparsable or below `1` becomes `1`.
- `pageSize` wins when both `pageSize` and `perPage` are present.
- A missing, unparsable or non-positive size falls back to the endpoint default.
- A size above the endpoint cap is clamped to the cap (it is not reset to the
  default), so a client asking for more rows still gets the largest page.

| Endpoint family | Default | Cap |
| --- | --- | --- |
| `/users`, `/isp/*`, system settings and other lists | 20 | 1000 |
| `/sessions`, `/accounting`, `/radius-profiles` | 10 | 100 |

The compact family keeps a smaller page and cap because those rows are the
widest the API returns.

## Filters

Filters are plain query parameters; React Admin sends them flat.

| Endpoint | Parameters |
| --- | --- |
| `/users` | `status` (exact, lowercased), `username`, `realname`, `email`, `mobile` (case-insensitive substring; LIKE wildcards escaped) |
| `/sessions` | `username`, `framed_ipv6_address`, `framed_ipv6_prefix`, `delegated_ipv6_prefix`, `mac_addr`, `acct_session_id` (substring); `nas_addr`, `framed_ipaddr` (exact); `acct_start_time_gte`, `acct_start_time_lte` |
| `/accounting` | `username`, `nas_addr`, `acct_session_id`, `framed_ipaddr`, `mac_addr`, `framed_ipv6_address`, `framed_ipv6_prefix`, `delegated_ipv6_prefix` (substring); `acct_start_time_gte`, `acct_start_time_lte` |
| `/radius-profiles` | `name`, `addr_pool`, `domain` (case-insensitive substring), `status` (exact) |
| `/isp/customers` | `status` (exact), `q` (substring over `customer_no`, `name`, `phone`) |
| `/isp/packages` | `status` (exact) |
| `/isp/subscriptions` | `status` (exact), `customer_id` |
| `/isp/invoices` | `status` (exact), `customer_id`, `invoice_no` (substring) |
| `/isp/payments` | `customer_id`, `invoice_id` |

Time-window bounds accept RFC 3339, HTML `datetime-local`, `2006-01-02 15:04:05`
and date-only values; an unparseable bound is ignored rather than rejected.

## Sorting

`sort` and `order` are honored by `/users`, `/sessions`, `/accounting` and
`/radius-profiles`. The sort field is matched against a per-endpoint allowlist
and re-derived from the allowlist key before it reaches the `ORDER BY` clause;
an unknown field falls back to the endpoint default. `order` is `ASC` or `DESC`
(case-insensitive); anything else falls back to the default direction.

| Endpoint | Default order |
| --- | --- |
| `/users` | `username ASC` |
| `/sessions` | `acct_start_time DESC` |
| `/accounting` | `acct_start_time DESC` |
| `/radius-profiles` | `id DESC` |
| `/isp/customers`, `/isp/packages`, `/isp/subscriptions` | `id DESC` |
| `/isp/invoices` | `invoice_date DESC, id DESC` |
| `/isp/payments` | `paid_at DESC, id DESC` |

The `/isp/*` lists do not support `sort`; sending it is accepted and ignored so
the React Admin dataProvider can always include its default sort parameters.

## Error envelope

```json
{ "error": "USER_NOT_FOUND", "message": "User not found" }
```

- `error` is a stable machine-readable code; `message` is human-readable;
  `details` is optional structured context.
- `details` is always stripped for 5xx responses (and logged server-side with
  the path and code) so database and schema details never reach a client.
- No other top-level keys are emitted.

Common codes: `UNAUTHORIZED` (401, no resolvable operator), `PERMISSION_DENIED`
(403, insufficient level), `PLATFORM_ACCESS_REQUIRED` (403, platform-only
route), `INVALID_ID` (400, non-numeric path id), `NOT_FOUND` / `USER_NOT_FOUND`
/ `SETTING_NOT_FOUND` (404), `VALIDATION_ERROR` (400/422, with `details`),
`DATABASE_ERROR` (500, details stripped).

## Contract tests

`api_contract_test.go` pins the above per endpoint:

- `TestContractListPaginationGrammar` — envelopes, `pageSize`/`perPage`
  precedence, defaults, caps, empty pages, and that `meta.total` counts the
  filtered rows.
- `TestContractListSortFallback` — an injection-shaped `sort`/`order` stays 200
  with the documented default order, the schema survives, and the user list
  never returns the stored password.
- `TestContractUnpaginatedConfigSchemas` — the one unpaginated collection.
- `TestErrorContractEnvelope` — 401/403/400/404 envelopes and validation
  mapping, with no extra keys.
- `TestDataProviderResourcesHaveContractCoverage` — parses `resourcePathMap`
  from the frontend dataProvider and fails when a mapped resource has no
  contract case, so a new UI resource cannot ship without a pinned payload.

To add a resource, add a `contractListEndpoint` entry (or an unpaginated case)
in `pagedListEndpoints()` / `unpaginatedListEndpoints()`; the coverage test
points at the missing path.
