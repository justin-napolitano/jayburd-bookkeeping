# Scoped Sandbox equality and zero-journal verifier

## Decision and scope

Coordinator-approved engineering profile for #272/#85/#87/#88, 2026-10-22;
review checkpoint preceded migration30 implementation. Fresh one-source synthetic book only.
No Production, raw-provider export, table SELECT grant, posting, source mutation,
provider request or cloud provisioning. No baseline delta or snapshot nonce.

Existing [synthetic lifecycle proof](../runbooks/plaid-85-verification.md) verifies
exact fixed values through the real adapter/SQL boundary, but hosted UI counts
cannot establish complete equality or absence of automatic journals. This slice
adds a separately enrolled bounded operational capability; deployment and actual
execution remain separately authorized work.

## Physical model and interfaces

New NOLOGIN `bookkeeping_verifier` capability: integration schema USAGE and EXECUTE
only on `integration.verify_sandbox_fixture(uuid,bigint,text,text)`. No external
snapshot or diagnostic capability is granted. Dedicated login is
`jayburd-dev-verifier@jayburd-bookkeeping-dev.iam`. No ADMIN/SET, other bookkeeping,
predefined pg_* or privileged memberships. Bootstrap must precreate the safe
capability before restricted migration; runtime never enrolls itself.

`integration.verification_mandate` grain: one actual verifier login × exact
entity/book/source/namespace/stream/Item × fixture and identity-map digest × pinned
binding/Item/completed generation interval. Columns: mandate_id UUID primary key;
entity_id/book_id/source_account_id/namespace_id/stream_id/sync_item_id composite
FKs; verifier_login name; approving_principal UUID FK; binding_revision,
item_generation,target_generation bigint; fixture_id text; fixture_sha256 and
identity_map_sha256 bytea32; enabled boolean; mandate_revision bigint;
created_by_login name; created_at,expires_at timestamptz. Lifetime at most60minutes
and no later than current source consent. Scope/pins/digests/expiry are immutable.
Only explicit revoke changes enabled true→false and increments revision. Renew
through a new owner mandate; expired history remains immutable.

`integration.verification_operator_receipt` grain `(session_user,operation_key)`:
normalized metadata, result and created_at only. No expected financial rows,
identity-map contents, raw provider bytes, baseline counts or tokens are stored.
Owner-only enrollment checks an explicitly attested approving principal and
current scope; approving principal is not runtime caller identity. Owner-only
revoke takes mandate, expected revision and stable key. Exact known-operation
replay retains its receipt; changed same-key input is rejected.

## Current identity, locks and revocation

Go checks actual session_user=current_user before calling SQL. SECURITY DEFINER
changes current_user, so SQL independently requires safe enrolled session_user and
`current_setting('role')='none'`, never equality against its definer. Reject
privileged attributes, database/schema CREATE, all predefined and other application
roles, direct/effective table/sequence authority, unexpected function capability,
and grant/delegation options. Fixed errors contain no dependency diagnostics.

The Go adapter imposes a four-second context deadline around the entire
SERIALIZABLE transaction, beginning before BeginTx. It configures three-second
statement/one-second lock budgets before the compare call and uses independent
rollback. SQL does not claim that changing statement_timeout within an already
running function bounds a direct caller. No automatic cloud/provider work.
Authority locks precede scope: current approving/consenting/service principals in
UUID order, entity/membership, book, source, Item, stream, mandate. Current read/
source-read authority, active scope, unexpired consent, exact binding/Item pins and
mandate enabled/revision/expiry are rechecked. Runtime source/Item/session/login
cannot be selected by caller parameters. One source belongs to the synthetic book;
other-namespace observations and unexpected review records make equality fail.

The stream must have no active lease and requested_generation=completed_generation=
the mandate's target. New work or scope changes make the snapshot ineligible.
Do not hold database locks across a source-provider request. Revocation serializes
at the existing authority rows and mandate; concurrent changes cause bounded
failure/retry-required evidence, never a false pass. No new financial write occurs.

## Independently authored fixture and identity map

