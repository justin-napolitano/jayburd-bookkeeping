# Plaid acquisition recovery

Issue #220; accepted owner scope, 2026-10-20: repair existing development Sandbox
acquisition without another enrollment. Parent #88 remains open for hosted proof.

## Evidence and decision

Observed: four delivery operations quarantined with `permanent_failure`; three
sets abandoned with `decoder_rejected`, one with `dispatch_quarantined`. HTTP
204 acknowledges a durable disposition, not complete publication. No original
provider response has been read; the precise rejected field is unproved.

Reviewed selected official [Transactions sync response and account-filter
sections](https://plaid.com/docs/api/products/transactions/#transactionssync),
2026-10-20, API request pin 2020-09-14. Account-filtered streams retain separate
cursors. Transactions can precede enrollment (default history request 90 days).
ISO currency validity must still cover the source date. A successful response is
not evidence that our bounded schema/date profile accepts it.

Keep strict material validation; do not guess the rejected field or relax money,
currency dates, source ownership, duplicate keys or account filtering. Persist a
closed decoder classification atomically with abandonment; expose only the safe
code and delivery disposition through authenticated stream status. Provider
values and unknown field names never enter diagnostics.

Alternative: blanket retry or reset quarantined rows. Rejected because it destroys
terminal evidence and can restart unknown effects. Explicit Request sync now may
instead release a known permanent failure only when SQL proves abandonment, no
completion and no live acquisition. Append an immutable recovery receipt; retain
the original operation, claim and attempts. Publisher creates a fresh operation
from the existing committed cursor. Unknown outcomes remain blocked.

The current actor/import/source/worker permissions and generation checks remain
mandatory. Existing idempotency receipts prevent a lost response or concurrent
retry from releasing another attempt. Old queue deliveries only acknowledge the
old terminal operation. No journal is posted by acquisition or recovery.

## Standards and storage

Reuse the mappings and limits in [Plaid decoder](plaid-decoder.md),
[dispatch SQL](acquisition-dispatch-sql.md) and [standards](standards.md).
ISO 4217 identifiers, minor units and source-date validity remain mandatory.
These local error/retry messages have no applicable ISO 20022 message definition;
they carry operational classifications, not a financial exchange. No audit export
or ISO conformance claim is introduced. Public vendor sections reviewed; paid ISO
normative text and real provider payloads were not accessed.

New integration-owned diagnostic/recovery sidecars use scoped keys and foreign
keys, immutable receipts, no raw payloads or credentials, and no direct runtime
table grants. Entry functions use current authorization and existing canonical
locks in serializable transactions. Migration 24 is additive; rollback must refuse
to erase recovery evidence. Terminal operation identities remain immutable.

Acceptance: safe decoder-code/atomic-abandon tests; manual recovery authorization,
same-key replay, unknown/live/completed refusal and fresh-claim SQL tests;
frontend quarantine versus queue state tests; complete browser/API/PostgreSQL
journey. Hosted acceptance requires completed generation, publication and reviewed
transactions, never deployment/HTTP acknowledgement alone. Exact compatibility
fix is contingent on the newly classified failure.
