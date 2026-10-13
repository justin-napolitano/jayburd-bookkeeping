<!-- stage-runbook: true -->
# Financial reporting contract review

## Status and boundary

2026-10-13: proposed design for #51 under epic #50. No SQL migration, endpoint,
OpenAPI operation, export, cloud change or real-data use is implemented here.
Lease REPORTS-51-20261013T134933Z covers only the reporting contract, this runbook
and its source register; coordinator owns shared index and issue updates.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Existing authority reviewed | Issue #51; AGENTS, index, standards/decision evidence, ADR 0005, multicurrency, migration 1/2 reporting/auth/schema and fixed corpus. |
| Research | FR01–FR05 selected public sections reviewed; normative audit/GAAP gaps retained. |
| Contract | Scoped SQL/API proposal, snapshot/revocation semantics, exact statements/CSV, independent constants and named planned tests. |
| Validation | `make check test build` PASS with Go 1.26.9: project/doc links, immutable migration checks, stage-runbook structure, vet, race-enabled existing tests and all four binaries. |

## Current evidence

[Contract](../contracts/financial-reporting.md) is the authoritative proposed behavior;
[source register](../research/financial-reporting-sources.md) states review limits.
Current SQL authorization uses row locks and cannot run in a READ ONLY transaction.
Chosen continuation protocol fails on book revision changes instead of claiming
historical metadata snapshots. No listed new report test is implemented by this PR.

## Remaining work

Owner reviews contract/applicability; #53 implements SQL, tests, OpenAPI, generated
clients and integration in that order. #1/#10 retain real policy and audit-export
mapping approval. Measure query/lock budgets; generated totals are not proof.
Integration review clarified revision zero and per-measure CSV sign fields.
The exact ordered CSV header and row-kind/nullability/measure mapping remain a
required review gate before export implementation; this is not a complete wire
schema. Root coordinator links sources/index and records the lease handoff.

## PR handoff

Design-only; no migration or generated OpenAPI impact. Required local commands:
`make check test build` with pinned Go 1.26.9. Stage this runbook with every commit
and run `bash scripts/check-stage-runbook.sh --staged`. No browser/API runtime
contract changed; proposed paths remain unadvertised until implementation.
