# Synthetic source-account registry

## Decision and scope

Issue #81 implements only the registry portion of [durable import](durable-import.md):
stable source accounts, immutable adapter identities, explicit read grants and
scoped SQL reads. No upload, observation, draft, posting, provider connection,
HTTP endpoint or cloud provisioning. Coordinator review of this precise contract
precedes migration 000009. This resolves synthetic operator enrollment only;
neither a UUID nor operator assertion proves ownership of a real bank account.

## Operator and runtime boundary

Reuse NOLOGIN `bookkeeping_owner`; no new bootstrap role or API self-enrollment.
Operator functions are SECURITY INVOKER, executable only as current_user
bookkeeping_owner in SERIALIZABLE. An authorized provisioning login explicitly
SET LOCAL ROLEs to that owner. Audit records session_user independently from
approved_by, the active enrolled entity reader named in the reviewed manifest.
Runtime cannot assume this role, call provisioning or grant source permissions.
No new ledger membership value is added. The vocabulary reserves bind/import/review/read for the parent design, but only
read has a grant operation or runtime meaning here. No acquisition authority.

Every mutation takes approved principal, entity, read membership and book locks
following the ledger authorization protocol, with explicit canonical UUID order
for all affected principals/memberships before the book lock. This owned helper
retains the locks needed to revoke a disabled reader without weakening approver
authority; it does not call ledger.authorize twice in inconsistent order. The book lock serializes receipt replay; new work then locks the source. State/permission
changes and runtime reads share the same source/grant lock boundary. Dates and
money are not inferred from names. Existing account semantics are immutable;
future acquisition must revalidate account activity for each booked date.

## Exact SQL operations

All IDs are UUID, all raw inputs strict unique-key JSON objects <=8192 bytes,
keys match `[A-Za-z0-9._:-]{1,128}`. UUID input normalization accepts the existing
hyphenated grammar; date strings use ledger.date_value. Revision strings are
canonical unsigned bigint. All functions use schema-qualified references and
`search_path=pg_catalog,pg_temp`. Error codes P2300 malformed, P2301 unavailable,
P2302 conflict, P2303 unsupported mapping/profile, P2304 limit/overflow;
messages contain no source values. Transaction isolation errors are P2300.

| Function | Arguments / result and authority |
| --- | --- |
| banking.register_source | `(approved_by,e,b,key,raw)` →jsonb. Owner-only. Exact raw keys ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,reader_principal_id,namespace. Namespace keys profile,synthetic_dataset_id,source_account_ref. Atomically create source, first namespace, read grant and receipt/audit. |
| banking.register_source_namespace | `(approved_by,e,b,source_id,key,raw)` →jsonb. Owner-only. Keys expected_binding_revision,namespace. Add immutable namespace, increment binding revision; subsequent namespace permanently carries overlap_review_required=true. |
| banking.set_source_state | `(approved_by,e,b,source_id,key,raw)` →jsonb. Owner-only. Keys expected_binding_revision,state. active↔paused, either→retired; retired never reactivates. Retire all namespaces, keep identities and read access. Same state is a conflict, not another transition. |
| banking.set_source_permission | `(approved_by,e,b,source_id,key,raw)` →jsonb. Owner-only. Keys expected_binding_revision,principal_id,enabled (JSON boolean). Grant/revoke only read; grant requires active current entity reader. No-op requested state conflicts. Increment binding revision; no ledger membership mutation. |
| banking.get_source | `(actor,e,b,source_id)` →jsonb. SECURITY DEFINER, API EXECUTE only. Repeatable-read or serializable; current entity read plus unrevoked source read; principal/entity/book/grant locks serialize revocation. Unknown/wrong-scope/denied returns uniform P2301. Paused/retired records remain readable history, never acquisition authority. |

Owner results are `{source_account_id,namespace_id?,binding_revision}`. Full
normalized request, operator login, approved_by, operation and idempotency key
identify replay. Same request returns the original result without applying grants
or state again; changed request conflicts. Replay still checks current approver authority and active grantee membership
for enrollment/grant. Revocation locks an existing reader even if disabled or
without current membership; otherwise a dormant source grant could survive later
identity reactivation. Returned revision is the operation's committed revision, not current
state. Unknown COMMIT requires replay with the same key and body; no HTTP recovery
operation is introduced. Retired identity cannot be reused with a new key.

Read result profile source-registry-v1 contains entity_id,book_id,source_account_id,
ledger_account_id,currency_version_id,currency,minor_units,binding_date,state,
binding_revision,source_revision and namespaces. Each namespace contains namespace_id,
profile,synthetic_dataset_id,source_account_ref,state,overlap_review_required.
Revisions are decimal strings, namespace order UUID ascending, max 25 and <=64 KiB
result. No grant roster, evidence reference, approval identity or database login
is exposed. No list/discovery endpoint in this slice.

