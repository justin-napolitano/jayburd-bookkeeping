<!-- stage-runbook: true -->
# Connection operations SQL delivery

## Status and boundary

SQL foundation for #103/#104, lease OPS-SQL-103-104. Migration 14 only; prior migrations
remain unchanged. Public recovery discovery, stream operations and dedicated verified
notification intake share existing acquisition generations. No real provider, cloud,
secret retrieval, scheduler deployment or financial mutation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source review | Selected official Plaid verification/Transactions and PostgreSQL 17 isolation sections reviewed; contract labels project inferences. |
| Contract approval | Coordinator approved interfaces, individual stream pause, authoritative unfinished count, separate stream pin and scoped signal outbox before implementation. |
| Initial PostgreSQL proof | Migrations 1–14 and first four test groups passed using disposable actual API/sync/webhook/operator logins. |
| Defaults | `make check test build` PASS; `/private/tmp/jayburd-connection-ops-default.log`. |
| Final regression | All eight actual PostgreSQL groups PASS, including completion/signal race, nulls/revoked replay, expired binding vs unknown claim and concurrent fresh enrollment. |

## Current evidence

`internal/connectionops/operations_integration_test.go` runs only with explicit
`-connection-ops-proof` against synthetic connection_ops_template databases. Tests
use immutable exact literals; no credentials or provider payloads are loaded.

- Discovery/guard: same-actor scoped pagination, global unfinished count, expiry,
  exact-key recovery, fresh attempt rejection and revoked access.
- Pause/resume: held-set fencing, independent stream revision, unchanged binding
  approval, unaffected sibling, stale operation denial and atomic resume intent.
- Webhook receipt: same-JWT replay once, later same-body notification preserved,
  conflicting replay rejected, paused siblings excluded and current revocation.
- Catch-up/roles: bounded pagination, stable tick, restricted function ACL and mixed
  role denial. Actual acquisition remains owned by the existing runner.
- Completion race: simultaneous real database completion and notification preserve
  a later generation, with complete timestamp and unknown provider update time.
- Malformed/revoked replay: required nulls, old issue time, unknown Item, revoked
  source bind and wrong scope leave no partial receipt or wakeup.

Proof log: `/private/tmp/jayburd-connection-ops-proof.log`. Test names here do not
claim independent provider verification; the Go verifier lane owns that evidence.

## Remaining work

Coordinator integrates webhook Go, HTTP/API, UI, schema-version and role bootstrap
changes and runs combined regressions. Deployment, real ingress/secret policy,
message transport, scheduler/backoff and live qualification remain separate. The
new wakeup outbox is a durable intent, not a claim of delivered or completed work.

## PR handoff

Owned files: migration14 pair/checksum, new connectionops SQL tests, contract and
this runbook. Copied scripts/test-connection-ops.sh remains untracked; root owns it.
No push or external mutation. Defaults, stage guard and diff check required before
commit; final named evidence recorded here.

Expiry boundary proof preserves old paused Item/reference history while allowing a new
attempt only for known expired binding_review; expired unknown claims remain blocked.
Concurrent fresh Begin proves exactly one attempt wins and one receives P2804.

## Follow-up: existing lifecycle fixture ordering

The combined regression exposed an older test that created a fresh new enrollment
after deliberately leaving a rotation claim unresolved. Migration 14 correctly
rejects that sequence. The fixture now prepares and expires the unrelated unclaimed
attempt before starting the competing lifecycle claims. All private-context expiry,
public-history readability and exclusive-claim assertions remain unchanged; no
production guard is relaxed. Existing connection-state PostgreSQL proof PASS: 12 top-level groups and 5 named
subtests against migrations 1–14. Log: `/private/tmp/jayburd-connection-ops-legacy-proof.log`.
This fixture-only adjustment does not change the final migration checksum.
