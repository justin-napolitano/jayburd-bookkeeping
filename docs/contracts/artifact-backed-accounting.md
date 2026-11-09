# Artifact-backed accounting

## Governing owner decision

Accepted2026-11-08: no standalone manual accounting entries. Every new application
posting starts from retained evidence and must link back to it. The owner approved
this requirement and implementation in conversation; GitHub issues own delivery
state. This supersedes the manual-journal application workflow, not immutable
history or the low-level synthetic ledger mathematics proof.

Artifact → review and categorize → confirm accounting treatment → posted entry.
An artifact's existence alone does not authorize posting, prove an economic event
or establish an accounting policy. Current actor/book/source permissions, source
revision, exact amounts/currencies, immutable draft, idempotency and explicit
prepare/execute/acknowledgement remain mandatory. Classification never posts.

## Supported first enforcement

Currently supported evidence is an imported bank/provider observation or imported
statement row retained by the existing banking source-review model. Only the
reviewed `source_post` command may create application postings. The source review,
draft, source/binding/book revisions and original units are validated by existing
SQL. Journal, review_posting link, source projection and receipt commit atomically
in banking.execute_source_post; replay cannot create a second journal.

The public application Handler rejects direct journal creation and unbound
reversal, and rejects new durable `post` commands. Before executing a saved
command, the artifact-aware service checks the current pending command within
the same serializable transaction; a prepared legacy `post` cannot execute.
Cancellation, acknowledgement, inaccessible recovery and already-committed
historical command receipt replay remain available under current authority.
Old manual registration-key replay itself is also rejected; receipt recovery
uses authorized pending/read/committed execute rather than reviving registration.
No historical
journal is deleted, rewritten or assigned invented evidence.

The general ledger primitive and its isolated mathematical/HTTP proof remain
internal test/library mechanisms. They are not exposed as a user-selectable
policy or alternative public Handler. This application restriction does not
claim that a privileged database owner is incapable of invoking internal SQL.
No migration or runtime-role widening is required for this first restriction;
1–47 stay immutable and48 remains unallocated. Existing low-level proof fixtures
must be clearly isolated from application acceptance.

## Product flow

Remove New journal and the blank manual form. Navigation exposes Transactions,
Accounts and existing Overview; Posted entries is a secondary bookkeeping view.
Users reach accounting review from an imported transaction and can follow its
source reference after posting. A prepared legacy manual request explains why
it cannot execute and offers its existing safe cancellation/acknowledgement.
Do not add a pretend Documents destination or upload button before ingestion exists.

## Documents, matching and corrections

Receipt, invoice, statement and supporting-document ingestion is tracked as the
next evidence capability. It requires retained original content or an authorized
versioned source, integrity hash, type/date/issuer metadata, access/retention,
durable extraction/review and immutable journal association. A filename, note,
arbitrary URL or caller-supplied artifact ID alone is insufficient evidence.

One economic event can have several artifacts: e.g. a receipt and matching bank
observation. Adding a supporting artifact must not automatically create a second
event or posting. Matching uses explicit reviewed identity/linkage, not amount
and date coincidence alone. A new artifact may initially remain unmatched.

Corrections require their own supporting evidence/approved reason, preserve the
original source and posting, and reference the original entry. Until a reviewed
artifact-linked correction workflow exists, the application rejects standalone
reversal rather than exposing an unsupported adjustment form. Existing historical
reversal evidence remains readable; nothing silently reverses or rewrites books.

## Standards and evidence assessment

This is an owner product policy and internal application command restriction,
not an assertion that ISO mandates this workflow. Existing ISO4217/exact original
currency mapping and the non-ISO provider/CSV usage profiles remain unchanged.
No new financial exchange or ISO20022 message/version is introduced. ISO21378:2019
audit-data collection is relevant to future document/audit export mapping; only
public abstract/metadata has been reviewed, so no field-level conformity is
claimed. Document ingestion/export remains blocked on its own reviewed mapping.
See standards.md, source-review-sql.md, posting-boundary.md and the existing source
register; implementation uses the actual source-post authorization/link contract.

Alternative considered: only hide New journal. Rejected because direct HTTP and
prepared legacy commands would still permit unlinked entries. A generalized
document registry is a dependent feature, not an excuse to retain manual entry
or label unsupported uploads as complete. Full database-wide evidence constraints
and privileged migration/import policy require a separately reviewed migration;
this slice enforces every public application posting path without changing old SQL.

## Named acceptance

### Threat review for this restriction

| Risk | Control and evidence |
| --- | --- |
| Forged manual operation or direct endpoint bypass, high | Public Handler has no selectable legacy profile; authenticated unsupported routes reject before posting. New-registration policy is closed; SQL still validates untouched known requests. ART01. |
| Saved legacy command or UUID spelling bypass, high | Inspect current pending inside the same SERIALIZABLE execution transaction; PostgreSQL canonicalizes accepted target UUID spellings before comparison. ART02/current-authority and retry tests. |
| Stale/foreign source, duplicate effect or amount substitution, high | Existing source_post SQL owns exact retained draft/revisions/current actor/source permissions and atomically stores its unique original source link/receipt. ART03 and existing source-post replay/precision/revocation proof. |
| Unknown commit reported as success or retried, high | Existing bounded SQL retry/unknown-outcome protocol remains; no result body until acknowledged commit. ART02 transaction failure injection. |
| Misleading artifact completeness/history rewrite, medium | No fake Documents destination or invented backfill evidence; keep historical entries readable and safe pending cancel/ack. ART04/ART05 and owner visual review. |

These controls are implementation requirements, not a compliance certification.
Security-sensitive public posting changes require independent source and owner
review before merge/release; the approved task authorizes implementation, not a
new live-bank/Production policy. No AppSec or other party is contacted automatically.

- ART-01 authenticated direct journal/reversal requests and new post commands
  fail without posting SQL; unauthenticated requests remain401.
- ART-02 a saved prepared manual command cannot execute through public transport;
  cancel/acknowledge and historical committed receipt recovery remain safe.
- ART-03 actual authenticated source review→draft→saved source_post→execute→ack
  creates one journal and original source link, with exact money and replay proof.
- ART-04 no New journal/manual form in desktop/mobile/standalone book navigation;
  legacy manual work has a clear blocked reason and safe exit.
- ART-05 duplicate receipt/bank matching and artifact-backed correction/document
  ingestion remain open until their own implemented end-to-end evidence exists.
