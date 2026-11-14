# Automatic source-backed accounting — 334

## Proposed implementation boundary (review required before SQL)

Owner explicitly requests routine eligible imported evidence to post without a
per-entry confirmation. One-time reviewed versioned accounting mappings authorize
automation; analytic labels or bank categories do not select accounting treatment.
This extends the artifact policy with a distinct durable automatic operation,
never a fabricated interactive command or automatic acknowledgement. Existing
manual source review, immutable journals, held evidence and retained history stay.

Proposed next migration49 is the sole SQL lane;1–48 remain immutable. Source-only,
synthetic verification; runtime opt-in remains disabled and no managed activation,
provider call, live record, account creation or policy inference is included.

## Financial policy and alternatives

A current actor with book read/prepare/post plus current source read/review grants
can save a complete immutable per-book mapping edition. It binds the actual
book_policy UUID, enabled flag, effective interval and up to100 unique rules.
Each rule has a stable UUID/name, exact source_account_id, direction, required
EXACT source description, existing offset account_id and treatment
expense/income/transfer/card_payment/loan. Exact matching is deliberately finite
SQL policy selection, not another merchant classifier. Account/direction/interval
and exact description must all match. More than one matching rule holds;
no priority guessing, first-match rule, provider-category account inference or
creation of a missing account. Policy versions and immutable operation provenance
identify the rule actually used, regardless of later edits.

Expense requires an outflow and an expense/debit offset; income requires an
inflow and revenue/credit offset. Source and offset must differ and be active.
Transfers/card payments hold as paired-transfer-required; loans hold as
split-treatment-required until reviewed pair/split evidence machinery exists.
Existing implemented posting/valuation policy is reused exactly: same-currency
USD identity valuation can post under the current explicit supported policy;
original EUR and other unsupported/FX facts retain original units and an explicit
currency/FX hold. No USD substitution or fabricated rate. This is a bounded
initial treatment profile, not full transfer/loan/FX automation.

Alternatives: deriving financial accounts from analytic topics incorrectly
conflates organization with recognition; replaying interactive commands/auto-ack
misrepresents owner review; a second journal writer duplicates money/permission
logic; broad merchant/regex matching adds inference and classifier scope. Choose
explicit source mappings plus shared source-draft/source-post SQL internals.

## Durability, reuse and correction protection

The reviewed create_review_draft financial builder becomes one private shared
helper called by its unchanged public manual wrapper and the automatic engine.
The shared source-post core retains its atomic ledger.post_journal, source/review
projection, review_posting, receipt/audit/outbox behavior. review_posting gains an
exclusive origin arm: existing interactive command OR automatic operation UUID.
Manual command rows/acks are never fabricated. Drafts/decisions explicitly record
automatic policy origin; they do not attest a human confirmed each item.

Separate automatic policy editions, idempotent settings receipts and durable
operations retain actor, source/review/observation revision, source/binding/book
revision, policy edition/rule and exact generated draft/body. Prepare commits
inert draft/provenance and operation identity; execution occurs in a fresh
serializable transaction and commits the balanced journal and linked receipt
atomically. Existing pending automatic identity always wins over selecting new
work. Unknown prepare/execute replies recover THAT identity. Exact committed
replay returns the stored linked result under current authorization. A unique
review_posting source-review link prevents a second effect after a policy edit,
new rule ID or replay; uniqueness is not merely per rule revision.

A current posted original is never regenerated. Changed/removed posted evidence
stays correction_hold (333); no unbound reversal, delta adjustment or replacement
is invented. Existing immutable anchor and original journal remain readable.
Existing manual draft_ready work is protected; auto-owned stale drafts may be
replanned under a new immutable operation, retaining all prior drafts/receipts.
Manual analytic category choices are neither read as financial authority nor
rewritten by this engine.

## Authorization, locks and holds

Every settings/read/prepare/execute/replay rechecks current principal/entity/book
and relevant permissions; worker work-list selection does not grant authority.
No acquisition/observer role gains posting functions. Local retained evidence
bookkeeping does not require a new provider egress consent; source/book/read/review
permission and current source state remain mandatory. Policy owner capabilities
cannot outlive revocation. No public caller supplies trusted actor/observation
results or arbitrary posting lines.

Lock order: principal strongest lock required by shared recovery, entity/current
memberships, book FOR UPDATE, scoped source permissions/source in UUID order,
automatic policy/operation/review as required by existing helpers. Existing shared
interactive saved-work reservation defers automatic execution; automation never
cancels, executes or acknowledges it. No later lock upgrade or competing order.
One source posting per transaction avoids stale sibling book/source revisions.
SERIALIZABLE, transaction statement/lock budgets<=4seconds; known40001/40P01 only
whole-transaction retries, bounded service rounds, cancellation before pool close.

Fixed hold reasons cover missing/ambiguous policy/rule/account, pending/zero/
held/overlap/removed source, manual draft, posted correction, missing FX or unsupported
currency/policy, closed period, inactive accounts, permission change and saved work.
Every hold has zero new journal/outbox; retry requires a changed relevant input.
Errors never expose raw SQL/provider data. Status/read routes do not create work.

## Proposed SQL and transport ownership

