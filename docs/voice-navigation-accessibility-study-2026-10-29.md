# FUT-016 — What a navigation-only voice layer can and cannot be asked to do

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `FUT-016` (`TR-F013`, P3, RISET, dependency `ENG-053`) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/voicenav` (read-only, imported by nothing).
Criterion under test: **Voice optional, salah transkripsi tidak membuat mutasi,
dan teks fallback ada.**

## The question

The card asks for *voice-assisted navigation for internal accessibility*, with
two hard edges on it: no permanent listening backend, and no automation of
payment. The criterion then makes three claims that have to be falsifiable:
voice is optional, a mis-transcription does not cause a mutation, and a typed
fallback exists. None of the three is answerable by demoing a microphone, and
one of them is not answerable by a browser at all - so this round prices them as
properties of the code instead:

1. **"Salah transkripsi tidak membuat mutasi."** Two independent guarantees, and
   the study measures both. The first is structural: the resolver's answer type
   carries a menu path and nothing else - no method, no record id, no payload -
   so a mis-transcription *cannot* be a mutation however bad the transcript gets
   (`TestTheAnswerTypeCarriesNoOperation`, which reflects over the type rather
   than trusting the prose). The second is behavioural: an operation word, derived
   from the server's own action guard and write-route tails, refuses the whole
   utterance before anything is scored (7 of 7 operation transcripts refused).
2. **"Teks fallback ada."** Measured per destination, not asserted: every speakable
   path is traced through `App.tsx`'s import graph to the file that renders it, and
   the typed controls on that file are counted. The answer turns out to be a
   split one - 140 text and search inputs across 23 screens, but 12 screens with
   none - which is why the fallback is a *navigation*-level one (the menu, a URL,
   a click) rather than a typed one everywhere.
3. **"Voice optional."** Not a design promise but a current fact the study can
   count: 0 references to a browser speech API anywhere in `web/src` or `web/e2e`,
   and 0 microphone or audio entry points anywhere in `internal`. A voice layer has
   nothing to attach to, so attaching one is necessarily opt-in.

What is *not* closable from the tree is registered rather than performed: real
recognizer error distributions, operator acceptance, latency and permission
prompts, and the device question the card's "no permanent listener" clause
implies. `G01`-`G06` name each and what asset it would need.

The deliverable is therefore a *measurement with a no-go register*: the vocabulary
a voice command is allowed to have, the reach of an answer, the refusal set with
its pruning shown token by token, a 58-transcript corpus scored per error class,
and the typed fallback destination by destination.

## What counts as evidence

The rules the `ENG-006`, `ENG-007`, `ECO-006`, `FUT-001`, `FUT-002`, `FUT-007`
and `FUT-015` studies used, each enforced by a test in this package:

- **A citation is a code line.** Comments are blanked rune-for-rune before a
  symbol is matched, so line numbers survive and prose cannot confirm a claim
  (`TestTheCommentBlankerKeepsLineNumbers`; the `.yml` dialect uses `#`, everything
  else `//` and `/* */`, which is why `isp.tsx`'s CSS keeps its `#printable-invoice`
  selector).
- **A whole identifier.** `containsBounded` refuses an adjacent word character, so
  a scan cannot be satisfied by a substring inside a longer name.
- **Drift is reported, never dropped.** A missing file is `vacated`, a citation
  that no longer lands is `broken`, and `TestEveryStatementHolds` fails the build
  on either.
- **Every number in this memo is spliced from the run.** Nothing in the prose is
  typed by hand; the tables below are the verbatim output of
  `VOICE_DUMP=1 go test -p 1 -count=1 -v -run TestReportMarkdownDump ./internal/voicenav/`,
  and `render_test.go` refuses an empty table, a `%!` verb, a statement with no
  line, or a limit that closed itself.
- **The absence scans skip the prototype.** `ownPackage = "voicenav"` is excluded
  from every walk, so the study cannot refute - or confirm - itself.
- **The digest is deterministic.** Two runs over the same tree produce the same
  sha256 (`TestTheDigestSurvivesATickingClock`). This round earned that test: the
  refusal list originally attributed each token to whichever source a map
  iteration happened to visit first, so the printed "derived from" column moved
  between runs.

Three rules specific to this study:

- **The vocabulary is not authored.** A speakable phrase is a label the shipped
  menu has, rendered by one of the two shipped locale files - 23 destinations,
  46 phrases, 0 destinations speakable in neither locale. Nothing in the corpus can
  name a screen the console does not print.
- **The mutation vocabulary is not authored either.** 17 action words come from
  the server's own guard line and from write-route trailing segments, and the
  refusal set adds the label words those actions render. The pruning is then
  *measured* rather than assumed: 32 derived tokens, 4 of which are also screen
  words (`management`, `manajemen`, `operator`, `operators`) and have to go, leaving
  28 the resolver refuses on.
- **The scorer was broken by the corpus, and the fix is reported.** A single word
  ("jaringan") scored well enough to be answered as a screen. A label the
  transcript does not cover word for word is now not a candidate at all, which is
  what makes the three truncation rows refuse instead of guess
  (`TestNoFragmentBecomesAConfidentScreen`). That rule came out of running the
  corpus, not in front of it.

## The prototype

| File | Role |
| --- | --- |
| `voicenav.go` | `Run`, the `Result` the memo splices, the 34-counter totals table and the sha256 digest. |
| `evidence.go` | Corpus constants, comment blanking per file dialect, bounded matching, line walking, importer and import-set helpers. |
| `surface.go` | The menu rows parsed from `CustomMenu.tsx`, the two locale label blocks, the admin API's read/write registrations, the derived action words and the pruning rules. |
| `resolve.go` | Normalization, the edit-distance similarity, per-label scoring, the fragment eligibility rule, the refusal order and the three thresholds. |
| `corpus.go` | The 58 authored transcripts in 14 error classes, each with the answer the study requires. |
| `fallback.go` | Destination-to-page tracing through the import graph, typed-field and filter counting, the speech-API and microphone absence scans. |
| `claims.go` | 15 statements with their checks, the 6-item gap register, the evidence gatherer and the mechanism pins. |
| `render.go` | Emits every table in this memo from one run. |

