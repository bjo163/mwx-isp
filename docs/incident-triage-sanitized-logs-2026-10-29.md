# FUT-002 — What incident triage over sanitized logs can name today

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `FUT-002` (`TR-F031`, P3, RISET, dependency `ENG-028` delivered) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/incidenttriage` (read-only, imported by nothing).
Criterion under test: **Dataset label dan hallucination rate dilaporkan, no
automatic remediate.**

## The question

The card asks for "eksperimen klasifikasi akar masalah dari data sintetis dengan
evidence backlinks dan opsi non-AI" over sanitized logs. Three things have to be
shown, not asserted:

1. **A labeled dataset** - incidents with a stated cause label and a stated reason
   for the label, in the shape the operator log export actually hands out.
2. **A hallucination rate** - which requires a definition of hallucination that a
   test can check, not a synonym for "wrong".
3. **No automatic remediate** - which has to be a property of the data model, not
   a promise in a README.

The design answer to the second and third is the same answer: an engine may only
name a cause it can point at. Every verdict carries the token that fired, the log
line it was found on, and the product code line that writes that token. A verdict
with no such support is *measured* as a hallucination, and the struct has nowhere
to put a command even if an engine tried to invent one.

The study is deliberately non-AI. The card says "opsi non-AI", and `claims.go`
records as an open gap that nothing in this tree calls a language model, so an
LLM comparison would have no production baseline to be compared against.

## What counts as evidence

The same three rules the `ENG-006`, `ENG-007`, `ECO-006` and `FUT-001` studies
used, enforced by tests in this package:

- **A citation is a code line.** Comment-only lines are blanked before a symbol is
  matched (`TestClaimsAllLandOnCodeLines` reads through `blankComments`).
- **A whole identifier.** Word-boundary matching, so `bind` cannot be proven by
  `Rebind` (`containsBounded`). This is load-bearing here: incident `c18` is a
  pool re-bind that the taxonomy must not read as the directory's service bind,
  and the bounded match is what makes it abstain.
- **Drift is reported, never dropped.** A vanished file is `broken`; a gap that
  got filled is `vacated` (`TestGapsStayOpenAndTheScanIgnoresItself`, and the
  absence walk skips `internal/incidenttriage/` so the prototype cannot refute
  itself).

Three rules specific to this study:

- **No invented token.** All 20 signals in the taxonomy must resolve to a line of
  product code that writes that string (`TestEverySignalResolvesToAProductCodeLine`),
  and the report prints the file:line for each one.
- **No unsanitized line reaches a rule.** `Triage` accepts `[]SanitizedLine`, and
  the only way to build one is `Intake.Sanitize`, which calls the product's own
  `internal/redact` - the same path `internal/adminapi/system_logs.go` uses before
  an entry leaves the process. "Triage ran on sanitized logs" is a property of the
  type, not of a test (`TestSanitizeRunsTheProductsOwnRuleSet`).
- **No action field exists.** `TestVerdictCarriesNoActionField` reads `Verdict` by
  reflection and expects an empty list of fields that could carry an action,
  command, verb, route, method, body, exec or remedy. If someone later adds one,
  that test fails and the claim has to be re-argued, not quietly dropped.

## The prototype

| File | Role |
| --- | --- |
| `intake.go` | Parse and sanitize: mirrors the export's own parser, then runs `internal/redact` over every value and the free-text message. Owns `SanitizedLine`. |
| `dataset.go` | The labeled corpus - 20 incidents, the secret values planted in them, and the raw JSON lines as the log file holds them. |
| `causes.go` | The ten-class taxonomy, each signal anchored to a string the product writes, plus the grouping from lines to incidents. |
| `engines.go` | Three deterministic engines: the token rules, a subsystem-name baseline, and a never-abstains counterfactual. |
| `score.go` | The hallucination definition, the accuracy and per-cause tallies, and the single `Run` that produces every number in this memo. |
| `remediate.go` | The mutating verbs measured from the admin API's own route registrations, and the reflection over `Verdict`. |
| `claims.go` | 16 positive statements and 6 gaps, plus the scans that re-check the namespace and job fields. |
| `evidence.go` | The comment-blanking, bounded-match and tree-walk helpers. |
| `render.go` | Emits every table in this memo from one run. |

Nothing in `internal/` or `cmd/` imports it, it opens no database connection, and
the only files it touches are opened for reading (`TestThePrototypeWritesNothing`).

## The corpus

20 incidents, 24 log lines, one label each for 17 of them and three that are
labeled `unresolved` because the honest answer is that the lines do not say. Every
message string is copied from a template the product writes; `claims.go` proves
each token still resolves to a code line.

The corpus is synthetic and says so. It is the dataset the criterion asks for, not
a sample of a live tenant's log file - which is why the accuracy figures below are
statements about these three engines over this taxonomy, not a production
precision claim.

## Verdict

| Measure | Value |
| --- | ---: |
| Incidents in the corpus | 20 |
| Log lines in the corpus | 24 |
| Incidents with a single labeled cause | 17 |
| Incidents labeled unresolved | 3 |
| Cause classes in the taxonomy | 10 |
| Signals in the taxonomy | 20 |
| Signals resolved to a product code line | 20 |
| Incidents the grouping produced | 20 |
| Grouping window | 2m0s |
| Namespace-plus-level keys the baseline fitted | 6 |
| Of those, keys whose corpus labels disagree | 3 |
| Namespace literals product code writes | 3 |
| Log lines that attach a namespace | 55 |
| Job literals product code writes | 0 |
| Log lines that attach a job | 0 |
| rule-token-match: named a cause | 17 |
| rule-token-match: correct | 17 |
| rule-token-match: wrong | 0 |
| rule-token-match: abstained | 3 |
| rule-token-match: hallucinated | 0 |
| rule-token-match: accuracy | 100.0% |
| rule-token-match: hallucination rate | 0.0% |
| namespace-baseline: named a cause | 19 |
| namespace-baseline: correct | 8 |
| namespace-baseline: wrong | 11 |
| namespace-baseline: abstained | 1 |
| namespace-baseline: hallucinated | 19 |
| namespace-baseline: accuracy | 42.1% |
| namespace-baseline: hallucination rate | 100.0% |
| always-name: named a cause | 20 |
| always-name: correct | 17 |
| always-name: wrong | 3 |
| always-name: abstained | 0 |
| always-name: hallucinated | 3 |
| always-name: accuracy | 85.0% |
| always-name: hallucination rate | 15.0% |
| Secret values the corpus planted | 14 |
| Secret values that survived redaction | 0 |
| Mutating verbs measured from the admin API | 9 |
| Verdict fields that could carry an action | 0 |
| Verdicts naming one of those verbs | 0 |
| Claims confirmed against product code | 16 of 16 |
| Gaps still open | 6 of 6 |

## How a batch becomes incidents

The export is a stream of lines, so the unit a verdict should speak for is a run of
lines that the operator would read together: same namespace and job, no gap wider
than two minutes. Grouping the 24 sanitized lines produced 20 incidents, which is
exactly the corpus's own case count - `TestGroupingProducesOneIncidentPerCase`
holds that, so the study is scoring incidents the grouping found, not the cases it
wrote down.

Two properties are tested rather than assumed:

- `TestGroupingMergesOnlyLinesInsideTheWindow` - the three lines of one rejected
  session (`c13`) become one incident, so a session that complains three times is
  triaged once, not three times.
- `TestGroupingDoesNotMergeDifferentSubsystems` - adjacency is not enough; a line
  from another namespace starts a new incident.

The timestamp path is the part that could have been silently wrong. zap writes
`ts` as a numeric second, and the export converts it before filtering; without that
conversion every line looks timestamp-less and the whole stream collapses into one
incident. `timestampField` mirrors the export's handling, and the grouping counts
above are only reachable because of it.

## The labeled corpus

| Label | Incidents |
| --- | ---: |
| credential-rejected | 3 |
| profile-unavailable | 3 |
| unresolved | 3 |
| malformed-packet | 2 |
| notification-failure | 2 |
| tenant-integrity | 2 |
| backup-failed | 1 |
| directory-unavailable | 1 |
| nas-unknown | 1 |
| rate-limited | 1 |
| tunnel-negotiation | 1 |

| Case | Subsystem | Job | Lines | Label | Why that label |
| --- | --- | --- | ---: | --- | --- |
<!-- (none) in Subsystem means the code path writing that message attaches no namespace field; no product line attaches a job. -->
| c01 | radius | (none) | 1 | credential-rejected | the reject tag is the only thing the auth validators return for a bad credential |
| c02 | radius | (none) | 1 | credential-rejected | an MS-CHAP2-Response whose peer value did not match is still a credential rejection |
| c03 | (none) | (none) | 1 | profile-unavailable | the profile cache is the only producer of this message |
| c04 | adminapi | (none) | 1 | profile-unavailable | a subscription whose package has no linked profile cannot authorize anything |
| c05 | radius | (none) | 1 | directory-unavailable | the service bind is the directory's own failure string |
| c06 | radius | (none) | 1 | malformed-packet | an AVP header running past the flight is a wire-format fault, not an auth fault |
| c07 | radius | (none) | 1 | tunnel-negotiation | EAP-TLS with no trust anchor is a configuration refusal, not a client failure |
| c08 | adminapi | (none) | 1 | tenant-integrity | the restore gate is the only place that names a cross-tenant reference |
| c09 | adminapi | (none) | 1 | backup-failed | the backup endpoint validates before it writes |
| c10 | (none) | (none) | 1 | notification-failure | the WhatsApp device store is the only producer of this string |
| c11 | radius | (none) | 1 | nas-unknown | the stage name in the pipeline log is the NAS lookup |
| c12 | radius | (none) | 1 | rate-limited | the rate-limit stage names itself, and shedding is not an auth failure |
| c13 | radius | (none) | 3 | credential-rejected | three lines of one rejected session still carry one cause |
| c14 | adminapi | (none) | 1 | profile-unavailable | a disabled profile behind a live subscription is the same class as a missing one |
| c15 | adminapi | (none) | 1 | tenant-integrity | an unresolvable tenant id is the boundary class, not a packet class |
| c16 | (none) | (none) | 1 | unresolved | a bare error with no taxonomy token must not be given a cause |
| c17 | (none) | (none) | 2 | unresolved | two causes match one token each, so naming either would be a coin flip |
| c18 | (none) | (none) | 1 | unresolved | re-binding a pool is not the directory's service bind; the match has to be whole |
| c19 | radius | (none) | 1 | malformed-packet | a packet shorter than its authenticator is a wire fault |
| c20 | (none) | (none) | 2 | notification-failure | the second notification incident keeps the class non-degenerate |

The three `unresolved` cases are the part of the dataset that does work: they are
there to make abstention measurable, and an engine that scores 100% accuracy only
by refusing to guess is distinguishable from one that guesses right by accident
because of them.

## What redaction touched

The corpus deliberately carries things that must not survive: NAS shared secrets,
customer e-mail addresses, framed IPv4 addresses and an inline password assignment.
14 secret values were planted and 0 appear in the sanitized text.

| Measure | Value |
| --- | ---: |
| Structured lines | 24 |
| Fields keyed secret | 7 |
| Fields keyed customer PII | 6 |
| Fields keyed address | 3 |
| Messages changed by the free-text sweep | 2 |

`TestTheCorpusCarriesSomethingToRedact` exists because a pre-clean corpus would
make the sanitization half of this card vacuous - a rule set that redacts nothing
has proven nothing.

## What the export's own filters carry

The operator log export lets an operator filter on namespace and job. Counted from product code, that is what those two fields actually hold:

| Field | Literal | Times written |
| --- | --- | ---: |
| namespace | "radius" | 45 |
| namespace | "adminapi" | 9 |
| namespace | "isp" | 1 |
| job | no literal written anywhere | 0 |

This measurement is what stopped the study from making a false claim. The first
version of this memo said only the admin API writes a namespace; the scan disagreed
- three literals across 55 write sites, and none of them in `internal/domain`,
`internal/notify` or `internal/ldapauth`, which produce four of the taxonomy's ten
classes. So the claim became the measured one (`namespace-is-coarse`) and the
absence became a gap (`no-namespace-on-the-supporting-packages`).

The consequence is the baseline's score below: three namespace literals and zero
job literals cannot address a ten-class taxonomy, and no amount of fitting changes
that.

## The taxonomy and where its tokens come from

| Cause | Signal | First product code line that writes it |
| --- | --- | --- |
| credential-rejected | "radus_reject_passwd_error" | internal/app/radius_metrics.go:17 |
| credential-rejected | "MS-CHAP2-Response" | internal/radiusd/plugins/eap/handlers/ttls_inner.go:89 |
| directory-unavailable | "service bind" | internal/ldapauth/ldapauth.go:283 |
| directory-unavailable | "BindDNTemplate is empty" | internal/ldapauth/ldapauth.go:155 |
| directory-unavailable | "starttls" | internal/ldapauth/ldapauth.go:257 |
| profile-unavailable | "failed to get profile from cache" | internal/domain/radius_user_getters.go:33 |
| profile-unavailable | "linked RADIUS profile is disabled" | internal/adminapi/isp_operations.go:159 |
| profile-unavailable | "package has no linked RADIUS profile" | internal/adminapi/isp_operations.go:149 |
| malformed-packet | "truncated AVP header" | internal/radiusd/plugins/eap/handlers/ttls_avp.go:66 |
| malformed-packet | "AVP Length" | internal/radiusd/plugins/eap/handlers/ttls_avp.go:85 |
| malformed-packet | "payload shorter than authenticator" | internal/radiusd/radsec_server.go:225 |
| tunnel-negotiation | "ClientCAs is required" | internal/radiusd/plugins/eap/tlsengine/engine.go:120 |
| tunnel-negotiation | "preferred EAP method disabled" | internal/radiusd/auth_stages.go:411 |
| tenant-integrity | "from another tenant" | internal/adminapi/system_backup.go:988 |
| tenant-integrity | "invalid tenant ID" | internal/adminapi/platform_tenants.go:81 |
| backup-failed | "Backup failed validation" | internal/adminapi/system_backup.go:526 |
| backup-failed | "Backup encryption passphrase is too short" | internal/adminapi/system_backup.go:369 |
| notification-failure | "load WhatsApp device" | internal/notify/whatsapp.go:86 |
| nas-unknown | "nas_lookup" | internal/radiusd/auth_stages.go:21 |
| rate-limited | "auth_rate_limit" | internal/radiusd/auth_stages.go:23 |

Two of these are worth naming because they are contracts rather than messages.
`radus_reject_passwd_error` is both the tag the auth validators return and a metric
name (`internal/app/radius_metrics.go:17`), so a credential-rejection rule is
matching on something other code already depends on staying stable. `nas_lookup`
and `auth_rate_limit` are stage names in the authentication pipeline, which is why
a NAS-lookup failure and a shed request are separable at all - and why the taxonomy
has a class for each rather than one "auth failed" bucket.

## The three engines

1. **`rule-token-match`** - the non-AI option the card asks for. Score each cause
   by how many of its signals appear on the incident's lines; name the leader;
   abstain when nothing matches **or** when the leader and the runner-up tie.
2. **`namespace-baseline`** - read only which subsystem complained and at what
   level, never what it said. It is *fitted on the corpus labels*, so the
   comparison is honest: it gets the same chance to learn that the rule table had
   by hand.
3. **`always-name`** - the same token rules with the abstention removed: on any
   gap it names the most frequent class in the corpus. It stands in for a
   classifier that would rather guess than say "I do not know". It is not a
   language model and is not described as one.

Hallucination is defined in code (`hallucinated` in `score.go`), and it is not
"wrong". A named cause is a hallucination when its evidence does not hold: the
verdict carries no evidence, or a token it cites is not in the incident, or the
line index it cites is not one of the incident's lines, or the token has no product
code line behind it. An engine can therefore be *correct and hallucinating* -
right about the class and supported by nothing - which is exactly the failure mode
the criterion wants counted.

## Three engines over the same corpus

| Engine | Named | Correct | Wrong | Hallucinated | Abstained | Accuracy | Hallucination rate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| rule-token-match | 17 | 17 | 0 | 0 | 3 | 100.0% | 0.0% |
| namespace-baseline | 19 | 8 | 11 | 19 | 1 | 42.1% | 100.0% |
| always-name | 20 | 17 | 3 | 3 | 0 | 85.0% | 15.0% |

Accuracy is measured over named verdicts only, so an abstention is not paid for as
a wrong answer - and not hidden either, because `Abstained` is its own column.

The three numbers say three different things:

- The rule engine's 100% accuracy is not a claim about intelligence. It is the
  statement that on this corpus it named a cause only when a taxonomy token was
  sitting in the lines, and it refused all three incidents where nothing supported
  an answer.
- The subsystem-name baseline answers 19 of 20 and every one of them is a
  hallucination, because reading `radius|error` and replying "credential-rejected"
  cites nothing in the log. Eight of them were even the right label - which is the
  whole reason "correct" is not the criterion.
- `always-name` buys three extra decisions and pays for them with a 15%
  hallucination rate. Coverage went 17 → 20, accuracy fell 100% → 85%, and the two
  extra correct answers came from guessing into classes the corpus happened to label
  that way. That is the trade the card needs to see before anyone ships a
  "summarize this incident" button.

## Per-cause detail for the rule engine

| Cause | Labeled support | Predicted | Exact | Precision | Recall |
| --- | ---: | ---: | ---: | ---: | ---: |
| credential-rejected | 3 | 3 | 3 | 100.0% | 100.0% |
| profile-unavailable | 3 | 3 | 3 | 100.0% | 100.0% |
| malformed-packet | 2 | 2 | 2 | 100.0% | 100.0% |
| notification-failure | 2 | 2 | 2 | 100.0% | 100.0% |
| tenant-integrity | 2 | 2 | 2 | 100.0% | 100.0% |
| backup-failed | 1 | 1 | 1 | 100.0% | 100.0% |
| directory-unavailable | 1 | 1 | 1 | 100.0% | 100.0% |
| nas-unknown | 1 | 1 | 1 | 100.0% | 100.0% |
| rate-limited | 1 | 1 | 1 | 100.0% | 100.0% |
| tunnel-negotiation | 1 | 1 | 1 | 100.0% | 100.0% |

Ten classes, ten rows, support of 1 for six of them. `TestEveryLabeledCaseReachesEveryCauseClass`
keeps that honest - no class is scored only as a distractor - but a class with one
incident behind it is a hypothesis, not a precision measurement. The table exists so
the thinness is visible per class instead of averaged away by the headline number.

## What each engine said, incident by incident

| Case | Label | rule-token-match | namespace-baseline | always-name |
| --- | --- | --- | --- | --- |
| c01 | credential-rejected | credential-rejected (1 links) | credential-rejected (no evidence) | credential-rejected (1 links) |
| c02 | credential-rejected | credential-rejected (1 links) | credential-rejected (no evidence) | credential-rejected (1 links) |
| c03 | profile-unavailable | profile-unavailable (1 links) | notification-failure (no evidence) | profile-unavailable (1 links) |
| c04 | profile-unavailable | profile-unavailable (1 links) | profile-unavailable (no evidence) | profile-unavailable (1 links) |
| c05 | directory-unavailable | directory-unavailable (1 links) | credential-rejected (no evidence) | directory-unavailable (1 links) |
| c06 | malformed-packet | malformed-packet (1 links) | credential-rejected (no evidence) | malformed-packet (1 links) |
| c07 | tunnel-negotiation | tunnel-negotiation (1 links) | credential-rejected (no evidence) | tunnel-negotiation (1 links) |
| c08 | tenant-integrity | tenant-integrity (1 links) | profile-unavailable (no evidence) | tenant-integrity (1 links) |
| c09 | backup-failed | backup-failed (1 links) | profile-unavailable (no evidence) | backup-failed (1 links) |
| c10 | notification-failure | notification-failure (1 links) | notification-failure (no evidence) | notification-failure (1 links) |
| c11 | nas-unknown | nas-unknown (1 links) | credential-rejected (no evidence) | nas-unknown (1 links) |
| c12 | rate-limited | rate-limited (1 links) | rate-limited (no evidence) | rate-limited (1 links) |
| c13 | credential-rejected | credential-rejected (2 links) | credential-rejected (no evidence) | credential-rejected (2 links) |
| c14 | profile-unavailable | profile-unavailable (1 links) | profile-unavailable (no evidence) | profile-unavailable (1 links) |
| c15 | tenant-integrity | tenant-integrity (1 links) | profile-unavailable (no evidence) | tenant-integrity (1 links) |
| c16 | unresolved | abstain | notification-failure (no evidence) | credential-rejected (no evidence) |
| c17 | unresolved | abstain | notification-failure (no evidence) | credential-rejected (no evidence) |
| c18 | unresolved | abstain | abstain | credential-rejected (no evidence) |
| c19 | malformed-packet | malformed-packet (1 links) | credential-rejected (no evidence) | malformed-packet (1 links) |
| c20 | notification-failure | notification-failure (1 links) | notification-failure (no evidence) | notification-failure (1 links) |

`c16`, `c17` and `c18` are the three the rule engine refuses, and each refusal has a
different reason: no token at all, a tie between two one-token causes, and a token
that only looks like one because the match is not whole. The baseline abstains on
`c18` for an unrelated accident (that namespace and level pair never appeared in
the labeled set) and names a cause with no support on the other two.

## No automatic remediate

Verdict fields that could carry an action: 0 - `TestVerdictCarriesNoActionField` reads the struct by reflection.

Mutating verbs the admin API accepts, measured from its own route registrations: activate, check, coa, disconnect, dispatch, retry, reverse, revoke, toggle.

Verdicts naming any of them: 0.

The verb list is not typed out by hand. It is the trailing segment of a non-GET
registration that addresses one record (`/sessions/:id/disconnect` → `disconnect`),
collected by walking `internal/adminapi`, and `TestMeasuredVerbsAreActionWords`
plus `TestTheVerbListIsMeasuredNotEmpty` keep the scan from silently returning an
empty list - an empty vocabulary would make "no verdict names an action" true only
because there was nothing to name. `TestNoVerdictNamesAMutatingVerb` then checks
every rendered verdict against it.

So the third quarter of the criterion holds in two independent ways: the verdict
type has nowhere to put an instruction, and the instructions the console does
accept are named by 0 of the 60 verdicts the three engines rendered over 20
incidents.

## What this shows

- **The labeled-dataset half is met.** 20 incidents, 17 single-cause labels, 3
  deliberate `unresolved` cases, and a stated reason per case. The corpus survives
  the export's own redaction path: 14 planted secrets, 0 survivors, across 7 secret
  keys, 6 PII keys and 3 address keys.
- **The hallucination half is met with a definition, not a number alone.** 0.0% for
  the rule engine, 100% for the subsystem-name baseline, 15% for the
  never-abstains counterfactual - all three from one checkable rule: a named cause
  whose evidence does not hold.
- **Evidence backlinks are real and reachable.** 20 of 20 taxonomy tokens resolve to
  a product code line, and each verdict prints the line index of the log entry and
  the `file:line` that writes the token, so an operator can be shown why the engine
  said what it said.
- **Abstention is what makes the accuracy number mean anything.** The rule engine
  refused exactly the three incidents labeled `unresolved`. Had it answered them,
  the same 17 correct answers would sit behind a 70-85% accuracy and a non-zero
  hallucination rate - which is the `always-name` row.
- **The subsystem name is not a cause.** Three namespace literals and zero job
  literals against ten classes; 3 of the 6 keys the baseline could fit saw
  disagreeing labels in the corpus. Its 42.1% accuracy is what a classifier with no
  access to the message text is worth, and it is the reason a triage surface has to
  read the lines, not the filters.
- **Part of the taxonomy arrives with an empty subsystem bucket.** Four classes are
  produced by code paths in `internal/domain`, `internal/notify` and
  `internal/ldapauth` that attach no namespace field at all, so an operator who
  filters the export by subsystem cannot see those incidents - a finding about the
  logging, not about the classifier.
- **No path from a verdict to a write exists.** Zero action-shaped fields on the
  struct, zero verdicts naming a measured verb, and no route, client or writer in
  the package.

## What this study does not close

| No-go | Blocking asset |
| --- | --- |
| A shipped triage surface in the console | `TR-F031` scope decision; `TR-N003` forbids turning bounded health checks into a general observability platform, and this prototype is research that no product code imports. |
| Precision on real tenant logs, or any claim that 100% would hold in production | A real incident corpus with labeled respondents. Everything above is measured on 20 synthetic incidents built from message templates the product writes; six of the ten classes have exactly one incident behind them. |
| Any AI/LLM comparison | A sanctioned model plus the data-handling decision about sending log lines to it. `no-model-client` is an open gap in this tree, so there is no production baseline to compare against - and the card asks for the non-AI option first. |
| Triage that respects per-tenant log access | A live seeded admin session per role. The prototype never opens the export endpoint, so it can show what a verdict would say, not which tenant's lines an operator is allowed to read. |
| Remediation suggestions, auto-executed or human-approved | A scope change beyond `FUT-002`, whose criterion is explicitly *no automatic remediate*. The gap `no-auto-remediation` stays open on purpose; closing it is a different card with a different risk profile. |

## Decisions taken here

1. Build it as a research package over the export's own sanitized shape, not as a
   product surface - no route, no UI, no schema, no model call.
2. Make "triage ran on sanitized logs" a type-level property (`SanitizedLine` is
   only producible through `internal/redact`) instead of a test that hopes the
   fixture was clean.
3. Define hallucination as unsupported evidence, not as a wrong answer, so that an
   engine which is right by accident is still reported - that is what makes the
   baseline's 42.1% accuracy and 100% hallucination rate two facts instead of one
   contradiction.
4. Prefer abstention over a guess, and prove the preference is load-bearing by
   running the never-abstains counterfactual over the same corpus.
5. Anchor every taxonomy token to a code line rather than to a plausible-sounding
   message, and print the `file:line` for all 20 in the report.
6. Measure the "no remediate" half from the admin API's real mutating vocabulary
   instead of a hand-written forbidden-word list.
7. Report the namespace/job measurement as it came out - and rewrite the claim that
   contradicted it, moving the absent half into the open-gap table instead of
   dropping it.

## Every statement and gap, as the scan found them

### Statements the memo leans on

| ID | Statement | Evidence | Status |
| --- | --- | --- | --- |
| export-redacts-before-return | the operator log export redacts every entry before it leaves the process | internal/adminapi/system_logs.go:118 | confirmed |
| export-parses-json-lines | the export reads structured log lines and keeps unstructured ones as free text | internal/adminapi/system_logs.go:114 | confirmed |
| redact-one-rule-set | one classifier decides secret, PII and address keys, so the export and the support bundle cannot drift | internal/redact/redact.go:61 | confirmed |
| reject-tag-is-stable | a credential rejection is tagged with one stable string the validators return | internal/radiusd/auth_plugin_runner.go:95 | confirmed |
| reject-tag-is-a-metric | the same string is a metric name, so it is a contract other code can depend on | internal/app/radius_metrics.go:17 | confirmed |
| profile-cache-error-exists | a profile that cannot be resolved from the cache is logged with its own message | internal/domain/radius_user_getters.go:33 | confirmed |
| pipeline-stages-are-named | the authentication pipeline names its stages, which is what makes a NAS-lookup failure addressable | internal/radiusd/auth_stages.go:21 | confirmed |
| rate-limit-stage-exists | shedding happens in a stage of its own, so a rate-limit incident is distinguishable from an auth failure | internal/radiusd/auth_stages.go:23 | confirmed |
| mschap-validator-rejects | the MS-CHAPv2 validators are a real source of the reject tag | internal/radiusd/plugins/auth/validators/mschap_validator.go:32 | confirmed |
| avp-truncation-is-a-fault | a truncated AVP header is reported as an inner-protocol fault | internal/radiusd/plugins/eap/handlers/ttls_avp.go:66 | confirmed |
| ldap-bind-missing-user | the inner tunnel distinguishes a missing directory record from a bad password | internal/radiusd/plugins/eap/handlers/ttls_inner.go:104 | confirmed |
| tls-needs-a-trust-anchor | EAP-TLS refuses to run without ClientCAs, which is a configuration class rather than a client class | internal/radiusd/plugins/eap/tlsengine/engine.go:120 | confirmed |
| restore-gates-cross-tenant | the restore path already detects a row that references another tenant | internal/adminapi/system_backup.go:988 | confirmed |
| backup-validates-first | the backup endpoint runs a validation gate before doing anything | internal/adminapi/system_backup.go:526 | confirmed |
| whatsapp-device-store | the notification channel loads a stored device, and its failure is a distinct incident class | internal/notify/whatsapp.go:86 | confirmed |
| namespace-is-coarse | the namespace field the export filters on is written with a handful of coarse values, so a subsystem-name classifier cannot separate the taxonomy | internal/radiusd/auth_stages.go:304 | confirmed |

### Gaps the scan re-checks on every run

| ID | Gap | Scope | Status | Evidence |
| --- | --- | --- | --- | --- |
| no-triage-in-product | no product code triages a log line into a cause class | internal/adminapi, internal/app, internal/radiusd, internal/domain, pkg, cmd | open | none |
| no-labeled-incident-dataset | the repository holds no labeled incident dataset to learn from | internal, pkg, cmd, test | open | none |
| no-model-client | nothing in the tree calls a language model, so the AI half of this card has no baseline in production | internal, pkg, cmd | open | none |
| no-auto-remediation | no path from a log line to an action exists, which is what keeps this card's no-remediation half true | internal/adminapi, internal/app, internal/radiusd | open | none |
| no-persisted-incident | incidents exist only as a window over the log file; nothing is stored, so a triage verdict has no record to attach to | internal, pkg | open | none |
| no-namespace-on-the-supporting-packages | the domain, notify and directory packages attach no namespace field to any log line, so part of the taxonomy reaches the export with an empty subsystem bucket | internal/domain, internal/notify, internal/ldapauth | open | none |

## Verification

```
go build ./...                                          # clean
go vet ./internal/incidenttriage/...                   # clean
go test -p 1 -count=1 ./internal/incidenttriage/       # 28 pass, 1 skipped (the dump)
gofmt -l internal/incidenttriage/                      # empty
golangci-lint run ./internal/incidenttriage/...        # 0 issues
grep -rn incidenttriage --include=*.go internal/ cmd/ | grep -v ^internal/incidenttriage/   # 0 importers
```

The tables above come from `TRIAGE_DUMP=1 go test -p 1 -run TestReportMarkdownDump
-v ./internal/incidenttriage/`, so every number in the prose is the same run, and
`TestRenderIsDeterministic` pins that the run repeats.

The tests were checked against mutations, because a green suite that cannot fail is
not evidence:

| Mutation | Property it attacks | Caught by |
| --- | --- | --- |
| keep the raw value instead of `redact.Value(k, v)` | sanitized-only intake | `TestSanitizeRunsTheProductsOwnRuleSet`, `TestTheCorpusCarriesSomethingToRedact` |
| skip `redact.Text` on the message field | the free-text sweep | `TestSanitizeRunsTheProductsOwnRuleSet` |
| treat empty evidence as supported | the hallucination definition | `TestHallucinationDefinitionNeedsSupportNotJustWardrobe`, `TestTheNamespaceBaselineIsBlindOnTheRadiusSide`, `TestAlwaysNamingBuysCoverageWithHallucination` |
| never take the tie branch | abstention over guessing | `TestTheRuleEngineIsRightWhereTheLinesSaySo`, `TestTheRuleEngineAbstainsRatherThanGuesses` |

All four were caught; the tree was restored and re-run before this memo was written.

## Residual limits

- **The corpus is synthetic.** 20 incidents assembled from message templates the
  product writes. It measures the engines, not the world; nothing here is a claim
  about how often a real tenant hits `credential-rejected`.
- **The taxonomy is ten classes chosen by reading this repository**, not a general
  theory of ISP incidents. A cause the product logs but this table has no token for
  is invisible to all three engines, and would show up as an abstention rather than
  a miss.
- **Support of 1 for six of the ten classes.** Per-cause precision and recall are
  therefore 1-of-1 or 0-of-1 statements.
- **The baseline is fitted on the same corpus it is scored against**, which flatters
  it - it is the ceiling a subsystem-name classifier reaches here, not a
  cross-validated estimate.
- **Grouping is adjacency plus a two-minute window over namespace and job.** Real
  correlated incidents spread across subsystems (a directory outage that produces
  credential rejections downstream) become separate incidents and get separate
  verdicts. Nothing in this study models causality between incidents.
- **The token table is hand-curated over measured strings.** It catches the exact
  strings the code writes; a message reworded in a future commit stops matching, and
  `TestEverySignalResolvesToAProductCodeLine` is what fails when that happens.
- **The verb vocabulary comes from route registrations only.** A mutation the
  console accepts only as a request-body action value - `suspend`, the shape
  `FUT-001` found in the subscription lifecycle handler - is not in the measured
  list of nine, so the "names no verb" check is stated against what this scan can
  see.
