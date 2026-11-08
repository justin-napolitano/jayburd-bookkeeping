# Bounded private Sandbox activation — 453

Status: coordinator-approved source profile, 2026-11-07. Source implementation and managed
execution are separate approvals. Existing SQL 1–44 remains immutable; this
profile allocates no migration, identity, permission or cloud resource.

## Authority and identity

Provisioning is an explicit operator operation, never an HTTP self-service
capability. A finite owner-invoked SQL stage must require the existing authorized
operator's SET authority for the safe NOLOGIN bookkeeping_owner role. Bind the
fixed Sandbox project/database/issuer, exact enrolled subject and expected
principal UUID, capabilities, workspace revision, evidence UUID, idempotency key
and a finite execution deadline. Match the active enrolled issuer/subject row to
the expected UUID before calling banking.provision_collection_owner. Do not
enroll, reactivate or infer identity from email or book ownership. Return only
workspace UUID, revision and capabilities; suppress subject and SQL diagnostics.

Adoption uses the existing managed Firebase authenticator, including current
revocation, TOTP and exact active issuer/subject resolution. An operator-approved
actor UUID is only a restriction compared with that verified result. It cannot
produce or override the authenticated actor. The database connection remains an
actual authorized API login; SET ROLE is not an identity proof.

## Finite transport

Inject one immutable approved Sandbox plan into a dedicated adoption service,
default absent. Bind plan UUID, actor/workspace/entity/book/Item UUIDs, expected
Item generation, complete reviewed active-source UUID/display-name set, SQL key,
evidence UUID and deadline. No provider Item ID, credential reference, token,
financial observation or generic SQL is an input.

POST /v1/collection-activation/adopt accepts exactly one plan_id UUID.
Authenticate before interpreting it; require equality with the approved actor
and plan. Execute only banking.adopt_collection_item using the plan's exact tuple
and body in SERIALIZABLE. SQL retains current workspace acquire, book read,
entity connect/import, source read/import and original consent-owner checks.
Generation mismatch, incomplete active streams, unresolved connection attempts
or staging acquisition work deny the entire operation.

Load(path, now) reads at most32768 bytes from a regular nonsymlink file. Its
closed envelope is {sha256,plan}; sha256 is64 lowercase hexadecimal characters
over the exact nested plan JSON bytes. This integrity value is not a signature;
approval depends on the operator-controlled file and explicit server composition.
The closed plan fields are profile=private-sandbox-activation-v1,
project=jayburd-bookkeeping-dev, plan_id, actor_id, workspace_id, entity_id,
book_id, sync_item_id, expected_generation (positive decimal string),
idempotency_key (1–128 ASCII letters/digits/dot/underscore/colon/hyphen),
evidence_ref, sources and expires_at. Sources contain1–25 unique
{source_account_id,display_name} entries with1–200 Unicode codepoints per name.
Expiry is an exact UTC seconds timestamp, future and at most one hour ahead at
load. Duplicate/unknown/case-alias fields, trailing JSON and invalid IDs deny.
The operator must independently review the hash and every target field; a
self-consistent hash alone does not approve a plan.

ParseApproved(raw, now) provides the exact same32KiB envelope/hash/expiry checks
for optional inline operator configuration, without filesystem or network I/O.
Load retains its regular-file/symlink/inode/size guards and delegates only its
validated file bytes to this shared parser. Neither delivery can supply a user
identity or extend approval. Root composition must reject simultaneous file and
inline configuration and retain the same Sandbox+CollectionEnabled gates.

Inline approval metadata in service command arguments is visible to authorized
project operators. Only opaque reviewed target UUIDs, idempotency/evidence
identifiers, generation/deadline and explicitly owner-approved safe aliases may
be delivered this way. Never put issuer subjects, UIDs, email, tokens,
credentials, provider IDs or source facts in it. Hash verification establishes
integrity/content binding, not operator authentication; trusted deployment
configuration is the approval boundary. This source path needs no new mount,
resource or IAM grant, but its availability does not authorize a managed update.
An expired configured plan is rejected at startup; an already running service
denies adoption after expiry. The operator must explicitly remove the temporary
approval configuration after its reviewed window or approve the same intent
again; the application never renews or silently discards approval itself.

### SDK argument delivery

Inline JSON contains commas. Ordinary shell quoting does not prevent the SDK's
own list parser from splitting a default-comma --args value. Use the reviewed
custom-delimiter syntax `^DELIM^arg1DELIMarg2` for the complete argument list,
quoting the entire SDK flag value for the selected shell so JSON quotes,
whitespace and metacharacters remain literal. Independently verify that the
chosen nonempty delimiter occurs nowhere in the complete approved envelope or
any other argument; do not assume a fixed character is safe. Do not use eval or
reparse the envelope as shell code. Alternatively use an operator-controlled
structured --flags-file with --args represented as a YAML/JSON list, preserving
each argument and the exact envelope bytes as one string.

