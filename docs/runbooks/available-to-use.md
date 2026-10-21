<!-- stage-runbook: true -->
# Available-to-use proposal delivery

## Status and boundary

2026-10-21. Bounded #251 design work under lease
`PLAN-CONTRACT-251-20261021T2325Z`, branch `issue/251-available-to-use-contract`.
No SQL, API, UI, automation, service grant, cloud or real financial data.
Contract remains proposed pending owner/accountant review and #248/#1/#56.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Authority | Read AGENTS/index, live #251 and open #248/#1/#56, accounting/financial-reporting/multicurrency/identity and standards contracts. |
| Research | [ATU-01–07](../research/available-to-use-sources.md) give selected portions, source limits and design inferences. |
| Contract | [Proposal](../contracts/available-to-use.md) compares models, defines provenance and unknown states, and fixes independent synthetic expected outcomes. |
| Validation | `git diff --check` passed. `make check test build` blocked at pinned Go 1.26.10 download; default proxy socket disconnected, then approved alternate proxy connection reset. Installed Go is 1.26.5 and cannot satisfy `go.mod`. No product tests ran. |
| Navigation | Added index and bibliography pointers to the bounded proposal and reviewed source record. |

## Current evidence

The proposal separates a posted cash input from a dated scenario estimate.
The fixed examples include sufficient, committed, uncertain, multiple-currency,
stale, negative and timing cases. No calculation has run against a ledger.
The source record relies on the existing reviewed bibliography and explicit
public ISO access limits; it makes no conformance claim.

## Remaining work

Owner/accountant selects eligible cash, purpose, obligations/reserves, timing,
uncertainty and foreign-currency treatment. #248 must approve Position semantics;
#1 must approve real-entity accounting policy; #56 must define any service
permission. Then split implementation into bounded SQL/API/UI and authorization
issues with fixed synthetic test oracles. No child implementation issue is
created from this unaccepted model.

## PR handoff

Design-only; no migration or generated OpenAPI impact. Retry `make check`,
`make test`, `make build` with pinned Go 1.26.10 when available; run staged
runbook gate before commit. Report exact results and commit SHA to coordinator.
Do not close #251 as completed until
owner-reviewed acceptance evidence exists.
