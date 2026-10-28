<!-- stage-runbook: true -->
# Production migrator profile — #316

## Status and boundary

PROD316-MIGRATOR; build/316-production-migrator from217b2ac.
Source-only migration transport wiring. No managed SQL, DDL, IAM grants, resources,
secret access, provider calls or spending. Existing merged SQL unchanged.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |316 open;353/354 merged, no active conflicting lease; current contracts/source reviewed |
| Design | Explicit Production migrator login and exact target, canonical finite expiry, pre-I/O rejection; context bounds execution |
| Implementation | cmd/migrate flags/injected transport tests; clouddb Production migrator allowlist only |
| Validation | Core check/test/build and migration-test PASS; focused race and38 release tests PASS |

## Current evidence

Local Go1.26.5 with external temporary modfile/cache; tracked1.26.10 unchanged.
Disposable synthetic PostgreSQL migration suite passed; no managed database used.
Synthetic identifiers only. Invalid/offline commands construct no connector;
Production status/up select correct profile and the earlier expiry/ten-minute
deadline. Default Sandbox pool and Production verifier prohibition retained.
No actual Production login/bootstrap/runtime or migration35 apply established.

## Remaining work

Human source review; Production build workflow, migration Job/IAM/role bootstrap,
trusted evidence collection and actual337 mandate/hosted qualification. Context
cancellation cannot undo committed DDL; inspect actual status before any retry.

## PR handoff

Refs #316, partial stage C. No migration files or product API changes. Required
local tests and toolchain limitations recorded before handoff; no live authority.