Inputs are proprietary `jayburd-sandbox-verification/1`, at most1000 revisions and
2MiB total. Author expected financial/state fields before provider injection.
Separately bind logical IDs to provider-assigned IDs using bounded injection
response identity metadata where supplied. The documented Sandbox create endpoint
returns no transaction IDs; the proposed first hosted posted-only run uses the
[strict tagged identity readback](hosted-sandbox-verifier.md#identity-readback-correction-engineering-profile-for-review)
instead, subject to that profile's human review and limits. Neither map method may
supply expected amounts/dates/state or regenerate fixture financial fields.
Do not copy raw provider records into the expectation. Complete-source equality
requires a complete controlled fixture, not a selected injected subset.

Fixture rows identify logical entry/revision and assert current pointer, original
numeric lexeme, exact canonical unsigned decimal magnitude, direction, source state,
booked date, ISO currency code and dated currency-version identity, description,
previous revision and current review disposition. Pending links are independent
logical-ID pairs. Identity-map rows contain only logical ID and transaction ID.
Require unique JSON keys, strict required fields, no unknown fields, unique
logical/revision pairs, exactly one current revision per logical ID, unique mapped
provider IDs, complete mapping and valid lineage references. Legitimate equal-value
purchases remain separate identities. Never deduplicate by amount/hash.

Owner enrollment pins SHA256 of exact fixture bytes and exact separate map bytes.
A verifier cannot vary candidate values or IDs to use mismatch counts as a probing
oracle. Changing either requires a new owner-reviewed mandate. Digests are of
independent synthetic expectations, never released hashes of actual observations.
Financial input and map contents stay ephemeral and are not logged or retained.

SQL alone owns exact comparison. Missing/extra revisions, quantity/lexeme/unit/date/
state/direction/description/current-pointer/lineage mismatches and review evidence
mismatches fail equality. Fresh profile requires exactly one review assignment per
observation, one distinct observation per review, and exactly one rendered evidence
entry with the exact observation/namespace/provider identity and expected revision.
Equal-valued purchases cannot share a review assignment. Compare all retained revisions, not only latest state.
No float conversion, rounding, minor-unit truncation or implicit FX translation.
Preserve original currency units; functional-currency accounting is not inferred.

## Output and zero-journal evidence

Only booleans and bounded count strings leave SQL/CLI. Counts saturate at1001
(meaning at least1001); supported actual history/review sets above1000 are rejected. No IDs, names, dates,
amounts, tokens, cursor values, source hashes or raw errors. Only the combined comparison is externally callable. It checks expected values
and counts journals plus `integration.outbox_message` with event_type journal.posted
inside the same authorized snapshot. Equal requires all mismatches zero and both
absolute financial counters zero. A fresh zero book with immutable journals/outbox
establishes no journal additions within that book; this does not attribute effects
in a preexisting nonzero book or qualify unrelated accounts.

## Sources, standards and alternatives

Existing [standards](standards.md), proprietary Plaid/API2020-09-14 mapping in
[provider sync](provider-sync-sql.md), dated ISO4217 semantics in
[history currency](plaid-history-currency.md), and P88-10/11/12 in the
[Plaid source register](../research/plaid-88-test-sources.md) apply. ISO20022 is not
an applicable registered message for this internal redacted operational comparison.
ISO4217 identity/date validity and exact financial mapping remain applicable.
ISO21378:2019 abstract review identifies audit-data relevance but this is not an
audit export or conformance claim; full normative text remains unreviewed.

PostgreSQL17 source basis: existing database foundations S06/S08/S12 and
[bootstrap review](webhook-bootstrap.md#sources-standards-and-alternatives) for
function privileges, serializable locking, role inheritance/delegation and powerful
predefined roles. These establish mechanisms; finite mandates, separate login,
complete independent fixture and absolute-zero gate are project acceptance choices.

Alternative unrestricted owner SQL/raw dumps violate the narrow evidence/privacy
boundary. Reusing sync/API identity couples comparison to acquisition/posting and
widens authority. A caller-selected baseline or unbound nonce can fabricate
causality. The scoped hash-pinned capability adds migration/provisioning work but
allows inspectable exact comparison without general financial reads. Nonzero-book
baseline/attribution remains explicitly deferred rather than weakly asserted.

## Required acceptance

Disposable real-role tests: valid complete lifecycle fixture, every financial/state/
lineage/review mismatch, missing/extra/duplicate identities, duplicate JSON/unknown
fields, hash change, excessive bytes/rows; no identity collapse for equal purchases;
wrong/SET/mixed/predefined/direct-privilege roles, revoked principal/source/mandate,
expired consent/mandate, foreign scope and changed generations. Controlled synthetic
posting must change absolute financial counters and fail zero-journal certification.
No output may contain input source markers. Permission denials require exact
SQLSTATE or closed classification, not connection errors. Migration, ledger, runtime
adapter and CLI inertness gates remain required. No hosted qualification from these.

## Bounded operator command

`cmd/sandbox-verify` without flags prints only `{"offline":true}` and performs no
file, identity, network or database operation. Explicit execution requires
`-action=compare -manifest=/absolute/clean/path/manifest.json`. The <=8KiB strict
metadata manifest has exactly these string fields: `profile` equal to
`jayburd-sandbox-verification-manifest/1`, `instance` for the reviewed development
Cloud SQL instance, `mandate_id`, positive canonical `mandate_revision`, and
lowercase raw-byte `fixture_sha256` / `identity_map_sha256`. It accepts no DSN,
caller actor, secret, provider or sync option. The component derives the verifier
IAM login; its pool has max1/min0.

The command reads only fixed siblings `fixture.json` and `identity-map.json`,
checks each final path is a regular nonsymlink file, enforces the combined2MiB
bound and verifies both digests before creating transport. Ancestor path symlinks
are not a security boundary; the owner supplies the reviewed absolute directory.
No financial values are normalized in Go. The adapter starts its four-second
context **before BeginTx**, applies local3s statement/1s lock limits, checks actual
session identity and runs one SERIALIZABLE comparison/commit attempt. Rollback has
an independent one-second cleanup context. The CLI has an overall15s transport
context; a function-local setting is not claimed to bound an already-running
statement from a direct SQL caller.

Successful committed equal/unequal comparisons print exactly the two booleans and
seven count strings. Unequal results exit nonzero after emitting that bounded
result. Invalid input, denied access, cancellation or uncertain commit emit no
prepared comparison and a fixed diagnostic; no automatic transaction retry occurs.
Owner enrollment/revocation is separate from this read-only command. Unit fakes
prove transport/commit failure handling; the disposable real-login integration
proof also invokes this Go adapter against the real comparison function.
