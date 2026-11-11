<!-- stage-runbook: true -->
# Source-derived correction contract — #332

## Current authority and preserved record

Reviewed2026-11-11 under486. This is the historical332 design-stage record,
not evidence that a customer correction workflow has shipped. Current owner
policy473 requires [artifact-backed accounting](../contracts/artifact-backed-accounting.md):
new public postings use retained source review and `source_post`; standalone
manual posting and unbound reversal are rejected. Low-level SQL posting/reversal
and their synthetic proofs are reusable mechanisms, not a public alternative.
Keep authorized historical receipt/read recovery and safe cancel/acknowledgement;
never execute prepared legacy manual work to resolve a correction.

The [correction contract](../contracts/source-journal-corrections.md) retains the
proposed evidence-linked atomic correction design. Supporting evidence/approved
reason, exact interfaces, implementation and end-to-end acceptance remain required
before exposing correction actions. Current delivery/task state belongs to GitHub
and the [artifact delivery record](artifact-backed-accounting.md). The dated
stages and remaining-work statements below describe the original handoff and
are preserved as history, including its then-current toolchain and dependencies.

## Status and boundary

Historical design-stage scope,2026-10-25:

CORRECTION332-CONTRACT through2026-10-25T03:45Z; dedicated
design/332-source-journal-corrections from main1bacecda. Design only: no SQL/API/
runtime feature, migration reservation, source/provider/cloud/secret/live action.
Owner approved bank-read-only access with ledger bookkeeping and linked corrections.
Actual policy, security review and activation remain separately scoped.

## Stage ledger

| Stage | Evidence / limit |
| --- | --- |
| Resume | Current332/104/333/65 and main checked;308 plan lease released,317 revised CI live,312 dependency not counted complete |
| Inventory | Migrations2/11/12/29, current posting/command/source-sync contracts and ADR0005 reviewed; original review_posting/posted anchor immutable, exceptions metadata only |
| Decision | Reuse atomic SQL journal invariants with linked exact reversal/replacement and immutable resolution chain; current authority, same-command replay and effective-journal targeting before DDL |
| Sources | Official Plaid transaction states plus PG17 Serializable/SECURITY DEFINER selected sections refreshed; engineering inference and ISO/real-policy limits explicit |
| Fixed examples | Independently authored USD50->55->60 and metadata no-op, EUR10/12USD->EUR11/13.20USD; named future negative/concurrency/unknown-commit tests, not implemented proofs |
| Validation | make check/test/build PASS, cached local Go1.26.5/modfile only; tracked1.26.10 unchanged. Bibliography32743bytes, whitespace/link/stage validation; no new runtime/correction behavior is exercised by documentation gates |

## Current evidence

The following is preserved historical design evidence, not current implementation
acceptance. Current authority is stated above.

Contract defines financial effects and failure invariants. Existing SQL can post/
reverse, but does not yet atomically resolve a changed posted source and expose
the full correction chain.333 remains blocked on this contract actually merging
and its exact SQL/API review. No dummy runtime implementation or weakened oracle.

## Remaining work

Original remaining work at the historical handoff:

Source/document checks and required core/test/build; publish for contract review.
333 owns forward DDL/API/Go,334 owner-enabled rules,335 browser and336 independent
report/reconciliation proof. Registration90 prioritizes existing308/311–316 path
and does not require completion of all automation. No live policy/approval inferred.

## PR handoff

Historical handoff record:

Refs332/104/333/334/335/336/90. Design only, no migration or public API change.
Marked runbook in every implementation commit and staged check before each commit.
Source contracts/test names are not execution evidence; no live readiness claim.
