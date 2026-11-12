# Source-derived journals and linked corrections — #332

## Current authority and historical scope

Reviewed2026-11-11 for486. The accepted2026-11-08 owner decision473 in
[artifact-backed accounting](artifact-backed-accounting.md) governs the current
public application and supersedes this design's former manual-posting/reversal
fallback. New postings require retained source evidence and the reviewed
`source_post` workflow. Direct journal creation, unbound reversal, new durable
`post` registration and execution of prepared legacy `post` commands are rejected.
Authorized historical reads/committed receipts and safe cancel/acknowledgement
remain available; manual registration replay does not revive that workflow.

Artifact-linked correction inspection, preparation and atomic execution below
remain a future design, not an available customer action. Existing SQL posting
and reversal primitives remain reusable internal mechanisms and isolated
synthetic proof subjects, not a public bypass or a correction workflow. Future
corrections require supporting evidence/approved reason and their own reviewed
implementation and end-to-end acceptance. See the current
[API authority](api.md#status-and-authority) and
[artifact delivery evidence](../runbooks/artifact-backed-accounting.md).

The dated inventory, worked examples and proposed interfaces below retain the
2026-10-25 design and its evidence limits. Historical issue sequencing is not a
current implementation-status source; GitHub owns live task state. No new
financial message, policy, mapping or standards-conformance claim is introduced.

## Historical decision and proposed implementation scope (2026-10-25)

Owner2026-10-25 approved bank-read-only access with ledger bookkeeping, retained
source revisions and reconcilable journals. Coordinator engineering design under
#332/#104; no SQL/API implementation, live accounting policy, security review or
activation approval is claimed. #333 implements corrections, #334 opt-in rules,
#335 UI and #336 independent end-to-end proof. The original design allowed initial personal
registration90 before the complete rule engine and proposed reuse of manual
posting/reversal. That application fallback is superseded by473; retained source
posting does not itself implement linked correction.

Bank observations are authoritative evidence of what the provider reports, with
immutable revisions and an explicit current pointer. The ledger is the durable
accounting record, including policy decisions and non-bank entries. Provider
categories/dates alone do not select recognition, accounts, FX or tax treatment.
No real book silently inherits dev-accrual-usd-v1. Reviewed per-book policy and
current prepare/post permissions determine eligible accounting effects.

## Historical source inventory and correction gap (main1bacecda)

At main1bacecda: migration11 banking.review_posting is immutable and keyed by
entity/book/review; it links one original draft/command/journal and cannot be
updated to a replacement. Migration12 banking.provider_posted_anchor preserves
the original posted observation/journal, and banking.provider_review_exception
records modified/removed source revisions. Subsequent precision/history functions
retain these anchors. Present exception rendering is bounded reference metadata,
not complete before/after financial inspection or correction resolution.

ledger.post_journal/reverse_journal use the existing balanced immutable posting
boundary, canonical receipts, audit/outbox, current authorization and book revision.
Reverse copies original historical money/rates; it does not reprice history.
ledger.interactive_command stores reviewed work before explicit execution and
terminal acknowledgement. These existing functions are reusable mechanisms;
calling two independent HTTP operations is not atomic linked correction.

Acquisition still never writes a journal or receives posting permission. Local
bookkeeping of authorized retained history need not stop merely because new
provider access expires or is paused. Source data access/review and book posting
permissions remain independently current; revoked access is not restored by
having a historical receipt. Provider fetch/retention authority is not bypassed.

## Proposed correction snapshot and logical interfaces

The #333 SQL/API review must freeze exact physical signatures/OpenAPI before DDL.
Logical boundaries are owned source-correction inspection, immutable reviewed
preparation, and same-command atomic execution/recovery; no arbitrary SQL selector
or caller actor. Expected SQL names are banking.get_source_correction,
banking.prepare_source_correction and banking.execute_source_correction, not
implemented guarantees. Public actor always comes from verified identity.

Inspection requires current book/source read; preparation adds current prepare
and source-review authority; execution adds current post. Current retained-source
policy must be checked explicitly rather than assuming active provider consent
is a prerequisite for every local ledger read or correction. Operator/superuser
access is outside runtime guarantees. No table DML or new implicit grant.

The persisted reviewed snapshot pins entity/book/source/namespace/observation,
original root journal, currently effective journal, expected correction-chain
version, exact original-posted and target source revisions, binding/source/review/
book revisions, exception identities, policy/rule version, action/reason, accounting
date, eligible category/ledger accounts and full normalized posting/valuation pins.
Currency versions, original amount lexemes and functional assertions remain exact
strings. Current bank state and current accounting state are separate fields.
Return bounded authorized structured facts, not a provider payload or credential.

First actions:

- replace: exact reversal of the effective ordinary journal plus one reviewed
  eligible replacement. No reversal of a reversal; the replacement becomes the
  next effective journal while the original root remains immutable.
- no_financial_effect: explicit reviewed evidence that changed metadata does not
  change accounting under the pinned policy. No journal, book revision or
  journal.posted intent; retain a resolution/audit receipt.
- reverse_only: explicit reviewed cancellation/removal under a permitted policy,
  confirming no replacement should be recorded. Exact historical reversal only;
  never infer this merely from a removed feed record or emit a zero-value journal.

Pending/unmatched replacement, unknown removal meaning, unmapped classification,
unsupported currency, missing FX/policy, zero replacement or closed period remain
actionable holds until a valid reviewed action exists. Missing capabilities stay
visibly unavailable. Old period data is not rewritten; a later correction date
needs the selected policy, not an automatic date/rounding plug.

## Durable operation, locking and atomicity

Register the exact reviewed correction on the existing principal interactive gate
before execution; explicit confirmation and terminal acknowledgement remain.
Registration/discovery/cancellation never post. A versioned deterministic rule
uses its separately reviewed durable operation under an approved current service
principal and scoped post authority; a webhook/acquisition wakeup is not that grant.

Serializable transaction, SQL-owned canonical full-body equality and whole-
transaction retry remain authoritative. First lock all participating current
principals in UUID order, then existing entity/membership/book order, source/
permission and namespace/observation/review, correction chain/command/receipt,
policy/period/accounts and effective journal in the reconciled existing order.
The exact lock table must be reviewed in #333 against current posting/import/
acquisition functions before implementation; never take a weaker lock first and
upgrade through a conflicting path. No provider/network/key work inside locks.

Reauthorize and compare all uncommitted expected pins before financial work. The
same transaction creates the original-value reversal, policy-valued replacement,
their existing journal receipts/audit/outbox, immutable correction resolution,
source-to-effective-journal links and committed command result. Either every
effect commits or none. Preserve each underlying journal's normal book-revision
increment: replace has two increments, not an invented single journal revision.
Do not expose a half-corrected report or acknowledge before COMMIT.

Preserve banking.review_posting, provider_posted_anchor and exception rows.
Append a scoped immutable resolution/correction chain with predecessor/root/
effective-journal/revision references. Unresolved status is derived from exception
plus resolution evidence; a counter or latest100 projection cannot silently drop
older unresolved work. Validate unique correction targets and no cross-book or
namespace chain, cycles, duplicate active effects or repeat reversal. Exact table/
constraint/grant definitions belong to the forward #333 migration review; no
migration version is reserved by this document.

Same committed command/body under current operation permission returns its exact
original result without revalidating old rates/source/current pins or new writes.
A newer bank revision does not invalidate knowledge of an already committed
correction. It creates new unresolved work. Changed body/same key conflicts; a
new key/stale snapshot cannot become another reversal. Lost result means inspect/
reconcile the same command, never invent a new transaction. If source60 arrives
before an uncommitted55 correction executes, stale pins deny; after55 committed,
read its receipt and prepare a new55->60 operation targeting effective journal55.

## Independent worked expectations

Fictional expense scenarios, same open period and explicitly eligible accounts/
policy, no opening-balance invention. These are fixed oracle values, not results
generated by implementation. Dr expense / Cr bank; reversals swap exact sides.

| Stage | New journals | Retained journal count | Net expense/bank credit |
| --- | --- | --- | --- |
| Booked source USD50, reviewed original post | original50 |1|USD50|
| Source changes to55, before correction | none |1|USD50, unresolved discrepancy5|
| Confirm replace with reversal50 and replacement55 |2|3|USD55|
| Replay event/command/lost terminal reply |0|3|USD55|
| Later source60, confirmed effective55 reversal and replacement60 |2|5|USD60; original50 and first correction retained|
| Metadata-only later change under unchanged recognition |0|5|USD60, explicit no-effect resolution|

EUR example: original EUR10 at explicitly selected synthetic EUR/USD1.20 ->
functional USD12.00. Changed source EUR11 with the same eligible policy/rate ->
replacement USD13.20. Exact reversal retains EUR10/USD12.00 and its historical
rate identity. Three journals net EUR11/USD13.20; no aggregation of unlike units.
A different replacement FX/date requires a separately reviewed eligible pin;
missing rate holds. Synthetic rates never authorize a live market-rate policy.

## Source/standards evidence, alternatives and threat review

[Posting sources P01–P07](../research/posting-sources.md) record exact source
sections/depth/limits. Plaid settled changes are vendor feed facts; this restricted
correction protocol is our engineering choice. Existing proprietary financial
mapping, ISO4217 dated currencies and original units remain. This internal
operational command is not a registered ISO20022 exchange; ISO21378 audit relevance
retains abstract-only limits and no export/conformance claim. SQL/Go profile
compatibility and versioned normalized correction input must be fixed before DDL.

Alternatives: rewriting original journals destroys audit evidence; reconstructing
all journals as a bank-table view loses manual/policy/non-bank accounting; two HTTP
calls expose partial correction. Delta adjustments can be legitimate but are not
selected without their own reviewed operation. Full linked reversal/replacement
reuses existing historical valuation and balanced posting boundaries.

STRIDE: spoofed source/tenant/rule (high) -> verified/current roles plus all scoped
FK/pins; stale/tampered/replayed work (high) -> immutable review/body equality,
serializable locks and unique targets; partial effects (high) -> one transaction/
same-command recovery; disclosure (high) -> current scoped reads, no-store/memory-
only browser state/fixed errors/no financial logs; abusive automation (high) ->
explicit versioned rule grant, holds and kill switch. Human review before sensitive
code merge and real mandates remain required; this is not independent signoff.

Named proposed evidence: TestSourceJournalCorrection50To55,
TestSuccessiveSourceCorrectionsTargetEffectiveJournal,
TestCorrectionMetadataNoEffectAndRemovalHold,
TestCorrectionOriginalEURAndHistoricalValuation,
TestCorrectionAtomicFailureAndUnknownCommit,
TestCorrectionConcurrentSourceUpdateAndRevocation,
TestCorrectionReplayAfterLaterSourceRevision and
TestCorrectionNoProviderIOForAuthorizedHistoricalWork. Real restricted SQL/API/
browser fixtures and exact reports in #333/#335/#336; no admin impersonation,
mock-only money totals or generator self-oracle. Core/ledger/migration/web/full-
stack gates follow changed boundary. No new feature is implemented by this plan.