Nothing in `internal/` or `cmd/` imports it (`TestNothingImportsTheStudy`, and a
grep over the tree returns 0), it opens no database connection, and the only
files it touches are opened for reading
(`TestRunningTheStudyWritesNothing` snapshots `web/src`, `internal/adminapi` and
`.github/workflows` before and after the run). Its 18 imports are stdlib plus the
test framework (`TestTheStudyImportsNothingThatCanSendOrWrite`, which now also
fails if the parse finds nothing rather than passing vacuously).

## What the run says

**The answer cannot be an operation, and that is a property of the type.** 78
write registrations sit behind the console's UI, 68 reads beside them. A verdict
carries one of 23 menu paths or nothing at all: `TestTheAnswerTypeCarriesNoOperation`
reflects over `Verdict` and fails if a field that could name an
operation ever appears. Over the 58-transcript corpus, 0 confident answers
reached an operation, and the 7 transcripts written as operations were refused
before scoring (statement `C07`).

**Refusal is derived, and the pruning is the interesting half.** The naive list
is 32 tokens; 0 were dropped for being under three letters and 4 were dropped
because a *screen label* uses them. `operators` is the case that matters: it is
the trailing segment of `ApiPOST("/platform/tenants/:id/operators")` - a write
route - and simultaneously the word `Manajemen Operator`, a screen the menu
prints. A deny-first rule that refuses on label words would make that screen
unspeakable, so the vocabulary wins over the route tail and `manajemen operator`
navigates. `C06` reports the counts; the per-token table shows exactly what was
kept and what was dropped.

**A damaged transcript usually still names the right screen.** 7 of the 8
transcripts with recognizer damage land where they meant, across 5 error classes:
a transposed vowel (`manajeman profil`, 94%), a dropped vowel (`pembayran`, 90%),
a truncated tail (`konfigurasi sist`, 83%), a word split into three tokens
(`lang gan an`, 100% - the joins recover it) and a filler inserted in front
(`anu voucher hotspot`, 100%). The one that does not is `peladangan`, a plausible
Indonesian near-word for `pelanggan`: it scores 70%, under the 78% accept floor,
and is refused. That is the floor pricing a 2-edit homophone as too far, and this
round reports the mismatch (57 of 58 transcripts match what the study requires)
rather than tuning the constant until the corpus agrees.

**A fragment is not a command, and that came from the corpus.** `paket`,
`manajemen` and `jaringan` each score against *something* - 50%, 33%, 50% - and
each names more than one screen, because the console has multi-word labels that
share those words. The first pass of the resolver answered `jaringan` with
`/operations`; the eligibility rule - a candidate must have no more words than the
transcript has tokens - is what refuses it now. The refuse rows print how many
labels were dropped as fragments (16-17 of 22-23 for a one-word transcript), so
the rule's effect is visible in the table instead of asserted in prose.

**Two real labels in one utterance ask instead of guessing.** `buka tagihan dan
pembayaran` and `pelanggan dan paket internet` both score 100% on two screens with
0% margin, and the resolver asks. `TestRefusalOnlyRemovesAnswers` pins the
direction: a refusal can only ever remove a screen from the answer set, never
promote one.

**The typed fallback is real on 11 screens and absent on 12.** All 23
destinations trace to a page file. `radiusUsers.tsx` carries 26 text or search
inputs, `nas.tsx` 15, `isp.tsx` 12 shared across 5 ISP screens - those are the
screens where typing does something. The other 12 (`Dashboard.tsx`,
`TroubleTicketsPage.tsx`, `IPAMPage.tsx`, `ODPPage.tsx`, `OperationsPage.tsx`,
`PlatformTenantsPage.tsx`, `BrandingPage.tsx`, `SystemConfigPage.tsx`,
`WebsiteEditorPage.tsx`, `HotspotVouchersPage.tsx`) declare none: the fallback
there is the menu, a URL or a click, not a text field. Two screens
(`/radius/accounting`, `/radius/online`) bind `setFilters` while declaring no
input of their own - they take react-admin's built-in list filter - so "filter
bound" and "reachable by typing" are reported as separate columns rather than
collapsed into one (statements `C12`, `C13`).

**Voice has nothing to attach to, which is what makes it opt-in.** 0 speech-API
references across `web/src` and `web/e2e`; 0 microphone, audio-stream or
recognizer entry points across `internal`. The card's "no permanent listening
backend" is therefore satisfied today by absence, and a prototype that adds a
recognizer adds the only listener there is (`C11`, `TestVoiceHasNothingToAttachTo`).

**Nine gated screens stay speakable, and that is a decision this study does not
make.** The vocabulary is role-blind exactly as the URL bar is: `/system/config`
is gated to a `platform_admin` session in the menu and still says "Konfigurasi
Sistem". A voice layer inherits the menu's visibility, not the server's authority;
whether a routed-but-forbidden screen then answers is a server question
(`G04`).

**Four Indonesian operation sentences are refused by nothing but the answer
type.** `hapus pelanggan`, `kirim whatsapp ke pelanggan`, `bayar tagihan pelanggan
itu` and `tangguhkan langganan billing yang menunggak` are instructions whose
verbs no route segment spells in Indonesian. Two are refused or asked because a
brand or route word carried over; two navigate to a *read* screen - which is safe
(screens only) and useless (the operator meant a mutation). Whether the voice
vocabulary should carry a curated synonym set for those is `TR-F013` scope
(`G05`), and this round records it as open rather than extending the deny list
until the corpus looks clean.

## The measured tables

Spliced verbatim from one run of `internal/voicenav`.

## The measured numbers