## Physical model and invariants

Book-owned rows use composite `(entity_id,book_id,id)` keys/FKs; no cascades.
Objects belong to bookkeeping_owner. All columns NOT NULL except revoked_at.
Timestamps are finite timestamptz. Server mints IDs; source_revision is zero until
a separately reviewed observation migration. Ordinary owner DML cannot rewrite identity or receipts/audit; controlled
provisioning is the supported path. The trusted schema owner can perform privileged
maintenance; this slice does not claim to contain a malicious database owner.

| Table | Fields and invariants |
| --- | --- |
| banking.source_account | source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by,created_at,state,binding_revision,source_revision. Unique entity/source UUID and scoped ledger_account_id. One source per ledger account is a deliberate first-profile restriction and survives retirement; it is not proof of real-world identity. Ledger FK preserves book scope. Binding requires asset/debit/monetary account active on binding_date; currency has nonnull ISO minor-unit metadata and validity covering date. No automatic USD default or FX. State active/paused/retired; binding_revision>=1. |
| banking.source_namespace | namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,state,overlap_review_required,created_by,created_at. Unique entity/profile/dataset/ref across books and source IDs. Profile exactly jayburd-synthetic-statement-csv/1; dataset UUID nonzero, ref `[A-Za-z0-9_-]{1,64}` case-sensitive. Source FK scoped; max 25; state active/retired. Identity never changes, including after retirement. |
| banking.source_permission | source_account_id,principal_id,capability (read/bind/import/review; only read used here),revoked_at nullable; scoped PK includes principal/capability. FK to principal and source. Grants convey no permission without current ledger membership. |
| integration.source_registry_receipt | scope,operator_login,operation,idempotency_key,approved_by,normalized_request,result,recorded_at. PK scope/operator/operation/key. Immutable, <=16384-byte normalized request (raw input <=8192 bytes) and <=64 KiB result. Records exact manifest and database operator as both replay receipt and provisioning audit; existing audit.action is journal-scoped and unsuitable. |

Registration atomically writes all four rows or none. Additional namespace/state/
grant changes write corresponding rows and receipt together. No journal, outbox,
account mutation or ledger book-revision increment. Separate source identity does
not prove two different ledger accounts represent different real bank accounts;
the operator must review that manifest. Cross-adapter continuity and actual Plaid
namespaces require their accepted profile and explicit overlap decision later.

## Standards, sources and alternatives

Source registry is internal authorization/provenance metadata, not an ISO 20022
financial exchange. The decoder's explicit synthetic CSV exception and SI01–SI06
remain authoritative. ISO 4217 applies through installed immutable currency versions;
full-date follows RFC 3339/ISO 8601 profile. No IBAN/BIC/LEI, camt message or audit
export conformance is claimed. Live metadata releases and provider ownership remain
unresolved gates.

Reviewed sources: [DI01–DI03](durable-import.md#standards-sources-and-alternatives),
[bibliography S07/S08/S12](../research/database-sources.md) and
[schema ADR 0004](../decisions/0004-schema-ownership.md). PostgreSQL 17 constraints, serializable locking and SECURITY DEFINER
sections explain scoped FKs, replay transactions and restricted entrypoints;
they do not establish bank ownership or tax retention. These previously reviewed
sections underpin this implementation; no new financial standard is inferred.
Owner-only enrollment is narrower than adding a global operator role/bootstrap
for one synthetic fixture slice. Read grants avoid circular bind self-provisioning.
Unrestricted JSON identity would admit unreviewed adapters; explicit CSV columns
are chosen now, with later additive reviewed provider identity support.

## Required evidence

TestSourceRegistryEnrollment: exact same-key replay, changed-body conflict,
duplicate ledger account/namespace conflict, concurrent replay and no financial
effects. TestSourceRegistryMapping: inactive/mixed-scope/nondeposit accounts,
unsupported currency/date/null minor units and malformed fields fail atomically;
USD/JPY/KWD retain metadata. TestSourceRegistryAuthority: allowed, missing/revoked,
disabled principal/entity/book, wrong scope and current grant replay checks.
TestSourceRegistryLifecycle: pause/resume/retire, stale revision, namespace reuse,
immutable rows, explicit grant/revoke (including disabled/revoked grantees) and cap. TestSourceRegistryPrivileges:
API/worker/operator-role separation, no direct tables/DDL/helper execution.
TestSourceRegistryAtomicity: late receipt-write failure rolls back all source/grant
rows; same-key retry then succeeds. TestSourceRegistryMixedScope: real cross-book
account/FK/namespace violations fail. TestSourceRegistryRevocationRace: actual competing sessions prove read-first and
revoke-first serializations. These are required tests, not PASS until the runbook
records execution.
