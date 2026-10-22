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
2. **Owner sign-in — pending.** The canonical deployed application is at its
   sign-in screen. The owner was asked to complete sign-in/MFA in Edge and reply;
   passwords and authenticator codes are not collected by the agent.
3. **Synthetic regression — complete with one volume failure.** Core and
   provider suites passed. The final 1,000-command browser volume check timed
   out; results below distinguish this from hosted qualification.
4. **Hosted API smoke — partial, expected boundary.** The existing diagnostic
   authenticated to Plaid, exchanged a disposable fictional Item, decoded initial
   and repeat sync, and removed that Item. Details below.
5. **Hosted application journey — pending sign-in.** No interactive Link session,
   application binding, acquisition, review or posting has been performed by
   this pass. The owner sign-in question remains pending.

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
| Real Hosted Link, initial/incremental sync, pending-to-posted, signed webhook, durable replay and provider-to-review/no-journal evidence | Pending. Synthetic passes do not satisfy these hosted stages. |

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

## Remaining work

Complete owner sign-in, verify the selected application environment, then follow
the Hosted Link operator journey. Record redacted stage outcomes only. A queued
sync request is not import evidence, and Plaid's success screen is not durable
credential/binding proof. The Scheduler is currently enabled, but the selected
account's worker and publication evidence still require observation. Do not
alter infrastructure under this lease. Keep #88 open for every unproved hosted stage and #90 blocked for
live-account qualification. The separate #89 capacity/overload matrix is not
implemented by these checks; the observed browser volume timeout remains an
open finding. No hosted scaling or cost/SLA result is claimed.

## PR handoff

Evidence-only branch; no runtime or migration change. Update issue #88 and
release the #65 lease after recording the completed test matrix and any precise
hosted blockers. Owner review remains separate from test success.
