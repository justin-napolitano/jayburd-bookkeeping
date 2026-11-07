# Collection labeling SQL appendix —304

## Status and evidence

Bounded implementation proposal after owner approval238/242 and merged239/447.
GitHub304 owns task state; this appendix owns SQL43's physical and concurrency
contract. No provider, deployment, economic spending recognition or journal action.

Reviewed2026-11-06: approved [private categorization](private-categorization.md),
[registry42](private-registry.md), [command recovery](command-recovery.md),
[fixed cases](../reviews/classification-433.md), database-foundations and bibliography.
Official PostgreSQL17 [row locks and deadlocks](https://www.postgresql.org/docs/17/explicit-locking.html#LOCKING-ROWS),
[repeatable read/serializable](https://www.postgresql.org/docs/17/transaction-iso.html),
and [recursive CTE evaluation](https://www.postgresql.org/docs/17/queries-with.html#QUERIES-WITH-RECURSIVE)
reviewed in relevant full sections. These establish row locking/snapshot errors,
whole-transaction retry and recursive traversal mechanisms; the bounded hierarchy,
private authority, command semantics and no-journal invariants are project choices.
Alternative lazy-created revision state rejected: primed RR could miss it.
Alternative book-bound ledger commands rejected: they encode different authority
and result scopes. Existing ledger command functions/tables remain untouched.

## Physical records

One precreated collection label state per workspace, migration-backfilled and
created by AFTER INSERT on42 workspace. It holds taxonomy revision1, decision
revision0 and fixed ruleset/map revision0. Immutable taxonomy editions contain full
ordered topic sets; stable perworkspace IDs survive rename/retirement. New editions
must retain all prior IDs; active parent references, no cycles, depth8, names1–80
Unicode codepoints/no control/blank, case-sensitive unique sibling names,
500active/1000total. Retired IDs cannot receive new labels. Immutable original
editions remain available to historical decisions, never erased.

Immutable preview owns actor/workspace, exact canonical action and composite
selection/revisions, six pins, safe review rows, creation/expiry (15minutes).
Label decisions append workspace/source/observation/source revision, action,
taxonomy edition, command identity and decision revision; latest is derived,
not destructive overwrite. Source correction makes prior decision stale.
Prepared command owns immutable preview/key, prepared/committed/cancelled state,
terminal receipt/timestamp and explicit acknowledgement. At most one unacknowledged
command per workspace; no automatic execute/ack on discovery. Audit appends
preview/prepare/execute/cancel/acknowledge provenance. All new history is immutable.

## Authority and locks

Runtime executes only reviewed SECURITY DEFINER entrypoints; fixed
pg_catalog,pg_temp search path. No public/helper/table access. Actual API login
required by42; explicit current principal-private read or classify capability,
never entity membership in its place. Queries return source facts under private
capability. Native assignment is unassigned/unposted. Legacy assignment and journal
projection require independent current active entity/read membership/book, locking
existing rows before workspace; denied metadata is unavailable, never unassigned.
No provider IDs, credentials, raw payloads or unauthorized book IDs/names projected.

Lock order: principal, ordered legacy entity/membership/book authority rows,
workspace, label state,42 Item/source/legacy source rows, immutable preview and
command. All writes SERIALIZABLE; reads RR orSERIALIZABLE, bounded locks/timeouts
and whole-transaction retry40001/40P01. Lock ALL present authority rows without
active predicates before evaluating current access. Changed rows after primed RR
must abort. Previously absent grants remain conservatively unavailable until fresh
read.42 workspace current-row lock detects changed source admission inventory.
No source storage or lifecycle path takes our label state before workspace.

Review source digest combines42's ordered full source snapshot with safe opaque
ordered authorized legacy book/review/assignment revision evidence; it is equality
metadata, never authority. Snapshot includes authority_revision,source_snapshot,
taxonomy_revision,ruleset_revision,map_revision,decision_revision. Query page1,
subsequent pages, preview and initial execute compare all pins. Label writes
advance decision revision; taxonomy writes advance taxonomy revision. Inert rule/map
pins remain0 until real integration; no fabricated provider suggestions.

## Recovery and lifecycle

Preview selects1–100 exact composite IDs/revisions for topic or explicit
Uncategorized; taxonomy action has an empty selection and full replacement topic
list. Preview creates no decision or journal. Prepare stores the exact preview
under stable key; equal retry returns same command, changed key/body conflicts.
Expiry/pins checked for preview/prepare/initial execute. Discovery/cancel/ack and
terminal replay require current private capability/current selected sources but
not old data/taxonomy pins, so stale work remains explicitly recoverable. Initial
execute validates every member and action before any decision; all writes and
terminal receipt commit atomically. Execute/cancel serialize and return whichever
terminal state already won; ack only terminal, never changes terminal outcome.
A changed source revision preserves history and shows stale owner label; removed
facts preserve original quantity and never generate a new journal.

SQLSTATEs reuse42: P2800 malformed; P2801 denied/notfound; P2802 immutable body,
pending gate or invalid transition conflict; P2803 stale pins/expiry; P2805 bounds.
Fixed diagnostics contain no private values. New down refuses retained domain,
preview, decision, command or audit records; empty migration roundtrip supported.

## Standards and required evidence

Private internal analytic profile, no suitable ISO20022 financial exchange.
Preserve ISO4217 dated identity/minor units exactly, no arithmetic/FX. No audit
export or ISO21378 conformance claim; reuse existing reviewed standards limits.
High disclosure risk addressed by current composite authority before count/projection,
high tampering by immutable editions/previews/decisions and atomic commands,
repudiation by current-authority replay/audit, DoS by finite taxonomy/page/selection.

Named proofs planned: principal denial, full3000pages/count/bothdateorders/filters,
hierarchy/cycle/retirement/bounds, taxonomy/source/decision snapshot conflicts,
exact100bulk atomicity and foreign member rollback, primedRR taxonomy/grant races,
execute/cancel and lost-reply recovery, source-corrected override stale,
posted retag unchanged original source/journal/outbox, restrictive function/table
privileges and emptydown/up plus retained refusal. Planned names are not evidence
until executed and recorded in the marked runbook.

## Frozen operation signatures

All return jsonb, only these six functions granted to bookkeeping_api:

- banking.get_collection_taxonomy(actor uuid,w uuid)
- banking.query_collection_review(actor uuid,w uuid,raw text)
- banking.preview_collection_labels(actor uuid,w uuid,raw text)
- banking.prepare_collection_label_command(actor uuid,w uuid,raw text)
- banking.pending_collection_label_command(actor uuid,w uuid)
- banking.transition_collection_label_command(actor uuid,w uuid,id uuid,operation text,raw text)

Actor is verified by Go, never trusted from JSON. Raw JSON has recursively unique
keys and closed nested objects. Read catalog has taxonomy revision/current topics.
Query uses the shared305 review profile. Preview has workspace,snapshot,selection,
action. Prepare body is exactly {preview_id,prepare_key}; transition body is exactly {}
with explicit execute/cancel/acknowledge operation.
Pending wrapper returns command:null or the same safe immutable saved command.
Historical command rendering additionally rechecks book-read authority for any
stored assigned/book labels; private source capability alone cannot disclose that
old metadata after a book grant is revoked.

A hidden projection revision on the precreated state row detects source admission
and authority/projection insertions absent from a primed snapshot. Additive43
AFTER triggers touch that fence for private source/Item changes and relevant
legacy entity/membership/book/review/assignment changes. Acquire that state row
AFTER authority/workspace/source rows, matching trigger order. The digest binds
this fence and full ordered42 source digest. Root froze this exact protocol beforeDDL under lease65; transition identity is
commandID plus currently authorized workspace, never a changed request body.

Root frozen wire: catalog profile private-collection-catalog-v1 plus workspace_id,
authority_revision,taxonomy_revision,ruleset_revision,map_revision, complete topics
(topic_id,name,parent_id,status) and complete sources(source_id,name). Query raw
is305 Query minus workspace_id including profile. Preview raw is PreviewRequest
minus workspace_id; response profile private-collection-preview-v1 plus all
persisted Preview fields. Prepare/pending/transition return profile
private-collection-command-v1, workspace_id and command (null only pending).
Request byte bounds query8192,preview524288,prepare4096,transition1024; all
responses at most1MiB. Taxonomy retirement is terminal.

## Implemented safe coverage and legacy unit profile

Partial describes source-history coverage only: true for any currently selected
visible native source not HISTORICAL_UPDATE_COMPLETE, including empty NOT_READY or
INITIAL_UPDATE_COMPLETE; legacy history remains conservatively partial. Denied
book metadata is independent and never reinterpreted as unassigned.

Legacy original facts whose original amount lexeme exceeds the dated minor-unit
scale are projected source_state held (removed remains removed), with held reason
invalid_precision and null interpreted amount/direction/currency fields. Unsupported
or out-of-date unit metadata uses unsupported_currency. Existing evidence,
precision exceptions and journals remain immutable; no rounding or source rewrite.
The initial command/preview displays this safe projection, not a fabricated zero.

Command inserts/transitions advance separate hidden command_revision on the same
precreated state row. This detects primed snapshot misses of pending commands,
while excluding that revision from the source digest so preparing a preview does
not invalidate its own execute pins. All14 named SQL proof groups and exact
observations are recorded in the marked runbook; remaining HTTP/UI proof is root-owned.

Read-only pending discovery: after current read authorization, command:null may be
returned only if no current unacknowledged command exists. A present saved command
still requires current classify capability and source/book reauthorization before
any payload disclosure. This preserves idle read-only browsing while keeping
inaccessible saved intent closed; no implicit clearing or auto-acknowledgement.

## Shared principal interactive reservation —root review follow-up

The collection and existing book command engines share one database exclusion:
one unacknowledged reviewed command per principal across both engines. The prior
browser coordinator alone could not enforce this across independently authenticated
tabs. Existing command recovery already locks the principal FOR UPDATE;43 now
acquires that same strongest principal lock FIRST for private writes, before
legacy scope/workspace/source/state locks. Read-only projections retain FOR SHARE.

New ledger.interactive_command_reservation has principal_id as nondeferrable
primary key and exactly one typed reference: legacy_command_id or the private
workspace_id/label_command_id pair. Both command FKs are deferrable so a BEFORE
INSERT trigger can claim the identity before the referenced command insert ends.
All API/helper/table access is revoked; the owned trigger is the only product
claim/release boundary. Existing ledger command rows and journal code are unchanged.
The gate covers these two interactive command engines; existing direct API and
worker idempotency boundaries keep their reviewed scope.

43 holds an admission lock on ledger.interactive_command and backfills every old
unacknowledged command before installing both command-table triggers. A BEFORE
INSERT attempts the shared current unique index last. Visible contention returns
fixed existing P2112 for old book preparation or P2802 for collection preparation.
An invisible conflict under a primed SERIALIZABLE snapshot aborts40001 and the
whole transaction retries; no stale EXISTS-only authority test. Current same-key
replay occurs before a new claim, preserving exact historical recovery while
another engine owns newly prepared intent. Execute and cancel retain the claim.
Only acknowledged_at null→nonnull releases its exact principal+engine+command.
Existing inaccessible book recovery uses that same acknowledgement path and keeps
its immutable audit. No source/label/journal data is erased by releasing a claim.

Release must delete exactly one expected reservation. If a transaction primed
before43 cannot see the newly backfilled claim, zero deleted rows raise fixed40001
retry_command_acknowledgement; acknowledgement cannot commit while leaving a
permanent invisible reservation. Already acknowledged replays do not UPDATE and
remain idempotent. Down refuses any retained reservation, including one created
solely from an old book command; after explicit acknowledgement, empty43 down can
preserve old acknowledged command/audit history while removing the new foundation.

Reviewed PostgreSQL17 [INSERT ON CONFLICT/RETURNING](https://www.postgresql.org/docs/17/sql-insert.html),
[unique/FK constraints](https://www.postgresql.org/docs/17/ddl-constraints.html) and
[isolation conflict/retry](https://www.postgresql.org/docs/17/transaction-iso.html)
relevant full sections2026-11-07. These establish unique arbitration, returned
inserted rows and deferred references; the cross-engine principal reservation,
fixed busy responses and acknowledgement-only release are project policy. A
precreated mutable principal pointer was considered; a shared unique claim also
arbitrates invisible-row insertions without modifying existing principal/history
rows or old migration source. Actual two-winner and upgrade schedules are required
beyond that source interpretation.
