# Migration tooling and database design review

<!-- stage-runbook: true -->

## Status and boundary

JB-003 partial slice: golang-migrate tooling and SQL-first design standard.
No application schema, ORM, cloud deployment or real financial data. No live DSN
accepted. JB-001/JB-002 still gate the first ledger schema.

## Stage ledger

| Stage | Result | Next action |
| --- | --- | --- |
| Select tool and design sequence | v4.20.1 release and image digest verified | Validate scripts |
| Add migration workflow and isolated fixture proof | PASS locally | Publish PR and verify CI |

## Current evidence

- `GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache make check test build`: PASS including the shell fault suite.
- `bash scripts/test-migration-files.sh`: PASS for checksum drift, missing pair,
  duplicate version, unfinished template, resealed-history and out-of-order denial.
- `bash scripts/migrate.sh check main`: PASS.
- `bash -n scripts/migrate.sh scripts/test-migrations.sh scripts/test-migration-files.sh`: PASS.
- `make migration-test`: PASS with PostgreSQL 17.11 and migrate v4.20.1 pinned images.
  Fresh/replay, synthetic down/up, transactional DDL rollback and dirty-state
  rejection verified. Product-schema execution explicitly SKIPPED (none exists).
- Sandbox denied the default Go cache and Docker image-pull access. Writable
  temporary Go caches and approved Docker execution resolved these limitations.
- `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/seed.yml`: PASS after approved registry access.
- Remote CI pending at this stage.

## Remaining work

Review accounting policy and first physical schema. Add application driver/query
generation, runtime compatibility tests, deployed-checksum receipts and cloud
migrator identity later. This slice does not complete JB-003.

## PR handoff

Review migration safety and the SQL-first contract sequence. Next: JB-001 policy
and JB-002 table/transaction review before the first product migration.
