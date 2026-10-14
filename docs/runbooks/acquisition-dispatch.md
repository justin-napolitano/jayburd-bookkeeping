<!-- stage-runbook: true -->
# Acquisition dispatch delivery

## Status and boundary

Implemented a bounded #7 dispatch slice on PR #108 head `8f64a78`. #108 remains
open with passing CI. This is a stacked branch, not merged predecessor evidence.
Default work uses synthetic credentials/providers and disposable PostgreSQL only.

## Stage ledger

- Reviewed project context, acquisition/worker contracts and current issue state.
- SQL and Go adapters have exclusive parallel lanes; root owns shared HTTP/queue
  profiles and integration. A third agent delivered the bounded HTTP host and independently reviews transport hazards.
- Refreshed official Cloud Tasks duplicate/order, OIDC and deadline documentation.

## Current evidence

- `make check test build`: PASS, including all-package race tests.
- `make sync-dispatch-test`: PASS, 21 groups and 9 subtests against migration 15.
  Includes private publisher HTTP → intercepted Cloud Tasks HTTP → private consumer
  HTTP → actual acquisition runner → restricted SQL and private evidence storage.
- `TestDispatchRealAcquisition`: exact `9007199254740993.01`, duplicate delivery,
  lost publication/settlement replies and current revocation; one provider fetch.
- `TestDispatchRealUnknownClaim`: ambiguous committed claim performs no provider work.
- `TestDispatchRealRetryHandoff`: real adapter 429 preserves 3,600 seconds durably,
  old delivery cannot fetch, new task runs after eligibility; no automatic journals.
- SQL tests prove initial demand, derived-claim live leases, eight-failure budget,
  unknown-outcome quarantine, expiry, immutable receipts, current authority and
  rejection of partial null state. `TestDeliveryCandidateFairness` prevents 30
  future/healthy operations from hiding later due work in a 25-item page.
- `TestAcquisitionProfileIsolation`, `TestAcquisitionHTTPRejectsBeforeWork`,
  `TestAcquisitionQueueProfile`: PASS. Existing observer routes remain isolated.
- `make web-check web-test web-build web-browser`: PASS; 244 unit tests and
  13 browser journeys. Loopback/Docker proofs required normal sandbox escalation.
- Eight ingestion/provider regression targets: PASS, 114 groups and 137 subtests;
  source/import, connection state/operations/service, webhooks, acquisition and flow.
- `make migration-test cloud-sql-test ledger-test observer-test sync-state-test`: PASS.
  Restricted-role/ledger/observer/sync suites: 146 groups and 286 subtests.
  The migration-status fixture now correctly restores version 15.

Independent review corrected exact deadline timestamps and due-candidate starvation.
Integration corrected shared test-flag registration; added explicit null-state
constraints and atomic next-task creation on durable retry handoff. Tests passed
for these corrections. Logs are local synthetic evidence under `/private/tmp/`;
CI runs the same committed proof scripts.

## Remaining work

Hosted executable/dependency composition, Cloud Tasks/Scheduler/IAM Terraform,
scoped Infisical writes and credential propagation, Link/reconnect hosting, real
OIDC/queue qualification, authorized #88 Sandbox proof and #90 live-pilot approval.
Quarantine reset needs an explicit operator contract. #89 owns measured scale.

## PR handoff

Local checks passed; prepare a PR targeting `build/connection-operations` while
#108 remains open. GitHub #65 owns assignments; release all four dispatch lanes
at PR creation. #7 remains open for hosted acceptance. No automatic merge, deploy
or live connection. Migration 15 adds sidecars/functions and five publisher/two
sync grants; migrations 1–14 and browser/public accounting contracts are unchanged.
