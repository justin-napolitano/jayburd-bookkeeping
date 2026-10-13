# Local observer runtime adapter

## Decision and scope

Implemented the receipt-only Go adapter for the nine functions in
[worker SQL](worker-sql.md), issue #7. SQL remains the sole authority for
membership, journal/event identity, retries, fences, leases and receipts. No HTTP,
OIDC, Cloud Tasks SDK, credential access, deployment or financial mutation is
included. The [background-work](background-work.md) STD-01 internal-wakeup
assessment applies unchanged; no ISO financial or audit exchange conformance.

## Owned interfaces and budgets

`Beginner.BeginTx(ctx, pgx.TxOptions)` supplies a configured database connection.
Separate `Publisher` and `Consumer` constructors receive different role-bound
beginners. A publisher also receives one explicit binding UUID and an injected
`Transport.Enqueue(ctx, Task)`. A task contains only deterministic hexadecimal
name, target version 1 and a `work-wakeup-v1` envelope. Target URLs and identity
configuration belong to the future transport adapter, not database data.

Publisher exposes Discover, Candidates, DispatchClaim, DispatchResult, Recover
and Tick; Consumer exposes Claim, Heartbeat, Finish, Fail and Handle. Every SQL
call uses SERIALIZABLE, transaction-local 5s statement/1s lock timeout, an overall
6s call deadline, immediate COMMIT and at most three full retries for 40001/40P01.
Cleanup has its own bounded 1s context; idle transaction timeout is set to 6s. Validate session/current-role equality,
exact capability and absence of owner/API/migrator/other-worker capability and
privileged role flags inside each transaction; direct application-table privileges are rejected. SQL validates enrolled session login
and current scoped authority. No caller-supplied principal is accepted.

Tick has a 60s overall deadline, discovers at most 25 facts, enumerates at most 25
operation UUIDs and processes them sequentially: Recover → DispatchClaim → COMMIT
→ Enqueue (at most 10s) → DispatchResult → COMMIT. No transport call holds a SQL
transaction. Queue result codes are accepted/already_exists/timeout/transient/
rejected. Errors map to fixed codes; arbitrary transport/SQL text is never returned.

Handle uses the same 60s bound and validates the exact three-key wakeup before SQL:
schema_version numeric 1, canonical lowercase operation UUID and positive signed-
bigint-range dispatch_generation string, maximum 1024 bytes, no duplicates,
unknown fields, nulls or trailing JSON. It commits Claim before invoking Finish.
SQL done/observed becomes completion only after successful COMMIT; busy/not_due
remain retryable. Heartbeat/Fail are explicit primitives; receipt-only Handle has
no external business step or fabricated failure receipt.

## Unknown commits and metadata

An unclassified COMMIT error is `commit_unknown`, never automatic transaction
replay, completion, or permission to enqueue. A subsequent Tick/Handle consults
SQL again: completed work returns done; an uncertain claim remains busy until
SQL recovery accounts for abandonment. Explicit server 40001/40P01 may retry.
Neither queue acknowledgement nor a prepared SQL result proves observation.
Validate every returned status, nonnull/null metadata shape, positive generation/
fence, bounded candidates and fixed target version. Before enqueue independently
recompute SHA-256(operation UUID|dispatch generation|1); reject digest mismatch.

## Evidence and sources

Existing BW02–BW04 and WS01–WS03 source sections cover duplicate delivery, task
identity, transactional locks and current-time leases. P05 in
[posting sources](../research/posting-sources.md) reviews pinned pgx transactions;
this adapter uses the repository's pgx/v5 v5.11.0. Unit fault tests and disposable
PostgreSQL adapter/fake-queue integration cover role separation, rollback,
whole-transaction retries, commit-loss suppression, no queue call before commit,
duplicate/reordered delivery, revocation, expiry and context cancellation.
Actual Cloud Tasks/OIDC/network-response-loss proof remains a separate gate.

**OR01**, PostgreSQL Global Development Group, PostgreSQL 17
[Appendix A, Error Codes](https://www.postgresql.org/docs/17/errcodes-appendix.html),
classes 08, 23, 25 and 40 reviewed 2026-10-13. Codes 08007 and 40003 explicitly
name unresolved transaction/statement completion; the presence of a `PgError`
alone does not establish rollback. Our conservative policy treats unclassified
COMMIT errors as unknown. Only explicit serialization/deadlock rollback,
constraint/owned-trigger rejection, failed-transaction state and pgx's explicit
commit-rollback sentinel count as confirmed failure. Only serialization/deadlock
is retried. Unit tests assert no enqueue, replay or exposed claim metadata for
both ambiguous SQLSTATEs. This code table is mechanism evidence, not a network
failure simulation or an exactly-once transport guarantee.
