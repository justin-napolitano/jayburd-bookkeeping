# Durable synthetic import delivery

<!-- stage-runbook: true -->

## Status and boundary

Implemented #92/#93/#94: synthetic staging → validated durable preview → explicit
recoverable import into immutable observations and review items. No automatic
journals, provider calls, cloud deployment or live-data execution. Planning #91
merged into still-open #83; this delivery stacks on that branch.

## Stage ledger

- Reviewed ingestion, source authority and command recovery contracts; #8 remains
  parent for the complete user workflow. Exclusive leases recorded in #65.
- Integrated independent properties b760fea, Go orchestration 109fc78 and SQL
  authority bad6842. Migration 10 has one writer; migrations 1–9 unchanged.
- Root integrated command wire types, safe recovery UI, CI, readiness fixtures and
  database-reported unknown-commit handling. Independent review found no blockers.
- Combined synthetic validation passed on 2026-10-13. Existing demos preserved;
  app proof used temporary ports 39581–39583, restored byte-for-byte afterwards.

## Current evidence

- `make check test build`: PASS, including migration integrity and default tests.
- `make migration-test ledger-test cloud-sql-test observer-test
  source-registry-test import-service-test app-integration`: PASS on disposable
  PostgreSQL. Includes forward preservation, restricted roles, readiness,
  actual lost-commit recovery, source isolation, replay and injected rollback.
- Source SQL proof: 21 groups / 47 subtests, including eight durable-import groups,
  1,000-row atomic imports and 1,001-row rejection. See [SQL proof](import-sql.md).
- Import service proof: seven PostgreSQL/filesystem plus ten unit groups; storage
  outside SQL lifetime, reauthorization, actual lock timeout and lost COMMIT reply.
  See [service proof](import-service.md). Confirmed imports create no journals,
  outbox events or book revision changes.
- `make web-check web-test web-build web-browser`: PASS, 153 unit tests and
  11 browser journeys. Full app integration: one real browser/Next/Go/PostgreSQL
  setup/post recovery journey PASS; this is not a browser upload/import journey.
- Independent decoder oracle: 7,726,993 fuzz executions across two 60-second runs,
  with fixed USD/JPY/KWD assertions; limits in [property proof](import-properties.md).
- Actionlint 1.7.7 and whitespace checks: PASS. Initial sandbox Docker/local-listener
  restrictions required escalation; combined authorized rerun passed.

Root fixed SQLSTATE 08007/40003 commit uncertainty: suppress unconfirmed results,
recover the same command, never mint another key. Named real regression:
`TestCommandDatabaseUnknownCommit`. Reviewed PostgreSQL 17
[Appendix A](https://www.postgresql.org/docs/17/errcodes-appendix.html), classes 08/40
on 2026-10-13; conservative recovery is our implementation inference.

Local logs: `/private/tmp/jayburd-combined-import.log`,
`/private/tmp/jayburd-combined-db.log`, `/private/tmp/jayburd-import-browser.log`.
The first log ends at the initial sandbox Docker restriction; the second contains
all successful database/application gates. SQL up checksum:
`214350cacf0b69dd0c3273f756a1d373ae016a828a5cfbf4d631ddd74319c8b8`.

## Remaining work

Expose authenticated upload/preview/review routes and UI, classification/drafts
and source-linked posting. Hosted storage, orphan retention/deletion, provider
lifecycle, reconciliation and measured scaling remain separate gates. The recovery
panel cannot display reviewed import rows yet, so it permits cancel/acknowledge
but disables import execution; no blind confirmation is offered.

## PR handoff

Ready for review after staged runbook guard. Migration 10 adds durable import
objects and extends the existing command gate; no merged migration is edited.
Close bounded #92/#93/#94 on merge; keep #8 and broader #84–#90 quality gates open.
Merge #83 first and retarget this stacked PR to main. No deployment or live-use
approval is implied by these synthetic proofs.