Private SQL owns closed body validation, source selection, mapping, draft creation,
posting and receipt equality. Restricted API functions: get/set automatic source
policy; list bounded automatic holds/status; trusted work inventory; prepare next
operation and execute exact operation. API has EXECUTE only, no table/helper grants.
Settings require a fresh expected revision/key and are explicit configuration,
not per-entry approval. Actual closed JSON signatures will be frozen after review.
Go transports trusted verified identity and runs the bounded background lifecycle;
new AutomaticSourcePosting runtime flag defaults false. No new service/database.

Reviewer frontend/API lane may add settings/status surface only after this profile
is accepted. No automatic postings are enabled by a GET, UI render or migration.
Root owns shared integration/OpenAPI/runtime flag composition and normal CI merge.

## Standards, evidence and limits

This is an internal retained-evidence accounting operation, not a bank exchange
or a new ISO20022 message. Existing dated ISO4217 currency identifiers, exact decimal
strings and strict YYYY-MM-DD dates are retained. ISO21378 audit-data semantics
remain relevant to later exports; no new audit export or conformity claim. The
owner still selects actual accounting framework, recognition and real-data policy.

Reviewed selected PostgreSQL17 CREATE FUNCTION security/privileges and Serializable
Isolation sections and ISO20022 FAQ About/Compliance on2026-11-14; existing posting
source register P02/P03/P07/P08 and accepted332 source-journal-corrections contract
supply precise mechanism/accounting limits. These establish mechanisms, not proof
of this implementation or live accounting approval.

Required independently fixed actual restricted SQL evidence:
- eligible USD outflow12.50 yields debit expense12.50 / credit source12.50; inflow
  yields debit source12.50 / credit revenue12.50, with no interactive command/ack;
- exact operation replay, restart/lost prepare/execute reply, reordered rounds,
  policy edits and new rule IDs do not duplicate journal/link/receipt/outbox;
- revoked current grants, stale source/binding/book/policy, manual draft, pending,
  overlap, zero, ambiguous/unmapped rule, inactive account and closed period hold;
- transfer/card/loan unsupported treatments hold explicitly; EUR original10.00
  remains EUR10.00 with FX/currency hold and no invented USD journal; JPY100 and
  KWD0.125 preserve zero-/three-decimal original units and original lexemes on hold;
- posted source50 then changed55 stays immutable original50 plus correction hold,
  never an invented automatic55 or source revision re-post;
- actual API-role table/helper denial, cross-actor/book isolation, saved-work
  deferral, rollback at each link/projection/receipt boundary and unchanged history.

These are planned tests, not completed guarantees.333 correction execution and
real provider/identity/deployment/financial cutover acceptance remain separate.

## Proposed closed configuration/HTTP profile

Package internal/autopost exposes Request{Actor,Entity,Book,Body string} and
Operations.GetPolicy/SetPolicy/Status(context.Context,Request)(json.RawMessage,error).
The authenticated actor is server context only. Public routes:
GET/POST /v1/entities/{entity}/books/{book}/automatic-accounting and
GET /v1/entities/{entity}/books/{book}/automatic-accounting/status. There is no
public run/prepare/execute route. Existing same-origin verified auth/origin boundary
applies. Closed duplicate-key-free UTF8 JSON, no query/encoding/idempotency header;
POST max128KiB, GET body empty. UUID path selectors are validated after auth.

POST body exactly {key,expected_revision,enabled,book_policy_id,effective_from,
effective_to,rules}; revisions are decimal strings, enabled boolean, effective_to
nullable strict date, others nonnull. Each rule exactly {rule_id,name,
source_account_id,direction,description,offset_account_id,treatment}. Direction
inflow/outflow, description exact nonempty<=500chars, name nonempty<=80chars;
max100 unique rule IDs/names, all existing source/account IDs and explicit policy.
No omitted description/wildcard or client posting lines. Complete immutable policy
edition keeps rule identities; removed rules are absent only from future editions,
never historical operations. Settings key actor/book scoped and normalized-body
exact replay; fresh expected revision required to change policy.

GET policy and successful POST return profile automatic-accounting-policy-v1,
entity_id,book_id,revision,enabled,policy(nullable configuration without key/
expected_revision/enabled), and supported_profile='source-identity-usd-v1'.
Absent policy is revision0/disabled/policy null. Status profile
 automatic-accounting-status-v1 has entity_id,book_id,enabled,pending_count,
attention_count decimal strings,blocked_reason nullable saved_work/permission_changed/source_unavailable,last_operation nullable run metadata and holds
bounded to100 closed {review_id,source_account_id,reason,operation_id} rows.
Status counts are complete; response explicitly reports holds_truncated boolean,
never presents100 as the complete count. Run metadata is {operation_id,state,
review_id,journal_id nullable,reason nullable}. Current permissions apply before
any disclosure; unsupported/denied returns fixed errors, never empty fake success.

Error vocabulary proposed P2A00 invalid_automatic_accounting/400,
P2A01 automatic_accounting_denied/404, P2A02 automatic_accounting_conflict/409,
P2A03 automatic_accounting_snapshot_changed/409,
P2A04 automatic_accounting_limit/422; unavailable503. Existing posting errors are
held by the trusted operation only when classified and known to have no effect;
serialization/deadlock/unknown commit remain transaction failures, not persisted
success or financial holds. No raw SQL diagnostics enter transport.
