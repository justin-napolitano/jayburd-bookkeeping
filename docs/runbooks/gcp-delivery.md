# Synthetic GCP delivery evidence
<!-- stage-runbook: true -->

## Status and boundary

Terraform runtime rollout implementation slice (#27) under [epic #25](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/25).
GitHub owns current task state; this ledger records decisions and observed review
results. Terraform foundation/database configuration, explicit IAM connector and
restricted migration runner are implemented. Foundation infrastructure is now applied;
application/database deployment, live build-token exchange and live-bookkeeping
approval remain unestablished.

## Runtime rollout slice — current branch

- Decision: separate migration/runtime roots and runtime-only state bucket; disabled independent deployment federation and reviewed manifest gate. See [rollout contract](../contracts/cloud-run-rollout.md).
- #34 merged at `452e76d`; all hosted checks passed. #23/#24 remain open; runtime deployment is intentionally blocked pending dependencies and owner configuration.
- Implemented bounded services, dedicated migration job, exact-digest release validation, migration-before-service orchestration and explicit compatible rollback without SQL changes.
- Initial validation caught HCL multiline syntax and an overbroad test log matcher (runtime bucket name mistaken for service mutation); corrected both. Eight release/smoke tests and synthetic shell ordering/failure/rollback proof pass.
- PASS: pinned Terraform 1.15.7/Google 8.6.0 validate and 16 mocked runs across foundation/database/migration/runtime. Registry DNS sandbox denial resolved through approved rerun.
- PASS: `make check test build release-test` with Go 1.26.9; nine Node release/smoke/readback tests and synthetic shell failure/rollback proof; workflow actionlint, shell syntax and diff checks. Initial actionlint registry denial resolved through approved rerun.
- Reviewed token lifetime, exact ready revision/digest/traffic receipts and fail-closed expiry recheck. Hosted PR CI pending. No cloud apply, token retrieval, image publication or protected state/plan access.

## Runtime identity/private API slice — prior branch

- Merge #33 verified at `9fbb5d0`; this slice changes application code only. Foundation stays applied; no new live provider request or apply is performed.
- Replaced build-time Firebase identifiers and static rewrite with dynamic public configuration and a fixed-target bridge carrying separate user/service tokens.
- PASS: initial 67 browser/server tests, including transport denials, failure, replay-byte preservation, cancellation, concurrency and late configuration disposal. Type checks identified a Next ProcessEnv augmentation in test fixtures; helpers now declare their actual string-map input contract.
- PASS: `make web-check web-test web-build web-browser`: generated API/type checks, 67 unit tests, production build with dynamic auth/bridge routes and three Chromium journeys.
- PASS: `bash scripts/test-web-container.sh`: non-root/read-only isolated image, denied unconfigured routes, static assets, graceful shutdown and same-image synthetic runtime configuration without rebuild.
- PASS: pinned Go 1.26.9 `make check test build`, including race tests, documentation links and immutable migration checks. No database contract or migration changes.
- Next: Terraform runtime settings/Invoker grants and controlled migration/rollout; real identity and Infisical acceptance remain open.

## Foundation bootstrap — 2026-10-12

Decision: owner explicitly authorized creating/linking the development project,
then bootstrapping Terraform and GitHub build identity. Existing merged source
`ea48cef` was used unchanged; no application/database provisioning or image
workflow dispatch was performed.

- Project `jayburd-bookkeeping-dev`, number `9088794002`, billing link verified.
- PASS: Terraform 1.15.7/provider 8.6.0 init/validate and foundation apply. Nine component service accounts, immutable image repository and active repository/owner/actor/main/workflow-dispatch restricted provider were read back through GCP APIs.
- PASS: monthly USD 50 budget, project filter and 50/80/100% thresholds read back. These are notifications, not a spending cap; delivery of notifications remains untested.
- PASS: foundation state migrated to `gs://jayburd-bookkeeping-dev-jayburd-dev-foundation-state`, prefix `foundation`. Application state has its separate bucket. State/plan contents were not retrieved or displayed.
- PASS: GitHub variables `GCP_PROJECT_ID` and `GCP_PROJECT_NUMBER` set and read back. Shared CLI account initially selected `JNA31A_AIT`; switched to existing `justin-napolitano` login and verified immutable repo/owner IDs before configuring.
- Initial applies were partial: Budget API rejected user ADC without an explicit quota project. `USER_PROJECT_OVERRIDE=true GOOGLE_BILLING_PROJECT=jayburd-bookkeeping-dev` resolved it; source G17 records the documented setting.
- A separate gcloud Storage metadata probe failed certificate validation in the local Python trust chain. TLS verification was preserved; Terraform GCS backend migration succeeded.
- PASS: remote-backend `terraform plan -detailed-exitcode` returned 0 (no changes); contents remained private. Actual GitHub token exchange, negative trust tests and image publication are not claimed.

Private operational directory: `/private/tmp/jayburd-dev-bootstrap/.local/bootstrap/foundation`.
Owner-only output logs and local recovery files remain under its parent; do not
open, stage, or upload them. Future operators can initialize the same merged
foundation source against the above bucket/prefix after owner authentication.
Use project/number/billing inputs and `enable_github_build_identity=true` explicitly
with the quota-project environment settings above; the repository default remains
disabled for new installations. Next: application auth/ingress, Infisical and
Terraform-owned rollout; cloud database and application are still unapplied.

## Image-delivery evidence — prior slice

- PR #30 merge verified at `567eabe`; #5 is closed in GitHub. #6/#11/#26 and PRs #23/#24 remain open; closure is not inferred from configuration merges.
- Added standalone frontend packaging, exact-image runtime checks, owner-only main workflow, pre-publication scan/SBOM and candidate digest receipts.
- PASS: receipt tests reject changed configuration digests, missing provenance, wrong repository and ambiguous published identities.
- PASS: `make check test build`, browser type checks/33 unit tests/production build and three Chromium journeys; actionlint; canonical standalone frontend container with static asset and unavailable-route checks. API/migrator host-build runtime checks pass; their canonical builders remain hosted CI gates.
- Fixed the merged #30 CI readiness race: socket readiness saw PostgreSQL’s temporary initialization server. Cloud SQL and ledger proofs now wait for TCP, matching the migration harness. `make cloud-sql-test migration-test ledger-test` passes after the fix.
- The first frontend proof caught a self-proxy loop (default API port matched frontend port); the unconfigured image now targets unused loopback port 1. Node SIGTERM exit 143 is accepted; forced-kill exit 137 remains failure.
- No workflow dispatched, identity enabled, registry push or cloud mutation. Trivy publication scans and real federation remain unexecuted until explicit owner setup/dispatch.
- PR #31 merged at `cda91ff`; its hosted foundation/frontend/validate checks all passed, including canonical Go images and the PostgreSQL readiness fix. The receipt compatibility fix follows in a separate PR. A local Trivy 0.75.0 image scan could not download its vulnerability database because the Docker VM lacked the outbound proxy CA; no scan result is claimed and TLS verification remains enabled.
- Local Buildx metadata proof exposed a Docker store difference: containerd reports the manifest/index ID and omits the configuration digest. Receipt validation now accepts either recorded content ID, rejects unrelated IDs, and never labels an index as a configuration digest. Named receipt tests cover both stores.
- Next: private service-token bridge/runtime browser configuration, then Terraform rollout. Current frontend is deliberately unconfigured; all receipts are candidate-only.

## Stage ledger

| Stage | Evidence / outcome | Next action |
| --- | --- | --- |
| Reconcile existing work | Reviewed #5/#7/#11/#12 and open PRs #23/#24; preserved existing evidence and identity | Use native dependencies before claiming a slice |
| Publish bounded work | Created epic #25, database #26 and app #27; updated existing issues with named acceptance and recovery requirements | Native relationships verified |
| Record architecture | Updated GCP contract, index and annotated sources; scoped early proof separately from full month-close | Review documentation PR |
| Implement foundation | Two state buckets, nine identities, image repository, disabled build WIF and budget alerts; no runtime resources | Review IAM and owner bootstrap |
| Mocked provider proof | Terraform validate and six tests pass; invalid project/number/billing/region denied | Review PR and hosted CI |
| Cloud SQL configuration | Three new mocked tests pass for PostgreSQL settings, transport, recovery and inputs | Live provider verification remains open |
| Restricted migration | Non-superuser administrator/bootstrap and restricted migrator/replay pass; API DDL/table denial, durable checksum drift, contention and dirty failure pass | Review PR; execute named cloud gates after owner setup |

## Current evidence

Current bootstrap results are recorded above; the following items preserve prior
planning evidence. Local `make check test build` also passed for this documentation update.

- PASS: issue creation/edit commands returned expected URLs; native sub-issue and dependency mutations succeeded.
- PASS: GitHub API read-back confirmed epic children 5/7/11/12/26/27; #26 blocked by #5; #27 blocked by #5/#6/#11/#26; #12 blocked by #5/#6/#7/#11/#26/#27. #13 retains #10/#12.
- PASS: `PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH GOCACHE=/private/tmp/jayburd-go-cache make check test build` (Go vet/race tests, documentation links, migration-file checks and binaries). Initial sandbox cache-write denial resolved by an approved rerun; no cloud credentials used.
- PASS: `git diff --check` and `bash scripts/check-stage-runbook.sh --staged`.
- PASS: native dependency read-back via `gh api repos/justin-napolitano/jayburd-bookkeeping/issues/12/dependencies/blocked_by --jq '.[].number'`; equivalent checks covered #26/#27/#13 and epic #25 sub-issues.
- Source review: [GCP register](../research/gcp-sources.md), selected sections only.

## Remaining work

No GitHub token exchange or image publication has been executed.

Next: Terraform runtime settings/Invoker grants and serialized migration/rollout.
Runtime browser configuration and private service-token forwarding are implemented
with synthetic evidence on this branch. Identity and Infisical acceptance remain
open. PRs #23/#24 were open at the last feature review; recheck before final cloud
acceptance. Final #12 proof still requires recovery, 1,000 postings and measured
spend. Default CI remains credential-free. No live-bookkeeping approval follows
from foundation provisioning.

## PR handoff

Review Terraform runtime/migration roots, deployment trust/state isolation and the
owner-dispatch manifest workflow. #27 stays open: this is implementation evidence,
not deployment or real identity acceptance. No SQL migration or accounting/API
contract changes. No provisioning or workflow dispatch performed. Hosted checks
pending PR creation; see current slice above for local evidence.

## Foundation evidence — prior slice

- PASS: Terraform 1.15.7 `validate` and `test -filter=tests/foundation.tftest.hcl`, six mocked runs; Google provider 8.6.0 schema validation caught and resolved an unsupported budget notification block before handoff.
- PASS: provider lock records signed darwin_arm64/linux_amd64 packages.
- PASS: isolated `make infra-test` (format, readonly lock initialization, schema validation and all six mocked tests); no backend or owner variable files are copied.
- PASS: current-branch `make check test build` with the pinned Go toolchain/cache command above; `bash -n scripts/test-infra.sh` and `git diff --check`.
- PASS: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/seed.yml .github/workflows/infrastructure.yml` with pinned Go 1.26.9.
- Sandbox provider handshake/registry DNS and Go checksum fetch failures resolved through approved reruns. No cloud credentials, state or plans were retrieved.
- Hosted CI remains pending PR creation; staged runbook gate is required before this commit.
- PARTIAL: `gcp-identity-boundary` proves configured trust and grants offline; live token allow/deny tests remain unexecuted. `gcp-cost-review` has an allocation envelope, not a verified regional quote. `gcp-promotion-review` confirms no branch protection/environments; build/app promotion implementation belongs to #27.

## Cloud SQL evidence — prior slice

- PASS: Terraform application format/validate and three mocked tests; existing six foundation tests pass through `make infra-test`.
- PASS: `make cloud-sql-test` uses an internal Docker network, non-superuser administrator with ADMIN on test login roles, unchanged product migrations and a runtime with no table/DDL authority. `TestRestrictedMigration`, `TestMigrationContention`, `TestMigrationDirtyFailure` and checksum cases pass.
- PASS: `make migration-test ledger-test` including transactional failure/dirty state, posting/replay, multicurrency, restoration and readiness. Additional readiness denial cases for deployment DDL/table access and cloud administrator membership pass.
- PASS: final `make check test build cloud-sql-test ledger-test` using pinned Go 1.26.9; actionlint and govulncheck pass (no reachable vulnerabilities).
- PASS: `scripts/test-migrator-container.sh --local-build` verifies embedded SQL offline as non-root/read-only and rejects down; `scripts/test-api-container.sh --local-build` proves inert routes and graceful shutdown.
- PARTIAL: canonical local Docker source build failed on the Docker VM outbound proxy CA during module download. TLS verification remains enabled. Explicit host-build fallback proves runtime only; hosted CI still requires canonical source builds for both images.
- Boundary: the first bootstrap fixture lacked ADMIN on IAM-style login roles; PostgreSQL rejected ALTER ROLE. Explicit platform-admin fixture grants resolved this, and required owner authority is documented. Real Cloud SQL bootstrap privilege compatibility remains unverified.
- No GCP apply, token retrieval, live restore or financial data access. #26 remains partially open for those named cloud evidence gates.