| number | what it counts | value |
| --- | --- | --- |
| destinations | menu rows a spoken command may name | 23 |
| phrases | labels the two locales render for them | 46 |
| en-phrases | labels from the English locale | 23 |
| id-phrases | labels from the Indonesian locale | 23 |
| unlabelled | destinations speakable in neither locale | 0 |
| gated | destinations the menu hides from a lower role | 9 |
| reads | GET registrations in the admin API | 68 |
| writes | POST, PUT and DELETE registrations in the admin API | 78 |
| action-words | operation words the server derives | 17 |
| ban-raw | tokens the naive refusal list holds | 32 |
| ban-short | tokens dropped for being under three letters | 0 |
| ban-screen | tokens dropped because a screen label uses them | 4 |
| ban-kept | tokens the resolver refuses on | 28 |
| fixtures | transcripts scored | 58 |
| navigated | transcripts answered with a screen | 39 |
| asked | transcripts answered with a question | 3 |
| refused | transcripts answered with nothing | 16 |
| matched | transcripts whose answer is what the study requires | 57 |
| mismatched | transcripts whose answer is not what the study requires | 1 |
| wrong-screen | confident answers naming a screen the transcript did not ask for | 0 |
| mutation-reached | confident answers that performed an operation | 0 |
| fallback-pages | destinations traced to a page file | 23 |
| typed-fields | TextInput and SearchInput elements across those pages | 140 |
| filter-bound | pages that bind a filter to typed input | 8 |
| no-typed-field | screens reachable only by a whole-label utterance or a click | 12 |
| speech-refs | speech-API references in the frontend | 0 |
| listener-refs | microphone-listener references in the backend | 0 |
| claims | statements pinned to a code line | 15 |
| confirmed | statements whose citation still lands | 15 |
| broken | statements that no longer hold | 0 |
| vacated | statements whose file is gone | 0 |
| limits | open questions this round cannot close | 6 |
| open | limits still open after this run | 6 |
| imports | packages this study builds on | 18 |

## The vocabulary: the menu rows and the labels the two locales render

| menu destination | label key | gate | English label | Indonesian label | phrases |
| --- | --- | --- | --- | --- | --- |
| / | dashboard | (none) | Dashboard | Dasbor | 2 |
| /isp/customers | customers | (none) | Customers | Pelanggan | 2 |
| /isp/invoices | invoices | (none) | Invoices | Tagihan | 2 |
| /isp/packages | packages | (none) | Internet Packages | Paket Internet | 2 |
| /isp/payments | payments | (none) | Payments | Pembayaran | 2 |
| /isp/subscriptions | subscriptions | (none) | Subscriptions | Langganan | 2 |
| /isp/tickets | tickets | (none) | Trouble Tickets | Tiket Gangguan | 2 |
| /isp/vouchers | vouchers | (none) | Hotspot Vouchers | Voucher Hotspot | 2 |
| /network/ipam | ipam | super,admin | IPAM & Subnets | IPAM & Subnet | 2 |
| /network/nas | nas_devices | (none) | NAS Devices | Perangkat NAS | 2 |
| /network/nodes | network_nodes | (none) | Network Nodes | Node Jaringan | 2 |
| /network/odp | odp_management | super,admin | FTTH ODP Assets | Aset ODP FTTH | 2 |
| /operations | operations | super,admin | Network & Alerts | Jaringan & Peringatan | 2 |
| /platform/tenants | tenants | platform_admin | ISP Tenants | Tenant ISP | 2 |
| /radius/accounting | accounting | (none) | Accounting Logs | Log Akuntansi | 2 |
| /radius/online | online_sessions | (none) | Online Sessions | Sesi Online | 2 |
| /radius/profiles | radius_profiles | (none) | Profiles Management | Manajemen Profil | 2 |
| /radius/users | radius_users | (none) | Users Management | Manajemen Pengguna | 2 |
| /system/branding | branding | platform_admin | Product Branding | Branding Produk | 2 |
| /system/certificate | certificates | platform_admin | Certificates | Sertifikat | 2 |
| /system/config | system_config | platform_admin | System Configuration | Konfigurasi Sistem | 2 |
| /system/operators | operators | super,admin | Operators Management | Manajemen Operator | 2 |
| /system/website | website_editor | platform_admin | Website & Home Editor | Editor Website & Beranda | 2 |

## What the console performs as an operation, and what the refusal list keeps

| action word | where it comes from | code line | English label the console renders | Indonesian label the console renders |
| --- | --- | --- | --- | --- |
| activate | the server's action guard | internal/adminapi/isp.go:1188 | (none) | (none) |
| apply-fup | an action route | internal/adminapi/isp.go:104 | (none) | (none) |
| check | an action route | internal/adminapi/network_monitor.go:101 | (none) | (none) |
| coa | an action route | internal/adminapi/sessions.go:270 | Change Authorization | Ubah Otorisasi |
| disconnect | the server's action guard | internal/adminapi/isp.go:1188 | Force Disconnect | Putuskan Paksa |
| dispatch | an action route | internal/adminapi/isp_operations.go:54 | (none) | (none) |
| operators | an action route | internal/adminapi/platform_tenants.go:52 | Operators Management | Manajemen Operator |
| reactivate | the server's action guard | internal/adminapi/isp.go:1188 | (none) | (none) |
| reset-fup | an action route | internal/adminapi/isp.go:105 | (none) | (none) |
| retry | an action route | internal/adminapi/sessions.go:274 | Retry | Coba Lagi |
| reverse | an action route | internal/adminapi/isp.go:101 | (none) | (none) |
| revoke | an action route | internal/adminapi/isp_operations.go:32 | (none) | (none) |
| send-whatsapp | an action route | internal/adminapi/isp.go:102 | (none) | (none) |
| simulate-pay | an action route | internal/adminapi/isp.go:108 | (none) | (none) |
| suspend | the server's action guard | internal/adminapi/isp.go:1188 | (none) | (none) |
| terminate | the server's action guard | internal/adminapi/isp.go:1188 | (none) | (none) |
| toggle | an action route | internal/adminapi/network_monitor.go:100 | (none) | (none) |

