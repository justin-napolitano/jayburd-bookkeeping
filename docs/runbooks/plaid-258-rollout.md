<!-- stage-runbook: true -->
# Plaid #258 approved Sandbox rollout

## Status and boundary

### Resumed owner-approved release — 2026-10-22T03:15Z

Owner explicitly requested migration29, merging ready work and reviewing the
transaction flow before personal-account enrollment. Lease
`PLAID-258-RESUME-20261022`, expiry08:00Z, reserves this bounded development
Sandbox release. Live-account readiness remains a separate #90 gate.

Merged reviewed #257, #264 and checking repair#263; reviewed and merged the
proposed source-first authority#245, fixture#252, planning proposal#254 and
Investigate proposal#256 without accepting their deferred implementation or
live policy. Position#260 merged as f334d06 after resolving only index and
bibliography conflicts, preserving both proposals and the bibliography budget.
#253 was incorporated by that merge. Combined product source passed core
check/test/build,300 web unit tests,23 browser cases, production web build,
app-integration and connection-browser-test on disposable migration29 SQL.

Fresh outside-sandbox npm audit completed successfully with zero reported
vulnerabilities. No scanner fallback or security-control change was implemented.
Earlier 503/timeouts remain historical failed checks. Candidate build39715438972
on f334d062e2ee4c572af52b823969c299cd37bf93 failed its first audit with HTTP503;
attempt2 retained passed core/full-stack results and reran frontend successfully,
including the unchanged npm audit. All four image proofs/scans passed; each scan
reported zero HIGH/CRITICAL findings. Native run metadata and component receipts
passed the strict migration and worker validators independently.
Superseded build39714745765 and duplicate main-push runs were cancelled; they
are not passing evidence. The migration execution result is recorded below;
matching application schema-status and worker/transaction acceptance follow it.

The managed browser session expired during the review read. Re-sign-in will be
needed after the matching frontend is deployed; no token or identity substitute
is used.

Real-account readiness review found explicit Sandbox restrictions in
`internal/connections/http_provider.go`, `internal/acquisition/plaid_http.go`,
`internal/acquisition/sql.go`, `internal/providersecrets/types.go`,
`internal/bankcredential/envelope.go` and webhook validation. The release
manifest cannot switch those boundaries to Production. #90 requires separate
implementation and approved source/privacy/retention scope after remaining
Sandbox correctness gates; owner handles real credentials/consent. No personal
account was connected by this release. Existing migration12/18 constraints also
restrict provider Items and encrypted credentials to Sandbox; any production
extension requires new forward SQL, not edits to merged migrations.

