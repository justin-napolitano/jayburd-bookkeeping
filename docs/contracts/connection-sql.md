# Connection lifecycle SQL

## Approved decision

Coordinator-approved #101/#102 foundation, 2026-10-13. Migration 13 adds durable
connection attempts and opaque credential-reference publication to migration 12.
Tokens never enter SQL. Go's trusted connection service verifies Link association,
provider Item/account metadata and secret-store acknowledgement through injected
interfaces. SQL authorizes scope and controls publication; it cannot make a remote
exchange transactional or certify live account ownership. Synthetic provider/secret
adapters are proof tools, not a hosted connection integration.

Provider Link/Items JSON is proprietary, not ISO 20022. Existing PI05–PI08, ISO 4217
and date rules apply. Refreshed official [Items API](https://plaid.com/docs/api/items/)
sections public-token exchange, access-token invalidation and Item removal, and
[Link update mode](https://plaid.com/docs/link/update-mode/), 2026-10-13. These establish
separate remote operations and same-Item update behavior; they do not guarantee
idempotent recovery after lost responses. Our inference is a one-shot durable claim,
verified manifest, deterministic secret reference and explicit uncertain state.
Blind remote retries and storing tokens in the database are rejected alternatives.
Only selected relevant sections were reviewed; no Sandbox/production qualification.

## Authority and identity

Existing `bookkeeping_api` is the trusted backend SQL boundary, as for decoder
publication. Connection functions take the server-resolved actor, never identity
from Link metadata. New explicit entity `connect` membership is owner-provisioned;
current read/connect/import is required for mutations and replay. Existing-source
binding additionally requires source read/import/bind. Worker principal/login is
owner-configured per book and cannot be selected by a runtime request. Provider-first
creation explicitly consents to bounded actor/service read+import and actor bind
source grants; no administrative permission, grant option or arbitrary grantee.
The service's entity read/import grants must already exist. Worker configuration
disable/replacement governs connection enrollment/management, not revocation of
previously enrolled sync streams; revoke their current principal/source permissions
or pause/disconnect the Item to stop existing acquisition. No implicit credential
retrieval/write permission follows from this configuration.

One Item belongs to one entity/book, at most 25 account streams. Existing source
mapping remains immutable; new namespaces on an existing source hold overlap.
Only a new provider-first source with no namespace/evidence starts without that
hold. Currency/depository/ledger mapping is explicit; provider account name/mask
never establish ownership. Mismatched reconnect Item cannot inherit old cursors.

## Frozen SQL interfaces

Scope/actor/attempt/claim/nonce/Item/stream identifiers are UUIDs; keys are bounded
text. All calls use SERIALIZABLE and fixed diagnostics. No new public raw-manifest
HTTP route is implied by these trusted-backend SQL functions.

- `begin_connection(actor,e,b,key,raw)` takes mode new/reconnect/rotate/remove,
  sync_item_id nullable UUID, expected_generation nullable decimal string,
  approved_stream_ids UUID array. New requires null/null/empty; other modes require
  exact current affected set and generation. Returns private context; execute_link
  true only for first new/reconnect insertion. Attempt/nonce/expiry are durable.
- `record_link_session(actor,e,b,attempt,nonce,raw)` takes link_token_sha256 and
  expires_at. Hash pins association, not ownership. Link/public tokens remain outside SQL.
- `claim_connection(actor,e,b,attempt,nonce,raw)` takes nullable link_token_sha256
  and public_token_sha256. New requires both; reconnect Link only; rotate/remove
  neither. Returns execute_provider true only for first claim. Replays are false;
  uncertain claim commit cannot authorize a repeated provider side effect.
- `record_connection_manifest(actor,e,b,attempt,claim,raw)` takes verified item_id,
  accounts [{account_id,type,iso_currency_code nullable}], consent_expires_at;
  persists credential_pending before secret write. Types are depository/credit/
  loan/investment/other; unsupported accounts remain unbound. No token fields accepted.
- `confirm_connection_credential(actor,e,b,attempt,claim)` follows confirmed secret
  write/readback. New enters binding_review; reconnect/rotate require same Item and
  all bound accounts still valid before reactivation. Remote exchange is never repeated.
- `connection_failure(actor,e,b,attempt,claim,reason)` records fixed uncertainty,
  preserving recovery and local fences instead of claiming success.
- `connection_context(actor,e,b,attempt)` is the trusted service projection below.
  `get_connection_attempt(actor,e,b,attempt)` is its safe browser projection.
- `bind_connection(actor,e,b,attempt,key,raw)` takes consent_grants:true and bindings
  [{account_id,ledger_account_id,currency_version_id,source_account_id nullable,
  expected_binding_revision nullable,binding_date,binding_evidence_ref}]. Only
  verified accounts are selectable. Scope/currency/date/deposit and current grants
  are checked before atomic source/namespace/stream publication.
- `list_connections(actor,e,b)` returns {items:[{sync_item_id,item_id,state,
  item_generation,accounts:[{stream_id,account_id,source_account_id,state,
  binding_revision}]}]}; bounded at 100 Items/25 accounts with explicit overflow.
- `disconnect_stream(actor,e,b,stream,key,expected_binding_revision bigint)` fences
  only that account; returns {stream_id,state:"disconnected"}. It never removes Item.
- `confirm_item_removed(actor,e,b,attempt,claim)` records confirmed remote removal.
  Whole-Item approval/fences precede provider I/O; unknown reply stays unresolved.

Private context fields: attempt_id,entity_id,book_id,mode,state,nonce,expires_at,
sync_item_id nullable,item_generation nullable string,expected_item_id nullable,
claim_id nullable,credential_ref nullable,credential_version nullable string,
previous_credential_ref nullable,previous_credential_version nullable string,
link_token_sha256 nullable,public_token_sha256 nullable,manifest nullable,
execute_link boolean,execute_provider boolean. Get/replay execution booleans are false.
Public attempt: attempt_id,entity_id,book_id,mode,state,expires_at,sync_item_id,manifest.
No hashes, nonce, secret refs or claim are exposed by status. Bind also returns streams
[{stream_id,source_account_id,namespace_id,binding_revision,overlap_review_required}].

States are link_pending, link_ready, exchange_claimed, credential_pending,
binding_review, active, credential_recovery_required, disconnect_pending and
disconnected. An exchange_claimed attempt after a crash is unresolved recovery work,
never authority to repeat the remote call. Attempt history may say active while the
current Item listing shows a later lifecycle state. Attempt handoff expires after
15 minutes, clamped to earlier Link expiry; expiry blocks a first claim and new
binding, while already reserved credential recovery still requires current grants
and unexpired verified consent. There is no automatic secret cleanup.

Owner-only SECURITY INVOKER provisioning functions are
`set_connect_permission(actor,e,b,subject,key,enabled)`,
`set_connection_worker(actor,e,b,key,raw)` with
{service_principal_id,worker_login,enabled}, and
`set_source_bind_permission(actor,e,b,source,key,raw)` with
{expected_binding_revision,principal_id,enabled}. Receipts record operator login and
attesting actor. These functions are not exposed to the API role.

Fixed `connection_rejected` diagnostics: P2700 malformed/isolation; P2701 current
authority; P2702 identity/replay conflict; P2703 stale/expired pins; P2704 wrong stage
or invalid verified selection; P2705 bounded overflow. PostgreSQL transaction retry
codes remain visible to the Go transaction owner. Keys accept 1–128 ASCII identifier
characters; raw envelopes reject unknown/duplicate keys and are bounded before casts.

## Credential and lifecycle invariants

SQL allocates immutable target reference/version before external effects. Verified
manifest commits before secret write. Confirmed write/readback precedes activation.
Lost exchange response remains credential_recovery_required; no claim reacquisition.
Lost secret acknowledgement can resolve only the SQL-pinned reference and verified
manifest; no replayed exchange/rotation. Reconnect uses existing reference and
verifies same Item/filter. Rotation fences all siblings before provider invocation;
replacement reference publication is atomic and leaves source identities/cursors.
Removal fences all approved streams first; local disconnect does not imply remote
removal. Revocation/expiry checks apply even to replay; journals/evidence are retained.

## Acquisition additions

`integration.sync_context(e,b,stream,set,fence)` is sync-role-only, rechecks current
session/permissions and live attempt, then returns stream_id,set_id,fence string,
provider,environment,item_id,account_id,item_generation string,credential_ref,
credential_version string,binding_revision string,starting_cursor,request_cursor,
next_page_index,page_count,byte_count,change_count,final_page,lease_until,
currency {code,version_id,minor_unit,valid_from,valid_to nullable}, and limits
{max_pages:10,max_changes:1000,max_bytes:2097152,max_page_changes:500}.

`integration.sync_resume(e,b,stream,claim_uuid)` reauthorizes before returning
{state:absent|staging|completed|abandoned|quarantined,set_id nullable,fence nullable
string,reason nullable string,completion nullable receipt}. Absent is an explicit
object, not JSON null. This enables resolution after an unknown commit without
unnecessarily repeating provider calls. Expired staging cannot gain a new fence;
new attempts restart C0 under the existing rules.

## Acceptance and limits

Planned actual PostgreSQL proof covers scoped role/grant revocation, one-shot claims,
unknown-commit replay, manifest/credential sequencing, provider-first vs overlap,
exact account/currency bindings, same-Item reconnect, sibling rotation fencing,
single-account disconnect, Item removal uncertainty and private context access.
Named executed evidence belongs in the [runbook](../runbooks/connection-sql.md).
No test names here imply PASS. Hosted adapters, secret policy and Sandbox/live
qualification remain explicit external gates.

Private context revalidates Item generation and the approved sibling set for every
nonterminal attempt before exposing credential references. Unclaimed attempts also
require unexpired handoff authority; public status remains readable with current
permissions after these pins become stale.

New and first-claimed reconnect/rotation/removal operations require Item state
active, paused or reauthorization_required. A claimed operation first moves the
Item into credential_recovery_required or disconnect_pending, excluding overlapping
remote mutations even when two attempts were prepared beforehand. Only that pinned
attempt may reconcile its recorded manifest/reference. Unknown operations without
a persisted manifest have no automatic recovery path; a new attempt cannot bypass
the unresolved remote outcome. Generation fencing alone cannot cancel an in-flight
provider request.
