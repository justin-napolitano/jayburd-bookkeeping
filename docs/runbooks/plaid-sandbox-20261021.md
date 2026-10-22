<!-- stage-runbook: true -->
# Plaid qualification pass — 2026-10-21

## Status and boundary

Issue #88; lease `PLAID-88-20261021T2358Z`; source baseline
`6a921869f7af22e15ade0cd567c9114b34739075`. The owner requested connecting
with Plaid and running the recommended tests. This pass uses fictional Sandbox
accounts and disposable synthetic databases. It does not authorize production
accounts, deployment, IAM changes, Scheduler activation, or automatic posting.
No agent retrieves credentials, raw provider responses, or private records.

## Stage ledger

1. **Readiness review — complete.** Rechecked GitHub main, #88/#85/#87/#90,
   #65 leases, project index, bibliography, sync quality plan, Sandbox
   qualification contract, Hosted Link and connection-release runbooks. The
   current application release contains loan/card enrollment. Its release runbook
   records a paused Scheduler; current metadata readback below supersedes that
   historical state. The separate source-first preview is a fixture and cannot
   connect to Plaid.
2. **Owner sign-in — complete.** The owner completed managed sign-in in Edge.
   No owner password or authenticator code was collected by the agent.
3. **Synthetic regression — complete with one volume failure.** Core and
   provider suites passed. The final 1,000-command browser volume check timed
   out; results below distinguish this from hosted qualification.
4. **Hosted API smoke — partial, expected boundary.** The existing diagnostic
   authenticated to Plaid, exchanged a disposable fictional Item, decoded initial
   and repeat sync, and removed that Item. Details below.
5. **Hosted application journey — partial with checking failure.** Interactive
   fictional Link and four reviewed bindings committed. Card, mortgage and
   student acquisition completed; checking failed exact decoding and published
   nothing. Card repeat acquisition completed. Read-only review evidence below;
   no draft creation or posting was attempted.

## Current evidence

| Check | Result |
| --- | --- |
| `make packet ID=JB-001` | PASS; seed only, no scheduling authority. |
| `make check test build` | PASS; all-package race tests, migration-file checks and binary builds. |
| Locked `npm ci` | PASS; 233 packages, zero reported vulnerabilities. |
| `make web-check web-test web-build web-browser` | PASS; generated API/type checks, 286 unit tests, production build and 17 Chromium component browser tests. |
| `make migration-test ledger-test sync-state-test connection-state-test acquisition-test connection-service-test provider-flow-test connection-ops-test webhook-test sync-dispatch-test sandbox-qualification-test` | PASS with cached pinned images. Includes migration 27 fresh/replay/down/up, guarded historical-currency rollback, golden month/restore, atomic publication, fence/wakeup races, one-shot exchange, exact money, dated-currency acquisition, account isolation, provider-to-review flow, signature/receipt rejection and dispatch recovery. |
| `make app-integration connection-browser-test` | PASS. Both real production bridge/Go/disposable PostgreSQL journeys passed; provider and identity remain synthetic. `TestSyncReviewPostJourney` confirmed exactly one journal, two source observations, one review-posting link and one outbox message after explicit posting/recovery. |
| `make observer-test source-registry-test import-service-test worker-bootstrap-test sandbox-bootstrap-test` | PASS. Disposable database checks cover observer/recovery, private source/import identity and authority, worker-role replay/DDL denial, scoped bootstrap and atomic rollback. |
| `make demo-test` | FAIL: 3 of 4 browser tests passed, including 3,000 source transactions, exact-cent guided flow and posted-result reload recovery. The 1,000-command volume test exceeded its 270-second completion assertion while still cycling through command lifecycle states. No completed total was verified. |
| `FuzzPlaidMoney` (`60s`, two workers) | PASS; 3,084,897 executions, 377 total interesting inputs, independent rational quantity oracle. |
| `FuzzPlaidPage` (`30s`, two workers) | PASS; 1,759,029 executions, 112 total interesting inputs, atomic rejection and source-byte checks. |
| Existing hosted Sandbox API smoke | PARTIAL by diagnostic design. Workflow [39698259990](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/39698259990) and execution `jayburd-dev-sandbox-probe-cvg7n` succeeded; fixed receipt outcomes below. |
| Real Hosted Link and reviewed bindings | PASS for four supported fictional accounts; three other loan subtypes explicitly unsupported. |
| Hosted account acquisition and read-only review | PARTIAL: card, mortgage and student complete; checking held on decoder rejection. Card repeat generation completed with source revision unchanged. |
| Dynamic pending-to-posted, signed webhook, durable replay and database-level provider-to-review/no-journal evidence | Pending. UI observations and synthetic passes do not satisfy these hosted stages. |