## The derived refusal set, and exactly what the pruning rules removed

| refusal token | derived from | kept or dropped |
| --- | --- | --- |
| activate | activate (route or guard word) | kept |
| apply | apply-fup (route or guard word) | kept |
| authorization | coa (English label) | kept |
| change | coa (English label) | kept |
| check | check (route or guard word) | kept |
| coa | coa (route or guard word) | kept |
| coba | retry (Indonesian label) | kept |
| disconnect | disconnect (route or guard word) | kept |
| dispatch | dispatch (route or guard word) | kept |
| force | disconnect (English label) | kept |
| fup | apply-fup (route or guard word) | kept |
| lagi | retry (Indonesian label) | kept |
| management | operators (English label) | dropped: a screen label uses it |
| manajemen | operators (Indonesian label) | dropped: a screen label uses it |
| operator | operators (Indonesian label) | dropped: a screen label uses it |
| operators | operators (route or guard word) | dropped: a screen label uses it |
| otorisasi | coa (Indonesian label) | kept |
| paksa | disconnect (Indonesian label) | kept |
| pay | simulate-pay (route or guard word) | kept |
| putuskan | disconnect (Indonesian label) | kept |
| reactivate | reactivate (route or guard word) | kept |
| reset | reset-fup (route or guard word) | kept |
| retry | retry (route or guard word) | kept |
| reverse | reverse (route or guard word) | kept |
| revoke | revoke (route or guard word) | kept |
| send | send-whatsapp (route or guard word) | kept |
| simulate | simulate-pay (route or guard word) | kept |
| suspend | suspend (route or guard word) | kept |
| terminate | terminate (route or guard word) | kept |
| toggle | toggle (route or guard word) | kept |
| ubah | coa (Indonesian label) | kept |
| whatsapp | send-whatsapp (route or guard word) | kept |

## The transcript corpus, scored

