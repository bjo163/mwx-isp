# EAP-PWD need study for milestone M12 (ENG-007)

Date: 2026-10-29 · Backlog: `ENG-007` · Feature: TR-F004 · Milestone: M12 · Prototype: `internal/eappwdneed/`

> **Question 2 was answered, then corrected the same day.** This study found that
> the archive carried no EAP-PWD text and filed the Network Access Identifier
> under its name. **M12.1** vendored RFC 5931 as `docs/rfcs/rfc5931-eap-pwd.txt`
> and renamed the NAI document to `rfc7542-network-access-identifier.txt`, so the
> `eap-pwd-spec` verdict below moved from **mislabeled** to **carried**, so M12
> stops being blocked on a document once the M12.1 commit merges. The tables and
> decisions here were updated in place; the answer to question 1 (11/11 claims
> confirmed) is unchanged.

## Question

M12 says "EAP-PWD password authentication", planned P3, on demand. Two things
have to be checked before anyone writes a handler:

1. Is the reason EAP-PWD was proposed still true in this codebase, or has the
   authentication surface moved on since the roadmap line was written?
2. Is the specification this milestone would implement actually in the
   repository, under the name the project files it under?

## What counts as evidence

The prototype enforces three rules, each with a test behind it:

- A claim is proved by an identifier the compiler sees in product code, not by a
  sentence that mentions it. Comment lines are blanked before matching, and a
  citation that only ever lands on a comment fails
  `TestEveryRealCitationIsACodeLine`; the blanking itself is
  `TestCommentMentionsAreNotEvidence`.
- A longer name cannot prove a shorter claim: matching is at word boundaries, so
  `GetPassword` does not confirm a claim about a stored `Password` field
  (`TestLongerNameDoesNotConfirmAShorterClaim`).
- A claim whose file moved or whose symbol disappeared is reported as `broken`,
  never dropped (`TestMovedFileIsReportedBrokenNotDropped`). A memo that kept
  citing it would be quoting a statement the code no longer makes.

## Prototype

Read-only, imported by nothing, writes nothing.

| File | Role |
| --- | --- |
| `internal/eappwdneed/need.go` | 11 pinned need claims, 3 negative claims, `Verify` |
| `internal/eappwdneed/vendoring.go` | reads the header each RFC prints about itself, audits the cited numbers |
| `internal/eappwdneed/render.go` | the markdown tables below |

## Verdict

**11 of 11 need claims confirmed, 3 of 3 gaps still open, 2 vendoring findings
across 56 carried specifications.** The need argument is intact; the
specification is not carried.

### The claims the need rests on

| Claim | Evidence | What it implies |
| --- | --- | --- |
| Password storage is recomputable | `internal/domain/radius.go:54` | EAP-PWD is the only method that authenticates a password without the server holding it in a recomputable form |
| EAP-MD5 reads the stored password | `internal/radiusd/plugins/eap/handlers/md5_handler.go:97` | every non-tunneled password method inherits this exposure |
| EAP-MSCHAPv2 rebuilds the NT response | `internal/radiusd/plugins/eap/handlers/mschapv2_handler.go:264` | the same exposure is the weak half of PEAP |
| PEAP's inner exchange rejects any inner type but MSCHAPv2 | `internal/radiusd/plugins/eap/handlers/peap_inner.go:148` | a password method that is not MS-CHAPv2 has no place in the tunnel that exists today |
| TTLS inner stops at password methods | `internal/radiusd/plugins/eap/handlers/ttls_inner.go:89` | inner PAP/CHAP/MS-CHAP/MS-CHAP-V2 exist, but the tunnel that would carry a tunneled EAP exchange is still half built |
| EAP-TLS grants only on a verified certificate identity | `internal/radiusd/plugins/eap/handlers/tls_handler.go:191` | this is exactly the requirement EAP-PWD is proposed against |
| The LDAP backend only serves cleartext methods | `internal/radiusd/ldap_backend.go:120` | a directory account cannot be proved by a hash-based method |
| The method enum has no certificate-free, hash-free option | `internal/app/config_manager.go:157` | the gap is a configuration-surface fact, not an opinion |
| An unregistered method is dropped, not accepted | `internal/radiusd/auth_stages.go:448` | adding EAP-PWD means handler → registration → enum entry, in that order |
| EAP-OTP already ships fail-closed | `internal/radiusd/plugins/eap/handlers/otp_handler.go:95` | there is precedent for a declared method that denies rather than guesses |
| Type constants stop at EAP-TLV (33) | `internal/radiusd/plugins/eap/interfaces.go:32` | `interfaces.go` is where a type 52 constant would be added |

