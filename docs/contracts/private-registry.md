# Private source registry —239 / forward42

## Decision, scope and review

Owner approved the source-first authority and private classification design on
2026-11-06; GitHub238 is the accepted operation matrix/fixed-case record,242 the
taxonomy direction,239 the implementation authority and65 the live writer lease.
This contract freezes the first physical private-registry slice. Root reviewed
the read/provision wire, exact snapshot and SQLSTATE plan before DDL. New source
review and actual restricted-role/migration proofs are required before merge.
No topic/label/rule table, HTTP route, provider call or cloud change belongs here.

Existing identity mapping remains exact verified issuer+subject to enrolled UUID.
An owner-only provision command explicitly grants a collection capability; book
membership never creates it. One private workspace belongs to one principal.
Sharing, legal-owner assertions, cross-book posting, destructive purge and live
credential/provider integration remain separate work. No holding book is created.

## Physical ownership and records

Every new table is owned by bookkeeping_owner. Runtime has no table privilege.
No new SQL role or schema is added. All UUIDs below are NOT NULL unless described
as nullable; generated IDs default gen_random_uuid(). Instants are timestamptz
with finite checks and recorded_at defaults clock_timestamp(). Counters bigint
are nonnegative or positive as named, transported as canonical decimal strings.

| Table / grain | Columns, keys and lifecycle |
| --- | --- |
| banking.collection_workspace / enrolled private owner | workspace_id UUID PK; owner_principal_id UUID unique FK ledger.principal; revision bigint positive default1; capabilities text[] subset read/acquire/classify, with read required if nonempty; created_at instant. UUID/owner immutable; only explicit revision+capability transitions; no DELETE. Empty capabilities revoke workspace access without erasing evidence. |
| integration.collection_item / canonical provider identity reservation | collection_item_id UUID PK; provider text=plaid; environment text sandbox/production; item_id case-sensitive text1–256 printableASCII; unique(provider,environment,item_id); workspace_id nullable FK until explicit legacy adoption; legacy_entity_id/book_id/sync_item_id nullable all-or-none composite FK integration.sync_item; native credential_ref UUID/credential_version bigint positive and consent_evidence_ref UUID/consent_expires_at finite instant only when legacy tuple absent; state active/paused/revoked; generation bigint positive default1. Legacy rows retain original credential context by reference, never copied/re-encrypted. State/generation/private-owner transition only; no identity rewrite/delete. |
| banking.collection_source / immutable private alias | workspace_id/source_account_id composite PK, workspace FK; collection_item_id FK; account_id case-sensitive provider account ID1–256; display_name text1–200 no controls; origin native/legacy; optional legacy_entity_id/book_id/source_account_id composite FK banking.source_account; unique legacy tuple and unique(collection_item_id,account_id); state active/revoked; revision bigint positive default1; source_revision bigint nonnegative default0; cursor_value nullable text<=256; readiness fixed known status; last_completed_at nullable instant. Private alias differs from globally nonunique legacy UUID; original tuples remain intact. Source lifecycle/cursor revisions only; no deletion/reparenting. |
| banking.collection_observation / native source identity | workspace_id/source_account_id/observation_id composite PK/FK source; external_entry_id case-sensitive text1–256; unique(workspace,source,external_entry_id); current_revision positive bigint default1; created_at instant. Only monotonic current pointer may change; all identity fields immutable. |
| banking.collection_revision / immutable native fact | workspace/source/observation/revision composite PK and scoped observation FK; previous_revision nullable for first, otherwise revision−1 scoped FK; booked_date bounded date0001–9999; description text<=500codepoints/2000bytes; merchant_name nullable text with same bounds; source_state booked/pending/removed/held; currency_code nullable bounded source text; currency_version_id nullable FK reference.currency; amount_lexeme text<=128 preserving exact normalized source magnitude; amount nullable ledger.amount; direction nullable inflow/outflow/zero; hold_reason nullable unsupported_currency/invalid_precision; normalized_fact jsonb bounded8192bytes. Held facts have no interpreted amount/currency; immutable original fields remain privately retained. |
| integration.collection_receipt / command replay | workspace_id FK; principal_id FK; operation bounded enumerated command; idempotency_key printable identifier1–128; request jsonb bounded2MiB; result jsonb bounded2MiB; recorded_at. PK(workspace,principal,operation,key). Immutable. Operator grant receipts use the subject principal plus fixed operator operation and include session_user in request. |
| audit.collection_action / audit decision | action_id UUID PK; workspace_id FK; principal_id FK; operation text; target_id UUID; evidence_ref nullable UUID; database_actor name=session_user; recorded_at. Immutable; source facts/tokens are not logged here. |

