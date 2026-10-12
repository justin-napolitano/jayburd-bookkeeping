# Migration-only release
<!-- stage-runbook: true -->

## Status and boundary

Continue #26/#27 after #44 merged at `6e6dd80`. SQL bootstrap/replay succeeded;
product migrations have not run. Full application promotion requires #26 closed,
so this stage needs an independently reviewed migration-only operation.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Build | Run 38289669417: validation, API and migrator pass; web scan fails before publication | Rebuild all three from merged repair; no partial release |
| Implementation | Existing hash-approved workflow now supports a separate migration manifest and immutable execution receipt | Review without weakening promote/rollback gates |
| Tests | 14 Node release tests and synthetic shell orchestration pass; repaired web image scan/runtime proof pass | Review PR, then run hosted build |

## Current evidence

- PASS: `make check test build release-test`, workflow actionlint and shell syntax.
- PASS: `make migration-test`: disposable synthetic fresh install/replay,
  down/up and failed-DDL rollback/dirty-state rejection, plus product SQL replay.
  Synthetic orchestration proves migration-only never reaches either runtime
  Terraform or service readback; failures stop promotion and rollback skips SQL.
- PASS: hosted build validation and API/migrator build/test/scan/publication. This
  is the first observed builder federation/publication success, not deployment
  federation or negative IAM trust testing.
- BLOCKED: web HIGH/CRITICAL scan exited 1. No web image was published and no
  complete release is approved. The original failed job did not retain its JSON
  report; this change adds a separate seven-day rejected-scan artifact, with the
  same blocking scan and publication order.
- PASS: private source-only reproduction located all 13 Node findings inside
  bundled npm, plus OpenSSL CVE-2026-14456 in libcrypto3/libssl3 3.5.7-r0.
  Runtime now excludes npm/Yarn and requires OpenSSL packages >=3.5.8-r0.
  Rebuilt image passed Trivy 0.75.0 HIGH/CRITICAL scan (including unfixed) and
  `scripts/test-web-container.sh`: non-root, read-only, static assets, denied
  finance routes, fail-closed identity config and synthetic runtime config.
  This private proof is not a hosted candidate receipt. No migration job or
  runtime service has been created or executed.
  Deployment federation remains disabled.

## Remaining work

Validate scoped manifest, prerequisites and execution receipt; preserve full
promotion gates. Prepare an exact-digest release only after the complete build
passes. Owner reviews the manifest before dispatch. No source credentials, state,
plans or private provider logs enter artifacts.

## PR handoff

Keep #26 open. Migration-only success will not prove API denial, identity enrollment,
restore, pool recovery or live bookkeeping readiness.