### Gaps that are not already duplicated

`eap-pwd`, `TypePWD` and `dragonfly` appear nowhere under `internal/`. The scan
excludes the prototype's own directory, because this package quotes the very
strings it tests (`TestAbsenceScanIgnoresThePrototypeItself`).

So M12 is a real gap, not a rename of something already present.

### The cited specifications

| Citation | Verdict | Text carried in | File named for the topic | What that file actually is |
| --- | --- | --- | --- | --- |
| `eap-pwd-spec` (RFC 5931 EAP-PWD) | **carried** (by M12.1; mislabeled when this study ran) | `docs/rfcs/rfc5931-eap-pwd.txt` | same | "Extensible Authentication Protocol (EAP) Authentication Using Only a Password" |
| `eap-md5-spec` (RFC 3748 EAP-MD5) | **carried-elsewhere** | `docs/rfcs/rfc3748-eap.txt` | `docs/rfcs/rfc3851-eap-md5.txt` | "Secure/Multipurpose Internet Mail Extensions (S/MIME) Version 3.1 Message Specification" |
| `teap-spec-control` (RFC 7170 TEAP) | carried | `docs/rfcs/rfc7170-teap.txt` | same | "Tunnel Extensible Authentication Protocol (TEAP) Version 1" |

The RFC numbers are not this study's recollection: each one is read back out of
the project document that cites it, and
`TestEverySpecNumberIsCitedByAProjectDocument` fails if a cited number does not
appear in its `CitedIn` file. When this study first ran, no file under
`docs/rfcs` printed `Request for Comments: 5931`, and the file whose name
promised EAP-PWD was RFC 7542. `TestEAPWDSpecIsCarriedUnderTheNameThatPromisesIt`
now pins the corrected state: RFC 5931 is carried, exactly one file claims the
`-eap-pwd` name, and the carried text contains the phrases an implementer needs
(`EAP-pwd`, `shared password`, `Ciphersuite`) rather than merely the right number.

TEAP is kept as a positive control, and the archive still supplies a live
carried-elsewhere case in EAP-MD5, so the audit has to keep distinguishing "the
text is here but filed under a misleading name" from "the text is not here" and
from "a file named for the topic is a different document". The third case has no
example in the real tree anymore, so it is exercised by a fixture
(`TestAuditReportsMislabeledUnderAFixture`) instead of being left untested.

The curated handbook already knew the citation was wrong:
`docs-site/src/en/rfc-index.md` carried a "Catalog accuracy note" naming
`rfc7542-eap-pwd.txt` as the NAI specification and RFC 3748 §5.4 as the real home
of EAP-MD5. What was still wrong was the archive itself and its own index —
`docs/rfcs/FILE-NAMING.md` mapped "RFC 7542 → `rfc7542-eap-pwd.txt` → EAP-PWD" —
and that mapping was corrected with the vendoring. RFC 7542 is a document this
project does need, for NAI and anonymous outer identities (M10.5 cites RFC 7542
§2.4); it is simply not the password method. The Indonesian side of the docs site
has no RFC catalog of its own — `docs-site/src/id/rfc-index.md` is a stub that
links to the English chapter — so there was no second copy of the note to fix.

