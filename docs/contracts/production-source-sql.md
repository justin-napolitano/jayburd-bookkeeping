# Production source SQL authority — #311

## Decision and boundary

Coordinator engineering review2026-10-25, before forward DDL. Owner assigns311
to this branch and313 separately. Accepted308/312 and current mainc843315 are
the baseline. This contract specifies source authority, not resource provisioning,
an approved real accounting policy, credential injection or live activation.
Only synthetic records are used in tests, including fictional Production-profile
rows; Production environment is a routing/provenance value, not a test-data claim.

Retain `jayburd-plaid-transactions/1`: its exact amounts/original lexemes,
ISO4217 currency-version/date/sign/state mapping and API2020-09-14 do not change.
Environment and Item already participate in namespace identity. Extend supported
environment values, not money semantics or an old row's origin. Existing Sandbox
records stay Sandbox/synthetic; newly enrolled Production Items have synthetic=false.
Data origin is derived from the immutable environment, never supplied by browser
JSON. No Production data is marked synthetic to pass a development constraint.

## Physical specification and administration

One new bookkeeping_owner-owned table `integration.provider_scope`, one row per
entity/book, FK to ledger.book, composite PK(entity_id,book_id): environment text
NOT NULL IN(sandbox,production); api_login/worker_login name NOT NULL;
bank_key/link_key text nullable only for Sandbox; enabled boolean NOT NULL;
revision bigint NOT NULL positive. Key resources are bounded canonical Cloud KMS
cryptoKey names, not secret values or mutable secret aliases. Production compares
envelope key resource exactly to the approved scope key before accepting ciphertext.
No new KMS key is created by configuration. Exact resources remain an owner313/337
decision; fixtures use fictional names. Separate new SQL logins are required for
Production; neither may also belong to an existing Sandbox scope. API login must
be a safe LOGIN with bookkeeping_api and no owner/sync/webhook/operator privileges;
worker must satisfy existing strict sync_login checks. No direct table grants.

Owner-only `integration.configure_provider_scope(actor uuid,e uuid,b uuid,k text,
raw text) RETURNS jsonb`, SECURITY INVOKER, requires current_user=bookkeeping_owner,
SERIALIZABLE and existing source_operator_authorize. Closed raw keys:
environment,api_login,worker_login,bank_key,link_key,enabled. This is an operator
function, never granted to API/worker or mounted as a browser endpoint. Initial
configuration requires no connection attempts, Items, source namespaces or journals
in that book: it cannot promote/relabel an existing Sandbox book. Environment,
logins and key resources are immutable after creation; subsequent calls may only
toggle enabled and increment revision. Same-key/body receipt replay is exact;
changed-body replay conflicts. No automatic role grants, retention/delete policy
or source consent is created. Lock principals/entity/book first, then scope row;
existing connection receipts retain operator audit facts. Revocation uses the same
book lock as ongoing work, so it cannot commit through an authorized transaction.

Internal `integration.provider_environment(e uuid,b uuid) RETURNS text` obtains
the current scope FOR SHARE and verifies session_user against its allowed API or
worker login, current role safety and enabled state. No `SET`/caller actor or JSON
can select a profile. An unconfigured book retains the legacy Sandbox behavior
only for logins not explicitly registered for Production elsewhere. A configured
Production book has no fallback: missing/wrong/revoked scope/login failsP2701.
Both connection_authorize and sync_authorize retain their existing current
principal/membership/book/source/consent/generation/lease checks and additionally
require this environment match. The configured connection worker must match the
approved scope worker. Production API cannot use an unconfigured Sandbox book.

API-only `integration.connection_provider_context(actor uuid,e uuid,b uuid)
RETURNS jsonb`, SECURITY DEFINER, reuses connection_authorize and returns only
environment, data_origin(test/provider), scope_revision as decimal string,
bank_key/link_key. #313 must compare this preflight to its explicit immutable
runtime profile and configured key resources BEFORE begin/provider I/O. It may not
reinterpret a missing scope as Production. No secret, token or envelope is returned.

## Persistence and existing function inventory

Add connection_attempt.environment NOT NULL, backfill ONLY existing rows to their
already-known sandbox origin, and preserve it in its immutable guard. New attempts
derive environment from authorized scope at INSERT; clients cannot write a value.
Private render_connection adds environment/data_origin; public projection unchanged.
sync_item keeps its existing synthetic boolean with the new two-way CHECK:
sandbox AND synthetic=true OR production AND synthetic=false. Extend
source_namespace provider profile check and bank_credential environment check to
the same closed enum. No update of old Item, namespace, receipt, ciphertext or journal.
Insertion guards require each Item/namespace/credential environment to match its
book scope and retain immutable identity on update. Scope cannot subsequently be
reassigned. Cross-table validation uses owned functions/triggers, not a CHECK that
reads other tables. Unique(provider,environment,Item[,account]) remains unchanged.

Latest implementations reviewed:13 connection_authorize/lock/item_pin and
sync_authorize;32 owned repair wrappers and renamed pre32 functions;28 bind_connection
and sync_context;20 put_bank_credential;33 put_link_handoff and its pre32 delegate;
18 bank_credential_authorize/get/worker reads;12 operator enroll and observation
provenance;25 installed currency history/validation;29 exact precision. Patch only
the actual current implementation or active pre32 delegate, preserving wrappers,
ACLs, exact replay/body equality and existing lock/receipt semantics.

