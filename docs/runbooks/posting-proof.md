# PostgreSQL posting proof

<!-- stage-runbook: true -->

## Status and boundary

Implemented synthetic manual identity/spot posting, journal reads, exact reversal
and trial balance under #4. #19 merged into #18's already-merged branch; this
branch carries that reviewed policy into main. No live accounts, public server,
production token verifier, setup API, close endpoint or background dispatcher.

## Stage ledger

| Stage | Result / evidence | Next action |
| --- | --- | --- |
| Carry reviewed contracts | Merge commit `c5107c9` includes #19 | Review implementation against ADR 0005 |
| SQL authority | Migrations 000001/000002; restricted functions and composite scope keys | Forward-only changes after merge |
| Go boundary | pgx 5.11.0 transactions and injected HTTP identity interface | Implement production identity separately |
| Fixture proof | `TestGoldenMonth`, `TestSecondBookIsolation`, `TestManualSpotPosting`, `TestCurrencyScalesAndRounding` pass | Full position accounting remains future work |
| Failure/concurrency | Write-stage/commit rollback, two blocked replay callers, close races, both revocation orders pass | Preserve tests with every accounting change |
| Recovery | Forward upgrade keeps seeded state; streamed synthetic restore replays receipt once | Live recovery objectives and cross-cluster role provisioning remain open |

## Current evidence

Observed locally with Go 1.26 and pinned PostgreSQL 17.11/migrate 4.20.1 images:

```sh
make check test build
make migration-test
make ledger-test
```

PASS default checks: doc/context links, vet, Go race tests, migration-file integrity,
and tooling builds. PASS pinned actionlint 1.7.7 on the modified workflow. PASS migration fresh install/replay, transactional failure and
dirty-state rejection. PASS integration suite and restore replay. CI runs the same
three targets. The Go adapter reserves one second of its five-second budget for rollback cleanup.
The Linux integration binary is built without CGO/race instrumentation
and executes on an internal Docker network; ordinary Go tests run with `-race`.
No database port is published, no inherited DSN or credentials are consumed, and
only resources created by each invocation are removed. Both database and test
containers mount only the required migrations, test executable and synthetic corpus.

`TestPostingInvariants` injects AFTER-write failures at book, header, line, audit,
receipt and outbox, plus a deferred COMMIT failure; all state rolls back.
`TestCommitAcknowledgementLoss` / `TestHTTPPostCommitBoundary` drop a real commit
acknowledgement: response is 503, then original receipt replays without duplication.
`TestSchemaPrivileges`, `TestTransactionIsolation`, `TestEntityReferencesAndPrivileges`,
`TestReceiptActorAndArchive` and `TestSchemaCompatibility` cover grants, scope,
revocation, archival and setup bounds. Remaining tests map to the
[acceptance matrix](../contracts/posting-boundary.md#postgresql-acceptance-matrix).

For local execution, only nonsecret build caches were set:
`GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache`.
Before commit run `bash scripts/check-stage-runbook.sh --staged`.

## Remaining work

Trusted user identity and controlled setup are required before exposing a server.
Real-LLC policy, ASC review, opening balances, accountant exports/cutover, close
review evidence, live rate selection, settlement/remeasurement positions and outbox
delivery remain open. Test-only close races do not implement the close workflow.
Restore proves this synthetic database with roles already provisioned, not a live
recovery SLA or complete disaster-recovery procedure. No scale benchmark is claimed.

## PR handoff

Review migrations, SQL privileges/locks, exact calculations and failure tests.
Down migrations intentionally fail; repair financial history forward. Runtime
cannot perform setup or table DML. Owner/superuser remains trusted administrative
authority. Keep broader #1/#2/#3 gates open; this is the bounded development proof,
not official-bookkeeping approval. Source review is recorded as P05/P06.
