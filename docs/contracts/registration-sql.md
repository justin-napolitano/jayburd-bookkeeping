# Recoverable selective registration SQL —392

Engineering design2026-11-02, before forward DDL; owner approved391 direction.
Human exact source/security review remains required before merge/apply. No managed
source maximum is increased by this implementation. Existing first checking
history/credentials/journals and original bind receipts remain immutable.

## Decision, sources and alternatives

Reuse ledger.interactive_command for the server-resolved registration plan instead
of inventing a parallel recovery table. Its existing principal lock, single pending
gate, immutable reviewed request/result, audit, cancel/acknowledge and inaccessible
recovery apply. Add only the register_accounts operation and restricted preparation/
execution hooks. Generic register_command continues rejecting that operation;
only the owned preparation function can persist its server-resolved body.

This refines391's conceptual plan/receipt into existing normalized storage.
An independent plan table would duplicate the command gate and cause cross-command
race/recovery ambiguity. Reopening an active bind attempt would corrupt its receipt;
new local append uses a new command and leaves that attempt untouched. Shared SQL
binding validation/mutation is extracted from34, not duplicated in Go/UI/jobs.

Reviewed sources: [SR01–04](../research/select-register-sources.md), existing
[database foundations](../research/database-foundations.md), S08 Serializable
Isolation Level and S12 function privilege sections;34 actual connection lock/
bind implementation;11 command authorization/transition and16 inaccessible recovery.
Vendor sources establish cached/shared account behavior and SQL mechanisms, not
Jayburd's access limits, accounting profile or exact-once effects. No full normative
ISO or real provider payload/record is read. Already-shared accounts only; no new
provider calls/update-mode/token exchange or bookless registry in this slice.

## Limit and shared binding mutation

integration.registration_limit(entity_id,book_id) FK ledger.book stores a maximum
smallint1–25 and positive revision. No row means Production1; Sandbox25. New
owner-only SECURITY INVOKER configure_registration_limit(actor,e,b,key,raw)
requires SERIALIZABLE/current source-operator authority and configured Production
scope, closed {max_sources:<decimal string>} input, exact body receipt/replay.
Revision advances on change; no deletion or API/worker grant. Existing provider
scope and its key/login immutability are unchanged. Limits never grant account
consent or change one-Item begin guard. Count non-disconnected streams (paused
included) plus proposed streams, under existing book lock. Known bind receipts
still return before new-work capacity checks.

Extract the current strict binding loop into an owner-only integration helper:
manifest/provider-account identity, account classification, dated exact currency,
new/existing source/account validation, grants, namespaces and streams remain
unchanged. Reject a provider account already present on the same Item, including
disconnected history. Rebinding a disconnected account must use explicit existing
source/reconnect recovery; creating duplicate lineage by registration is denied.
Legacy bind retains its initial active-attempt prohibition and receipt/result;
only its capacity guard now uses the scoped maximum. Sandbox compatibility stays.

## Prepare server-owned plans

integration.prepare_registration(actor,e,b,attempt_id,key,request_id,raw) returns
the existing rendered Command. Closed request {account_ids:[string,...]}1–25,
unique nonempty immutable IDs, no environment/money/currency/code injection.
Retain normalized selection in the immutable command body for same-key equality.
Existing chart, manifest and reference snapshots determine the exact proposal.
Current verified actor owns the attempt; current connect/import/read and worker/
scope checks apply. Attempt is new-mode binding_review or active, with verified
manifest/confirmed Item. Initial setup requires the existing confirmed paused
Item and still obeys attempt expiry; append requires
active, unexpired consent on the same owned Item. No new Item/credential created.

Choose the unique currently effective book policy in its own timezone, then its
local binding date; missing/ambiguous policy fails. Resolve exactly one installed
currency version on that date with known minor units, never defaultUSD or change
transaction dates. Existing supported asset/liability profile supplies category/
normal side and neutral subtype label. Allocate stable collision-free chart code
proposals within a bounded search and distinct random account/evidence IDs; no
ledger/source objects or grants created at prepare. Advanced mappings retain the
existing direct bind API; this narrow prepare input is automatic new-account setup.

