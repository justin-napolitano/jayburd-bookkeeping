# Production release execution plan — #316

Status: PLANNED2026-10-26; no316 source implementation or provisioning started.
Hard gates: merged308/311–315 and accepted337 mandate, plus applicable actual
qualification/dispositions. PR348 is now merged6be6c48 with all five exact-head checks PASS.
Do not close316 from this plan. Exact targets/cost/expiry/retention remain pending
[337 owner/security decision](../contracts/production-pilot-mandate.md).

## Ordered bounded work

| Stage | Files/boundary to review | Acceptance evidence |
| --- | --- | --- |
| A: settle mandate |337 targets, identities, account/platform, policy, budget/retention and gate table | Accepted decision with named owner/security reviewer; no placeholders |
| B: source/release contract | New Production manifest validator/synthetic fixtures; deploy docs; source/profile allowlist and compatibility | Canonical exact hash, explicit Production profile, numeric versions, source/image/schema ancestry; null/omitted/mixed/expired/extra fields deny before cloud auth |
| C: identity/bootstrap | Extend existing migration/SQL-role bootstrap and scoped workload identity for Production, preserving development | Actual intended Production migrator/operator/runtime names; no implicit dev-derived login; fresh35 floor/status/checksums and least-privilege tests; synthetic-only default |
| D: runtime delivery | Production composition for existing runtime/worker/webhook/foundation roots or reviewed reusable profile modules | Fixed nondev targets, exact API/origin/audiences/keys/static versions; private API/workers, approved signed receiver exposure; mocked IAM/role/cross-profile negatives; separate state |
| E: build/workflow | Production repository/federation image publication and owner-only exact-hash dispatch; do not silently reuse dev builder/deployer | Same-build scanned exact images incl webhook command; complete provenance and successful gate artifacts; no keys/state/plans/raw payloads uploaded |
| F: recovery/stop | Manifest compatibility/expiry, paused scheduler/queue defaults, restore/kill/retention and candidate receipt | Compatible Production rollback only; no prior receipt means stop/forward repair; deny fake Sandbox rollback and retained-data down migration |
| G: qualification/release | Record applicable88/274/275/276/62/89 outcomes; private identity/secret delivery/resource readback | Concrete candidate with real public metadata and all applicable evidence, exact review hash, duration/cost/retention and human approval |
| H: actual90 pilot | Separately authorized apply/migration/status/bootstrap/runtime/receiver/paused-worker rollout, verified sign-in, one fresh Item | Owner privately authenticates; controlled first sync; complete-set reconciliation with opening/closing balances and explained timing differences |

StagesB–F are implementation after accepted predecessors, not performed tests.
Claim one bounded issue/path lease at a time; create reviewed children if a stage
cannot remain one bounded delivery. Coordinator serializes common CLI/CI/schema
integration; no SQL migration planned unless a reviewed defect demands one.

## Current source inventory and concrete gaps

The Go source now has explicit Production API/worker/receiver transports, keys,
finite approval and SQL context. Migration35 is merged source, not a managed
Production apply. #346 fixed first empty-sync receipt0; #315 composes that path.

Existing .github/workflows/deploy-dev.yml, deploy-workers.yml and
deploy-webhook.yml and their validators are development/Sandbox contracts.
infra/gcp/runtime, worker-runtime and webhook-runtime still derive jayburd-dev
service names, identities, instance/static paths and origins. build-images.yml
publishes to jayburd-dev with development federation. These must gain an explicit
reviewed Production delivery; passing Production Go tests cannot make the current
deployment templates suitable. No secret/URL replacement or all-dev rename.

Production migrator/bootstrap must be inventoried explicitly: current Cloud SQL
helper only admits Production api/sync/publisher/webhook components; do not deploy
a dev-derived migration identity into Production. Runtime/worker/receiver image
schema/key/environment compatibility must be enforced before rollout.

Frontend Hosted Link instructions still say Sandbox despite profile-aware backend.
Issue349 owns correcting that copy and its tests before actual Production use, without making a
browser-selected environment authority. Prototype profile strings or backend
configuration do not grant frontend permission to choose provider destinations.

316 must preserve the owner's already-entered Infisical Production secret. Need
only owner-delivered numeric version/reference mapping, not secret contents or a
repeat request to enter it. Actual resource/SQL/secret readback remains separate
from source tests; keep private data out of public receipts and CI.

## Required planned tests

These names describe proposed new tests, not existing implemented guarantees:
TestProductionReleaseRejectsMixedTarget; TestProductionReleaseRequiresCurrentProof;
TestProductionSecretVersionPins; TestProductionMigrationIdentity;
TestProductionRuntimeArguments; TestProductionWebhookReleaseProfile;
TestProductionPausedScheduling; TestProductionRollbackCompatibility;
TestProductionExpiredApprovalBeforeAuth; TestProductionEvidenceRedaction.
Prove no provider/cloud mutation from default tests and preserve existing Sandbox
manifest hashes/role/rollout behavior. Run required core/release/mock-infrastructure
gates and affected app/web/migration/ledger tests, with exact-head CI.

No financial recognition/currency/message change: STD-01 operational release
applicability, Plaid API2020-09-14, ISO4217 and current audit-access limits persist.
PP10 documents selected cloud mechanisms; resource/cost/retention choices are
project proposals. No successful deployment is live-bookkeeping approval.
