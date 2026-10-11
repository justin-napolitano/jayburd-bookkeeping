# PostgreSQL posting proof

<!-- stage-runbook: true -->

## Status and boundary

Owner authorized implementation after #18/#19 merged. #19 merged into #18's
already-merged branch; this branch carries that reviewed policy into main.
Work under #4: synthetic PostgreSQL manual identity/spot posting, reads, reversal
and trial balance. No live accounts, public server, close endpoint or dispatcher.

## Stage ledger

| Stage | Result | Next action |
| --- | --- | --- |
| Verify merges | #18 in main; #19 merged into former parent | Carry reviewed contracts into implementation branch |
| Read contracts | ADR 0005 and posting boundary selected | Implement SQL authority, then Go transaction adapter and tests |

## Current evidence

Docker available; contract/source review complete. Implementation checks pending.

## Remaining work

Migrations, restricted operations, Go adapter, disposable database integration
suite, failure/concurrency/restore proof, HTTP adapter tests and documentation.

## PR handoff

Keep broader #1/#2 policy and workflow gates open. No production readiness or
accounting/ISO conformance claim. Each implemented guarantee requires named tests.