Number hygiene is otherwise clean: every carried file agrees with the number it
prints about itself (`TestCarriedArchiveHasNoNumberMismatch`), and the audit reads
59 specification documents under `docs/rfcs`.

## Decisions

1. **Keep M12 as a real, open milestone.** The premise ("every password method
   here needs the server to hold the password") is true today, verified claim by
   claim, and no certificate-free password method exists.
2. **M12 was blocked on a document, not on code — and M12.1 cleared it.** Before
   any EAP-PWD work, the actual specification had to be fetched into `docs/rfcs`
   under its correct number and `FILE-NAMING.md` corrected. This study did not add
   or rename anything in the archive; that was a separate, explicitly approved
   step, taken the same day as M12.1: RFC 5931 is carried, the NAI document is
   named for itself, and the type code the sequence below relies on is now
   quotable from the archived text ("IANA has allocated a new EAP method type for
   EAP-pwd (52)", RFC 5931 §5).
3. **Sequence if M12 is scheduled**: obtain the specification → `TypePWD = 52` in
   `interfaces.go` → DRP/dragonfly primitives → a handler + registration + enum
   entry (the fail-closed contract in `docs/eap-otp-contract.md` is the template)
   → an inner-method slot in TTLS, which is where `ErrTTLSInnerNotImplemented`
   already reserves the place.
4. **Do not treat PEAP as a host for it.** PEAP's inner state machine is
   MSCHAPv2-shaped; retrofitting EAP-PWD there is a redesign of `peap_inner.go`,
   not an addition.
5. **Stay on the checklist's own boundary**: TR-F004 marks EAP-PWD "on demand…
   avoid sinking into a maintenance swamp for the sake of protocol completeness".
   Nothing in this study argues for building it because the number is missing.
   TR-N001-TR-N005 untouched; no charging, ERP or finance surface appears here.
6. **This study cannot and does not claim any client support.** It reads this
   repository only; `TestNoClaimAssertsClientSupport` forbids a claim from naming
   a platform or an interoperability result.
7. **The EAP-MD5 filename is reported, not fixed.** `rfc3851-eap-md5.txt` holds
   the S/MIME specification; the EAP-MD5 method is RFC 3748 §5.4, already carried
   in this archive. Renaming or re-filing it is product documentation work
   outside ENG-007's scope, so it is proposed here for a separate decision:
   either rename the file to what it is (`rfc3851-smime.txt`) or drop it and point
   the checklist at `rfc3748-eap.txt`. The audit keeps a live verdict for it, so
   whichever way that goes, a regression is caught by a test rather than by
   memory.

## What this means for M12

M12 stays Planned, with a cheaper precondition than M11: it is one specification
file plus a naming-table correction before the first line of protocol code, and
the code work is a new method rather than a retrofit. The gate that makes the
work safe to start is documentation, so it is worth doing even if the milestone
is never scheduled.

## Verification

- `go test ./internal/eappwdneed/` - 25 tests, all passing; the tables above are
  the output of `RenderMarkdown` on the real tree (`TestRenderMarkdownRealTree`).
- `golangci-lint run ./internal/eappwdneed/...` - 0 issues.
- `internal/eappwdneed` has no importers outside its own tests.
- No product code, RFC text, roadmap or checklist content other than the M12 and
  TR-F004 status lines was changed by this item.

## Residual limits

- The prototype proves claims by identifiers. It cannot show that a method
  behaves correctly, only that the code that a claim depends on is present.
- Whether the supplicants a deployment actually uses offer EAP-PWD is outside
  this repository's reach; it belongs to the `ENG-002` hardware matrix, which is
  blocked on that asset.
- Whatever the current normative text of the EAP-PWD specification says -
  including any revision or security caveat - cannot be stated here, because the
  document is not carried. The absence is the finding; no property of the method
  is claimed beyond what a certificate-less password exchange means by
  definition.
- The negative claims cover `internal/` Go files. A future vendor SDK under
  another tree could carry DRP code unnoticed.
