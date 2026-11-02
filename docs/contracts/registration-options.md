# Server-owned registration options —393

2026-11-02, bounded existing-projection increment after400/401. Read registration
SQL/HTTP, account-display-labels, bibliography, database foundations, decision
evidence and standards. Sources SR01–04 and their existing selected-section access
limits apply; no new vendor behavior, normative ISO text or real record reviewed.

Problem: a selection-only UI needs current eligibility/registered/capacity reasons
without copying SQL policy into Go/TypeScript. Alternatives: client classification
duplicates authority; trial preparation mutates the single pending-command gate;
raw table access widens runtime privileges. Choose a restricted SQL snapshot read.

Forward37, never edit1–36: owner-only immutable registration_profile(jsonb) factors
the exact36 automatic product/category/side/code/neutral-label mapping. The37
replacement preparer uses it with unchanged request/receipt/pins/mutation behavior;
down restores36's exact preparer before removing helpers. Final execution's strict
existing binding validation remains unchanged. No new account/product policy.

integration.registration_options(actor,entity,book,attempt) is SECURITY DEFINER,
fixed pg_catalog/pg_temp search_path, EXECUTE only bookkeeping_api, no runtime
table privileges. Existing connection_lock/Item pin/current scoped authorization
protect both initial and active owned attempts. Go also checks finite runtime,
environment and private key/profile context without provider/crypto I/O. The read
uses a serializable snapshot and ordinary scope locks, creates no saved plan,
account, stream, grant, receipt, audit action, demand or journal. It is advisory:
prepare and execute always revalidate; no new consent or source limit is granted.

GET /v1/entities/{entity}/books/{book}/connection-attempts/{attempt}/registration
returns200 application/json/no-store. Exact canonical verified scope, no query,
body or caller actor/environment accepted. Bridge allowlists only this new exact
GET. Optional service absent fails503. API/OpenAPI/types remain synchronized.

Closed result: entity_id/book_id/book_name/attempt_id/sync_item_id; binding_date
(null when no unique current policy); max_sources/current_sources/remaining_sources
as decimal strings (max_sources=null in Sandbox; remaining_sources=25 is its
per-request bound); history_target_days="90", history_request_mode="provider_default",
history_guaranteed=false. The current CreateLink adapter does not set explicit
transaction history days; this is a product target, not a claim of a90-day request;
blocked_reason (null or closed reason), can_register boolean; sorted accounts1–25.
Each account includes immutable account_id, neutral display_label, existing
type/subtype/ISO currency, registered boolean, optional stream/source ID/state,
eligible boolean and closed reason. Display labels are current-view subtype labels
plus sorted ordinals, not bank names/masks or identity. No new provider metadata.

Reasons: already_registered, source_disconnected, unsupported_account,
currency_unavailable, currency_ambiguous, policy_unavailable, connection_not_ready,
attempt_expired, consent_expired, capacity_exhausted, code_space_exhausted.
Registered/disconnected identity is reported before new-work blockers, so it cannot
be selected anew. Paused sources count. Production capacity uses36's explicit
scope/default1 and one-Item condition; Sandbox capacity is25 per request. Unique
dated currency/current policy and bounded free-code search match preparation.
Global reasons apply to new rows; row reasons do not imply final authorization.
An initial/active owned verified manifest is required; otherwise409. Current
authority/profile denial remains existing payload-free404/503 behavior.

STRIDE: spoof/escalation(high)→verified actor/current SQL login/scope/Item pins;
disclosure(high)→minimal existing fields/no-store/no raw errors or provider calls;
tampering(high)→closed GET path/no input data and final prepare/execute rechecks;
DoS(medium)→25 rows/bounded code search/deadlines; repudiation→no new mutation,
existing persisted command audit remains execution evidence. New API/data-access
source needs exact human security review before merge/apply; no deployment here.

Standards: internal setup snapshot has no ISO20022 registered message. Existing
Plaid API2020-09-14, installed ISO4217 dated identifiers/minor units and original
money/date semantics unchanged; no ISO21378 export or conformance claim. STD01
applicability here; STD02 closed HTTP fixture; STD03 fixed multicurrency profiles;
STD04 no mutation/current authority; STD05 forward37/down36 compatibility.

Acceptance: restricted SQL TestRegistrationOptionsNoEffects,
TestRegistrationOptionsExpiry, TestRegistrationOptionsRegisteredRecovery,
TestRegistrationOptionsAuthority and TestRegistrationOptionsPolicy;
TestRegistrationAPIRecovery proves actual HTTP→Go→SQL options and prepare
agreement; auth/query/body/method/bridge denial; zero provider/crypto calls; core,
migration, ledger, connection-state, locked web and app-integration gates. These
are passes only as recorded in stage evidence.393 stays open for advanced
selection override,290 for persistent metadata,394 picker,395 hosted qualification.
