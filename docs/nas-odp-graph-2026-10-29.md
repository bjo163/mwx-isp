# Static NAS-to-ODP graph, a bounded study (FUT-006)

Status: research note and prototype, 2026-10-29 on `dev`.
Scope: `TR-F031` network operations inventory; builds on the tenant
consistency work in `ENG-018`.
Non-goal: a new admin page, topology discovery, SNMP/LLDP probing, or any
device write. Nothing in this study changes the ODP registry, the NOC impact
panel, or the API.

## Question

The ODP registry already records which customers sit on which port, and each
ODP has an `olt_name` and `pon_port` field. The question is whether those
operator-entered relations are enough to draw a **static** NAS-to-ODP graph, or
whether the graph would have to invent links the operator never entered.

## What is actually stored

| Relation | Column | Strength |
| --- | --- | --- |
| ODP to its OLT | `isp_odp.olt_name` | Free text, no uniqueness, no FK |
| ODP PON port | `isp_odp.pon_port` | Free text, not parsed |
| Customer to ODP | `isp_customer.odp_id`, `odp_port` | Validated by `resolveODPAssignment` (tenant, capacity, port occupancy) |
| RADIUS user to node | `radius_user.node_id` -> `net_node.id` | FK by convention |
| NAS to node | `net_nas.node_id` -> `net_node.id` | FK by convention |
| ODP to NAS | — | **Does not exist** |

The only path from an ODP to a NAS is the OLT name, and the only operator-facing
device names are `net_nas.name`, `hostname` and `identifier`. Both tables are
tenant-scoped, and `ENG-018` established that the same NAS IP or username can
exist in two tenants, so any name match must stay inside one tenant.

## Prototype

`internal/topology` (analysis only, imported by nothing) builds the graph:

- an ODP links to a NAS when its trimmed, case-folded `olt_name` equals the
  NAS `name`, then `hostname`, then `identifier`, **within the same tenant**;
- when the winning field matches several NAS the edges are kept with
  `Ambiguous` set — the tool never picks one;
- an ODP that matches only another tenant's NAS is reported as
  `olt_name_cross_tenant` and left unlinked;
- ODP occupancy comes from distinct non-terminated customer ports, matching
  `listODPs`; more occupied ports than `total_ports` is reported as
  `odp_port_overflow`;
- `RenderSVG` draws the result as one static SVG document with no script and no
  external reference, with dashed edges for candidates and dashed boxes for
  unlinked ODPs.

Finding kinds: `odp_unlinked`, `olt_name_ambiguous`, `olt_name_cross_tenant`,
`odp_port_overflow`.

## What a demo tenant looks like

Five ODPs, two NAS, one typo and one empty OLT name:

```
2 NAS, 5 ODP, 3 linked, 2 unlinked, 0 ambiguous, 4 customers (0 without ODP)
```

```
<line x1="320" y1="134" x2="640" y2="190" stroke="#3b82f6" stroke-width="1.5"/>
...
<rect x="640" y="284" width="280" height="36" rx="4" fill="#1e293b" stroke="#f59e0b" stroke-dasharray="6 4"/>
<text x="652" y="300" fill="#e2e8f0">ODP-SDR-002 (unlinked)</text>
```

`ODP-SDR-002` records `OLT-ZTE-2` while the NAS is `OLT-ZTE-02`; `ODP-MJL-001`
has no OLT name at all. Both are real operator mistakes that a "best guess"
graph would have hidden behind an edge.

## Findings

1. **The graph is only as good as the OLT name.** A single character difference
   silently unlinks a whole splitter, and the registry accepts that state
   without a warning.
2. **Names are not unique identifiers.** Two NAS in one tenant may share a name
   (`net_nas` has no unique constraint on `name`); the graph keeps both
   candidates instead of selecting the first row.
3. **Cross-tenant name collisions exist by design.** `ENG-018` showed two
   tenants can register the same NAS identifier; a global name index would
   silently join an ODP to another tenant's device. The tenant boundary is
   enforced in the matcher before any edge is considered.
4. **`pon_port` is opaque.** `0/1/2` is stored as text and no code parses it, so
   the graph cannot place an ODP on a PON port or detect two ODPs on the same
   port.
5. **Occupancy is derivable and already derived.** `listODPs` computes used
   ports from customers; the graph reproduces that rule, and the only new
   signal is an overflow, which the registry itself never reports.

## Decisions

1. **Do not ship the graph as product UI yet.** Rendering a name-matched edge as
   a fact would turn a typo into a topology. The precondition for a product
   page is an explicit ODP-to-NAS relation (a nullable `nas_id` on `isp_odp`,
   validated on write, or an OLT registry row), not a better matcher.
2. **A static SVG is sufficient when the edge set is trustworthy.** The
   renderer is small, deterministic, script-free and printable, so the visual
   side is not the blocker; the data model is.
3. **Keep ambiguity visible.** Dashed candidate edges and dashed unlinked boxes
   are deliberate: an operator can see the uncertainty instead of a confident
   wrong line.
4. **Match only inside one tenant.** Cross-tenant near-misses are reported,
   never drawn, following `ENG-018`.

## Verification

```
go test ./internal/topology/ -count=1
ok  	github.com/bjo163/mwx-isp/internal/topology	1.420s

golangci-lint run ./internal/topology/...
0 issues.

go test ./internal/tenancy/ -count=1      # the raw SQL ratchet is unaffected
ok  	github.com/bjo163/mwx-isp/internal/tenancy	1.452s
```

Unit coverage (`internal/topology/graph_test.go`):

- `TestBuildLinksODPByOLTName` — trimmed case-insensitive name match.
- `TestBuildFallsBackToHostnameAndIdentifier` — field priority after the name.
- `TestBuildReportsUnlinkedAndEmptyOLTName` — an empty name and a typo.
- `TestBuildReportsAmbiguousOLTName` — two same-named NAS produce two candidate
  edges, not a silent choice.
- `TestBuildDoesNotCrossTenant` — a name that only exists in another tenant is
  reported and not linked.
- `TestBuildCountsDistinctPortsAndOverflow` — duplicate ports count once,
  terminated customers release theirs, and overflow is reported.
- `TestRenderSVGIsDeterministicAndEscaped` — identical output for identical
  input, XML-escaped labels, dashed candidates.
- `TestBuildFromDBReadsStoredRows` — the SQLite loader builds the same edge from
  persisted rows.

## Residual limits

- Matching is by device name only; there is no IP, MAC, SNMP or LLDP evidence,
  and adding any of those is discovery, which `TR-F031` explicitly excludes.
- `net_node` is ignored: it groups devices but carries no ODP relation, so it
  cannot create an edge the operator did not type.
- The SVG is a fixed two-column layout sized for a handful of devices; a tenant
  with hundreds of ODPs needs grouping by OLT/zone and pagination before it is
  readable.
- Optical loss, coordinates and `pon_port` are stored but not used; placing ODPs
  geographically or on a PON would need a coordinate/PON layout, not a graph
  edge.
- The study ran against hand-built SQLite fixtures, not a production inventory;
  the share of unlinked ODPs in a real tenant is unknown.