| transcript | error class | why this transcript is in the corpus | answer | kind | why it stopped there | score | margin | the study requires | match |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| pelanggan | canonical | the Indonesian label for the customer list, spoken alone | /isp/customers | navigate | id-ID label, 100% | 100% | 44% | navigate:/isp/customers | yes |
| paket internet | canonical | two-word Indonesian label, in order | /isp/packages | navigate | id-ID label, 100% | 100% | 50% | navigate:/isp/packages | yes |
| langganan | canonical | the subscriptions label | /isp/subscriptions | navigate | id-ID label, 100% | 100% | 44% | navigate:/isp/subscriptions | yes |
| voucher hotspot | canonical | the Indonesian word order, not the English one | /isp/vouchers | navigate | id-ID label, 100% | 100% | 63% | navigate:/isp/vouchers | yes |
| tiket gangguan | canonical | trouble tickets in Indonesian | /isp/tickets | navigate | id-ID label, 100% | 100% | 33% | navigate:/isp/tickets | yes |
| tagihan | canonical | the invoices label in Indonesian, which is not the English word | /isp/invoices | navigate | id-ID label, 100% | 100% | 56% | navigate:/isp/invoices | yes |
| pembayaran | canonical | the payments label | /isp/payments | navigate | id-ID label, 100% | 100% | 50% | navigate:/isp/payments | yes |
| manajemen profil | canonical | the RADIUS profiles label | /radius/profiles | navigate | id-ID label, 100% | 100% | 38% | navigate:/radius/profiles | yes |
| manajemen pengguna | canonical | the RADIUS users label, one word from the profiles label | /radius/users | navigate | id-ID label, 100% | 100% | 44% | navigate:/radius/users | yes |
| sesi online | canonical | the online sessions label | /radius/online | navigate | id-ID label, 100% | 100% | 63% | navigate:/radius/online | yes |
| log akuntansi | canonical | the accounting logs label | /radius/accounting | navigate | id-ID label, 100% | 100% | 67% | navigate:/radius/accounting | yes |
| perangkat nas | canonical | the NAS devices label, with an acronym at the end | /network/nas | navigate | id-ID label, 100% | 100% | 33% | navigate:/network/nas | yes |
| aset odp ftth | canonical | three acronyms in one Indonesian label | /network/odp | navigate | id-ID label, 100% | 100% | 57% | navigate:/network/odp | yes |
| node jaringan | canonical | the network nodes label | /network/nodes | navigate | id-ID label, 100% | 100% | 20% | navigate:/network/nodes | yes |
| konfigurasi sistem | canonical | the system configuration label | /system/config | navigate | id-ID label, 100% | 100% | 65% | navigate:/system/config | yes |
| manajemen operator | canonical | the operators label - a word that is also the tail of a write route | /system/operators | navigate | id-ID label, 100% | 100% | 38% | navigate:/system/operators | yes |
| sertifikat | canonical | the certificates label | /system/certificate | navigate | id-ID label, 100% | 100% | 70% | navigate:/system/certificate | yes |
| branding produk | canonical | the branding label, with the Indonesian word order | /system/branding | navigate | id-ID label, 100% | 100% | 68% | navigate:/system/branding | yes |
| tenant isp | canonical | the platform tenants label | /platform/tenants | navigate | id-ID label, 100% | 100% | 61% | navigate:/platform/tenants | yes |
| dasbor | canonical | the dashboard label, which is also the route the menu numbers 1 | / | navigate | id-ID label, 100% | 100% | 67% | navigate:/ | yes |
| invoices | canonical-en | the English invoices label, exact | /isp/invoices | navigate | en-US label, 100% | 100% | 67% | navigate:/isp/invoices | yes |
| hotspot vouchers | code-switch | an English plural over an Indonesian-rendered screen | /isp/vouchers | navigate | en-US label, 100% | 100% | 60% | navigate:/isp/vouchers | yes |
| online sessions | code-switch | the English word order, which is the opposite of the Indonesian label | /radius/online | navigate | en-US label, 100% | 100% | 54% | navigate:/radius/online | yes |
| network nodes | canonical-en | the English nodes label | /network/nodes | navigate | en-US label, 100% | 100% | 42% | navigate:/network/nodes | yes |
| system configuration | canonical-en | the English configuration label, two words | /system/config | navigate | en-US label, 100% | 100% | 67% | navigate:/system/config | yes |
| buka halaman manajemen pengguna | lead-in | the most common Indonesian phrasing, with a verb and a noun in front | /radius/users | navigate | id-ID label, 100% | 100% | 44% | navigate:/radius/users | yes |
| tolong tampilkan dasbor | lead-in | a polite prefix that carries no destination | / | navigate | id-ID label, 100% | 100% | 44% | navigate:/ | yes |
| go to internet packages | lead-in | an English navigation prefix before an English label | /isp/packages | navigate | en-US label, 100% | 100% | 63% | navigate:/isp/packages | yes |
| ke halaman tiket gangguan | lead-in | two Indonesian lead-in words in front of the label; the score only asks for the label | /isp/tickets | navigate | id-ID label, 100% | 100% | 33% | navigate:/isp/tickets | yes |
| peladangan | homophone | a plausible Indonesian near-word for pelanggan | (refused: too far from any label - Pelanggan at 70%) | refused | a label was nearly recognized, but not closely enough to act - best candidate /isp/customers at 70%; 16 of the 22 labels that resemble it were dropped as fragments, since the transcript has only 1 word(s) | 70% | 20% | navigate:/isp/customers | no |
| manajeman profil | homophone | a transposed vowel in a label word | /radius/profiles | navigate | id-ID label, 94% | 94% | 38% | navigate:/radius/profiles | yes |
| pembayran | deletion | a dropped vowel, the most common ASR slip in Indonesian | /isp/payments | navigate | id-ID label, 90% | 90% | 46% | navigate:/isp/payments | yes |
| konfigurasi sist | truncation | the recognizer cut the last two letters of the second word | /system/config | navigate | id-ID label, 83% | 83% | 45% | navigate:/system/config | yes |
| lang gan an | boundary | one word, three tokens: what a Spanish-trained model does to langganan | /isp/subscriptions | navigate | id-ID label, 100% | 100% | 22% | navigate:/isp/subscriptions | yes |
| sesi on line | boundary | online rendered as two words | /radius/online | navigate | id-ID label, 100% | 100% | 62% | navigate:/radius/online | yes |
| anu voucher hotspot | insertion | a filler word inserted before the label | /isp/vouchers | navigate | id-ID label, 100% | 100% | 50% | navigate:/isp/vouchers | yes |
| manajemen profil radius | insertion | an extra true word that belongs to no label | /radius/profiles | navigate | id-ID label, 100% | 100% | 38% | navigate:/radius/profiles | yes |
| dashboard | code-switch | the English label, spelled with the American 'd' the id locale does not use | / | navigate | en-US label, 100% | 100% | 78% | navigate:/ | yes |
| paket | truncation | one word of a two-word label: a fragment, not a command | (refused: too far from any label - Payments at 50%) | refused | a label was nearly recognized, but not closely enough to act - best candidate /isp/payments at 50%; 16 of the 22 labels that resemble it were dropped as fragments, since the transcript has only 1 word(s) | 50% | 28% | ask-or-refuse | yes |
| manajemen | truncation | the word three different labels share | (refused: too far from any label - Langganan at 33%) | refused | a label was nearly recognized, but not closely enough to act - best candidate /isp/subscriptions at 33%; 17 of the 23 labels that resemble it were dropped as fragments, since the transcript has only 1 word(s) | 33% | 11% | ask-or-refuse | yes |
| jaringan | truncation | the word two different labels share | (refused: too far from any label - Tagihan at 50%) | refused | a label was nearly recognized, but not closely enough to act - best candidate /isp/invoices at 50%; 16 of the 22 labels that resemble it were dropped as fragments, since the transcript has only 1 word(s) | 50% | 6% | ask-or-refuse | yes |
| buka tagihan dan pembayaran | two-targets | a coordinator that joins two real labels at equal distance | (asks: Tagihan or Pembayaran?) | ambiguous | two screens are equally close, so the console asks - /isp/invoices 100% vs /isp/payments 100% | 100% | 0% | ambiguous | yes |
| pelanggan dan paket internet | two-targets | two Indonesian labels in one utterance | (asks: Pelanggan or Paket Internet?) | ambiguous | two screens are equally close, so the console asks - /isp/customers 100% vs /isp/packages 100% | 100% | 0% | ambiguous | yes |
|  | silence | an endpoint that heard nothing | (nothing: the console asks again) | refused | silence or noise: nothing to compare - no word recognized | 0% | 0% | refused | yes |
|     | silence | noise that normalized to nothing | (nothing: the console asks again) | refused | silence or noise: nothing to compare - no word recognized | 0% | 0% | refused | yes |
| hm ah uh | noise | a filler-only transcript | (refused: too far from any label - Tagihan at 29%) | refused | a label was nearly recognized, but not closely enough to act - best candidate /isp/invoices at 29% | 29% | 6% | refused | yes |
| saya mau lihat sesuatu yang biasa saya buka | noise | a fluent sentence that names no screen at all | (refused: too far from any label - Tagihan at 50%) | refused | a label was nearly recognized, but not closely enough to act - best candidate /isp/invoices at 50% | 50% | 6% | refused | yes |
| putuskan sesi pelanggan | action | the Indonesian label the server's own disconnect action renders | (refused: navigation only - disconnect (Indonesian label) is performed by a write route) | refused | names an operation the server performs with a write route - disconnect (Indonesian label), heard as putuskan | 0% | 0% | refused | yes |
| suspend pelanggan ini | action | the action word the API guard accepts, spelled exactly | (refused: navigation only - suspend (route or guard word) is performed by a write route) | refused | names an operation the server performs with a write route - suspend (route or guard word), heard as suspend | 0% | 0% | refused | yes |
| reactivate langganan | action | another guard-accepted action word | (refused: navigation only - reactivate (route or guard word) is performed by a write route) | refused | names an operation the server performs with a write route - reactivate (route or guard word), heard as reactivate | 0% | 0% | refused | yes |
| terminate subscription | action | the guard's fifth verb, in English | (refused: navigation only - terminate (route or guard word) is performed by a write route) | refused | names an operation the server performs with a write route - terminate (route or guard word), heard as terminate | 0% | 0% | refused | yes |
| reset branding | action | a write route's trailing segment plus a screen noun: the action wins | (refused: navigation only - reset-fup (route or guard word) is performed by a write route) | refused | names an operation the server performs with a write route - reset-fup (route or guard word), heard as reset | 0% | 0% | refused | yes |
| reverse pembayaran | action | the payment-reversal route's segment, mixed with the Indonesian payments label | (refused: navigation only - reverse (route or guard word) is performed by a write route) | refused | names an operation the server performs with a write route - reverse (route or guard word), heard as reverse | 0% | 0% | refused | yes |
| disconnect | action | the action word alone | (refused: navigation only - disconnect (route or guard word) is performed by a write route) | refused | names an operation the server performs with a write route - disconnect (route or guard word), heard as disconnect | 0% | 0% | refused | yes |
| bayar tagihan pelanggan itu | action-uncovered | a payment instruction in words no route segment uses | (asks: Pelanggan or Tagihan?) | ambiguous | two screens are equally close, so the console asks - /isp/customers 100% vs /isp/invoices 100% | 100% | 0% | any-safe | yes |
| hapus pelanggan | action-uncovered | a deletion instruction; the English verb is a route label but the Indonesian one is not | /isp/customers | navigate | id-ID label, 100% | 100% | 44% | any-safe | yes |
| kirim whatsapp ke pelanggan | action-uncovered | the send-whatsapp route described in Indonesian, with only the brand word carried over | (refused: navigation only - send-whatsapp (route or guard word) is performed by a write route) | refused | names an operation the server performs with a write route - send-whatsapp (route or guard word), heard as whatsapp | 0% | 0% | any-safe | yes |
| tangguhkan langganan billing yang menunggak | action-uncovered | an operation sentence whose verb is Indonesian and whose only screen noun is a whole label: the resolver lands on the read screen | /isp/subscriptions | navigate | id-ID label, 100% | 100% | 40% | any-safe | yes |

