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
| Implementation | cmd/migrate explicit profile; dedicated manual Production build workflow and exact-hash public target/receipt helper |
| Validation | Core check/test/build and migration-test PASS; focused race and38 release tests PASS |

## Current evidence

Local Go1.26.5 with external temporary modfile/cache; tracked1.26.10 unchanged.
Disposable synthetic PostgreSQL migration suite passed; no managed database used.
Synthetic identifiers only. Invalid/offline commands construct no connector;
Production status/up select correct profile and the earlier expiry/ten-minute
deadline. Default Sandbox pool and Production verifier prohibition retained.
No actual Production login/bootstrap/runtime or migration35 apply established.
Manual Production workflow is source only: no target file exists and no dispatch
occurred. Production WIF/registry setup and reviewed target remain required.
Publication must pass local runtime tests/scan before cloud auth and recheck expiry
before auth and push; receipt generation retains independent image binding.
Public target/receipt tests and YAML syntax PASS. Final make check/test/build
and release-test PASS. Local non-root/no-network migrator container verifies
offline and rejects down: PASS. Host-built binary fallback proves runtime only;
canonical source-builder remains a CI gate. Migration-test passed before the
workflow addition; migration engine/SQL remained unchanged afterward.

## Remaining work

Human source review; actual build target/builder federation, migration Job/IAM/role bootstrap,
trusted evidence collection and actual337 mandate/hosted qualification. Context
cancellation cannot undo committed DDL; inspect actual status before any retry.

## PR handoff

Refs #316, partial stage C. No migration files or product API changes. Required
local tests and toolchain limitations recorded before handoff; no live authority.
