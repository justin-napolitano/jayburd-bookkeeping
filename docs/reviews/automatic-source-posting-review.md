# Automatic source-backed accounting review —334

## Authority and implemented baseline

Owner direction in334 (2026-11-12, reaffirmed2026-11-14) authorizes implementing
routine eligible posting without a per-entry confirmation. Policy configuration
can require an explicit review; automatically clicking or acknowledging an
interactive command is not the implementation. This source review does not
authorize real data, provider access, a migration against managed data or rollout.

Reviewed source baseline88fce16: AGENTS/index/current capability map,
artifact-backed-accounting, source-review-sql, source-journal-corrections,
posting-boundary and command-recovery contracts; posting source records P01–P04
and P08 and the independent corpus README. These are repository/code/design
sources, not a fresh review of normative accounting standards or vendor runtime
behavior. Existing source_post already supplies immutable drafts, exact posting,
current grants, balanced journals and atomic retained-source linkage. Automatic
analytic categories488 are separate and do not establish accounting treatment.
Artifact-linked correction333 remains unimplemented; do not claim its delivery.

## Contract acceptance conditions

| Boundary | Required behavior | Independent acceptance case |
| --- | --- | --- |
| One-time authority | A versioned reviewed mapping identifies book, source scope, eligible ledger accounts, treatment/direction, accounting-date rule, policy/valuation and effective interval. An analytic topic, merchant or provider label alone is insufficient. | An otherwise categorized source with no accounting mapping is held with zero journals and posting intents; enabling one exact eligible mapping permits only its scope. |
| Trusted actor | Public identity is verified by the existing authenticator. Policy author/current delegated actor and entity/source read, prepare, post and review authority are rechecked at execution and authorized replay. Acquisition/observer login gains no posting grant. | Cross-owner/source/book attempts and revocations after plan preparation deny or hold without disclosure/effects; a source-import capability cannot post. |
| Evidence and exact units | Use a current retained booked observation and existing source-to-journal construction; preserve its amount, original dated currency, date, source binding and immutable journal links. No arbitrary public posting body or inferred FX. | Independently fixed USD50 yields Dr selected expense50 / Cr bound bank50; JPY/KWD original precision remains exact; foreign-currency evidence without reviewed valuation produces fx_required and no invented zero/rate. |
| Replay and uniqueness | One initial economic posting per retained source-review identity survives new keys, retries, process restart, rule/policy editions and analytic category changes. A rule revision is not permission to post the same evidence again. | Commit then lose the reply; retry/restart/re-enable and advance rule edition. Exactly one journal, source link, audit result and journal-posted intent remain. |
| Changed posted evidence | Existing posted anchors remain immutable. Modified/removed posted observations require the explicit correction workflow; they never become another original journal. | Original50, source changes55: original50 remains, zero new journal, actionable correction hold. No claim that net55 is implemented by334. |
| Holds | Pending, removed, zero, overlapping, ambiguous, stale, unsupported currency, missing policy/valuation, unavailable accounts, source review conflicts and closed periods have finite explanatory reasons. Unknown reads are not zero/empty success. | One fixed fixture per hold reason with unchanged journal/line/outbox/book counts; resolve the actual missing configuration and demonstrate bounded reevaluation. |
| Transfers and splits | Transfers/card payments/loan splits need their own explicit reviewed mapping/evidence policy; a generic expense rule must not authorize them. | Same-amount opposite bank/card observations do not produce duplicate expense postings. Unsupported split/matching cases remain held. |
| Interactive recovery | Automatic effects have their own durable run/operation identity and never register, execute, cancel or acknowledge a human command. A conflicting existing reservation defers automatic work in the established lock order. | Prepared/unacknowledged human work remains unchanged while automation defers; recovery resumes the same automatic operation after release, without creating a fresh duplicate. |
| Atomicity and concurrency | Posting, source link/projection, immutable automatic receipt/audit and outbox commit in one transaction; known whole-transaction aborts retry boundedly. Unknown commit is reconciled through the same durable identity. | Fail at link/audit/outbox/commit, race two runners and revoke/change the mapping between phases. No orphan journal or partial source state and no duplicate effect. |
| Runtime and policy gates | Runtime default-off composition and explicitly enabled scoped policy are separate. Reads do not run accounting. Bounded work and cancellation/drain recover durable progress without a new service. | GET/list/status remain effect-free; absent runtime/policy never posts; controlled synthetic invocation proves zero per-entry confirmations and explicit disabled/blocked/pending/idle states. |
| Customer surface | Configure and review policy once, show counts/reasons and source-linked posted results. Routine eligible entries bypass the manual review workflow. Denied/stale/unknown outcomes remain distinct and recoverable. | Actual browser/API/restricted-SQL fixture configures once, imports or releases retained synthetic evidence, observes automatic posting and follows the journal back to its artifact. |

## Alternatives and standards scope

Reusing the atomic source-posting primitive is preferred to duplicating account,
valuation and balance logic. A separately durable automatic operation preserves
truthful consent and recovery; fake manual commands would misstate human review.
Stopping changed posted evidence at the existing correction boundary is preferred
to inventing reversal/replacement behavior in334 without333's independent proof.

This is an internal accounting-policy/execution contract, not an ISO20022 exchange
or ISO21378 audit export. Existing ISO4217 dated identifiers, exact decimal money
and source provenance remain mandatory. No new accounting framework, live policy,
FX convention, tax inference or ISO conformance is established by this review.

## Review and delivery status

The matrix is a concrete implementation/review checklist, not a report that its
new cases already passed. The core lane owns the SQL allocation and frozen
physical/wire contract. This independent lane owns contract/threat review and,
after agreement, a disjoint API/customer surface or independent public-contract
proof. Root delegated HTTP/server composition, generated contracts and the compact YAML customer surface to this lane; root retains final integration.
No SQL implementation or deployment belongs to this reviewer lane.

## Implemented boundary checkpoint

Default-off API and runner flags are independent; status runtime_enabled is server-owned, and policy_revision joins concurrent policy/status reads. Configuration import is a bounded inert YAML adapter into reviewed closed JSON, never a financial matcher. Auto-owned draft variant is explicitly versioned and inspect-only; manual draft keyset remains compatible. Machine-only terminal receipt observation is recovery metadata, not a public route or human posting acknowledgement. Actual SQL atomicity/authority/recovery proof remains owned by core and canonical integration.