## Per error class

| error class | transcripts | answered with a screen | asked | refused | matched the requirement |
| --- | --- | --- | --- | --- | --- |
| action | 7 | 0 | 0 | 7 | 7 of 7 |
| action-uncovered | 4 | 2 | 1 | 1 | 4 of 4 |
| boundary | 2 | 2 | 0 | 0 | 2 of 2 |
| canonical | 20 | 20 | 0 | 0 | 20 of 20 |
| canonical-en | 3 | 3 | 0 | 0 | 3 of 3 |
| code-switch | 3 | 3 | 0 | 0 | 3 of 3 |
| deletion | 1 | 1 | 0 | 0 | 1 of 1 |
| homophone | 2 | 1 | 0 | 1 | 1 of 2 |
| insertion | 2 | 2 | 0 | 0 | 2 of 2 |
| lead-in | 4 | 4 | 0 | 0 | 4 of 4 |
| noise | 2 | 0 | 0 | 2 | 2 of 2 |
| silence | 2 | 0 | 0 | 2 | 2 of 2 |
| truncation | 4 | 1 | 0 | 3 | 4 of 4 |
| two-targets | 2 | 0 | 2 | 0 | 2 of 2 |

## The typed fallback, destination by destination

| menu destination | page file it traces to | typed fields on it | filter bound to input | reachable by typing |
| --- | --- | --- | --- | --- |
| / | web/src/pages/Dashboard.tsx | 0 | no | no |
| /isp/customers | web/src/resources/isp.tsx | 12 | no | yes |
| /isp/invoices | web/src/resources/isp.tsx | 12 | no | yes |
| /isp/packages | web/src/resources/isp.tsx | 12 | no | yes |
| /isp/payments | web/src/resources/isp.tsx | 12 | no | yes |
| /isp/subscriptions | web/src/resources/isp.tsx | 12 | no | yes |
| /isp/tickets | web/src/pages/TroubleTicketsPage.tsx | 0 | no | no |
| /isp/vouchers | web/src/pages/HotspotVouchersPage.tsx | 0 | no | no |
| /network/ipam | web/src/pages/IPAMPage.tsx | 0 | no | no |
| /network/nas | web/src/resources/nas.tsx | 15 | yes | yes |
| /network/nodes | web/src/resources/nodes.tsx | 7 | yes | yes |
| /network/odp | web/src/pages/ODPPage.tsx | 0 | no | no |
| /operations | web/src/pages/OperationsPage.tsx | 0 | no | no |
| /platform/tenants | web/src/pages/PlatformTenantsPage.tsx | 0 | no | no |
| /radius/accounting | web/src/resources/accounting.tsx | 0 | yes | no |
| /radius/online | web/src/resources/onlineSessions.tsx | 0 | yes | no |
| /radius/profiles | web/src/resources/radiusProfiles.tsx | 13 | yes | yes |
| /radius/users | web/src/resources/radiusUsers.tsx | 26 | yes | yes |
| /system/branding | web/src/pages/BrandingPage.tsx | 0 | no | no |
| /system/certificate | web/src/resources/certificates.tsx | 8 | yes | yes |
| /system/config | web/src/pages/SystemConfigPage.tsx | 0 | no | no |
| /system/operators | web/src/resources/operators.tsx | 11 | yes | yes |
| /system/website | web/src/pages/WebsiteEditorPage.tsx | 0 | no | no |

## The absence scan: anything already listening

| scan | files naming it | hits |
| --- | --- | --- |
| browser speech APIs across web/src and web/e2e | (none) | 0 |
| microphone or audio entry points across internal | (none) | 0 |

## The mechanism lines

