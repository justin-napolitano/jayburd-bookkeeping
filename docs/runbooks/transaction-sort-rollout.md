<!-- stage-runbook: true -->
# Transaction sort Production rollout —410

## Status and boundary

ROLLOUT410, ops/410-sort-rollout from merged409/30ddfc1. Owner explicitly requested
latest deployment after concrete409/24fed7a review and five-green exact-head CI.
Existing Production project/shared bookkeeping_prod, forward39 and matching five
services only; current one-account cap and bank expiry2026-11-03T04:00:00Z stay.
Operator lease/execution deadline03:45Z does not renew bank consent. No live DOM/
records/provider payloads, new account/cap/consent, date repair, reset/relink/token
rotation, reimport, posting or resource expansion.40+ unclaimed;1–39 immutable.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume/source | Live410/406/65 read; dedicated branch/lease/JB-001 packet. PR409 exact24fed7a all five checks PASS, explicit latest-deploy approval, merged30ddfc1 |
| Preflight | Shared SQL RUNNABLE; backup/PITR enabled7days, recent managed backup1793653746728 SUCCESSFUL. Existing builder pool/provider disabled; exact owner/repository/main/manual/workflow numeric claim condition verified |
| Predecessor | Existing restricted38 image65151bbe… status execution fsk2c succeeded2026-11-03T01:53:34Z, exact Production shared target/checksums/clean38; read-only execution override through03:45Z |
| Recovery | Fresh managed backup1793670958354 SUCCESSFUL2026-11-03T01:57:29Z; no dump/export/record read, actual restore unproved |
| Baseline | Pinned official2.26.0 Cloud Shell IAM proxy; correct database/operator/owner membership predicates true. Scoped initial1attempt/Item/stream,187 observations/review,0journals/leases; private hashes retained in operator tmp files only, not returned or committed |
| Build | Trusted owner/main run41562555623/attempt1 from30ddfc1: all seven validation/image/runtime/HIGH+CRITICAL scan/publication jobs PASS. Four receipts independently recomputed against authenticated metadata/current main; both existing builder pool/provider restored and read back DISABLED. No new grants |
| Frozen state | Original ENABLED Scheduler/RUNNING queue paused and read back PAUSED; scoped frozen baseline exactly equal to initial,0leases. Production effective maximum1 independently true. Existing restricted job pinned to verified39 migrator with read-only default/03:45Z deadline |
| Migration | Restricted up39 execution4l9hp succeeded2026-11-03T02:12:02Z; independent default status7pwxn succeeded02:15:21Z, clean39/all embedded checksums. Post-migration private baseline equality and effective maximum1 true |
| Rollout | API00006-882, web00005-pt6, publisher00005-cm7, sync00005-8xs, webhook00005-6fz all Ready/current/100%. Independent projected service identity/args/probes/resources/concurrency/timeouts exactly equal to predecessor except image |
| Smoke/preservation | Public web/auth-config200; unauthenticated API/publisher/sync403; unsigned webhook401. Post-rollout frozen baseline equality true,187 observations/review,0journals; maximum1 true |
| Restoration | Original queue RUNNING and Scheduler ENABLED restored and independently read back after all gates within unchanged04Z bank authority |
| Cleanup | Scheduled POST on new publisher returned200 at2026-11-03T02:18:00Z, request metadata only. Temporary Cloud Shell SQL proxy stopped/read back absent; builder pool/provider disabled again |
| Local validation | make check/test/build PASS with existing installed1.26.5/external modfile adapter; tracked1.26.10 unchanged. Exact-source hosted validation/application integration and scanned image jobs all PASS |

## Current evidence

Approved build target hashb58f53138cf632257f9c18ef1035622e59b757ab759b8b39dc0780ad096f75b9
is unchanged/valid through04Z. Existing source-only367 web units/30 browser,
all three full-stack journeys, core/source/migration/ledger and3000-row pagination
passed; GitHub billing-start failures were resolved by owner-directed public
visibility, followed by five-green latest source. Managed39 up/status and private
preservation checks passed.

Recomputed four-receipt bundle SHA256:
f7047078874ddd2dc73cb60d7aafc54557da84c3fd957f725c8dc5eb26e93313.
All pins use us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod:

| Component | SHA256 digest |
| --- | --- |
| API |0e0451fc765d4fb78903ee2f76ebfcb29772c39d5fc767581715a3ad869cb5dc |
| Web |ae0e22f02098b97c83020123546f21103d027d76c866ffa49cda415190646b13 |
| Worker |ee537e6c951808220b6dc3655cbd3c3889fe9f1460d1a6079d2fa2e6e3a0aab8 |
| Migrator |8bdfdc91b5b6d460ac051f63063353e22ebeb9b0ed140bbd83f04a365acb4cae |

Image-only SDK updates require matching reviewed pins in the next operator
Terraform inputs. Never apply stale variables; no private state/plans/inputs read.
Migrator default remains read-only status through03:45Z. No down/force or old
runtime rollback; pause and forward repair on mismatch. Backup/PITR preparedness
does not prove a restore. No new IAM grants, secret/environment or resource changes.

## Remaining work

#404 existing historical availability repair, #407 navigation, #408 search/coverage,
#395 fresh Sandbox append qualification and #90 owner-private reconciliation remain
independent. Count/hash equality and deployment do not prove bank completeness,
transaction-value reconciliation or signed Production webhook processing.

## PR handoff

Refs410/406/409/395/404/90. Operations evidence only, no new product source or
migration. Deployment/counts are not bank-value reconciliation or new consent.