In either delivery, retain the full known prior container argument list and
change only the explicitly reviewed activation argument. Do not discover, print
or change environment/secret settings. Keep the approved envelope and argument
projection out of public evidence; record only its reviewed digest and bounded
operation outcome. This guidance does not authorize an SDK update or execution.

Official Google Cloud CLI references reviewed 2026-11-07:
[gcloud topic escaping](https://docs.cloud.google.com/sdk/gcloud/reference/topic/escaping),
DESCRIPTION and basic/multi-character delimiter examples (page updated2026-05-27);
[gcloud topic flags-file](https://docs.cloud.google.com/sdk/gcloud/reference/topic/flags-file),
DESCRIPTION and YAML list/quoting examples; and
[gcloud run services update](https://docs.cloud.google.com/sdk/gcloud/reference/run/services/update#--args),
Container Flags/--args. These selected sections establish SDK parsing and list
delivery, not application approval or hosted correctness. Requiring absent
delimiters, exact prior arguments and digest-only evidence is this profile's
bounded operating policy. No SDK command or managed configuration was executed.

The source-only flags collection-activation-plan-file and
collection-activation-plan (inline envelope) both default empty and are mutually
exclusive. Composition requires the Sandbox project, managed mode, existing
CollectionEnabled setting and a non-Production provider environment. The inline
configuration has the same32768-byte limit before dependency construction.
NewService uses only an existing restricted pool. Requests have1024-byte bodies;
responses are bounded32768 bytes and contain profile
private-sandbox-activation-result-v1, plan_id and result. Result is the exact
SQL-owned {collection_item_id,item_generation,sources} receipt; each source entry
contains source_account_id, source_revision and revision, never provider IDs.
No GET discovery or financial listing route is added.

Adoption explicitly pauses legacy acquisition and advances its generation. It
does not copy facts, change credentials/AAD, cursor/source/journal anchors, resume
acquisition, renew consent, contact a provider or create journals.

## Replay and operational limits

Retry only complete known-aborted transactions. A lost commit reply returns a
fixed unavailable result; recovery resends the identical approved plan/key/body
and rechecks current identity and SQL authority. Changed intent needs a different
reviewed plan, never a substituted replay body. Expiry prevents execution and
replay. If outcome remains unknown after expiry, an operator must explicitly
reapprove the same immutable SQL intent for a fresh finite window. Absence,
timeouts or expired approvals never establish success. A separate read-only
receipt operation would require an additional reviewed SQL boundary.

The existing migrator accepts only verify/status/up and cannot execute this
operator stage. Source availability does not establish managed operator SET
authority or approve a job/identity/grant. Managed execution remains blocked
until an authorized operator mechanism is selected and reviewed.

The dedicated db/bootstrap/sandbox-private-owner.sql inputs are project,
subject, principal_id, idempotency_key, body and expires_at. Body is the exact
existing provision JSON {expected_revision:null|revisionstring,capabilities,
evidence_ref}. Capture the returned workspace UUID before reviewing the adoption
plan; do not guess it. The script checks expiry again immediately before commit.
Its existing operator SET authority is a prerequisite, not a permission it grants.

## Evidence and applicability

Reviewed existing contracts: [identity/setup](identity-setup.md),
[private registry](private-registry.md), [Sandbox bootstrap](sandbox-book-bootstrap.md),
[migration status](migration-status.md); reviewed source entrypoints in forward42,
the managed Firebase adapter and restricted collection transport. The existing
[identity source register](../research/identity-sources.md) records SDK verification
semantics and access limits. These sources establish mechanisms; the finite
approval restriction and operational split are project policy proposed here.

Alternative runtime auto-provisioning would widen authority. Operator adoption
under an impersonated user would remove the trusted identity boundary. Generic
SQL would exceed the finite reviewed operations. None is selected.

This internal authority operation is not an ISO 20022 financial message or an
ISO 21378 audit export. Existing dated ISO 4217 evidence remains unchanged; no
recognition, currency conversion or standards-conformance claim is introduced.

Acceptance must cover foreign/disabled/unenrolled identity, expiry/default denial,
changed targets/replays, current revocations, generation/source-set mismatch,
pending work, atomic failure, both canonical identity races and exact lost-reply
recovery with actual restricted PostgreSQL roles. Synthetic tests do not replace
managed Firebase sign-in or owner financial review.
