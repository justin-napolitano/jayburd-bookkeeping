<!-- stage-runbook: true -->
# Outbox and worker contract review

## Status and boundary

2026-10-13: #63 proposal under #57, coordinated with #70. No worker, SQL migration,
queue, runtime endpoint, provider call or cloud change implemented. Lease
W63-20261013T143600Z owns only background-work contract, sources and this runbook.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Authority | AGENTS/index, issue63, existing background contract and standards/source requirements read. |
| SQL review | Immutable outbox v1, post/reverse receipt and interactive-command boundaries retained. |
| Research | BW01–BW05 selected sections reviewed; technical facts separated from proposals. |
| Coordination | #70 shares connection generation/fence and requested/target/completed generation vocabulary; source revision is not book revision. |
| Validation | `make check test build` PASS with Go1.26.9: project/doc checks, immutable migration verification, runbook structure, vet, race-enabled existing tests and binaries. |

## Current evidence

[Contract](../contracts/background-work.md) proposes separate mutable dispatch/work
records, fenced effects/receipts, uncertainty handling and SQL-owned recovery.
[Source register](../research/background-work-sources.md) states review limits.
All new fault-test names are planned; current outbox existence is not delivery proof.

## Remaining work

#7 requires reviewed physical SQL/roles for its journal-observer consumer, concrete mandates,
retention/recovery and queue/pool settings before implementation. #70 owns provider
pagination/lifecycle. Root coordinator updates bibliography/index and issue status.

## PR handoff

Design-only with no generated OpenAPI or migration impact. Required commands:
`make check test build` using Go1.26.9 and
`bash scripts/check-stage-runbook.sh --staged`. No product/ISO/deployment readiness
claim. Preserve immutable outbox and scoped business authorization during implementation.
