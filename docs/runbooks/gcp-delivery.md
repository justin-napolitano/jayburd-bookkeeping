# Synthetic GCP delivery evidence
<!-- stage-runbook: true -->

## Status and boundary

Foundation implementation slice under [epic #25](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/25).
GitHub owns current task state; this ledger records decisions and observed review
results. Terraform foundation and synthetic validation are implemented. No infrastructure
has been applied and no cloud authentication, deployment or live-bookkeeping
approval is established.

## Stage ledger

| Stage | Evidence / outcome | Next action |
| --- | --- | --- |
| Reconcile existing work | Reviewed #5/#7/#11/#12 and open PRs #23/#24; preserved existing evidence and identity | Use native dependencies before claiming a slice |
| Publish bounded work | Created epic #25, database #26 and app #27; updated existing issues with named acceptance and recovery requirements | Native relationships verified |
| Record architecture | Updated GCP contract, index and annotated sources; scoped early proof separately from full month-close | Review documentation PR |
| Implement foundation | Two state buckets, nine identities, image repository, disabled build WIF and budget alerts; no runtime resources | Review IAM and owner bootstrap |
| Mocked provider proof | Terraform validate and six tests pass; invalid project/number/billing/region denied | Review PR and hosted CI |

## Current evidence

- PASS: issue creation/edit commands returned expected URLs; native sub-issue and dependency mutations succeeded.
- PASS: GitHub API read-back confirmed epic children 5/7/11/12/26/27; #26 blocked by #5; #27 blocked by #5/#6/#11/#26; #12 blocked by #5/#6/#7/#11/#26/#27. #13 retains #10/#12.
- PASS: `PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH GOCACHE=/private/tmp/jayburd-go-cache make check test build` (Go vet/race tests, documentation links, migration-file checks and binaries). Initial sandbox cache-write denial resolved by an approved rerun; no cloud credentials used.
- PASS: `git diff --check` and `bash scripts/check-stage-runbook.sh --staged`.
- PASS: native dependency read-back via `gh api repos/justin-napolitano/jayburd-bookkeeping/issues/12/dependencies/blocked_by --jq '.[].number'`; equivalent checks covered #26/#27/#13 and epic #25 sub-issues.
- Source review: [GCP register](../research/gcp-sources.md), selected sections only.

## Remaining work

Review and merge the foundation slice, then complete the linked delivery slices;
all named cloud checks remain unexecuted.
Project, billing, auth metadata and apply authorization block provisioning.
PRs #23/#24 were open at review; final cloud acceptance requires their merged
capabilities. Open historical issues are not closed or declared complete here.
Default CI remains credential-free; application/database behavior is unchanged.

## PR handoff

Review foundation Terraform, synthetic tests, credential-free workflow and IAM
contract against #5.
No SQL/API or runtime changes; no data migration needed. Foundation is unapplied;
owner privately reviews plan/state and completes bootstrap. Pool budgets, recovery
objectives and spending allocations are proposals requiring acceptance, not measured
results. #5 remains partially open.
Next unit: #26 database implementation after foundation review, with #7 local outbox work independently
schedulable after its existing prerequisites are evidenced. Final proof requires
1,000 postings, exact balances, delivery receipts/faults, denial tests, isolated
restore, compatible rollback and seven-day measured spend. See each issue for
named acceptance checks. Full-month #10 remains a live-pilot prerequisite.

## Foundation evidence — current branch

- PASS: Terraform 1.15.7 `validate` and `test -filter=tests/foundation.tftest.hcl`, six mocked runs; Google provider 8.6.0 schema validation caught and resolved an unsupported budget notification block before handoff.
- PASS: provider lock records signed darwin_arm64/linux_amd64 packages.
- PASS: isolated `make infra-test` (format, readonly lock initialization, schema validation and all six mocked tests); no backend or owner variable files are copied.
- PASS: current-branch `make check test build` with the pinned Go toolchain/cache command above; `bash -n scripts/test-infra.sh` and `git diff --check`.
- PASS: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/seed.yml .github/workflows/infrastructure.yml` with pinned Go 1.26.9.
- Sandbox provider handshake/registry DNS and Go checksum fetch failures resolved through approved reruns. No cloud credentials, state or plans were retrieved.
- Hosted CI remains pending PR creation; staged runbook gate is required before this commit.
- PARTIAL: `gcp-identity-boundary` proves configured trust and grants offline; live token allow/deny tests remain unexecuted. `gcp-cost-review` has an allocation envelope, not a verified regional quote. `gcp-promotion-review` confirms no branch protection/environments; build/app promotion implementation belongs to #27.