Persist body: selection; attempt_id; Item id/generation; worker/provider-scope/
limit revisions; manifest digest; current book revision; resolved policy/date;
expires_at=min(now+10minutes, consent expiry, initial attempt expiry when initial);
initial/append mode; exact bindings; digest of the resolved proposal. Same key/
same normalized selection/target returns the same reviewed command, including
after execution; changed body/target conflicts. Another unacknowledged command
blocks preparation under existing principal lock. Plan IDs/maps never rebuilt
on retry. No financial request/token browser persistence.

## Execute, cancel and recovery

Existing ledger.transition_command branches register_accounts to the private
executor; all other operations unchanged. Current operation-specific authorization
maps revoked connection permission to normal ledger denial so inaccessible clear
works without exposing payload. Read/ack/cancel never require an unexpired plan
or pinned stale Item generation; execution of NEW work does.

Executor compares current attempt/Item/manifest/generation, worker/scope/limit and
book revision, policy/date and references, expiry/consent and supported mappings.
Conflict never silently regenerates a mapping. For initial binding use ordinary
bind_connection with command-owned key, preserving existing activation/receipt.
For append use shared mutation on the active Item, without updating original
attempt/bind receipt/Item generation/credential or existing stream cursors/fences.
New streams start requested1/completed0 with ordinary durable publisher discovery.
No acquisition task or journal outside its atomic SQL mutation.

Committed command result contains entity/book/Item and new streams, no tokens.
Exact execute replay reads the immutable result through current authority before
checking new-work pins/expiry. Cancel prepared then explicitly acknowledge;
committed cannot be canceled. Known stale/expired prepared plan can be canceled/
acknowledged and replaced; unknown commit must be resolved through same command.
Current access loss uses existing payload-free inaccessible clear. Existing
immutable commands, receipts, audit actions and journals cannot be rewritten.

## Standards, controls and acceptance

Internal setup/recovery and access configuration have no applicable registered
ISO20022 message/version. Proprietary Plaid API2020-09-14 and existing transaction
mapping unchanged. Currency identifiers/version/minor unit use installed ISO4217
records; original transaction amounts/units/dates preserved. ISO21378:2019 audit
relevance retains existing abstract-only access limits; no new export/ISO claim.
STD01 applicability: above; STD02 closed/profiles/wrong fields; STD03 currency/date/
asset/liability fixed fixtures; STD04 replay/immutable provenance; STD05 default1,
old client compatibility, immutable1–35 and new-command rollout gate.

STRIDE: spoof/privilege(high)→verified SQL login/current actor/scope/manifest;
tamper/replay(high)→immutable server-resolved command/revisions/exact result;
disclosure(high)→no raw metadata/tokens/public IDs; denied reads payload-free;
cost/DoS(medium)→1–25 limit, bounded code search/request bytes/expiry;
repudiation(high)→existing command/connection audit. No new dependency, external
integration, resource or crypto. Human source review before merge/apply.

Named synthetic restricted-login proofs: RegistrationScopeLimit,
RegistrationAppendPreservesExisting, RegistrationPlanAtomicity,
RegistrationPlanReplayAndUnknownCommit, RegistrationPlanCurrentAuthority,
RegistrationCurrencyAndProfile. Include concurrent appends/capacity, paused and
disconnected duplicate rejection, foreign owner/Item, unsupported/missing currency,
changed chart/policy/consent, cancellation/ack/global pending gate, no journals.
Required migration-test/ledger-test/connection-state/app-integration/core gates;
exact-head CI. These remain proposed until stage runbook records execution.

Forward migration36 only; down refuses if register_accounts commands or explicit
limit rows exist (even acknowledged/unchanged), preserving reviewed intent/history.
Empty/Sandbox compatibility down restores exact34 bind and11 command functions,
original operation constraints and removes only new helpers/limit table. No live
down-migration/reset/relink to repair a conflict.393 owns API,394 picker,395 hosted
qualification and concrete additional-account approval. No managed apply here.
After a registration command exists, runtime rollback requires a version that can
render/recover register_accounts; older command-unaware images are not a compatible
rollback. Pause/forward repair preserves the saved work.