| code line | mechanism | status |
| --- | --- | --- |
| web/src/components/CustomMenu.tsx:26 | a menu row is the only thing a command can name | pinned |
| web/src/components/CustomMenu.tsx:40 | a gated row is still a row the vocabulary carries | pinned |
| web/src/i18n/en-US.ts:23 | the English phrase set is the rendered label block | pinned |
| web/src/i18n/indonesian-overrides.ts:24 | the Indonesian phrase set is the rendered label block | pinned |
| web/src/i18n/indonesian-overrides.ts:31 | a damaged transcript is scored against a real label | pinned |
| web/src/App.tsx:132 | a resource declaration gives a destination its page | pinned |
| web/src/App.tsx:208 | a custom route gives a path its page | pinned |
| web/src/App.tsx:121 | the home dashboard is the root path | pinned |
| internal/adminapi/isp.go:1188 | the server's own guard closes the action vocabulary | pinned |
| internal/adminapi/isp.go:101 | a mutation an operator could speak is a write route | pinned |
| internal/adminapi/sessions.go:269 | the plainest operation is a POST on a record | pinned |
| internal/adminapi/platform_tenants.go:52 | a route tail that is also a screen label | pinned |
| web/src/resources/isp.tsx:5 | the typed fallback the console ships is a list filter | pinned |
| internal/voicenav/resolve.go:15 | the resolver's thresholds, printed with the run | pinned |
| .github/workflows/ci.yml:97 | the repository's gate is a build and test, never an audio run | pinned |

## The statements, pinned to code

| id | statement | code line | status | measured | asserts an absence |
| --- | --- | --- | --- | --- | --- |
| C01 | the words a command may name are the words the console prints, so a phrase exists only because a locale renders it | web/src/components/CustomMenu.tsx:26 | confirmed | 23 menu rows, 46 phrases | no |
| C02 | both shipped locales contribute the vocabulary, because either one is what an operator hears on screen | web/src/i18n/en-US.ts:23 | confirmed | 23 en-US + 23 id-ID phrases, 0 destinations speakable in neither locale | yes |
| C03 | every path the resolver can answer with is a route the app shell already declares, so voice adds no application | web/src/App.tsx:132 | confirmed | 23 of 23 destinations resolve to a declared route | yes |
| C04 | the console's write surface is larger than its read surface, and none of it is reachable through the answer type | internal/adminapi/sessions.go:269 | confirmed | 78 write registrations vs 68 reads; a verdict carries a menu path, never a method, id or payload | no |
| C05 | the mutation vocabulary is taken from the server's own guard and action routes, never typed by hand | internal/adminapi/isp.go:1188 | confirmed | 17 action words, each traced to a guard line or a write route tail | no |
| C06 | a derived refusal list also contains screen words, so the pruning rules are measured rather than assumed | internal/adminapi/platform_tenants.go:52 | confirmed | 32 derived tokens, 0 dropped as too short, 4 dropped as screen words, 28 kept | yes |
| C07 | refusal is checked before scoring, so an utterance that names an operation never reaches a screen | internal/adminapi/isp.go:101 | confirmed | 7 of 7 operation transcripts refused, 0 navigated to a mutation | yes |
| C08 | no transcript that the study expects to stop is answered with a confident screen | web/src/i18n/indonesian-overrides.ts:31 | confirmed | 0 wrong-screen answers, 0 label fragments guessed into a screen, over 58 transcripts | yes |
| C09 | a transcript the recognizer damaged still names the screen it meant, across every error class the corpus carries | web/src/i18n/en-US.ts:39 | confirmed | 7 of 8 damaged transcripts land on the intended screen, over 5 error classes | no |
| C10 | the vocabulary is role-blind: a gated row stays speakable, exactly as it stays reachable by URL | web/src/components/CustomMenu.tsx:40 | confirmed | 9 of 23 destinations are gated in the menu and still in the vocabulary | yes |
| C11 | voice is necessarily opt-in: the product contains no speech API reference and no server-side listener | web/src/App.tsx:117 | confirmed | 0 speech-API references in the frontend, 0 microphone-listener references in the backend | yes |
| C12 | the text fallback is measured per screen, and it is a navigation-level fallback rather than a typed one on most pages | web/src/App.tsx:121 | confirmed | 23 of 23 destinations traced to a page, 140 typed inputs in total, 12 screens with none | yes |
| C13 | the one screen that carries a typed filter is the one the corpus can reach with it | web/src/resources/isp.tsx:5 | confirmed | 8 screens bind setFilters, 140 text or search inputs across the resolved pages | yes |
| C14 | the resolver's thresholds are constants the run prints, so a tuning change shows up as a different measured outcome | internal/voicenav/resolve.go:15 | confirmed | accept floor 78%, margin 6%, deny floor 82%; 57 of 58 transcripts match the answer the study requires | no |
| C15 | no product code imports this study, and the study imports nothing that can write | .github/workflows/ci.yml:175 | confirmed | 0 importers outside internal/voicenav, 28 refusal tokens held in memory only | yes |

## What this study does not close

| id | what the study cannot close | what it would need | kind of asset | verdict |
| --- | --- | --- | --- | --- |
| G01 | the transcripts here are authored, not heard: what the study prices is the resolver's behavior over a mis-transcription, not a real recognizer's error distribution | a recorded corpus of operators speaking the two shipped locales, run through the ASR the product would use | audio corpus | no-go |
| G02 | the score a transcript gets is a string-similarity figure; whether an operator would accept that answer, and how long a wrong one takes to undo, is a human measurement | a panel of operators using a prototype push-to-talk control | human respondents | no-go |
| G03 | the resolver runs on a machine reading the tree; latency, permission prompts, and what a browser actually transcribes of an Indonesian label are only visible in a session | a browser session with the Web Speech API against the dev server and a seeded login | live browser session | no-go |
| G04 | 9 gated destinations stay in the vocabulary, so a lower-level operator can be routed at a path the menu hides; whether that path then answers is a server question this study does not ask | a signed session per role, spoken at each gated screen | live per-role sessions | no-go |
| G05 | 4 operation sentences in Indonesian are refused by nothing but the answer type; a synonym or glossary layer is a product decision about what the voice vocabulary should cover | a decision on TR-F013 scope: a navigation-only vocabulary, or a curated synonym set under the same deny-first rule | product owner | no-go |
| G06 | the device is unstudied: a headset, a phone speaker in a NOC, and a car microphone would produce different damage, and the card forbids a permanent listener without saying what an allowed one costs | a device and duty-cycle decision, plus the battery and data measurements that follow from it | hardware and a policy decision | no-go |