confirm_connection_credential's pre32 delegate creates a new Item using a.environment
and synthetic=(environment=sandbox); bind28 uses i.environment for namespace
creation. connection_lock/item_pin reject mismatched attempt/Item scope. Credential
authorization additionally matches env to the exact attempt. put credential and
new-Link handoff use the approved Production key; Sandbox keeps the existing
canonical dev-key grammar. Worker reads derive env/ref from sync_context as before.
Production owned reconnect remains unavailable until314's matched signed receiver
and313's repair runtime exist; do not broaden32's Sandbox-only proof functions.
Webhook ingress, Sandbox diagnostics/verifier and operator enroll remain Sandbox.

Pilot-bound source review refinement before DDL change: Production bind permits
exactly one selected account, and rejects another Item while a non-disconnected
stream already exists in the book (paused counts as bound). Reuse the existing
book lock; no additional lock or race-prone counter. Known committed bind receipt
replay is handled before this new-work limit. Sandbox retains its existing
multi-account behavior. TestProductionSingleAccountPilotLimit names both the
multi-selection and second-Item rejection with unchanged source counts.
An outer begin_connection wrapper rejects a fresh new attempt when any existing
Production Item is not disconnected, before Link/provider work. Existing-key
recovery remains allowed; the owned32 wrapper/delegate is preserved underneath.

No Go credential/runtime/CLI changes belong here. Acquisition's internal SQL
context validator may accept the two closed environments only after the new SQL
scope/worker checks are tested; no default Production constructor is activated.
The runner already propagates trusted environment from312. #313 owns crypto AAD
and composition matching; it must not use this SQL change as credential-origin proof.

## Posting, migration and acceptance

No new posting grant or blanket posting ban. Acquisition alone has zero journal
and journal.posted effects; current legitimate prepare/post/reversal/replay remains
subject to existing book policy and authorization. Synthetic USD/EUR tests do not
approve dev-accrual-usd-v1 for a real book. Corrections remain333 and closed-period/
policy restrictions remain unchanged. No automatic category/model posting.

Forward migration34 only after fresh sole-writer register check. Down refuses if
any Production scope/attempt/Item/namespace/credential exists, including disabled
scope: deleting/rewriting retained provenance is not rollback. On empty/Sandbox-only
state restore exact prior definitions/constraints and remove additive objects;
roundtrip and existing installed-migration checks must pass. SQL files1–33 immutable.

Named tests on actual restricted login sessions and disposable PG17:
TestProductionRecordsNotSynthetic; TestProductionProfileCrossEnvironmentDeny;
TestProductionScopeRevocationAndReplay; TestProductionCredentialSQLBinding;
TestProductionPostingAuthorityAndReplay; TestProductionScopePrivileges. Include
missing/foreign login/book/key/env, attempted JSON profile injection, revoked
current permissions/consent, duplicate commands, existing Sandbox regressions,
independent zero-journal acquisition and exact single authorized post/replay.
Tests must not infer success from administrator calls or mocked PostgreSQL.
Run migration-test, ledger-test, connection-state/credential/acquisition regressions
and core check/test/build. Proposed names are obligations until actually executed.

## Sources, alternatives and threat review

PP01/04/06 and new PP07 in the annotated source register record review depth.
PostgreSQL17 CREATE FUNCTION SECURITY DEFINER section266–292 establishes safe
search_path and atomic PUBLIC revoke/selective grants; Constraints5.5.1 paragraphs
on NULL/cross-table CHECK establish why typed NOT NULL and insertion guards are
needed. These sources establish mechanisms, not this authorization design.
Plaid API host/storage sections establish separate nontransferable Items and
sensitive tokens. Full paid ISO/GAAP texts remain unreviewed; no conformance claim.
This operational authority/AAD protocol has no applicable registered ISO20022
message; proprietary financial mapping and ISO4217 versioned currencies unchanged.
ISO21378 audit relevance retains existing abstract-only access limits.

Rejected alternatives: a global production boolean weakens Sandbox boundaries;
caller-selected environment permits confused deputy use; rewriting rows/ciphertext
destroys origin/AAD; a new DB/service without337 approval expands infrastructure.
Book-scoped explicit authority preserves existing components and legacy Sandbox
while requiring a separately configured new Production book/runtime identity.
Tradeoff: reassignment of Production logins/keys is deliberately unavailable in
this initial scope and needs a separately reviewed rotation/recovery operation.

STRIDE: spoofed runtime/tenant (high) -> session_user + scoped role/login match;
source relabel/tamper (high) -> immutable env/two-way origin/namespace uniqueness;
replay/escalation (high) -> existing current grants and immutable receipts; key
substitution/disclosure (high) -> exact configured key and no plaintext SQL/logs;
revocation races (high) -> SERIALIZABLE/current book locks and whole retry;
unintended egress (high) -> no provider calls/default activation in this change.
Human source/security review remains required before merge; operator resource and
live policy approval are separate. This is coordinator design, not independent signoff.