Primary/unique indexes serve exact owner lookup, provider identity reservation,
source enumeration, scoped external transaction identity and revision lookup.
Additional observation/source indexes support bounded original-date pagination.
Native and legacy source observations are never copied to one another. A safe
owner-only fact projection reads legacy observations through the immutable alias
and native observations through their scoped current pointer; it returns no book,
journal, credential, external provider ID, raw payload or acquisition cursor.

## One provider identity across old and new paths

Migration42 reserves every existing legacy Item in collection_item, without a
workspace owner or credential copy. It holds SHARE ROW EXCLUSIVE on the old Item
table through backfill/trigger installation and first checks duplicate canonical
identity groups with a fixed P2802 failure, never raw unique-constraint identity
details or a guessed deduplication. An AFTER INSERT trigger on integration.sync_item
reserves new legacy identities under the same unique(provider,environment,item_id)
key used by native registration. Therefore a legacy enrollment racing a native
registration cannot create two authorities. A conflict reports only fixed P2802.
This is an authority/identity index, not another provider enrollment or ledger.

Explicit adoption updates that existing reservation row; it never inserts a
second provider identity. The actor must be the original consent principal,
currently authorized for legacy read/connect/import and each selected source.
The reviewed source selection must exactly match all active legacy streams.
No unresolved connection attempt or staging acquisition set may exist; unknown
provider outcomes are not guessed. Adoption pauses/increments the old Item and
installs immutable source aliases without changing source/observation/cursor/
journal tuples or credentials. A trigger fences subsequent legacy Item reactivation
and sibling enrollment after adoption. Native publication into a legacy alias is
denied; new acquisition integration/reconnect belongs to240 with original AAD.

## SQL operations and current authority

All entrypoints use fixed search_path pg_catalog,pg_temp, bounded strict unique-key
JSON and explicit grants. Functions/helpers are revoked from PUBLIC and unrelated
runtime roles. API wrappers additionally require the existing actual-login
provider_api_login(session_user), including mixed-role/direct-DML denial. Actor
arguments come from the trusted identity resolver, never browser actor claims.

### Operator

banking.provision_collection_owner(subject UUID,key text,raw text) is SECURITY
INVOKER, owner-only. Raw exact fields: expected_revision null|string,
capabilities array of distinct read/acquire/classify strings, evidence_ref UUID.
Initial expected_revision=null creates one workspace; existing writes require its
current revision and increment it. The enrolled subject must remain active.
Same-key/body replay rechecks current principal and current resulting capability
state, never silently reactivates revoked access. Changed body conflicts. Result:
{workspace_id,workspace_revision,capabilities}. No runtime self-grant exists.

### Safe API reads

- banking.get_collection_workspace(actor UUID) returns
  {profile:private-source-workspace-v1,workspace_id,workspace_revision,capabilities}.
- banking.list_collection_sources(actor UUID,w UUID,raw text) takes exact
  {limit:integer1..100,cursor:null|string,snapshot:null|snapshot}. Returns
  {profile:private-source-page-v1,workspace_id,workspace_revision,snapshot,
  items:[{source_account_id,display_name,state,source_revision,origin}],next_cursor}.
- banking.query_collection_observations(actor UUID,w UUID,raw text) takes exact
  {source_account_id:null|UUID,starts_on:null|date,through_date:null|date,
  state:null|booked|pending|removed|held,search:text<=100,limit:integer1..100,
  cursor:null|string,snapshot:null|snapshot}. Returns
  {profile:private-source-query-v1,workspace_id,workspace_revision,snapshot,
  selector:{source_account_id,starts_on,through_date,state,search},matching_count,
  items:[{source_account_id,observation_id,observation_revision,source_state,
  booked_date,description,merchant_name,amount,direction,currency,hold_reason}],
  next_cursor}. currency is null for held unsupported money, otherwise
  {version_id,code,numeric_code:null|string,minor_unit,valid_from,valid_to} from the exact dated
  reference. Null is never a fabricated zero.

