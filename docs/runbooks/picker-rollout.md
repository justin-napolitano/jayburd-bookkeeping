<!-- stage-runbook: true -->
# Migration-first picker rollout —395

## Status and boundary

ROLLOUT395; ops/395-picker-rollout from reviewed403/11b9074. Owner explicitly asks
to apply36–38 then roll out. Existing Production project/shared isolated database,
one already-bound checking/default1, existing finite deadline2026-11-03T04:00:00Z.
No new account, cap increase, token rotation/relink/reset, consent renewal, source
date repair, infrastructure expansion or personal-record/provider-payload access.
Fresh Sandbox provider qualification remains expired/unapproved, not inferred.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live395/65/PRs/contracts/current runbooks read; dedicated finite release lease/worktree and JB-001 packet |
| Source |403 human-reviewed and merged11b9074 after all five exact-head checks;36–38 source/checksums reviewed;362 web units/29 browser/42 SQL and three full-stack journeys passed |
| Preflight | Existing service revisions match prior release; SQL RUNNABLE, backups/PITR enabled7days, latest automated backup1793599200000 SUCCESSFUL; builder registry Writer only/no project grants, exact repository/actor/main/manual-workflow claims |
| Recovery backup | On-demand managed backup1793653746728 SUCCESSFUL2026-11-02T21:10:38Z; no dump/export/record read. Actual restore remains unproved |
| Predecessor | Restricted old35 migrator status execution jayburd-prod-migrate-tm6f4 succeeded2026-11-02T20:56:41Z: clean35 and every recorded checksum, without SQL mutation |
| Preservation baseline | Read-only scoped owner SQL through official pinned2.26.0 Cloud Shell IAM proxy: one scope/attempt/Item/stream,187 observations/review,0journals/live leases. Private identity and observation-revision hashes retained only in the operator session; no records/amounts/provider IDs returned |
| Existing history | Closed predicate confirms historical observations precede the already-bound ledger account availability;38 deliberately leaves existing accounts unchanged. Follow-up404 records a separate reviewed preserving repair, not authority to mutate existing dates |
| Local evidence | make check/test/build PASS using installed1.26.5/external modfile; tracked1.26.10 unchanged. Exact-source hosted1.26.10 validate/frontend/application-integration all PASS; four exact image/runtime/scan/publication jobs now running |
| Build | Trusted Production run41523269516/attempt1 at reviewed11b9074/current main: all seven jobs PASS. Four receipts independently recomputed against authenticated run/job metadata; HIGH/CRITICAL scans and exact runtime steps PASS. Existing builder pool/provider both independently confirmed disabled again |
| Migration | Acquisition Scheduler/queue paused from independently observed ENABLED/RUNNING; frozen baseline equal/leases0. Restricted up execution jayburd-prod-migrate-hc77x succeeded2026-11-02T21:25:15Z; independent status6rqjg succeeded21:26:48Z: clean38/every embedded checksum. Migrator default remains read-only status, finite operator deadline00:30Z |
| Preservation | Post-migration closed baseline comparison exactly equal, including internal identity/history hashes,187 observations/review and0journals. Production effective registration maximum independently confirmed1. Existing availability limitation unchanged, tracked404 |
| Rollout | API00005-mb5, web00004-gc9, publisher00004-mrh, sync00004-7lq, receiver00004-d84 all Ready/current/100% at matching pins. Independently compared predecessor service identity/arguments/probes/resources/concurrency/timeouts: unchanged except image |
| Smoke/preservation | Public web/auth-config200; unauthenticated API/publisher/sync403; unsigned receiver401. Post-rollout private baseline exactly equal again; no signed Production webhook/value-reconciliation claim |
| Restoration/cleanup | Original queue RUNNING and Scheduler ENABLED restored/read back only after all gates; scheduled POST on new publisher returned200 at21:30:00Z (request metadata only). Local and Cloud Shell temporary SQL proxies stopped; builder pool/provider both disabled |

## Current evidence

Managed migrations36–38 applied/independently verified; matching services promoted.
Current independent evidence preserves the existing first checking import; metadata must not be
substituted for transaction reconciliation. No secret/config/state/plan file read.
Use existing SDK/workload identity; Infisical remains application secret owner.

Build target SHA256b58f53138cf632257f9c18ef1035622e59b757ab759b8b39dc0780ad096f75b9
retains its existing04Z expiry. Recomputed four-receipt bundle SHA256
e2163534e61b4210471f1d7c2b6a2954a34d1a9c710da4f51363d83d846dfcb6.
All images use us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod:

| Component | SHA256 digest |
| --- | --- |
| API |08ad2e215a6d6e041500a257c8fc7662ce4fdc72f1c57b55554f5406f41a3d78 |
| Web |d622c1b5af142534dd873d95443cfd150827eca88ccf1bf83bb2df31e40e2a86 |
| Worker |072ac3131c32b5c4076729ea74dcc1f56e83091cc333219fd3860ba430c2632e |
| Migrator |65151bbe03c488b6be322750881353764e08a00789579f1b2cf34126c634688d |

Executed promotion method: authenticated exact-source image/runtime/scan receipt
comparison, pause of the existing enabled publisher Scheduler and running queue,
zero-lease/frozen baseline verification, existing restricted migrator image update
(status default), up then independent status through the finite operator deadline.
Only image pins changed on the five services through SDK updates. Non-secret
identity/argument/probe/resource readbacks matched the captured predecessor;
only the originally enabled/run states were restored.
No IAM grant, secret/environment change, resource sizing change, bank authority
increase or bank-consent deadline extension.
These SDK image changes require matching reviewed pins in the operator's next
Terraform inputs; never reapply stale variables or revert to schema35 binaries.
No state/plan or owner configuration file is inspected/copied. On dirty status,
identity/history/count mismatch or failed compatibility, keep acquisition paused
and forward repair; never down/force/reset/relink/reimport or restore over newer
history. Managed backup/PITR is recovery preparedness, not a tested restore.

## Remaining work

Existing account availability is not rewritten by38;404 owns its separately
reviewed preserving repair. Fresh explicitly scoped Sandbox one→append provider
qualification and independent comparison/recovery remain395; expired consent
has not been renewed. The one-account Production limit/04Z expiry stay unchanged;
no second-account activation is approved. Actual private reconciliation remains90,
not passed by these count/hash, deployment or health results.

## PR handoff

Refs395/391/394/393/290/404/90. Operations/evidence only; no new product migration or
runtime source. Do not down-migrate accepted intent/history or reuse old images
that cannot render/recover register_accounts. Pause/forward repair preserves data.