## The import set this package builds on

| package import |
| --- |
| crypto/sha256 |
| encoding/hex |
| fmt |
| github.com/stretchr/testify/assert |
| github.com/stretchr/testify/require |
| go/parser |
| go/token |
| io/fs |
| os |
| path/filepath |
| reflect |
| regexp |
| sort |
| strconv |
| strings |
| sync |
| testing |
| unicode |

## What the digest locks

The digest is the sha256 of the tab-separated rows below, so the memo's tables
are reproducible from the tree alone and move the moment the tree moves.

- 23 vocabulary rows
- 17 action words
- 28 refusal tokens kept of 32 derived
- 58 scored transcripts
- 23 fallback rows
- 15 mechanism lines
- 15 statements
- 6 limits

- digest: `443ac7d1d52ff577c7d47b2189cdc9abc03f6a8861dc7037e4f8e38e7b803b95`

## Verification

```
go build -p 1 ./...                                # clean
go vet -p 1 ./internal/voicenav/...                # clean
gofmt -l internal/voicenav/                        # empty
golangci-lint run ./internal/voicenav/...          # 0 issues
go test -p 1 -count=1 ./internal/voicenav/         # ok (15 statements confirmed)
go test -p 1 -count=1 ./internal/...               # ok, whole internal tree
grep -rn voicenav --include=*.go internal/ cmd/ | grep -v ^internal/voicenav/   # 0 importers
```

The tables above come from
`VOICE_DUMP=1 go test -p 1 -count=1 -v -run TestReportMarkdownDump ./internal/voicenav/`,
so every number in the prose is the same run. The digest
`443ac7d1d52ff577c7d47b2189cdc9abc03f6a8861dc7037e4f8e38e7b803b95` locks 23
vocabulary rows, 17 action words, 28 refusal tokens kept of 32 derived, 58 scored
transcripts, 23 fallback rows, 15 mechanism lines, 15 statements and 6 limits; it
changes the moment the tree changes.

## No-go register

Six limits stay open, and none of them is closable from the tree: three need an
asset this round cannot produce (a recorded audio corpus, a human panel, a live
browser session), one needs per-role signed sessions, one is a `TR-F013` scope
call, and one is a hardware-and-policy decision. Each is recorded as **no-go**
rather than as done:

| id | needs | kind of asset | verdict |
| --- | --- | --- | --- |
| `G01` | a recorded corpus of operators speaking both shipped locales, through the ASR the product would use | audio corpus | **no-go** - the transcripts here are authored; the study prices the resolver over a mis-transcription, not a recognizer's error distribution |
| `G02` | a panel of operators using a prototype push-to-talk control | human respondents | **no-go** - a similarity score is not acceptance, and the cost of undoing a wrong screen is not derivable from source |
| `G03` | a browser session with the Web Speech API against the dev server and a seeded login | live browser session | **no-go** - latency, permission prompts and what a browser actually transcribes of an Indonesian label are only visible in a session |
| `G04` | a signed session per role, spoken at each of the 9 gated screens | live per-role sessions | **no-go** - the vocabulary is role-blind; whether a routed-but-hidden path then answers is a server question |
| `G05` | a decision on `TR-F013` scope: navigation-only vocabulary, or a curated synonym set under the same deny-first rule | product owner | **no-go** - 4 Indonesian operation sentences are refused by nothing but the answer type; extending the deny list is a product call, not a finding |
| `G06` | a device and duty-cycle decision, plus the battery and data measurements that follow | hardware and a policy decision | **no-go** - the card forbids a permanent listener without saying what an allowed one costs |

## Residual limits

- **The 58 transcripts are written by the study.** They are shaped like real
  recognizer damage - transposition, deletion, truncation, word-boundary splits,
  insertions, code-switching, lead-ins, fillers - and every one names a screen the
  console actually prints. None of them is an ASR output. `G01`.
- **The scores are string similarity, not comprehension.** `1 - edit distance /
  longer rune length` over per-word best matches, with two- and three-token joins
  allowed. A label word that the transcript never approximated costs its share of
  the score; extra transcript words cost nothing. That is what makes `buka halaman`
  harmless and `hm ah uh` only 29%.
- **The thresholds are this prototype's tuning.** 78% accept floor, 6% margin, 82%
  deny floor, stated in `resolve.go` and printed with the run (`C14`). A shipped
  voice layer would need them re-fitted to a real recognizer, which is exactly what
  `G01` and `G02` refuse to fake here.
- **The mutation guarantee is "no path to it", not "it was blocked".** The answer
  type cannot express a write, and the deny list refuses operation words before
  scoring; neither is a server-side authorization check. A real implementation
  still has to answer with a route the shell declares and let the API enforce the
  role - which is `G04`'s question.
- **Screen tracing follows `App.tsx`'s import graph.** A destination whose page is
  rendered through a component the parser cannot attribute is reported as
  untraced, and `TestEveryDestinationTracesToAPage` fails rather than letting the
  fallback table quietly shrink. Today all 23 trace.
- **Typed-field counts are element counts in one file.** 12 screens show 0 because
  their input comes from a child component or from react-admin's built-in list
  filter, not because typing is impossible there. The two columns are kept apart
  for exactly that reason.
- **23 destinations, 46 phrases, 68 read and 78 write registrations are today's
  totals.** Every table moves with them, and the claims fail the build when they
  do.
