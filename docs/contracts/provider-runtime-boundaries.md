# Provider runtime implementation boundaries

## Decision

Coordinator-reviewed synthetic implementation profile, 2026-10-13, #70/#100/#102.
Extend the [provider proposal](provider-ingestion.md) through separate SQL,
decoder and transport contracts. This record resolves the boundary with merged
migrations 9–11; it does not approve provider access or credential enrollment.

## Existing authority and required extensions

Migration 9 owns scoped source bindings and synthetic CSV namespaces. Migration 10
owns immutable booked observations with revision 1, strictly positive amounts and
recoverable CSV imports. Migration 11 adds category drafts and atomic source-linked
posting. None of those contracts implies provider pending/removal/revision support.
Only a new forward migration may introduce it; existing migration files stay intact.

Provider changes require immutable observation revisions plus a controlled current
projection. A posted journal and its original source evidence remain immutable;
later changes create explicit exceptions. Pending, removed and zero-valued source
items cannot enter the positive journal-line workflow. Original signed numeric
lexemes remain separate from unsigned exact magnitude and accounting direction.
Changes invalidate stale drafts; old prepared commands cannot bypass new state.

One account-filtered acquisition stream owns its cursor. The database checks
current binding, consent, service identity and generation at every mutation. A
dedicated restricted sync role receives narrow functions, not API posting authority.
Owner-only synthetic stream enrollment is a fixture boundary, not Link, consent
verification or a production provider connection. #101 owns those later operations.

## Implementation profile and alternatives

The first SQL complete-set boundary is at most 10 pages, 1,000 changes and 2 MiB of
normalized data. These are project limits, not Plaid guarantees or scalability
results. Exceeding a bound rejects the whole set without advancing the published
cursor. The broader proposal's 50,000-change target remains unproved. A short atomic
publication is preferred to partial page publication, which exposes incomplete
pending replacement and correction state. Larger workloads require measured limits
and a reviewed continuation design, not silently skipping source history.

Decode exact numbers before any SDK float conversion. Decoder output is untrusted
input to SQL authority; successful parsing grants no connection, review or posting
permission. Decode failures return fixed codes without retained unauthorized data.
The decoder does not claim a finished update set from a single page.

The existing journal observer remains receipt-only. Its HTTP/queue transport proves
private invocation and durable acknowledgement semantics independently; it is not
the provider acquisition runner. #102 must compose acquisition with its own state
and complete-set publication before claiming working server-to-server sync.

## Credential lifecycle gate

No current lane reads or writes real credentials. Later #101 must bind exchange
attempts to authorized scope, securely store a new version, then publish its opaque
reference. Uncertain provider exchange or secret-store acknowledgement stays in
credential recovery. Local reference publication cannot make a remote exchange
transactional. Rotation fences every sibling stream before replacing a shared Item
reference; single-account disconnect cannot implicitly remove the Item. Test these
failure points through injected interfaces before any separately authorized Sandbox
qualification under #88.

## Sources, standards and acceptance

Refreshed selected official sections on 2026-10-13: Plaid
[Transactions API](https://plaid.com/docs/api/products/transactions/), account-filtered
cursors, pagination restart, amount fields and initialization; PostgreSQL 17
[transaction isolation](https://www.postgresql.org/docs/17/transaction-iso.html),
serializable retries; and [function privileges](https://www.postgresql.org/docs/17/sql-createfunction.html),
security-definer search path and default execution privileges. These explain
mechanisms; the limits, role split and immutable exception model are project choices.
Full source register and earlier lifecycle review: [PI01–PI08](../research/plaid-ingestion-sources.md).

ISO currency/version and date rules follow [standards](standards.md). Provider JSON
and internal work requests are not labeled ISO 20022 messages. Applicable official
message mapping and accounting policy gates remain unchanged.

Named proof and remaining limits belong in the SQL, decoder and transport lane
runbooks and the [combined delivery record](../runbooks/sync-foundations.md).
#85/#87 own independent fault/security acceptance; #89 owns measured scaling.
