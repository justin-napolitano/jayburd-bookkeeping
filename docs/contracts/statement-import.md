# Synthetic statement-import contract

## Decision and scope

**Proposed, not approved or implemented**, 2026-10-13; owner: project maintainer.
[Issue #61](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/61)
designs the first bounded slice of [#8](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/8).
Propose `jayburd-synthetic-statement-csv/1`: an internal fixture template for one
synthetic deposit account, producing uncategorized import drafts only. No real
statement uploads, card/liability accounts, balances/reconciliation completion,
pending items, provider calls, OCR, automatic journals, fees/FX splitting or
source corrections in this version. Foreign-currency deposit accounts are allowed;
this is not a USD-only importer. [Sources SI01–SI06](../research/statement-import-sources.md)
record review depth and alternatives.

## Standards applicability and approval gate

| Assessment | Profile/mapping and unresolved evidence |
| --- | --- |
| ISO 20022 applicable to eventual bank statement exchange | No bank, registered message version or usage profile is selected for production. This custom CSV is **not** an ISO financial message. Proposed deviation: allow only an internal synthetic ingestion fixture while camt artifact/profile review remains open in [#17](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/17). Owner approval of this restriction is required before implementation. No implied camt field-level mapping or conformance. |
| ISO 4217 applicable | Currency column maps to installed alphabetic code, valid booking date and immutable currency-version ID; source release must be retained. Fixtures explicitly select reviewed metadata. Currency lists do not authorize rates/rounding. |
| ISO 8601 date semantics applicable through RFC 3339 full-date subset | `booked_on` is a valid Gregorian YYYY-MM-DD date, not an instant. No locale guessing or UTC shift. Operational timestamps use RFC 3339 UTC. Full ISO normative text not reviewed. |
| ISO 21378/21926 audit semantics unresolved for exports | Retain provenance; no audit-export format or conformity claim. No IBAN/BIC/LEI fields are accepted, so their identifier standards do not govern our opaque synthetic reference. A real identifier adapter needs a separate review. |

STD-01/03/05 review evidence must resolve this proposed internal-template exception,
metadata snapshot and mappings below. STD-02/04 future fixtures cannot substitute
for that review. SI01 supports syntax; SI06 supports separating statements from books.

## Binding and exact field contract

Before upload, an authorized actor explicitly selects entity, book and **monetary
asset deposit** ledger account. A separately reviewed binding records stable
source-account UUID, synthetic source namespace, exact `source_account_ref`, selected
currency/version, target IDs, and binding revision. Never infer ownership from
filename, last four digits, amount, description or upload actor. Binding and current
read/import permissions are checked on preview, import, replay and retrieval.
The import permission and controlled binding operation are proposed extensions,
not existing grants. A source account cannot simultaneously bind to different
books; rebinding needs a future reviewed history/migration procedure, not a new
namespace to evade duplicates. No cross-entity discovery from duplicate errors.

CSV requires this exact header/order and seven fields per record:

```csv
source_account_ref,entry_id,booked_on,currency,direction,amount,description
SYNTH-CASH-01,P001,2026-01-05,USD,outflow,12.50,Paper shop
SYNTH-CASH-01,P002,2026-01-05,USD,outflow,12.50,Paper shop
SYNTH-CASH-01,D001,2026-01-06,USD,inflow,100.00,"Owner funding, synthetic"
```

These are original fictional examples. Actual fixture bytes use CRLF, optional
final CRLF, no BOM, and printable ASCII; quoted comma/escaped double quote allowed,
embedded line breaks/control bytes forbidden. No blanks, comments, extra headers,
columns or silent whitespace trimming. Header counts toward size but not row count.
`text/csv; charset=us-ascii; header=present` describes this narrow profile; `.csv`
and MIME alone are insufficient validation. Reject unsupported encodings rather
than replacing characters. Non-ASCII real descriptions require a new profile review.

| Field | Mapping and rejection boundary |
| --- | --- |
| source_account_ref | Exact binding match; 1–64 ASCII letters/digits/hyphen/underscore. One account per file. |
| entry_id | Stable source transaction ID, 1–64 of same characters. Identity survives overlapping exports. Missing IDs are rejected; do not synthesize IDs from row order or a content hash. Synthetic fixture authors maintain IDs across revisions. |
| booked_on | Actual valid date, within caller-declared statement interval `[starts_on, ends_before)`. Source-booked evidence only; accounting date is later explicit review. |
| currency | Three uppercase ISO letters, matches binding currency, unambiguous valid installed version on date. One currency per file; other bindings can use EUR/JPY/KWD. |
| direction | Exactly `inflow` or `outflow` from the account holder's deposit balance perspective. Suggested deposit-account leg: inflow debit, outflow credit. Neither determines an offset account, expense category or a posted journal. |
| amount | Positive unsigned decimal string matching `[0-9]+(\.[0-9]+)?`; up to 29 integer/9 fractional digits, significant fractional digits no finer than currency minor unit. No sign, exponent, grouping, zero, NaN or float conversion; excess precision fails without rounding. Preserve original lexeme; canonical equality strips leading/trailing insignificant zeros. |
| description | 1–500 printable ASCII characters; preserve exact decoded text, render escaped inert text. Never execute formulas/HTML or export to a spreadsheet in this slice. Description differences are material to replay equality. |

No functional amount/rate is invented. E.g. EUR `10.00` remains original EUR 10;
a later USD journal requires explicit reviewed valuation. JPY `125` and KWD
`1.005` are valid with 0/3-decimal metadata; JPY `1.1` and USD `12.501` fail.
The suggestion above applies only to asset deposits, not bank-side debit/credit
labels or liability accounts. Closed book periods do not destroy source evidence;
later posting independently applies period/account/FX policy.

## Durable identity and preview/import transitions

Request metadata includes binding revision, profile version, synthetic namespace,
statement reference (1–64 characters), interval, and a server-computed SHA-256 of
exact bytes. A scoped file occurrence records these plus server object ID and
upload actor/time; filename is display-only. Same exact file and metadata returns
its existing authorized occurrence. Same statement identity with changed content
is a visible revision conflict, never a silent replacement. Byte identity is not
row identity: line endings/quoting/reordering may produce different files.

The durable row uniqueness key is `(entity_id, stable_source_account_id, entry_id)`.
Keep decoded immutable fields and canonical exact amount for equality, plus original
file/record ordinals and lexemes. Hashes accelerate lookup; compare full canonical
fields, never accept hash-only equality. A row key with equal content links the new
file occurrence to the existing draft; changed content is a source-correction
conflict. Repeated entry IDs **within one file** reject the file even if equal.
Different IDs with identical date/description/amount remain distinct purchases.
Identity claims from this template are trusted only as synthetic input, not proof
that a bank issued IDs. A live adapter must establish stable ID semantics first.

1. `received → validating → preview_ready | rejected`: parse the entire bounded
   file; preview lists each new/existing row and all bounded errors. Preview writes
   no drafts, accounting facts or posted entries. The durable upload/validation
   record survives retries; repeated validation is deterministic for pinned versions.
2. Preparation itself requires current read/import permission and the same entity/
   book/account binding; caller-supplied actor IDs never grant authority. Explicit
   import confirmation prepares a recoverable operation naming immutable
   preview ID/revision, binding revision, profile and content identity. New operation
   types require review of [command recovery](command-recovery.md); existing setup/
   post endpoints do not already implement import. Discovery never executes it.
   Proposed replay scope is principal + import operation + idempotency key; compare
   the complete validated command identity. Changed identity under the same key
   conflicts, and a different actor cannot replay another actor's receipt.
3. Execution rechecks principal/grants/binding/reference versions and row identities
   in one transaction; create **all** new drafts, file-row links, receipt and audit,
   or none. Changed preview/dedup state returns conflict requiring refreshed preview
   and explicit confirmation. Do not silently skip a newly discovered conflict or
   commit a valid prefix. Receipt replay resolves before mutable preview validation,
   while current authorization still applies.
4. `imported` reports created/existing counts and stable draft IDs. Timeout/unknown
   commit recovers the same operation; acknowledgement never imports again. Cancel
   before import creates no drafts; a commit winning cancellation remains imported.
   Draft classification/splits and later explicit posting are separate operations;
   every split retains source-row linkage. No posted journal is mutated or auto-created.

No background worker is introduced by this contract. Parsing is bounded and must
not retain a database transaction during object access. If asynchronous validation
is selected under #8/#7, its operation/outbox intent, fences and receipt follow
[background work](background-work.md); the existing #7 dependency is not waived.
Concrete SQL, object-storage coordination, HTTP commands and worker ownership need
separate accepted contracts before implementation.

## Security, provenance and retention

Proposed hard limits: 1 MiB exact bytes, 1,000 data records, 2,048 bytes per physical
record, seven fields and limits above; at most 100 displayed errors plus a truncated
flag/total count when determinable. First structural failure may end parsing; never
claim unparsed rows valid. Stable error code, ordinal and field name only: no raw
financial values, filesystem paths, object URLs or another scope's identities in
logs/errors. No archives, remote URL fetching, macros or executable parsing. Exceeding
any bound rejects the file before draft writes; retries cannot expand limits.

Raw bytes use generated private object IDs outside public web assets, encrypted
storage and scoped retrieval; no public links, analytics payloads or browser
persistence. Preserve uploader/time, binding/format/parser/currency versions,
byte digest, record ordinals, original values, canonical values, immutable preview,
import operation/receipt and draft links. Object publication must complete before
preview availability; failure leaves no draft and a discoverable failed upload.
Orphan objects are privately quarantined and swept, never interpreted as imports.

Synthetic-only proposed retention: rejected/orphan bytes at most 24 hours;
unconfirmed previews/raw bytes seven days; imported raw bytes 30 days. Keep
normalized imported evidence, dedup keys and receipts for the synthetic dataset's
lifetime, even after raw deletion; clearly mark raw evidence unavailable. Deletion
cannot free an ID for accidental reimport. Explicit dataset purge retires its
namespace permanently and erases associated synthetic evidence; it is not selective
bookkeeping-history deletion. No tax/statutory retention duration is inferred.
Live retention, backup deletion, legal holds, owner/audience and incident handling
remain approval gates; raw expiry alone cannot authorize live records.

## Independent examples and future acceptance

All names below are **planned**, not implemented/passing tests. Expected outcomes
are hand-authored here, independent of parser/draft code. Each fixture verifies
that ledger journals/lines and `journal.posted` outbox intent do not change merely
from preview/import. Import-specific audit records, receipts, drafts and file-row
links change only as specified; those audit writes must not be mistaken for
accounting postings.

| Named future evidence | Required expected result |
| --- | --- |
| `TestStatementProfile` | Exact three-row example creates preview with three new rows; quoted comma preserved. Invalid header/encoding/date/sign/scale and missing entry ID reject, zero drafts. |
| `TestStatementBinding` | Allowed binding succeeds; mixed accounts, foreign entity, missing/revoked grant, disabled/deleted principal and changed binding fail without revealing source content. |
| `TestStatementMoney` | Example total USD inflow100/outflow25, net increase75; both 12.50 purchases remain separate. EUR10 preserved without USD rate; JPY/KWD cases above exact. Totals never combine currencies. |
| `TestStatementReplay` | First confirmed import creates3 drafts; identical replay creates0 additional; reserialized same IDs in a new statement creates0 and3 links. Overlap P002 plus new P003 creates1 and links1. Same P001 changed12.51 conflicts; no silent edit. |
| `TestStatementAtomicity` | Valid rows1/3 with invalid row2 create0 drafts; injected failure after each draft/link/audit/receipt write rolls all back. Retry creates exactly3. Two concurrent equal imports have at most3 total drafts. |
| `TestStatementPreviewRace` | Membership revocation, changed binding/reference metadata or concurrent dedup change between preview and execution prevents stale new work; authorized known receipt replay has no additional effects. |
| `TestStatementRecovery` | Lost commit response recovers same draft IDs; cancel/import race has one terminal outcome; lost acknowledgement cannot create a second import. |
| `TestStatementBoundsAndRetention` | Each limit+1 rejects; diagnostics redact contents; text is inert; private retrieval denies wrong scope; raw expiry retains dedup identity, preview expiry blocks first execution, dataset purge prevents namespace reuse. |

## Handoff and blockers

Owner must accept/revise the internal CSV exception, binding and stable source-ID
requirements, permission, limits and synthetic retention. #17 remains a real camt
integration blocker, not a prerequisite proof magically supplied by this CSV.
#8 needs reviewed draft/provenance SQL and API/recovery contracts, storage consistency
and cleanup design, applicable #7 behavior, then these fixtures. This contract
neither closes #8 nor authorizes live data. Review evidence and actual checks are
in the [living runbook](../runbooks/statement-import.md).
