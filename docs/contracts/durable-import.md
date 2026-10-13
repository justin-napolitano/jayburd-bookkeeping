# Durable source import and review contract

## Decision and implementation boundary

**Proposed physical/API contract for review**, 2026-10-13, owner project maintainer;
[#8](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/8), with
[#61](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/61) and
[#70](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/70).
No DDL, endpoints, upload storage, review queue or source posting is implemented
by this document. First implementation remains synthetic deposit-account CSV.
The implemented [decoder](statement-decoder.md) owns byte/field semantics;
this contract owns proposed durable identity, transactions and review boundaries.
The [provider contract](provider-ingestion.md) owns future acquisition, consent,
fences and complete-set publication. It must not call the CSV upload endpoints.

Store immutable evidence separately from mutable review projections and posted
journals. Import confirmation creates uncategorized review items, not journals.
Explicit draft review and explicit posting are separate operations. This refines
the earlier phrase “import drafts”: an uncategorized review item is not yet a
valid ledger posting request. Standards, permission, retention and physical-design
review below must be accepted before the corresponding implementation.

## Identity and scope

Stable source account is a server-issued UUID for one real-world deposit account
within an entity, bound to exactly one book/account in this first profile. No
rebinding or namespace reset endpoint. A second adapter for the same account
must select that UUID and undergo overlap review; creating another UUID to avoid
duplicate detection is prohibited. Ownership cannot be proved by name/mask/hash.
Binding approval records the actor and synthetic evidence reference; real source
ownership and duplicate-account detection remain provider qualification gates.

An adapter namespace is a registered immutable identity tuple, not caller text:
CSV `(profile, synthetic_dataset_id, source_account_ref)`; later Plaid
`(provider, environment, item_id, provider_account_id)`. Its UUID is only a handle;
the full tuple has a unique constraint and case-sensitive equality. A new Item
creates a different namespace and requires overlap review; token/run/lease/parser
versions never change transaction identity. Retired datasets/namespaces cannot
be reactivated or reused.

Observation business key is `(entity_id, source_account_id, namespace_id,
external_entry_id)`, extending the earlier CSV three-part proposal explicitly.
CSV IDs obey its 64-character grammar; future provider IDs retain their adapter
grammar. Repeated byte uploads and repeated observations are different cases:

| Input | Required result |
| --- | --- |
| Same statement reference, metadata and exact bytes | Return the existing authorized batch; no new evidence or review item. |
| Same statement reference, changed bytes/metadata | `statement_identity_conflict`; no replacement. |
| New statement reference, same IDs and canonical fields | New file provenance links to existing observation revisions; no new review items. |
| Same ID, changed material fields | Whole CSV batch held as `rejected` with `source_correction_required`; existing observation unchanged. Future explicit correction intake can append a revision. |
| Different IDs, identical date/amount/description | Two observations and two review items. Never deduplicate by similarity. |
| CSV and Plaid IDs for a suspected shared purchase | Preserve both observations; require explicit overlap decision before the newly overlapping evidence can produce a journal. |

Canonical equality compares all decoded fields, canonical exact amount, currency
version and booked date. Original amount lexemes remain evidence, so `0012.500`
and `12.50` can compare equal without erasing original input. SHA-256 identifies
exact bytes and assists integrity checks; it is neither row identity nor a
substitute for full canonical equality. Duplicate IDs inside one file still
reject the entire file under the decoder contract.

## Proposed SQL dictionary

Schema/naming follow [ADR 0004](../decisions/0004-schema-ownership.md). Below,
`scope` means `entity_id uuid, book_id uuid`, present in every book-owned PK/FK;
IDs are server-generated UUIDs, `rev` is nonnegative bigint serialized as a
canonical decimal string over HTTP, and `instant` is timestamptz. Columns are
NOT NULL unless `?`; dates use the decoder's Gregorian bounds. All referenced
history uses RESTRICT/NO ACTION, never cascading deletion. Text is bounded by
the selected profile; metadata JSON cannot hide financially material fields.

| Table / grain | Columns beyond scope and scoped primary key | Constraints / lifecycle |
| --- | --- | --- |
| `banking.source_account` / stable account | `source_account_id`, `ledger_account_id uuid`, `currency_version_id uuid`, `binding_revision rev`, `source_revision rev`, `state text`, `created_by uuid`, `created_at instant`, `binding_evidence_ref uuid` | Entity/source UUID unique independently of book; same-scope ledger-account FK; positive binding revision; state active/paused/retired. Binding accepts only active monetary asset deposit mapping. Currency/version/date and account category checked under locks, not cross-table CHECK. |
| `banking.source_namespace` / registered adapter identity | `namespace_id`, `source_account_id uuid`, `adapter text`, `identity jsonb`, `state text`, `created_by uuid`, `created_at instant` | Exact allowlisted identity keys/types; unique entity/adapter/identity with full tuple equality. Composite FK to source account; immutable identity/account; active/retired; CSV profile only initially. |
| `banking.import_batch` / statement occurrence | `batch_id`, `source_account_id uuid`, `namespace_id uuid`, `statement_ref text`, `binding_revision rev`, `profile text`, `parser_version text`, `starts_on date`, `ends_before date`, `object_id uuid`, `byte_sha256 bytea?`, `byte_count integer?`, `state text`, `created_by uuid`, `created_at instant`, `expires_at instant`, `raw_deleted_at instant?` | Unique namespace/statement_ref within scope; namespace/source composite FK; 32-byte digest and 1..1048576 bytes required after publication. Immutable identity/metadata; starts<end; lifecycle below. Object ID generated before upload, never URL/path from caller. |
| `banking.import_preview` / immutable validation result | `preview_id`, `batch_id uuid`, `preview_revision rev`, `source_revision rev`, `binding_revision rev`, `rows jsonb`, `diagnostics jsonb`, `error_count integer`, `truncated boolean`, `created_at instant`, `expires_at instant` | Unique batch/preview_revision; 1..1000 decoded rows on success, zero usable rows on rejection; rows retain decoder values/ordinals and new/existing classification plus referenced observation/revision. At most100 redacted diagnostics. Max 2 MiB serialized preview, no arbitrary JSON keys. |
| `banking.source_observation` / adapter transaction identity | `observation_id`, `source_account_id uuid`, `namespace_id uuid`, `external_entry_id text`, `current_revision rev`, `created_at instant` | Unique scoped source/namespace/external ID, and entity/source/namespace/external ID. Composite FK enforces namespace belongs to source; current revision references an existing scoped observation revision via deferred FK. Identity never changes or frees for reuse. |
| `banking.observation_revision` / immutable source version | `observation_id, revision rev`, `source_state text`, `booked_date date?`, `currency_version_id uuid?`, `direction text?`, `amount_lexeme text?`, `amount numeric(38,9)?`, `description text?`, `previous_revision rev?`, `recorded_at instant` | Composite PK scope/observation/revision; previous revision same observation. CSV requires booked/date/currency/direction/positive amount/description. source_state booked for CSV; direction inflow/outflow. Finite exact lexical validation precedes typmod; no rounding. Reserved future pending/removed revisions require separately reviewed profile; removed rows retain prior evidence and cannot invent missing values. |
| `banking.import_row` / provenance occurrence | `batch_id, record_ordinal integer`, `observation_id uuid`, `observation_revision rev`, `original_fields jsonb` | PK scope/batch/ordinal; ordinal 2..1001; seven exact decoded CSV fields plus amount lexeme only; immutable scoped FK to revision. Every imported row links, including duplicates from another file. |
| `banking.review_item` / bookkeeping event group | `review_id`, `source_account_id uuid`, `review_revision rev`, `state text`, `retired_into uuid?`, `created_at instant`, `updated_at instant` | Same-scope FK retired_into; only unposted groups can retire; retired groups cannot draft/post. State needs_review/overlap_hold/correction_hold/draft_ready/posted/ignored. One group can retain multiple adapter observations; status is not source state. Unresolved pending/removed evidence cannot produce a draft. |
| `banking.review_evidence` / historical group observation membership | `decision_id, observation_id`, `review_id uuid`, `linked_revision rev` | Immutable scoped PK decision/observation; FKs decision/group/revision; old memberships remain history. |
| `banking.review_assignment` / current observation group | `observation_id`, `review_id uuid`, `decision_id uuid` | PK scope/observation gives one current group. Scoped FK to review_evidence decision/observation and group; controlled reassignment only with a new matching decision. Source-account equality checked under locks. |
| `banking.review_decision` / immutable action | `decision_id`, `review_id uuid`, `expected_review_revision rev`, `kind text`, `actor_id uuid`, `reason text`, `body jsonb`, `created_at instant` | kind initialize/match/distinct/ignore/draft; exact versioned body, <=64 KiB; reason1..500 except generated initialize reason. Contains pinned observation/review revisions and evidence IDs; no credentials or raw object content. |
| `banking.review_draft` / immutable proposed posting | `draft_id`, `review_id uuid`, `review_revision rev`, `binding_revision rev`, `source_revision rev`, `book_revision rev`, `posting_body jsonb`, `created_by uuid`, `created_at instant` | Complete existing posting schema <=1 MiB; source deposit leg is exact, remaining lines explicitly chosen; unique review/review_revision. No journal exists at creation. Revisions make old drafts stale without rewriting them. |
| `banking.review_posting` / immutable posted linkage | `review_id`, `draft_id uuid`, `command_id uuid`, `journal_id uuid`, `posted_at instant` | Unique scoped review/draft/command/journal as appropriate: one initial journal per event group. All same-scope FKs. Corrections link separately reviewed reversals/adjustments; initial link cannot be overwritten. |

Initialization decisions and normalized observations are written only on confirmed
import, not preview. A repeated-row import leaves the existing group/state alone.
Existing generic audit and operation receipts are used only after a reviewed
operation-code/shape extension; do not assume their current allowlists accept
imports. No speculative event is added to `journal.posted`.

## Permissions and owned functions

Propose new entity permission `import`, plus source-account-specific grants
`banking.source_permission(scope,source_account_id,principal_id,capability,
revoked_at?)`, PK including capability. Capabilities bind/import/review/read;
bind permission is assigned by a separately reviewed operator function, never
self-granted by uploading. Existing setup does not grant these permissions.
Import requires current entity read+import and source import; retrieval requires
entity read and source read; decisions/drafts require read+prepare and source
review; source posting additionally requires post. Binding requires entity
read+import plus source bind. Namespace registration is controlled binding work,
not a free-form request parameter. Owner/operator provisioning is an open gate.

`bookkeeping_api` receives EXECUTE on narrow `banking` functions only, no table
DML/DDL. Objects/functions are owned by existing NOLOGIN `bookkeeping_owner`;
the deployment/migration login is not the runtime owner. Preserve the reviewed
migration role path and keep API identities outside owner membership. Revoke
PUBLIC and fix trusted search_path.
Proposed function family `(actor uuid, entity uuid, book uuid, raw_request text)`:
`create_import_batch`, `get_import_batch`, `publish_import_preview`,
`prepare_import`, `execute_import`, `cancel_import`, `acknowledge_import`,
`list_review`, `get_review`, `decide_review`, `create_review_draft` and
`prepare_source_post`. Internal upload publication additionally receives a trusted
object receipt, never a caller-supplied path. SQL revalidates strict unique-key
JSON and scoped identities; Go owns authenticated identity, bounded object I/O
and decoder invocation. Provider workers receive no CSV/API impersonation grant;
their later publication function needs its own fenced service authorization.

All mutations use SERIALIZABLE and existing principal/entity/membership/book lock
order, then source-account locks sorted by UUID, operation/batch/review locks,
and observation locks sorted by UUID. Source-grant changes take compatible locks.
Recheck current auth before receipt replay or returning evidence. Lock source
accounts even for disjoint namespaces to serialize dedup/overlap decisions.
Source changes increment source_revision; review changes increment review_revision.
Neither increments ledger book revision unless a journal actually posts. This is
the provider contract's explicit refinement of the older blanket book-revision
rule; future close/reconciliation evidence must pin both revision domains.

## Upload, import and recovery transactions

1. Register immutable batch metadata and generated object ID with actor-scoped
   idempotency receipt. State `received`; current auth applies on replay. PUT raw
   bytes to the owned API; it streams at most 1 MiB + sentinel to private staging with
   a 15 s deadline, outside SQL. No URL fetch, public upload URL or DB transaction
   spans object I/O. A 20 s total request deadline bounds this synchronous slice.
2. Server verifies stored bytes/digest and decodes against pinned binding/currency
   metadata. Short SQL publication rechecks auth/binding and object receipt,
   records `validating → preview_ready|rejected` with immutable preview. Publication
   failure leaves no visible preview; same PUT can retry/reconcile the same object
   and bytes. Different bytes conflict; compare retained bytes on ambiguous hash equality rather
   than trusting a caller digest. Orphans are private and swept. Expiry or
   cancellation winning publication prevents a late successful preview.
3. Explicit confirmation registers an `import` recoverable operation containing
   batch, preview/revision, binding/source revision and profile. Extend the existing
   one-unacknowledged-command gate across setup/post/import/source_post; discovery
   never executes. Preparation, execution, cancellation and acknowledgement use
   the same principal-scoped recovery semantics and must be reviewed together.
4. Execution reauthorizes, resolves an existing receipt before mutable checks,
   then verifies exact pinned revisions and every row identity. One transaction
   writes all new observations/revisions, all row links, new review groups and
   initialization decisions, source revision, audit and receipt; batch becomes
   `imported`. Any changed preview/dedup state returns 409 and requires refreshed
   preview plus explicit confirmation, not silent skipping or partial import.
5. Lost commit response is 503 with recovery of the same command. Cancel-before-
   commit creates no observations; commit-before-cancel returns committed. Explicit
   terminal acknowledgement releases the gate, never deletes history. Retry whole
   transactions at most 3 times within 5 s for serialization/deadlock only; do not
   retry object side effects inside the transaction. Unknown commit never means
   “create another key.” Expired/cancelled/rejected batches cannot first execute.

Batch state progression is received → validating → preview_ready|rejected;
preview_ready → imported|cancelled|expired. Received/validating may also expire or
cancel. `rejected` is terminal. A new statement_ref does not permit changed content under
an existing observation ID: it still rejects as source_correction_required. Only
a future separately reviewed correction-intake operation may admit that revision.
Refresh of an unexpired preview creates a new immutable preview_revision and
invalidates prepared stale work. Raw-object lifecycle is separate from batch
state; deletion never changes imported to expired.

## Proposed HTTP boundary

Prefix `P=/v1/entities/{entity}/books/{book}`. Authenticated bearer/private bridge,
strict duplicate/unknown-field rejection, no-store and safe RFC9457 errors follow
existing API contracts. Server owns all actors/IDs/counts/digests. Dates/UUIDs and
money retain existing string profiles. No routes are mounted by this proposal. These signatures are review boundaries,
not generated OpenAPI or an executable exhaustive validator; implementation must
add strict schemas and contract fixtures without silently broadening the profile.

| Request | Exact new body / response boundary |
| --- | --- |
| `POST P/import-batches`, Idempotency-Key | `{source_account_id,namespace_id,binding_revision,statement_ref,profile,starts_on,ends_before}` →201  Batch. Profile fixed to decoder profile; revisions strings; statement_ref follows CSV ID grammar. |
| `PUT P/import-batches/{batch}/content` | Raw bounded CSV, fixed content type; no multipart or query. →200  Batch after validation publication, including rejected outcome. Retry identical bytes only. |
| `GET P/import-batches/{batch}` | No body/query →200  Batch; no raw object URL or bytes. |
| `POST P/import-batches/{batch}/preview`, Idempotency-Key | `{expected_preview_revision}` →200  Batch with refreshed immutable preview; only unexpired preview_ready batches. |
| `POST /v1/commands`, Idempotency-Key | Proposed operation `import`, scoped body `{batch_id,preview_id,preview_revision,binding_revision,source_revision,profile}` →existing Command envelope; execute/cancel/acknowledge remain existing recovery routes after extension. Import result `{batch_id,created_count,existing_count,source_revision}`; paged rows expose stable IDs. |
| `GET P/import-batches/{batch}/rows` | Query `preview_revision`, optional `cursor`, `limit` 1..100 (default 50); →`{profile,batch_id,preview_revision,source_revision,rows,next_cursor}`. Rows `{record_ordinal,disposition,observation_id,observation_revision,review_id,fields}`; new IDs nullable before import; disposition new/existing. Fields exactly the decoder's original/canonical values. |
| `GET P/review-items` | Required source_account_id; optional state, limit 1..100 (default 50), cursor; →`{profile,source_revision,items,next_cursor}`. First slice requires source_account_id; no mixed-source global revision invented. |
| `GET P/review-items/{review}` | No body/query →Review with provenance and current pinned revisions, never raw credentials/object URL. |
| `POST P/review-items/{review}/decisions`, Idempotency-Key | `{expected_review_revision,source_revision,kind,reason,other_review_id,other_review_revision}`; kind match/distinct/ignore, other fields null for ignore. →Review. Match requires two same-source groups and both revisions. Distinct may name another group or use both other fields null for an explicit new-event attestation after overlap review. |
| `POST P/review-items/{review}/drafts`, Idempotency-Key | `{expected_review_revision,source_revision,binding_revision,expected_book_revision,posting_body}` →201  `{draft_id,review_id,review_revision,source_revision,binding_revision,book_revision,posting_body}`. This is draft creation only. |
| `POST /v1/commands`, Idempotency-Key | Proposed operation `source_post`, body `{review_id,draft_id,review_revision,source_revision,binding_revision,expected_book_revision}` →existing Command envelope. Execution atomically revalidates the stored draft/source and invokes owned posting logic, writes review_posting and marks posted with the journal/audit/receipt/outbox commit. Never call existing HTTP post followed by a separate link write. |

Batch is `{profile:"durable-import-v1",batch_id,source_account_id,namespace_id,
state,binding_revision,source_revision,preview_id,preview_revision,created_count,
existing_count,diagnostics,error_count,truncated,raw_available,created_at,expires_at}`;
unavailable preview/count fields are null, not guessed zero. Diagnostics retain
decoder's code/record/field only. Review is `{profile:"source-review-v1",review_id,
source_account_id,state,review_revision,source_revision,binding_revision,
evidence,draft_id,journal_id,blockers,created_at,updated_at}`. Evidence entries are
`{observation_id,observation_revision,namespace_id,adapter,external_entry_id,
source_state,booked_date,currency_version_id,direction,amount,description,
raw_available}`; values follow the source profile, no float. Max 100 evidence
entries/group; additional matches fail with review_limit rather than truncate.
List items omit evidence and contain summary currency/direction/amount only when
unambiguous; mixed/conflicting groups return null summaries and blockers.

Page cursors bind scope, complete filters, source/preview revision and last UUID
(batch rows use ordinal), <=2048 bytes with canonical opaque encoding. Changed
revision returns 409 `source_snapshot_changed`, no partially fresh list. Response
limit 2 MiB; diagnostics at most 100. All mutation JSON except full drafts <=64 KiB;
draft/request posting limit 1 MiB and existing 1000-line cap. SQL/request row limits
must reject before partial effects. No raw download endpoint in first slice.
Errors: 400 shape ; 401 identity ; 403 current permission ; 404 nonexistent/inaccessible
object uniformly ; 409 identity/revision/state/replay conflict ; 413 bytes ; 422 profile/
review limit ; 503 retryable/unknown commit. Never include source values or raw SQL.

## Overlap, corrections and approval

Registering an additional namespace puts newly acquired rows into overlap_hold
until explicit distinct/match decisions; similarity may rank candidates only.
Match is allowed only within one source account and compatible currency/direction/
amount/date evidence. It creates an immutable decision, moves unposted evidence
into the retained group under both group locks and retires the redundant projection
without deleting its history. A group with an existing journal is retained; two
already-posted groups cannot be merged through this endpoint and require correction
review. A distinct decision preserves two legitimate events. Contradictory evidence
stays correction_hold. No silent auto-match across adapters or new Items.

Merge appends review_decision and review_evidence rows, changes review_assignment
under both group locks, and sets the unposted donor's retired_into to the retained
group in the same transaction. All existing unposted drafts on either group become stale; review revisions
increment. When the retained group is already posted, preserve its posted state,
journal_id, original draft and immutable review_posting; only the donor draft
becomes stale. The new evidence/review revision grants no second posting authority. Retired groups remain readable with their original
history but cannot receive new assignments or commands. Match cannot target a
retired group or create a cycle; resolve and explicitly review its current group.
Distinct with no candidate is an explicit new-event attestation, not a conclusion
from a failed similarity search. Incorrect matching reversal remains a separate
reviewed compensating operation, never deletion.

Draft creation requires needs_review with no unresolved blockers. It checks exact
source deposit leg, explicit offset lines, accounting date and valuation, and
sets draft_ready. Foreign transaction currency requires explicit supported FX
references; never replace source units with functional currency or guess a rate.
Posting rechecks pinned source/review/binding/book revisions and permissions,
including prior match decisions, then atomically creates the initial journal/link.
Ignoring requires a reason and preserves source evidence. Reconsidering ignored
items needs a separately reviewed decision; deleting/reimporting cannot reset it.

Future provider modifications append revisions; source removal appends a tombstone.
Unposted draft approval becomes stale and correction_hold; an already-posted
group retains journal_id and exposes correction_hold with an exception. Existing
journals never change; explicit reversal/adjustment is separate accounting work.
The first CSV profile rejects source corrections instead of inventing provider
precedence. A manual journal made outside this workflow is not automatically
recognized as duplicate; manual-to-source linkage is a separate acceptance gate.

## Evidence storage and retention

Apply the [statement proposal's synthetic retention](statement-import.md#security-provenance-and-retention):
24 h rejected/orphan bytes, seven days unconfirmed evidence, 30 days imported raw.
Normalized revisions, receipts, retired identities and posting provenance persist
for dataset lifetime. Raw deletion records raw_deleted_at and keeps digest/metadata;
raw_available means confirmed retained object, not proof of current retrievability.
Private generated object identifiers, encrypted storage, authorization and no
logs/browser persistence remain mandatory. No raw public URL, third-party scan,
remote fetch or analytical payload. Metadata publication never precedes complete
object publication. Idempotent cleanup rechecks retention/hold before deleting,
records unavailable after confirmed deletion, and reconciles uncertain deletion;
it cannot delete a newly imported object's evidence using a stale preview expiry.
No cleanup runner or cloud bucket is introduced here. Live retention/legal holds,
backup erasure, source ownership and accountant cutover remain explicit gates.

## Standards, sources and alternatives

Internal durable-import-v1/source-review-v1 are not bank exchange messages; no ISO
20022 identifier/namespace is invented. The selected synthetic CSV exception is
already recorded for decoder scope; durable acquisition and later Plaid each need
their reviewed usage decision. ISO 4217 currency-version references apply;
RFC 3339 full dates/UTC timestamps profile ISO 8601 semantics. SI01–SI06 and the
[standards contract](standards.md) retain source versions/access limits. No IBAN,
BIC or LEI interpretation is added. ISO 21378:2019/21926:2026 audit mapping and camt
version/usage profile under #17 remain unresolved, not supplied by these APIs.

Primary sections reviewed 2026-10-13; references establish mechanisms, not approval:

| ID / exact source sections | Finding and project interpretation |
| --- | --- |
| DI01 — PostgreSQL17 [§5.5.1–5 keys/constraints](https://www.postgresql.org/docs/17/ddl-constraints.html) and [§13.2.3 serializable](https://www.postgresql.org/docs/17/transaction-iso.html) | Composite keys/FKs enforce referential scope; cross-table CHECK is unsuitable; serializable transactions can require complete retry. Inference: serialize import publication and use immutable revisions plus scoped uniqueness, not application-only dedup. |
| DI02 — OWASP [File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html), File Storage Location, User Permissions, Upload and Download Limits | Storage segregation, authorization and bounded uploads are layered controls. Inference: private staging plus separate SQL publication and explicit orphan cleanup. Numerical caps/retention are project proposals, not OWASP requirements. |
| DI03 — Plaid [Transactions API](https://plaid.com/docs/api/products/transactions/), transaction_id, pending and pending_transaction_id fields | IDs are case-sensitive; pending details can change; pending linkage is conditional. Inference: stable adapter identity and immutable revisions, no amount/name identity or assumed pending lineage. Mutable API docs reviewed only these fields; provider version/mapping remains the separate provider contract. |

These sections were read directly; no full ISO normative text or real provider
payload was accessed. Alternative hash-only dedup conflates identity and evidence;
similarity dedup loses identical purchases. Direct import-to-post removes explicit
accounting review. Storing files in PostgreSQL would simplify atomic bytes/metadata
but grows ledger backups and broadens runtime access; private object staging is
chosen provisionally, subject to tested publication/cleanup. Entire provider event
sourcing is unnecessary: append-only source revisions and current projections
preserve corrections without replacing the transactional ledger.

## Acceptance and implementation sequence

All tests below are **planned**, not executed evidence. Independent fixed fixtures
must inspect journal/line/book-revision/outbox counts after every import operation.

| Required named evidence | Fixed expected outcome |
| --- | --- |
| `TestDurableImportScope` | Wrong/revoked/disabled/mixed entity/source denies retrieval, replay and publication; runtime table DML and namespace self-registration fail. |
| `TestDurableImportIdentity` | Two USD 12.50 rows with different IDs create 2 items; repeated file creates 0; new-file overlap links 2; changed 12.51 under same ID rejects whole batch; forced equal hash cannot bypass full-field comparison. |
| `TestDurableImportAtomicity` | Inject each observation/link/review/audit/receipt failure:0 partial rows and0 journals; concurrent equal imports produce one observation per identity. |
| `TestDurableImportRecovery` | Lost commit returns same IDs/counts; cancel/execute and revoke/execute races follow serialization; stale preview requires explicit refresh/reconfirmation. |
| `TestDurableImportObjects` | Upload/publish/deletion crash boundaries, byte/record/deadline limits + 1, orphan/expiry races and redaction; raw expiry retains identities, receipts and normalized evidence. |
| `TestSourceReviewOverlap` | CSV/Plaid overlap requires explicit match; already-posted group retains one journal; two distinct identical purchases remain 2; contradictory/mixed-scope match denied. |
| `TestSourceDraftPosting` | Import and draft create0 journals; explicit source_post creates 1 linked journal atomically; replay adds 0; source/book/binding/review changes reject stale approval; lost commit recovers same linked result. |
| `TestSourceCorrectionHistory` | Pending/removed not eligible; modified posted evidence leaves original journal unchanged and raises hold; foreign amounts retain EUR/JPY/KWD units and explicit FX gates. |
| `TestSourceReviewPagination` | Exact scoped cursor, limits, stale revisions and denied actor return no mixed snapshot or leaked evidence. |

First review these physical/API choices, permission provisioning and object-store
ownership. Then implement binding + private
batch/preview + atomic import with synthetic tests; then review decisions/drafts
and recoverable source_post; then Plaid complete-set publication and qualification.
The journal-observer worker does not satisfy provider acquisition/cleanup gates.
Issue #8 remains open through its actual end-to-end acceptance. This contract does
not authorize live imports, cloud resources, automatic posting or accounting policy.