Reviewed [compatibility evidence](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/258#issuecomment-6332019643)
pins the source, migration checksums, all four image digests, unchanged identity
and consent, old-worker compatibility and bounded pause/restore procedure.
Migration-only manifest hash
`b6fac6db38f49e4dda636acf89f71370e3f1f544f55eb23b0c1dae82cb07428a`;
matching paused-worker manifest hash
`2ad178d2dfa5d455caedbb1e6e0f1af6e35176db2c4150694879713813deee6e`.
Both expire12:00Z and use successful build39715438972 attempt2. API digest
`9e39f015fc293bf6b00afbf2ad46d218e81c65102edab631abfa074fa6fb97de`,
migrate `28b21087201e92c5bd5dce4ab5a1067118b31be64a46dae4a0965acd1d50acab`,
web `ddddaeeb341cbf90a016e2337d2f64f04a025a83e163ec05ffa5a40d719aea27`,
worker `3db89dd3f8962cf2369a36c2568a52552820e8b345ad834589cc32c791c4a0ef`.
Runtime manifests remain distinct from migration-only execution. No image is
rebuilt between migration and runtime promotion.
Local `make check test build release-test`, exact-byte validators, diff check
and staged runbook guard pass for this metadata stage. The documented external
Go1.26.5 modfile is used locally; the successful candidate provides pinned
Go1.26.10 evidence. No security workflow or dependency was changed.

PR265 merged as ba82a3f. Migration-only workflow39717677086 succeeded. Its
validated receipt reports `migration-execution-passed`, exact approved migrator
digest28b21087…, execution `jayburd-dev-migrate-cwkwx`, completed
2026-10-22T03:37:26.338233Z on the same build/source. No service update occurred
in that operation. Application schema-status verification is the separate next
gate, not inferred from the transport receipt. Application manifest hash
`b5768900ab9a6de5f162f05c7e398f9faf4436f6e26ca033a3fd34418f9a56de`
is prepared for the same build; strict validation passes. Independent comparison
preserves connection-policy digest
`d1b79512c169809470eb3071755fb963d4b4b7542edb0e5836e93af810d72148`
and worker-policy digest
`ab3b19d91dbd682e6b2f3945a8fb718e39bb3e3de475ae78246e9b8617e54688`.
Fresh Scheduler readback was ENABLED on the reviewed one-minute/90-second
schedule; paused at2026-10-22T03:36:07Z, readback PAUSED, first BASIC queue-name
count zero. No task payload was requested. Fresh known-account lease reads
remain unavailable because the managed browser session expired. No universal
drain claim is made.
Second BASIC queue-name count zero at03:37:56Z, over90seconds after the pause;
pipeline errors were checked and Scheduler readback remained PAUSED. The
application metadata also passed local `make check test build release-test`.

**Historical disposition (2026-10-22T02:48Z):** three additional fictional loans
are registered and synced. The shared checking repair is PR #263 (`d88d8a0`),
unmerged and not deployed. npm audit returned HTTP 503 on the first complete CI
attempt; the frontend-only retry reached the audit and hit its 15-minute job
limit. Independent public audit requests timed out outside the local sandbox,
including an empty IPv4 request. This is an external-gate blocker, not a
successful vulnerability result. Keep the audit requirement. No migration29,
new checking acquisition or live account is authorized by a passing partial run.

Owner subsequently requested review of PR #263 and application of migration29
if needed, and permitted canceling foundation tests. Review of exact head
`d88d8a0` found no blocking code findings; migration29 is required before the new
worker. [Review evidence](https://github.com/justin-napolitano/jayburd-bookkeeping/pull/263#issuecomment-6331500206).
Foundation already completed successfully; the existing passed result is reused.
Fresh npm audit check still timed out outside the sandbox. The successful trusted
candidate required by the release workflow is not yet available, so hosted
migration29 remains unexecuted. Scheduler readback remains ENABLED.

Owner approved migration 28, matching API/web/worker rollout, one held-checking
retry and registration of the three fictional additional loan types. Lease
`PLAID-258-ROLLOUT-20261022T0102Z`, expiry 2026-10-22T06:00Z. No production bank,
posting, IAM expansion, credential inspection, raw payload retrieval or SQL
force/down operation. Prior acquisition scheduling was ENABLED; pause, check
quiescence and restore that same schedule only as bounded rollout controls.

## Stage ledger

1. **Implementation acceptance — complete.** PR #259 merged as
   `7737991e386d3ddfcee6385289ffc52f4091693a` after all checks passed. CI run
   39703004273 passed core validation, frontend and both full-stack/demo paths;
   Infrastructure/GitGuardian also passed. #258 and #88 stay open.
2. **Candidate publication — complete.** Build-images run 39704183980, attempt1,
   completed successfully on the exact source commit above. All validation,
   runtime/container proofs and four image-security jobs passed. Independent
   strict validators matched every candidate receipt to the trusted run and all
   four scan reports contained zero HIGH/CRITICAL vulnerabilities.
   Duplicate push validation 39704176243 was canceled to release the shared
   main-branch concurrency slot. It is not passing evidence; the candidate's own
   complete validation remains mandatory. Position CI was left running.
3. **Migration — executed.** Exact-image migration-only manifest hash
   `5b6a88a2c6d3c0a85c393e884c2cdeb6ee0a84d8a919d995ef15aca7d3f5946d`
   passes strict metadata and same-build receipt validation. Worker manifest hash
   `579254e64c77244927bbcabfdf50d304133556d4931f60e596d0ca64bed9feb8`
   is prepared for the later paused rollout. Both expire 2026-10-22T12:00Z.
   PR #261 merged as `0292c2d`; deployment39706130730 succeeded. Its validated
   receipt reports `migration-execution-passed`, execution
   `jayburd-dev-migrate-r9hsl`, exact migrate image
   `c41940cd583fc0f3934d686534c2ab0e86a8476df9aa1753c0e9b9a8d672159e`,
   completed `2026-10-22T01:34:13.166403Z` on the same build/source.
4. **Application and paused workers — executed.** App promotion manifest hash
   `99dd25963b6925b48e0513b6ec6ee9ae6c578ff018914cce1486d9e1b898c825`
   passes strict same-build metadata/evidence validation. Reuse unchanged approved
   identity/secret-version metadata. Review additive schema/API compatibility,
   exact digests and expiry. Pause scheduling before application promotion;
   check queue empty and known account leases inactive before worker replacement.
   This prevents new UI-supported loans being acquired by an older worker.
   A paused Scheduler alone is not
   drain evidence. Restore its prior state only after successful readback.
   Scheduler paused at `2026-10-22T01:34:40Z`; readback PAUSED and first BASIC
   queue-name count zero. Second BASIC count zero at `01:36:21Z`, more than the
   90-second scheduling deadline later; pipeline failures were checked. No task
   payload was requested. Fresh known-account UI lease reread was unavailable:
   managed browser session expired before the read. Do not claim all SQL slots
   were inspected. Runtime safety also depends on the reviewed unchanged old-work
   semantics and existing fence/retry proofs, not a universal drain assertion.
   PR #262 merged as `6492846`; app workflow 39706827117 succeeded with
   `transport-smoke-passed`, API `jayburd-dev-api-00018-gzt` and web
   `jayburd-dev-web-00018-v57`, on the same verified candidate. Worker workflow
   39707303805 succeeded with `paused-workers-observed`: publisher
   `jayburd-dev-acquisition-publisher-00006-d5m`, consumer
   `jayburd-dev-acquisition-sync-00006-2xf`, exact worker digest
   `ca4ba9813666c6fb94fd63ba6b14871cf2555e882c8b96bd27c6795f3d99ac60`.
   Scheduler readback remained PAUSED before restoration; resume succeeded and
   readback ENABLED. Owner signed in again and selected the same Sandbox book.
   A single explicit checking retry (generation 2, completed 0) returned
   `currency_precision_exceeded`, decoder rejected, no publication and no active
   worker lease. No source payload or amount was inspected. Shared exact-source
   retention/row-hold repair is PR #263, head `d88d8a0`; synthetic gates passed
   and pinned-toolchain CI is pending.
5. **Diagnosis and registration — in progress.** Inspect only
   fixed diagnostic/status fields, and repair based on evidence. Register only
   auto, consumer and line of credit in a fresh reviewed fictional attempt; keep
   the existing four accounts out of the new bindings. Verify all seven account
   types and source publication outcomes without posting.
   Fresh fictional Hosted Link succeeded. After inspecting the existing attempt
   to resolve an unconfirmed completion result, its verified state reached
   binding review. Bound only auto/consumer/line of credit, respectively new
   liability codes 2202/2203/2204 and descriptive Sandbox names. The other four
   accounts were explicitly unselected, including their currency choices.
   Binding date 2026-10-22 and USD version from 2026-09-17 were reviewed; scoped
   acquisition acknowledgement confirmed. Attempt readback is active; auto's
   first acquisition subsequently completed. All three are active, generation
   1/1, HISTORICAL UPDATE COMPLETE, no queued generation or active lease:
   auto publication `02:14:08.689624Z`, consumer `02:14:12.276241Z`, line of credit
   `02:14:10.560721Z` (2026-10-22). Each review list displayed 47 needs-review rows;
   all three pagination controls ended and each showed source revision 1.
   Refreshed book revision 16 displayed 14 configured accounts, including all
   seven named dynamic account types, with zero posted trial-balance values.
   This is not SQL proof of zero journal records. These are UI observations, not raw-provider reconciliation or a
   ninety-day guarantee. No journal was prepared/posted.

6. **Shared checking repair — CI review.** PR #263 retains source precision up
   to the existing numeric storage bound and holds sub-minor-unit rows from
   drafting, with exact independent SQL eligibility/equality checks. Local
   core, migration/ledger, acquisition, web (290 unit /18 browser) and full-stack
   proofs pass; independent rational fuzz executed 1,682,846 cases. Initial CI
   caught a missing transaction-local owner role in migration 29. Both migration
   directions now explicitly select the existing restricted capability role;
   local cloud-SQL/bootstrap gates and the new CI restricted-migration stage
   pass. No new grants or weaker validation. Full CI remains required before
   merge/candidate publication and hosted migration 29.
   Final migration29 checksums: up
   `cb844e10e3b25857041c69f553104c31e1dc9d3333bb28436e06acee78e3b450`;
   down `1fca62a93ab61eb017be1cd2c59cfd800b1e6f0f25c1e3618f7cf5eda80353d3`.
   CI application integration passed (9m43s). Frontend tests/image proof passed
   but its audit service returned HTTP 503 from npm after five minutes; this is
   a failed gate, not a passing vulnerability check. Retry the completed failed
   job after the parent run finishes; retain the audit requirement unchanged.
   Frontend retry (run39710841287 attempt2) again passed behavior and exact image
   proof, reached explicit audit at `02:45:12Z`, then was canceled at its
   15-minute limit (`02:48:04Z`). No audit waiver, alternate registry, dependency
   change or forced merge was introduced. Core validation (12m2s), application
   integration (9m43s), foundation and GitGuardian passed on the repair head.

## Current evidence

The [rollout source review](../research/plaid-rollout-sources.md) records selected
Scheduler v1 deadline/retry/state sections and local handler budgets. Queue and
lease observations will be bounded evidence, not a universal drain guarantee.

PR #259: validate 11m15s; application integration 8m52s; frontend 2m04s;
foundation 42s; secret scanning passed. Release tooling tests also pass locally.
Additional seven-account, three-currency synthetic money matrix passes using an
external Go test overlay, leaving the candidate source unchanged; retain it in
the checking-repair PR. This is baseline parser evidence, not a fixed hosted
checking cause. The earlier Edge session was signed in to the fictional Sandbox
book; it expired during the resumed review. Scheduler readback at03:15:35Z
remains ENABLED.

[Compatibility review](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/258#issuecomment-6330452107)
binds migration28 checksums, exact source/build, current revisions, preserved
old-worker semantics and forward-repair limitations. Current app/worker/public
secret-version policy is unchanged. Strict release CLI evidence checks pass for
both manifests. `make check test build release-test`, diff validation and the
staged runbook guard pass locally; candidate CI provides the pinned-toolchain
and full-stack/runtime evidence for the unchanged product source.
The application-promotion manifest independently passes the same strict evidence
checks and local core/release gates. It uses the same candidate, current consent
and identity/secret metadata. Owner re-sign-in is required after new-web reload;
no token, password or test identity substitutes for verified access.

## Remaining work

The repaired source and Position are merged and the complete candidate is
successful. Merge and dispatch the reviewed migration-only manifest, then
matching application and paused-worker metadata through the existing release
workflows. Migration29 must precede the new worker; pause and
restore the existing Scheduler only around the reviewed matching rollout.
Checking's next explicit request is generation 3, after the deployed repair;
verify complete publication, held-row visibility, repeat equality and all seven
account types. Existing three new loans have baseline generation 1/1, source
revision1 and 47 needs-review rows each for the post-repair repeat comparison.
Preserve every unproved #88 webhook/lifecycle/reconciliation/no-journal gate.

## PR handoff

Issue #258. Runtime source is the merged implementation above; migration 28 is
additive and must precede its new fixed decoder codes. New web must understand
codes before new worker emits them. No money limit or posting policy changed.
The same exact tested candidate images must be used without rebuilding between
migration and runtime stages. Keep receipts bounded; never upload state/plans,
private provider diagnostics or financial records.