Current principal/workspace read authority precedes all rows/count/cursor details.
Only active source aliases under non-revoked Items are visible. Paused/expired
acquisition consent does not prevent stored-fact reads; explicit source/Item or
workspace revocation does. Explicit guessed inaccessible source IDs deny uniformly.
Book/journal projection is absent;304 must separately check current book permission.

Every page uses one RR or SERIALIZABLE snapshot and fixed dateDESC/sourceASC/
observationUUIDASC ordering (sources use aliasUUIDASC). Snapshot is exactly
{workspace_revision:string,source_snapshot:64lowercasehex}. The source digest is
core PostgreSQL SHA256 over the exact ordered authorized source inventory with
private source lifecycle/data revisions, Item generation/state and current legacy
source binding/data revision. It is not a sum. Legacy importer/manual-review
changes are therefore seen even though legacy provider acquisition is fenced.
Cursor embeds workspace, selector, snapshot and last composite key; a changed
snapshot rejects P2803 before continuation. Digests/cursors are equality metadata,
never grants. Queries reject more than100 sources or result envelopes over1MiB.

### Trusted normalized storage commands

banking.register_collection_item(actor,w,key,raw) creates a native Sandbox identity
and1–25 distinct source accounts only after current acquire capability. Exact raw:
{provider:plaid,environment:sandbox,item_id,credential_ref,credential_version,
consent_evidence_ref,consent_expires_at,accounts:[{account_id,display_name}]}.
Opaque references grant no secret access. No browser route exposes this trusted
verified-manifest boundary;240 must supply the actual Link/credential protocol.
Production native acquisition is deliberately not activated by this slice.

banking.publish_collection_batch(actor,w,source,key,raw) is a trusted, complete,
account-filtered normalized source publication, not the Plaid wire/worker adapter.
Raw={expected_source_revision,expected_item_generation,starting_cursor,next_cursor,readiness,changes:[...]},
up to1000 changes/2MiB. Each added/modified row is exact
{operation,transaction_id,booked_date,description,merchant_name,source_state,
currency_code,amount,direction}; amount is original unsigned decimal text,
source_state booked/pending, direction inflow/outflow/zero. A removed row is only
{operation:removed,transaction_id} and retains prior financial/date evidence.
Missing/dated-gap currency and unsupported precision retain a held immutable fact
with original lexeme and no interpreted amount, atomically with the complete cursor.
No partial-page publication;240 must complete provider pagination before this call.
Unknown modified/removed identity rejects. Duplicate IDs reject the entire batch.
Same identity/full normalized fact is a no-op; corrections append a revision.
Distinct IDs remain distinct even when money is equal. Empty initialization cannot
claim proven history completeness; readiness is attribution only.

banking.set_collection_state(actor,w,key,raw) takes
{kind:item|source,target_id,expected_revision,state:paused|active|revoked}.
Source accepts active→revoked only; Item supports active/paused and terminal revoke.
Current acquire capability is required, same-body receipt replay reauthorizes.
No provider Item removal or credential deletion is implied by local revocation.

banking.adopt_collection_item(actor,w,e,b,item,key,raw) takes
{expected_generation,sources:[{source_account_id,display_name}],evidence_ref}.
The exact old scope is explicit and currently reauthorized; aliases retain it
privately. Full-set adoption, old Item fencing, aliases, receipt and audit commit
atomically. Lost response resolves by same-key replay, not repeated adoption.

## Locks, retries, retention and safety

