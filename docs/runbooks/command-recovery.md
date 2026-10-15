# Command recovery and application integration

<!-- stage-runbook: true -->

## Status and boundary

Implemented for #9 on `feat/command-recovery`, based on merged #22/main56d7c38.
Scope: durable reviewed setup/post recovery and real browser/API/SQL synthetic
proof. No cloud, credentials, real accounts or provider authentication calls.

## Stage ledger

| Stage / owner | Work | Evidence / next |
| --- | --- | --- |
| Coordinator | Research, shared contracts, API generation, CI and review | Source sections reviewed; contract written before implementation |
| SQL/API agent | Migration5 and durable command transitions | PASS real PostgreSQL lifecycle, concurrency, revoked scopes and rollback |
| UI agent | Register/recover/execute/acknowledge workflow | PASS type/build, 36 unit tests and six mocked browser journeys; stale-session completions ignored, denied recovery clears views |
| Integration agent | Real Next rewrite, Go service and disposable PostgreSQL | PASS dropped post-commit response, remount recovery, exact database counts |
| CI portability | Use portable `grep -qi` for owned readiness headers | Hosted run38174592229 failed because `rg` was unavailable; replacement hosted verification pending |

## Current evidence

Use [recovery contract](../contracts/command-recovery.md) and its annotated sources.
Observed with Go1.26.9, Node24.18.0, locked web dependencies and Chromium:

- PASS `make check test build migration-test`: race tests, vet, checksums,
  immutable history checks, fresh migration/replay and intentional failure rollback.
- PASS `make ledger-test`: full synthetic accounting/regression/restore and
  readiness suite, including `TestCommandLifecycle`, `TestCommandConcurrency`,
  `TestCommandLostAcknowledgement`, `TestCommandValidationAndRollback`,
  `TestCommandScopeIsolation`, `TestCommandExecutionValidation`, authorization
  and `TestCommandHTTPBoundary`.
- PASS `make web-check web-test web-build web-browser`: generated types,
  TypeScript, 36 units, production build and six mocked recovery journeys.
- PASS `make app-integration`: real SQL commit followed by dropped HTTP response,
  component remount and pending discovery without re-execution. SQL asserts one
  journal, receipt and outbox message, two acknowledged committed commands and
  two execute audit records. Missing bearer and foreign origin are rejected.
- PASS OpenAPI lint (existing license warning only) and actionlint.
- PASS after the CI portability fix: `bash -n scripts/test-app-integration.sh`
  and `make check test build` with Go1.26.9. Full hosted integration rerun pending.

The integration harness uses a test-only identity and rendered bookkeeping
component; the production Next rewrite and Go/SQL boundaries are real. It does
not prove managed sign-in, MFA or deployed browser login. A bounded raw HTTP
bridge over `docker exec` supports local daemons without host port publishing;
no database port is exposed. All disposable resources are cleaned up.

## Remaining work

Final real integration rerun passed against the hardened UI; ready for PR review.
Hosted [run38174592229](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38174592229)
failed before browser execution: Ubuntu lacked `rg` for the readiness header
check. Both header checks now use `grep -qi`; hosted verification of this fix
remains pending.
Hosted first-book acceptance later identified an inaccessible unacknowledged
command. Issue #133 adds explicit payload-free clearing in migration16; its current
evidence and deployment status live in
[the focused runbook](inaccessible-command-recovery.md).

## PR handoff

One branch/runbook, exclusive agent file ownership, coordinator commits only.
No production authentication bypass; synthetic auth lives only in test artifacts.
Merged migrations1–4 stay immutable. No live deployment or bookkeeping approval.
Migration5 remains immutable. One pending command per principal deliberately
serializes interactive work across tabs; legacy direct callers retain their
existing idempotency obligations. Migration16 adds an explicit owner action that
clears only an inaccessible command without reading its payload or changing books.
