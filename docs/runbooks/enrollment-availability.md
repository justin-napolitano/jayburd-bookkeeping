# Optional enrollment availability delivery
<!-- stage-runbook: true -->

## Status and boundary

#281; EXPIRY281-20261024T0515Z; fix/281-expired-enrollment from c1cc86e.
Synthetic implementation only. No consent renewal, provider/cloud action, SQL
migration or private data. [Decision](../contracts/enrollment-availability.md)
separates malformed config from expired optional authority. Sensitive authority
changes require review before release; no live pilot gate is cleared here.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Current #90/#281/#132/#65, current main, bibliography, connection-release/command-recovery and actual app recovery reviewed |
| Design | Strict deadline, no cold expired secret I/O, provider-entry/transport fences; preserve SQL metadata/local controls and unrelated command recovery |
| Implementation | Structural validation accepts well-formed expired optional metadata; real runtime composes SQL-only service without static reads; managed provider entries and dependency HTTP requests use strict deadline fences |
| Core validation | `make check test build` PASS; deterministic before/at/after/cold restart/during initialization/running service and malformed expired config tests PASS |
| Web validation | Locked `npm ci`, installed locked Chromium; web-check/test/build/browser PASS:326 unit tests and24 browser tests |
| Actual synthetic full stack | `make app-integration` PASS: real browser -> production Next bridge -> Go -> restricted disposable PostgreSQL; expired enrollment returns503; dropped setup/post commit responses recover same commands; exactly one explicit journal |
| Additional SQL regressions | `make connection-service-test connection-ops-test` PASS against disposable PostgreSQL, including revoked/scope/claim/recovery/stream-control regressions |

## Current evidence

Existing incident: expired optional enrollment metadata rejected before server
startup. Existing constructors load static secrets before composing routes.
SQL-only discovery/status controls retain independent authority. Named proofs:
`TestExpiredEnrollmentStartup`, `TestEnrollmentExpiresDuringStartup`,
`TestEnrollmentExpiresAfterStartup`, `TestApprovalTransportDeadline`,
`TestExpiredApprovalDeniesProviderEntries`,
`TestExpiredApprovalPreservesSQLAuthority`, `TestApprovalDeadlineTransition`.
The synthetic expired runtime uses dependency tripwires, not a fake successful
bank operation. The first full-stack attempt exposed an overly specific test
expectation for an intentionally sanitized bridge error; only that expectation
was corrected, then the complete journey passed. Go vet also caught an unkeyed
scope literal in a new test, corrected before final passing core gates.

Local cached Go1.26.5/modfile override, tracked1.26.10 unchanged. No new hosted,
independent provider financial equality or real-account evidence claimed. No
ledger/DDL behavior changed, so no new migration or accounting-policy test is
substituted for existing acceptance. Bibliography remains within32KiB.

## Remaining work

Complete staged runbook check and PR review/CI. Hosted release/acceptance remains separate scope, including actual
revision outcomes; #281 stays open until that evidence is recorded.

## PR handoff

Refs #281/#88/#90. No migration or OpenAPI change anticipated. Review exact
authorization boundary and complete synthetic evidence before release; this
slice does not complete the broader Sandbox or live-account gate.
