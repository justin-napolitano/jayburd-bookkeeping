# Reporting comparisons, drill and review exports

<!-- stage-runbook: true -->

## Status and boundary

Build on merged #76 (`0aaad5a`) toward #51/#53. Add comparison controls, pinned
GL-line journal detail and internal review CSV. No audit-export mapping, live
policy approval, worker runtime, provider or cloud action.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Dispatch | Three isolated agents and exact paths in #65; no competing PR | Review contracts before implementation |
| Contract review | Root approved fixed long CSV measure rows, strict field grammars and full-selection limits; pinned journal selector checked in one transaction | Implement independently |
| Drill integration | `e746192` integrated as `c9fc507`; migration 7 alone adds restricted journal selector wrapper | Combined migration/ledger checks |
| Transport | Required revision/selector queries, bounded CSV framing and OpenAPI; focused transport tests passed | Integrated |
| UI/export integration | UI `231678c`, CSV `abca33f` integrated; independent transport/export review fixes applied | Complete |
| Combined verification | Default, migration, ledger, cloud-SQL, web and full-stack synthetic gates PASS | PR review |

## Current evidence

Combined `make check test build migration-test ledger-test cloud-sql-test` passed.
`make web-check web-test web-build web-browser` passed: 148 unit tests and nine
browser journeys. `make app-integration` passed with actual CSV download bytes,
comparison values and pinned journal detail. Existing review ports were occupied;
the harness used 39181–39183, then all three harness files were restored exactly.
Identity remains a synthetic test fixture.

`TestReportReviewHTTPPostgres` checks all four CSV summaries against independent
golden-month expectations, journal membership, stale revisions and revoked access.
Export SQL tests cover 101 committed cent journals, pagination, FX/reversals and
authorization changes; the unchanged thousand-posting report proof also passes.
A rejected fixture account UPDATE was replaced with a dedicated inserted account;
ledger immutability was preserved. Lane runbooks contain focused evidence.

Initial frontend CI exposed an existing auth-fixture timing race: the test entered
credentials before the new session reset effect flushed. The enrollment test now
awaits React configuration/effect setup before input; production auth is unchanged.

Root reviewed
RFC 4180 section 2 and OWASP CSV Injection guidance on 2026-10-13, alongside
FR03/FR05. The selected profile omits arbitrary labels and serializes exact sign
and magnitude separately. Quoting alone is not formula protection, and spreadsheet
numeric import is not promised lossless. See the export contract for precise fields.

## Remaining work

PR #77 is open; coordinator releases agent leases in #65. Full subtotal/history drill and normative
accountant exports remain outside this bounded slice. CSV is a bounded internal
review profile, not an ISO accountant-exchange conformance claim.

## PR handoff

PR: https://github.com/justin-napolitano/jayburd-bookkeeping/pull/77.
Combined review and validation passed; review before merge. Existing migrations
1–6 are unchanged.
Migration 7 is forward-only; runtime and schema must be released compatibly.