Native writes require SERIALIZABLE and lock active principal, workspaceFORUPDATE,
canonical Item then sorted source aliases, then observations. Reads lock principal
and workspaceFORSHARE before source diagnostics, then all private Item rows in UUID
order, all private source rows in aliasUUID order, and adopted original source rows
in entity/book/source order, each FORSHARE with no visibility-state predicate.
This is bounded100Items/100aliases and deliberately checks updated rows against a
primed RR snapshot: native Item/source revocation or legacy source pause commits
force40001 rather than allowing stale authority after a workspace wait. Legacy
book permission is neither read nor granted by that source-provenance row lock.
Provision uses principal then
workspace. Legacy adoption first uses current legacy authority/book/source locks
in existing order, then workspace/Item; it alone crosses both domains. Raw private
reads never acquire a legacy book lock. Future mixed private/book projections must
take legacy book authority before workspace to avoid reverse ordering. The shared
provider-identity unique key serializes legacy/native admission; transactions must
retry whole on40001/40P01/55P03, with bounded timeouts and no network effects inside.
No retryable exception is translated into a fake receipt.

Only monotonic pointers/lifecycle fields update; evidence, receipts and audit are
immutable. Reads never append an audit/command/journal. No new purge API exists,
and downgrade permits an empty-only42→41 rollback. Every new table, including legacy identity reservations, must be empty; any retained record rejects before schema destruction. Exact trigger/function/table removal uses no CASCADE. Reapply42 must pass afterward. No legal retention duration is
invented: revoke prevents access/acquisition and retains original/audit/posted
anchors. A later reviewed deletion policy must handle legal/audit holds, provider
credential removal and immutable accounting separately before destructive cleanup.

SQLSTATE fixed collection_rejected: P2800 malformed/isolation; P2801 current
authority/notfound; P2802 identity/replay conflict; P2803 stale pins/snapshot;
P2804 invalid lifecycle/publication; P2805 work/envelope bounds. Inputs/errors
never echo source data. Unknown SQLSTATE remains infrastructure/retry failure.

## Standards, evidence and alternatives

Purpose is internal private source storage/query, not a new registered financial
exchange. ISO20022 not applicable to this command/query profile; no camt message
or ISO designation invented. ISO4217 dated identifiers/minor units apply to validated
money; unknown/gapped mappings are held. Gregorian dates use the existing bounded
ISO8601 profile. ISO21378 export mapping is outside this non-export slice, not a
claim of conformance. Plaid transport/version/mapping and actual complete-loop
integration remain240/413; no historical payload extraction/refetch is authorized.

Selected PostgreSQL17 public sections revisited2026-11-06:
[CREATE FUNCTION](https://www.postgresql.org/docs/17/sql-createfunction.html),
security-definer/search-path/default-EXECUTE precautions;
[explicit locking](https://www.postgresql.org/docs/17/explicit-locking.html),
row lock modes/deadlock ordering;
[transaction isolation](https://www.postgresql.org/docs/17/transaction-iso.html),
RR/SSI retries;
[binary functions](https://www.postgresql.org/docs/17/functions-binarystring.html),
sha256/encode. These establish mechanisms; our key/authority/retention choices
and test oracles are project decisions, not vendor accounting/privacy approval.
Existing source-first/identity/provider-sync research retains its recorded limits.

Alternatives rejected: dropping old PKs or making book nullable (breaks existing
FK/grant/provenance contracts); fake holding book (invented accounting scope);
duplicate registry with no legacy trigger (race admits duplicate provider authority);
copying old observations (duplicate identities and divergent posted anchors);
workspace revision alone or revision sums (misses legacy changes/collides);
book membership as private access (leaks another owner's classifications).

STRIDE: high disclosure/elevation→explicit owner provision/current principal/source
authority before counts; high tampering→closed SQL, immutable evidence, exact keys
and replay; high stale authority→workspace/source/Item fencing and complete source
snapshot; medium DoS→100sources/100rows/1000changes/2MiB and deadlines; high
repudiation→immutable operator/source receipts/audit with session_user. No raw
tokens, payloads, secrets, browser persistence, new network or ledger write path.

Named independent proofs: TestUnassignedScope, TestCrossPrincipalDeny,
TestDuplicateItemAndAccount, TestLegacyNativeIdentityRace, TestRevisionReplay,
TestRevokeAndReconnect, TestExistingBookMigration, TestLegacySnapshotChange,
TestCurrencyHoldAtomicity, TestNoJournalFromAcquisition, TestCollectionPrivileges,
TestCollectionRevocationRace, TestCurrentSourceReadLocks and TestCollectionPagination3000. These must execute
against real restricted sessions/disposable PostgreSQL, not only owner-role fixtures.
