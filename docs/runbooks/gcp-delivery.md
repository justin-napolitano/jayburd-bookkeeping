# Synthetic GCP delivery evidence
<!-- stage-runbook: true -->

## Status and boundary

Image-delivery implementation slice (#27) under [epic #25](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/25).
GitHub owns current task state; this ledger records decisions and observed review
results. Terraform foundation/database configuration, explicit IAM connector and
restricted migration runner are implemented. No infrastructure
has been applied and no cloud authentication, deployment or live-bookkeeping
approval is established.

## Image-delivery evidence — current branch

- PR #30 merge verified at `567eabe`; #5 is closed in GitHub. #6/#11/#26 and PRs #23/#24 remain open; closure is not inferred from configuration merges.
- Added standalone frontend packaging, exact-image runtime checks, owner-only main workflow, pre-publication scan/SBOM and candidate digest receipts.
- PASS: receipt tests reject changed configuration digests, missing provenance, wrong repository and ambiguous published identities.
- PASS: `make check test build`, browser type checks/33 unit tests/production build and three Chromium journeys; actionlint; canonical standalone frontend container with static asset and unavailable-route checks. API/migrator host-build runtime checks pass; their canonical builders remain hosted CI gates.
- Fixed the merged #30 CI readiness race: socket readiness saw PostgreSQL’s temporary initialization server. Cloud SQL and ledger proofs now wait for TCP, matching the migration harness. `make cloud-sql-test migration-test ledger-test` passes after the fix.
- The first frontend proof caught a self-proxy loop (default API port matched frontend port); the unconfigured image now targets unused loopback port 1. Node SIGTERM exit 143 is accepted; forced-kill exit 137 remains failure.
- No workflow dispatched, identity enabled, registry push or cloud mutation. Trivy publication scans and real federation remain unexecuted until explicit owner setup/dispatch.
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

- PASS: issue creation/edit commands returned expected URLs; native sub-issue and dependency mutations succeeded.
- PASS: GitHub API read-back confirmed epic children 5/7/11/12/26/27; #26 blocked by #5; #27 blocked by #5/#6/#11/#26; #12 blocked by #5/#6/#7/#11/#26/#27. #13 retains #10/#12.
- PASS: `PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH GOCACHE=/private/tmp/jayburd-go-cache make check test build` (Go vet/race tests, documentation links, migration-file checks and binaries). Initial sandbox cache-write denial resolved by an approved rerun; no cloud credentials used.
- PASS: `git diff --check` and `bash scripts/check-stage-runbook.sh --staged`.
- PASS: native dependency read-back via `gh api repos/justin-napolitano/jayburd-bookkeeping/issues/12/dependencies/blocked_by --jq '.[].number'`; equivalent checks covered #26/#27/#13 and epic #25 sub-issues.
- Source review: [GCP register](../research/gcp-sources.md), selected sections only.

## Remaining work

Foundation #29 and database #30 are merged. Review image delivery, then complete the ingress/rollout slices;
all named cloud checks remain unexecuted.
Project, billing, auth metadata and apply authorization block provisioning.
PRs #23/#24 were open at review; final cloud acceptance requires their merged
capabilities. Open historical issues are not closed or declared complete here.
Default CI remains credential-free; cloud behavior remains unverified until explicit provisioning.

## PR handoff

Review the [image contract](../contracts/image-delivery.md), build workflow and
standalone packaging against #27. Public APIs and merged migration SQL are unchanged.
Frontend remains unconfigured, receipts do not authorize deployment, and builder
WIF remains owner-disabled by default. Next: runtime browser configuration and
private service-token forwarding, then serialized Terraform migration/rollout.
#5 is closed in GitHub; real cloud gates remain unverified. Final #12 acceptance
still requires identity/secrets, recovery, 1,000 postings and measured spend.

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
