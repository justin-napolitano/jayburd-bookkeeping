# Closed Plaid provider transport profile — #308/#312

## Source decision

Engineering baseline reviewed by coordinator2026-10-25 before implementation.
Owner authorizes source work toward real-account registration with bank-read-only
access and ledger bookkeeping; this does not accept deployment targets or secrets.
Source acceptance closes #308 only after this baseline is merged/checked. #337
retains the actual unresolved resource/isolation/retention/pilot decision; #316
depends on it. Splitting this gate does not count an absent predecessor as done.

[PP01/PP04/PP06](../research/production-pilot-sources.md) record reviewed official
host, request/version, sync/state and privacy sections and limits. Two closed
profiles: sandbox -> https://sandbox.plaid.com; production ->
https://production.plaid.com. API2020-09-14, pinned SDK49.0.0 unchanged. No arbitrary
URL, ambient discovery, case folding, whitespace normalization, redirect, fallback
or moving an Item between environments. Standard root TLS remains intact.

## Construction and request boundary

Use one small shared immutable profile value with private state. Unknown/empty
explicit profile and zero value reject before I/O. Existing public constructors
remain legacy Sandbox wrappers with identical behavior; new explicit constructors
require a valid profile. No new CLI/runtime/API/browser field or default enables
Production. #313 later wires approved managed composition after #311 SQL authority.

Connection allowlist retains only POST link/token/create, link/token/get,
item/public_token/exchange, item/get, accounts/get, item/access_token/invalidate
and item/remove. These support enrollment, readback and separately authorized
credential/Item lifecycle; none is money movement. Reject test/refresh/payment/
transfer/other paths, userinfo/query/fragment/opaque/encoded-path substitutions.
Use the exact profile host and API header, current bounded SDK transport, no
retries/redirect and existing pre-float account projection. Products stay US
Transactions only. Source312 Production rejects the Sandbox webhook URL and
HostedRepair capability; reviewed Production receiver configuration and repair
qualification belong to #314/#313/#275. Empty Production webhook permits isolated
construction tests; it is not a release-ready notification configuration.

Acquisition permits only fixed POST transactions/sync for the configured profile.
Keep account filtering/count500/cursor, raw bounded bytes, exact original-currency
lexemes, response/error budgets, pagination-mutation and Retry-After behavior.
Add explicit trusted environment metadata to FetchRequest and propagate the SQL
execution context in the runner. Production requires exact production; Sandbox
accepts exact sandbox; omitted metadata is accepted only by the legacy constructor,
not the new explicit Sandbox constructor.
Mixed/unknown context/profile denies before transport. Existing SQL validContext
continues to reject Production until #311; source312 must not weaken SQL checks.
Credentials are opaque: this layer cannot prove their origin from a token prefix.
Addressed credential/AAD/current SQL binding is #313, and must be qualified before
activation. Profile matching is not a substitute secret-origin or consent proof.

## Standards and unchanged semantics

Environment/host selection is operational authorization, not a registered ISO20022
message. Existing proprietary Plaid2020-09-14 financial mapping and ISO4217 dated
currency identities/original units remain unchanged; no float, FX fallback, new
recognition rule or ISO21378 export/conformance claim. SQL true data-origin/profile
version compatibility must be reviewed in #311 before DDL; source312 does not
relabel evidence, add a namespace or authorize real-account posting policy.

## Alternatives, risks and evidence

Changing credentials or one global URL was rejected: it does not carry scope/AAD/
webhook/SQL authority and risks mixing environments. Duplicating all clients was
rejected because financial decode/error rules would drift. Shared closed routing
keeps existing Go/PostgreSQL/Next components; no new DB/service or cloud spend.

STRIDE: spoofed profile/destination (high) -> private immutable closed value,
exact request allowlists and mismatch negatives; secret/evidence substitution
(high) -> current source limitation explicit, #311/#313 binding required; response
tampering/disclosure (high) -> unchanged bounded exact decoder/redacted failures;
unintended capability (high) -> no runtime/API selector, no payment/test/refresh
path and Production repair/notification guard. Human security review before
sensitive code merge/activation remains required. This is engineering review,
not AppSec or actual provider qualification.

Named independent tests: TestClosedPlaidProfiles rejects unknown/zero/malformed
and arbitrary host/path; TestProductionConnectionProfile checks exact injected
request/version/Transactions body and no Sandbox receiver/repair; existing Sandbox
request/SDK/decimal/recovery fixtures unchanged; TestProductionSyncProfile checks
matching/mixed/absent metadata and literal original numeric bytes, pagination and
retry/redaction/redirect negatives. All injected transports: no real provider,
secret reads or cloud I/O from default tests. Core/race/build and appropriate
existing integration regressions before handoff; no migration in #312.
