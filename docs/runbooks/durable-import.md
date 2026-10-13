# Durable import design review
<!-- stage-runbook: true -->

## Status and boundary

Design-only #8/#61/#70 slice; lease IMPORT-20261013T175000Z, branch
design/durable-import. The [contract](../contracts/durable-import.md) proposes
durable CSV/source identity, scoped SQL, transport, recovery and review boundaries.
No migration, endpoint, worker, cloud or provider behavior is delivered here.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source review | Existing decoder/import/provider/accounting/standards contracts, fresh issue snapshots and DI01–DI03 primary sections reviewed 2026-10-13. |
| Design | Registered namespaces and full-field equality preserve identical purchases; separate import, draft and explicit atomic source-post gates. Source/review revisions remain distinct from ledger book revision. |
| Coordinator review | Corrected object ownership to NOLOGIN bookkeeping_owner and clarified that changing statement_ref cannot bypass correction rejection. Required list source scope and posted-group merge invariants clarified. |
| Validation | `make check test build` PASS on 2026-10-13; named contract tests are future requirements. Staged runbook guard PASS before commit; final prose/contract review corrections included. |

## Current evidence

Document/source review and existing repository checks are complete.
`make check test build` passed with the pinned Go 1.26.9 toolchain; log
`/private/tmp/jayburd-durable-import-check.log`. These existing tests do not
implement or prove the proposed import acceptance matrix. Coordinator owns shared index and
bibliography integration. Review-state names were sent for the synthetic queue
prototype; that prototype must not imply working import endpoints.

## Remaining work

Accept physical/API design, review append-only matching/current-assignment invariants,
permission provisioning, object-store publication/cleanup and retention review.
Implement and prove named tests before claiming durable imports or posted linkage.
Provider/ISO profile, live-data permission and accountant cutover remain separate.

## PR handoff

Only docs/contracts/durable-import.md and this runbook are owned by this lane.
Coordinator integrates review and opens PR; no GitHub push or cloud action here.