### Execution environment

Tracked Go/toolchain/dependency files are unchanged. Installed Go 1.26.5 runs
with an external copy of the module files under `/private/tmp/jayburd-88-go.*`
that changes only the Go directive. Exact repository toolchain verification
remains CI evidence. Local listener/Docker checks use approved unsandboxed
execution.

Initial database checks stopped because Docker Desktop was not running. After
starting it, the registry refresh failed with Docker Hub EOF, including an
approved direct retry. Both exact repository-pinned PostgreSQL and migrate image
digests were already cached. A temporary external Docker adapter accepts only
those two exact `pull` references as cache inspections; all other commands go
to the original CLI. The unmodified test scripts and assertions run against
those cached immutable images. This is offline local evidence, not a registry
availability test. Test resources are disposable and isolated per invocation.

### Hosted probe boundary

The owner-requested test scope includes one invocation of the existing
`sandbox-probe` job via the fixed manual workflow on main, with no overrides.
Metadata readback confirmed zero task retries, a dedicated probe identity and
immutable image `sha256:caf1001dc84b0850f5bf08b0e79e15d2f563e03860cd63207ad7a53fe2540b07`.
This is the older diagnostic image used for the previous successful API smoke;
it cannot attest the latest application revision. It creates/exchanges/removes
only its disposable fictional Item, uses runtime-managed secrets, and reports
fixed checks/reasons/counts. No raw provider response or credential enters
agent context. Workflow submission success is not execution success.

Execution completed successfully at `2026-10-22T00:14:21.309624Z`, with one
succeeded task and no configured task retry. The profile
`plaid-sandbox-diagnostic-v1` receipt at `2026-10-22T00:14:10.481702Z` reported
overall `partial` with these allowlisted outcomes:

| Receipt check | Outcome |
| --- | --- |
| `sandbox_api_auth` | PASS — `fictional_item_exchanged` |
| `initial_sync` | PASS — `exact_decode_complete`; 1 page, 6 changes |
| `repeat_delta` | PASS — `empty_delta_observed`; 1 page, 0 changes |
| `historical_readiness` | PASS — `provider_reports_complete` for this disposable Item; no 90-day or institution history guarantee follows. |
| `sandbox_item_cleanup` | PASS — `removal_acknowledged` |
| `real_link_exchange` | BLOCKED — `application_proof_required` |
| `provider_to_review_no_journals` | BLOCKED — `application_proof_required` |
| `dynamic_pending_posted` | BLOCKED — `application_proof_required` |
| `signed_webhook_delivery` | BLOCKED — `application_proof_required` |

Only selected execution status and the exact receipt profile's fixed
checks/reasons/counts were read. No general logs, raw responses, financial
fields, Item/account identifiers, cursors, credential values or process
environments were inspected. Probe success does not complete #88.

### Current runtime metadata readback

On 2026-10-22, selected Cloud Run metadata confirmed API revision
`jayburd-dev-api-00017-znq` and web revision `jayburd-dev-web-00017-b42`, each
serving 100% traffic. The acquisition publisher Scheduler is **ENABLED**, on
`* * * * *`, with last attempt `2026-10-22T00:16:07.715341Z`. Five selected
request-status records from 00:12–00:16 UTC returned HTTP 200 on publisher
revision `jayburd-dev-acquisition-publisher-00005-w97`. No Scheduler or runtime
configuration was changed. This current readback supersedes the release
runbook's historical PAUSED state; endpoint acceptance does not prove a
selected account's source publication, queue completion or SQL/no-journal
invariants.

### Browser volume finding

`web/tests/demo/volume.spec.ts:15` waited 270 seconds for `Volume run verified`
or `Stopped for review`, but received `Saving immutable request`. The call log
continued alternating save, SQL execute and receipt acknowledgement; it does
not establish a deadlock, wrong amount or duplicate posting. The error snapshot
did not capture a committed count, and the disposable database was cleaned up;
no completed count or USD 495.55 oracle pass is claimed. The other three demo
tests passed, and the independent SQL report test committed 1,024 cent journals
and checked nine GL pages in 17.68 seconds.

