# FUT-001 — What a read-only operator intent search could resolve today

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `FUT-001` (`TR-F013`, P3, RISET, dependency `ENG-052` delivered) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/intentsearch` (read-only, imported by nothing).
Criterion under test: **20 prompt fixture mapped dengan alasan, salah tafsir tidak
memicu side effect.**

## The question

The card asks for a bounded intent parser that "menuju menu / menyusun filter
tanpa menjalankan mutasi NAS atau billing". Two halves, and the second one is the
hard one:

1. Given what the console actually exposes today, what could a typed prompt
   resolve to - which screens exist, and which filters can be composed for them?
2. When the parser is wrong - it misreads a mutation as a view, it ties between
   two screens, it matches nothing - what does "no side effect" mean in code, and
   can it be shown rather than promised?

The 20-fixture corpus is the deliverable, with a stated reason per prompt. The
safety half is answered by construction plus tests: a resolution can only ever
name a `GET` registration, so a misread produces an empty target list rather than
a partially-applied write.

## What counts as evidence

The same three rules the `ENG-006`, `ENG-007` and `ECO-006` studies used, enforced
by tests in this package:

- **A citation is a code line.** Comment-only lines are blanked before a symbol is
  matched (`TestClaimsAllLandOnCodeLines` reads through `blankComments`).
- **A whole identifier.** Word-boundary matching, so `q` cannot be proven by
  `query` (`containsBounded`).
- **Drift is reported, never dropped.** A vanished file is `broken`; a gap that
  got filled is `vacated` (`TestGapsStayOpenAndTheScanIgnoresItself`, and the
  absence walk skips `internal/intentsearch/` so the prototype cannot refute
  itself).

Three rules specific to this study:

- **No invented menu.** Every intent target must be a `<Resource>` name in
  `web/src/App.tsx` with a real read route behind it; the parser refuses to be
  constructed otherwise (`TestEveryCataloguedTargetExistsInTheTree`).
- **No invented filter value.** A status value only appears when the page's own
  `StatusValue[]` array contains it (`TestFilterValuesOnlyComeFromThePageVocabulary`).
- **No write is ever named.** The corpus is asserted to produce zero targets that
  are not a `GET` route (`TestEveryNamedTargetIsAReadRoute`,
  `TestTheVerdictLineStatesTheSafetyProperty`), and the package itself is asserted
  to contain no writer, no HTTP client and no database import
  (`TestThePrototypeWritesNothing`).

## The prototype

| File | Role |
| --- | --- |
| `surface.go` | Scans the admin API registrations, the menu, the filter arrays and the console's own menu-name-to-path table. |
| `intent.go` | The intent catalog and the resolver: deny first, then score, then absorb, then compose. |
| `fixtures.go` | The 20 prompts with their reasons, and the scorer that pairs expectation with output. |
| `claims.go` | 18 positive statements and 5 gaps, plus the totals the memo quotes. |
| `render.go` | Emits every table in this memo from one run. |
| `evidence.go` | The comment-blanking and bounded-match helpers. |

Nothing in `internal/` or `cmd/` imports it, it opens no database connection, and
the only files it touches are opened for reading.

## The measured surface

| HTTP method | Routes |
| --- | ---: |
| GET | 68 |
| POST | 46 |
| PUT | 17 |
| DELETE | 15 |

| Menu entry | list | show | edit | create |
| --- | --- | --- | --- | --- |
| isp/customers | yes | yes | yes | yes |
| isp/packages | yes | no | yes | yes |
| isp/subscriptions | yes | yes | no | yes |
| isp/invoices | yes | yes | no | no |
| isp/payments | yes | yes | no | yes |
| radius/users | yes | yes | yes | yes |
| radius/online | yes | yes | no | no |
| radius/accounting | yes | yes | no | no |
| radius/profiles | yes | yes | yes | yes |
| network/nas | yes | yes | yes | yes |
| network/nodes | yes | yes | yes | yes |
| system/operators | yes | yes | yes | yes |
| system/certificate | yes | yes | yes | yes |

Read routes are the only things a resolution may name, and they are less than half
of the registrations - which is the whole reason the second half of the criterion
matters more than the first.

## The vocabulary a prompt can be built from

Twelve filter inputs exist across the whole console, declared by six list pages.
The table that maps menu names to API resource names holds ten entries: four
rename, five repeat the menu name, and one is not a menu entry at all; four of the
thirteen menu entries never appear in it.

| Group | Query key | Widget | Offered values | Declared at |
| --- | --- | --- | --- | --- |
| customerFilters | q | SearchInput | none | web/src/resources/isp.tsx:181 |
| customerFilters | status | StatusSelectInput | pending, active, inactive, suspended, terminated | web/src/resources/isp.tsx:181 |
| packageFilters | status | StatusSelectInput | active, inactive | web/src/resources/isp.tsx:420 |
| subscriptionFilters | customer_id | ReferenceInput | none | web/src/resources/isp.tsx:574 |
| subscriptionFilters | status | StatusSelectInput | pending, active, suspended, terminated | web/src/resources/isp.tsx:574 |
| invoiceFilters | customer_id | ReferenceInput | none | web/src/resources/isp.tsx:691 |
| invoiceFilters | invoice_no | TextInput | none | web/src/resources/isp.tsx:691 |
| invoiceFilters | status | StatusSelectInput | draft, issued, partial, paid, overdue, void | web/src/resources/isp.tsx:691 |
| paymentFilters | customer_id | ReferenceInput | none | web/src/resources/isp.tsx:1268 |
| paymentFilters | invoice_id | ReferenceInput | none | web/src/resources/isp.tsx:1268 |
| settingsFilters | name | TextInput | none | web/src/resources/systemSettings.tsx:46 |
| settingsFilters | type | TextInput | none | web/src/resources/systemSettings.tsx:46 |

The name a menu entry shows and the resource name sent to the API are not always
the same string; the console owns that translation in one table (`dataProvider.ts:22`
→ `resolveResource`), and the resolver reads it rather than guessing:

| Menu entry | Resource name sent to the API | How |
| --- | --- | --- |
| isp/customers | isp/customers | declared as itself |
| isp/invoices | isp/invoices | declared as itself |
| isp/packages | isp/packages | declared as itself |
| isp/payments | isp/payments | declared as itself |
| isp/subscriptions | isp/subscriptions | declared as itself |
| network/nas | network/nas | not in the table |
| network/nodes | network/nodes | not in the table |
| radius/accounting | accounting | renamed |
| radius/online | sessions | renamed |
| radius/profiles | radius-profiles | renamed |
| radius/users | users | renamed |
| system/certificate | system/certificate | not in the table |
| system/operators | system/operators | not in the table |

Table entries that are not menu entries: `system/config/schemas`.

## The resolver

Four steps, in this order, and the order is the safety property:

1. **Deny.** If the prompt contains a mutation word the product itself uses, stop.
   That word list is measured, not typed out by hand, and it comes from two places:
   nine words are the trailing segment of a non-GET registration
   (`/…/:id/<verb>`), and five more - `suspend`, `reactivate`, `terminate`,
   `activate`, `disconnect` - are values a subscription lifecycle handler compares
   its action against. Two of those five also name a route, so the union is twelve.
2. **Score.** Count the cataloged words per screen.
3. **Absorb, or refuse.** A tie is resolved only when one side's words are the
   other side's filter keys - `tagihan pelanggan 42` means the invoices *of* a
   customer, not two screens. Anything else that ties is a coin flip and names
   nothing.
4. **Compose.** A filter key appears when the page declares it; a value appears
   only when the page offers it.

## The words the resolver refuses on

Action words are measured from two places: the trailing segment of a non-GET registration (`/…/:id/<verb>`) and the value a mutation handler compares its action against. The alias list is the only hand-written half.

| Measured action word | Routes it closes off |
| --- | --- |
| activate | POST /platform/tenants/:id/operators/:operator_id/activate |
| check | POST /network/monitor-targets/:id/check |
| coa | POST /sessions/:id/coa |
| disconnect | POST /sessions/:id/disconnect |
| dispatch | POST /isp/tickets/:id/dispatch |
| reactivate | value guard at internal/adminapi/isp.go:1188 |
| retry | POST /session-actions/:id/retry |
| reverse | POST /isp/payments/:id/reverse |
| revoke | POST /isp/vouchers/:id/revoke |
| suspend | value guard at internal/adminapi/isp.go:1188 |
| terminate | value guard at internal/adminapi/isp.go:1188 |
| toggle | POST /network/monitor-targets/:id/toggle |

Hand-written aliases: aktifkan, batalkan, bayar, blokir, buat, buatkan, ganti, hapus, kirim, matikan, nyalakan, putus, putuskan, reset, restart, tagih, tahan, tambah, tambahkan, ubah.

## The 20 fixtures

Navigation, filter, refusal - each with the reason the mapping is right, and the
parser's own words beside it.

| # | Prompt | Expected | Parsed | Match | Reason and refusal |
| --- | --- | --- | --- | --- | --- |
| 1 | daftar user radius | navigate | navigate GET /users | yes | the account list is the only screen the words name; the page has no filter array, so nothing is composed |
| 2 | profil radius untuk bandwidth 10 Mbps | navigate | navigate GET /radius-profiles | yes | profil and bandwidth both sit on the profile screen, which carries more mentions than the RADIUS account list |
| 3 | sesi yang sedang online | navigate | navigate GET /sessions | yes | the live session list is a screen, not a filter; the resource declares no filter array |
| 4 | daftar perangkat NAS | navigate | navigate GET /network/nas | yes | NAS is the device inventory; perangkat is not a cataloged word and cannot add a second target |
| 5 | pohon node dan ODP | navigate | navigate GET /network/nodes | yes | node and ODP are both words of the node tree, so one intent matches twice and no rival exists |
| 6 | operator yang punya akses admin | navigate | navigate GET /system/operators | yes | the operator register is the screen; the words describe the list, not a query key the page offers |
| 7 | sertifikat radsec yang hampir kedaluwarsa | navigate | navigate GET /system/certificate | yes | the certificate store is the only screen named; expiry is not a filter this page declares |
| 8 | list pembayaran bulan ini | navigate | navigate GET /isp/payments | yes | the ledger is the screen; the payments page offers customer and invoice references, neither named here |
| 9 | rekaman accounting kemarin | navigate | navigate GET /accounting | yes | accounting records are a screen; yesterday is a date the page cannot filter on |
| 10 | cari pelanggan bernama rani | filter | filter GET /isp/customers [q=] | yes | cari and bernama are the cues for the free-text key the customer page declares; the parser names the key and leaves the value to the operator |
| 11 | pelanggan yang statusnya suspended | filter | filter GET /isp/customers [status=suspended] | yes | suspended is one of the five values the customer page itself offers, so the filter can be filled, not just named |
| 12 | invoice status belum dilunasi | filter | filter GET /isp/invoices [status=] | yes | status names the key, but the page's vocabulary is English (paid, overdue, partial) and no offered value matches |
| 13 | invoice nomor INV-2026-0001 | filter | filter GET /isp/invoices [invoice_no=] | yes | nomor is the cue for the invoice number input; the value is text the operator must confirm before the list runs |
| 14 | tagihan pelanggan 42 | filter | filter GET /isp/invoices [customer_id=] | yes | pelanggan would also open the customer register, but it is a reference input of the invoice page, so it reads as a column here |
| 15 | daftar langganan yang pending | filter | filter GET /isp/subscriptions [status=pending] | yes | pending is offered by the subscription page's own status vocabulary |
| 16 | paket yang aktif | navigate | navigate GET /isp/packages | yes | the page offers active and inactive in English; aktif is the same word in Indonesian and the parser will not invent a translation, so it opens the screen and composes nothing |
| 17 | suspend pelanggan macet | refuse | refuse: "suspend" is an action the console accepts as a value at internal/adminapi/isp.go:1188 | yes | suspend is a mutation the console accepts only as an action value, not a route; the parser names where it found the word and aims at nothing |
| 18 | bayar invoice pelanggan 7 | refuse | refuse: "bayar" asks for a change of state, not a view | yes | bayar is a change of state, and a payment the console records through a POST; refusing is the only safe reading |
| 19 | pelanggan dan user | refuse | refuse: ambiguous between customers and radius-users; naming one of them would be a guess | yes | one mention each for two screens, and neither word belongs to the other page's filter vocabulary, so picking one would be a guess |
| 20 | tampilkan ringkasan performa jaringan | refuse | refuse: no word in the prompt matches a cataloged menu entry, so the parser names nothing | yes | no cataloged word matches: the dashboard is a custom route, not a Resource, so there is no menu entry to name |

### What each refusal said

- `suspend pelanggan macet`
  - "suspend" is an action the console accepts as a value at internal/adminapi/isp.go:1188
- `bayar invoice pelanggan 7`
  - "bayar" asks for a change of state, not a view
- `pelanggan dan user`
  - ambiguous between customers and radius-users; naming one of them would be a guess
- `tampilkan ringkasan performa jaringan`
  - no word in the prompt matches a cataloged menu entry, so the parser names nothing

## What this shows

- **The read half is real.** All 20 fixtures behave as expected, and the corpus
  reaches 13 of 13 cataloged screens. Six prompts compose a filter, two of them
  with a value taken from the page's own vocabulary.
- **A misread cannot write.** Zero targets in the whole corpus are not a `GET`
  route, and the three refusal classes - mutation word, tie, no match - all return
  an empty target list. There is no code path from a prompt to a body, a verb or a
  credential.
- **A composed filter needs no new endpoint.** The data provider already appends
  the filter object to the read URL (`dataProvider.ts:47`), and `ActiveFilters.tsx`
  already renders whatever is in the URL back to the operator. The missing piece is
  the front door, not the plumbing.
- **The verbs live in two places.** Reading only route segments would have missed
  `suspend`, which is the mutation operators ask for most and the console accepts
  only as an action value (`isp.go:1188`). Any deny list built from paths alone is
  incomplete by construction.
- **The language gap is the binding constraint on value filling.** The pages name
  their states in English (`active`, `paid`, `overdue`) and operators write
  `aktif`, `lunas`, `belum bayar`. Two of six filter prompts could be filled; the
  rest name a key and stop, because a translation table would be the parser
  inventing a value the product never declared (`paket yang aktif` is the fixture
  that shows it).
- **Some prompts have no target at all.** "tampilkan ringkasan performa jaringan"
  matches nothing, because the dashboard totals live at `GET /dashboard/stats` with
  no `<Resource>` entry - a screen the operator can see in the custom layout but an
  intent resolver cannot point at.
- **The customer list cannot answer a date.** It reads `q`, `status`, `page`,
  `perPage` and nothing time-shaped, so "pelanggan yang mendaftar bulan lalu" is
  unresolvable today regardless of the parser.

## What this study does not close

| No-go | Blocking asset |
| --- | --- |
| A shipped command palette in the console | `TR-F013` scope decision; this prototype is research and no product code imports it. |
| Measured resolution quality with real operators | Named respondents; nothing in this tree records what an operator actually types, so the 20 prompts are constructed, not observed. |
| A translation layer between Indonesian state words and the English vocabulary | A product decision about who owns that mapping - the pages, the i18n catalogs, or the API - and it is not this card's scope. |
| Any natural-language search beyond the cataloged screens | `TR-N001`/`TR-N002` guardrails plus a scope decision; the refusal classes exist precisely so the prototype does not quietly widen this. |
| Intent resolution that respects per-role visibility | A live seeded admin session per role; `ENG-052` proved the button matrix in the browser, but a headless run cannot show which screens a given operator is allowed to reach. |

## Decisions taken here

1. Build the parser as a research package, not a product surface; no route, no UI,
   no schema changed.
2. Take the deny vocabulary from the tree (route segments plus action-value
   guards) instead of a hand-listed word set, and disclose the one hand-written
   half - 20 Indonesian aliases - as hand-written.
3. Prefer an empty result over a guess: ties and unmatched prompts are refusals, so
   "salah tafsir tidak memicu side effect" is a property of the data model, not of
   the keyword table.
4. Report the language gap and the missing dashboard resource as findings instead
   of patching them with an alias table or a synthetic menu entry.
5. Keep the corpus at exactly the 20 prompts the criterion names, and make the
   coverage of the catalog a test.
6. Record the role-visibility question as a no-go rather than assume `requireAdmin`
   makes the answer safe.


## Every statement and gap, as the scan found them

### Statements the memo leans on

| ID | Statement | Evidence | Status |
| --- | --- | --- | --- |
| active-filter-render | the UI already renders the composed filter set back to the operator | web/src/components/ActiveFilters.tsx:31 | confirmed |
| certificate-list-page | the certificate page is a declared resource | web/src/resources/certificates.tsx:428 | confirmed |
| customer-list-handler | the customer register is a read handler the console already calls | internal/adminapi/isp.go:81 | confirmed |
| customer-list-page | the customer list page is the menu entry a navigation resolves to | web/src/resources/isp.tsx:182 | confirmed |
| dry-run-precedent | the product already has a no-write preview for a mutating path | internal/adminapi/system_backup.go:567 | confirmed |
| filter-becomes-query | the data provider appends the filter object to the read URL, so a composed filter needs no new endpoint | web/src/providers/dataProvider.ts:47 | confirmed |
| invoice-list-handler | invoices are listed through query parameters, not through a bespoke search | internal/adminapi/isp.go:96 | confirmed |
| lifecycle-action-words | not every mutation verb is a route segment; the subscription lifecycle takes them as a value | internal/adminapi/isp.go:1188 | confirmed |
| menu-name-map | a menu name becomes an API path through one table the console already owns | web/src/providers/dataProvider.ts:22 | confirmed |
| payment-list-handler | the payment ledger is reachable by a read route | internal/adminapi/isp.go:98 | confirmed |
| platform-gate | platform-only screens are gated separately, so an intent cannot flatten them | internal/adminapi/platform_tenants.go:47 | confirmed |
| query-param-read | the console reads its filters back as query parameters | internal/adminapi/isp.go:119 | confirmed |
| role-gate | an intent result still has to pass the role gate the admin API installs | internal/adminapi/authz.go:45 | confirmed |
| session-action-route | the live session page has a mutating action route, so a read-only resolver must name it and stay off it | internal/adminapi/sessions.go:269 | confirmed |
| session-list-page | the live session page is a declared resource, not a custom route | web/src/resources/onlineSessions.tsx:2028 | confirmed |
| status-vocabulary | the values a prompt may fill come from the page's own array | web/src/resources/isp.tsx:107 | confirmed |
| status-widget | the status filter is a widget with a declared value list | web/src/resources/isp.tsx:113 | confirmed |
| subscription-list-handler | subscriptions are a read surface with a status column | internal/adminapi/isp.go:92 | confirmed |

### Gaps the scan re-checks on every run

| ID | Gap | Scope | Status | Evidence |
| --- | --- | --- | --- | --- |
| dry-run-only-for-restore | the no-write preview exists for the restore path only, not for the ISP mutations an intent would refuse | internal/adminapi | open | none |
| no-command-palette | the console has no command palette or fuzzy launcher for an operator to type into | web/src | open | none |
| no-date-filter-on-customers | the customer list offers no date range, so a prompt about last month cannot be composed | internal/adminapi/isp.go | open | none |
| no-intent-parser | no product code parses a free-text operator intent | internal | open | none |
| no-search-endpoint | no admin route takes a natural-language query; free text only arrives as a search box value | internal/adminapi | open | none |

### Read but not real source

The scan walks `web/src`, so these copies are read as text and reported:

- `web/src/resources/radiusUsers.tsx.backup`

## Verification

```
go build ./...                                    # clean
go vet ./internal/intentsearch/...                # clean
go test -p 1 -count=1 ./internal/intentsearch/    # 21 pass, 1 skipped (the dump)
gofmt -l internal/intentsearch/                   # empty
golangci-lint run ./internal/intentsearch/...     # 0 issues
grep -rn intentsearch --include=*.go internal/ cmd/ | grep -v ^internal/intentsearch/   # 0 importers
```

The tables above come from `INTENTSEARCH_DUMP=1 go test -run TestReportMarkdownDump
-v ./internal/intentsearch/`, so every number in the prose is the same run.

## Residual limits

- The keyword table is hand-written Indonesian and English aliases over measured
  targets; the tests prove the targets exist, not that the aliases are the ones
  operators use.
- Scoring is a word-count over a normalized prompt. It is deterministic and
  explainable, and it is not a language model; anything requiring paraphrase
  resolution is out of reach by design.
- The scan reads `web/src` as text, so `web/src/resources/radiusUsers.tsx.backup`
  is parsed as if it were a page; the report names it rather than quietly skipping
  it.
- Route counting uses the registration line, so a route registered through a helper
  this scan does not know about would be invisible. The four method helpers
  (`ApiGET`, `ApiPOST`, `ApiPUT`, `ApiDELETE`) are the only ones the admin API uses.
- 12 filter inputs and 13 menu entries are today's totals; a new page can change
  every number in this memo. The tests fail loudly when the tree drifts.
