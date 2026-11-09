<!-- stage-runbook: true -->
# Artifact policy independent build evidence

## Status and boundary

VERIFIED: all four trusted candidate images pass independent receipt, source,
runtime, scan and inventory acceptance for Issues 473/436. Root owns the evidence
commit and all cloud actions. This file proves candidate build acceptance only;
API/web deployment and owner visual review remain separate. Worker and migrate
are build evidence only, with no rollout in this change. No credential, private
environment, state, plan, financial record or provider access occurred. SQL 1–47
remain immutable and migration 48 is unallocated.

## Stage ledger

| Stage | Observed evidence |
| --- | --- |
| Trusted identity | Run 42541738958 attempt 1, workflow_dispatch on main, exact merged source 1d8cae6f25eabd87159ba3e88bd962e2b04732bf. Created/started 2026-11-09T00:43:40Z. Repository and head repository 1414515318; owner, actor and triggering actor 55423093, justin-napolitano. Workflow 381644107, .github/workflows/build-images.yml. Independently verified public metadata. |
| Prerequisites | Frontend 128917961782 SUCCESS 00:47:11Z; full application-integration 128917961939 SUCCESS 00:51:40Z; backend validation 128917961950 SUCCESS 00:59:50Z, all 2026-11-09. |
| Runtime, scans and publication | Every image job SUCCESS, including exact candidate runtime, HIGH/CRITICAL scanner gate, dependency inventory, publication without rebuilding and successful candidate evidence retention. Final image job completed2026-11-09T01:02:02Z; overall SUCCESS was observed with final run metadata updated01:02:03Z. updated_at is not treated as an exact overall completion timestamp. |
| Independent recomputation | Every complete candidate.json deep-equals existing scripts/image-receipt.mjs recomputation with fixed expected project/component/source/run/attempt and downloaded build/repository metadata. Four-component completeness is mandatory. |
| Source, scan and inventory | All scanned image IDs equal tested IDs; architecture is amd64. OCI revision and BuildKit source/revision/entrypoint match this source and each Dockerfile. All nonempty CycloneDX inventories match tested ID and source revision and identify Trivy 0.75.0. Every report has 0 HIGH and 0 CRITICAL findings. |

## Current evidence

[Trusted build 42541738958](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/42541738958)
completed SUCCESS. The workflow builds once before cloud authentication,
proves the exact local candidate runtime, applies the pinned scanner gate,
records inventory and publishes the tested image without rebuilding. Public
artifact enumeration contains four successful candidates and no rejected-scan
artifact.

| Component | Image job | Artifact ID | HIGH | CRITICAL | SBOM components |
| --- | --- | --- | ---: | ---: | ---: |
| api | 128922340808 | 13642227568 | 0 | 0 | 106 |
| migrate | 128922340793 | 13642237343 | 0 | 0 | 44 |
| web | 128922340869 | 13641889324 | 0 | 0 | 46 |
| worker | 128922340770 | 13642108184 | 0 | 0 | 301 |

Downloaded public evidence is under `/private/tmp/jayburd-473-build-42541738958/`.
Each `image-COMPONENT-1d8cae6f25eabd87159ba3e88bd962e2b04732bf-42541738958-1/`
contains candidate.json, build.json, repo-digests.json, vulnerabilities.json and
sbom.cdx.json. The independently authored verify.mjs imports the existing receipt
function, uses assert.deepStrictEqual on each complete receipt, requires all four
components, and separately checks tested/scanned identity, architecture, source
provenance and SBOM identity/source/tool. verified-summary.json records numerical
results; trusted-run.json, jobs.json and artifacts.json preserve selected public
metadata. Initial pending identity metadata is separately retained.

Verified immutable candidates:

| Component | Registry image |
| --- | --- |
| api | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:9747304373c1c9923518b0675cb85a2fcdde1ed1f397244df59208683fc790e0` |
| migrate | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/migrate@sha256:e56861790df600d41df4482800d258134909422cbf87cea94812bd5f4e9586f6` |
| web | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:31873a8552222fabf12963c704e943f1559d9f1f6663a3f67b0778cf52334a34` |
| worker | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:20e2b507c6403f69f2b17189a7ed5f5ce7e96a05003f8096355e8c787e34651d` |

Exact tested image identities:

| Component | Tested image ID |
| --- | --- |
| api | `sha256:5a05c35946f874621332718db53c866385a7baf4c5cb5f2ebd1746101d7b9686` |
| migrate | `sha256:0d81bbe9fb31ae4ba35febc7af2de91764bddd55185e93b2c58781f91985e28b` |
| web | `sha256:b2215e9a88bc8ded11a9437776a1c890baec450679ab379ed81606449750803d` |
| worker | `sha256:c38dd4ccfb7c9fed5e14243bbba7e7634866669bc9d47bb8ee5b1c322a179834` |

Published repository digests can differ from BuildKit buildDigest values. The
existing receipt binds the tested image/config-or-manifest identity to a single
published repository digest; no tag-based substitution occurs. These are unsigned
candidate metadata, not a SLSA attestation. Zero HIGH/CRITICAL refers to the retained
build scan reports, not an unlimited future vulnerability guarantee.

## Remaining work

Candidate build/runtime/scan/publication acceptance is complete for this exact
source/run. Root separately owns API/web image-only rollout, unchanged settings
and traffic checks, rollback pins and post-rollout probes. This file does not
claim managed deployment acceptance, owner visual acceptance, live bookkeeping
approval, document ingestion support or worker/migrate deployment.

## PR handoff

Independent artifact verification complete. Only this allocated new marked
evidence file is edited by this lane. Root owns staging, commit, publication and
all cloud mutations. Stage-runbook structure and git diff --check PASS for the
completed evidence file; no application code was changed by this lane.