`TestAppIntegrationBridge` launches a `docker exec ... busybox nc` process for
each HTTP connection. That is a plausible contributor to this local volume
latency, not a measured root cause or a production-capacity conclusion. #89
owns a controlled timing/progress breakdown across browser command lifecycle,
bridge, SQL and resources. The test timeout was not raised and no retry was
used to relabel the failed gate. The synthetic error artifact is retained
locally as `/private/tmp/jayburd-88-volume-error-context.md`.

### Interactive hosted application — 2026-10-22 UTC

Resumed under lease `PLAID-88-INTERACTIVE-20261022T0028Z`, after owner sign-in
and explicit approval to use the public fictional Sandbox phone shown by Link.
Used First Platypus Bank, non-OAuth `ins_109508`, with its public dynamic
Transactions Sandbox user. No personal phone, real bank credential, private
financial record, raw provider response or credential store was inspected.

Plaid displayed seven fictional account types and reached its success screen.
The application then reached verified binding review and committed four selected
bindings: checking, credit card, mortgage and student loan. Auto, consumer and
line-of-credit accounts explicitly could not bind. The selected pre-existing
Sandbox book and its dated currency policy were retained; enrollment did not
create a new entity or book. The attempt became active with no unfinished attempt.
One completion/status read initially showed the generic unconfirmed outcome;
subsequent explicit inspection recovered the completed state. No duplicate
enrollment or exchange was initiated to work around uncertainty.

| Account | Observed acquisition and review outcome |
| --- | --- |
| Credit card | Initial generation 1/1, HISTORICAL UPDATE COMPLETE, publication `00:38:27.856894Z`. Explicit scoped repeat request completed generation 2/2 at `00:42:06.619626Z`, no queued request or active lease. Source revision remained 1. After repeat, needs-review pagination contained 50 + 5 rows; posted filter contained zero rows. The full pre-repeat count was not measured, so no count-equality assertion is claimed. |
| Mortgage | Generation 1/1, HISTORICAL UPDATE COMPLETE, publication `00:38:23.450744Z`; source revision 1 and 47 needs-review rows visible. A transaction detail opened its category/accounting form; no category selection, draft creation or posting was performed. |
| Student loan | Generation 1/1, HISTORICAL UPDATE COMPLETE, publication `00:38:21.929770Z`; six needs-review rows visible. |
| Checking | Generation 1/0, NOT READY; stopped after failure, no active lease or completed publication. Fixed UI reason `Data validation: amount out of range`; last outcome `decoder rejected`. All-status review showed source revision 0 and no transactions. Left held; no retry or rounding workaround. |

The checking result is a qualification failure with an undiagnosed cause, not
proof that the provider value was invalid or that the configured money limit is
correct. A bounded follow-up must distinguish provider decimal representation,
approved range and decoder behavior using synthetic reproductions and redacted
runtime evidence. Preserve exact money and atomic publication requirements.

Read-only review demonstrated imported unposted observations, including card
dates before the book's current policy start. It did not prove a full 90-day
window, exact provider-to-review equality, cursor state or absence of all
database journals. UI posted-filter evidence is narrower than SQL evidence.
The application's scoped sync request does not invoke the Sandbox refresh tool;
dynamic pending-to-posted changes were not induced. Signed webhook delivery,
restart/replay and fixture-known reconciliation remain open. No deployment,
configuration, Scheduler, IAM, disconnect or posting action occurred.

## Remaining work

Resolve the held checking decoder failure under a separate bounded lease;
qualify dynamic transitions, signed delivery, persisted cursor/replay and exact
provider-to-review/no-journal SQL evidence. Record redacted outcomes only.
Do not alter infrastructure under this lease. Keep #88 open for every unproved hosted stage and #90 blocked for
live-account qualification. The separate #89 capacity/overload matrix is not
implemented by these checks; the observed browser volume timeout remains an
open finding. No hosted scaling or cost/SLA result is claimed.

## PR handoff

Evidence-only branch; no runtime or migration change. Update issue #88 and
release the #65 lease after recording the completed test matrix and any precise
hosted blockers. Owner review remains separate from test success.
