<!-- stage-runbook: true -->
# Customer UI independent build evidence

## Status and boundary

VERIFIED: all four trusted candidate images pass independent build receipt,
source, runtime, scan and inventory acceptance for Issue 479. Root owns all cloud
writes, publication and this evidence commit. The release scope is WEB ONLY.
Existing API00026-5nt remains pinned to its accepted artifact-policy image:
`us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:9747304373c1c9923518b0675cb85a2fcdde1ed1f397244df59208683fc790e0`.
The three newly built API/migrate/worker candidates below are evidence only and
are not rollout inputs. No private environment, credential, financial record,
provider, state or plan access occurred. SQL 1–47 remain immutable and migration
48 is unallocated. Build acceptance does not establish managed deployment or
owner visual acceptance.

## Stage ledger

| Stage | Observed evidence |
| --- | --- |
| Trusted identity | Run 42561705439 attempt 1, workflow_dispatch on main, exact merged source 9b2a6568ffb05ed4e66c77857caa0ad07cc5509b. Created/started 2026-11-09T04:25:05Z. Repository/head repository 1414515318; owner/actor/triggering actor 55423093, justin-napolitano. Workflow 381644107 at .github/workflows/build-images.yml. Independently verified selected public run metadata. |
| Prerequisites | Frontend 128979347020 SUCCESS 04:28:33Z; full application-integration 128979347016 SUCCESS 04:33:17Z; backend 128979346833 SUCCESS 04:49:22Z, all 2026-11-09. |
| Runtime, scan and publication | Every image job SUCCESS, including exact candidate runtime, HIGH/CRITICAL scanner gate, dependency inventory, publication without rebuilding and successful candidate evidence retention. Last image job completed 2026-11-09T04:51:30Z. Overall SUCCESS was observed with final public run metadata updated_at 04:51:31Z; updated_at is not treated as an exact completion time. |
| Independent receipt recomputation | All four complete candidate.json objects deep-equal existing scripts/image-receipt.mjs recomputation using expected public project/component/source/run/attempt and downloaded build/repository metadata. Four-component completeness required. |
| Source and inventory binding | Every scan ID equals its tested ID and architecture is amd64. OCI revision and BuildKit source/revision/entrypoint match the exact merged source and component Dockerfile. Every nonempty CycloneDX inventory matches tested ID/source revision and identifies Trivy 0.75.0. Each report has 0 HIGH and 0 CRITICAL findings. |

## Current evidence

[Trusted build 42561705439](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/42561705439)
completed successfully. The workflow builds once before cloud authentication,
proves the exact local runtime, applies the pinned scanner gate, records inventory
and publishes the tested image without rebuilding. Artifact enumeration contains
four successful candidates and no rejected-scan artifact.

| Component | Image job | Artifact ID | HIGH | CRITICAL | SBOM components | Release scope |
| --- | --- | --- | ---: | ---: | ---: | --- |
| api | 128986039471 | 13650089864 | 0 | 0 | 106 | Build evidence only |
| migrate | 128986039422 | 13650582458 | 0 | 0 | 44 | Build evidence only |
| web | 128986039365 | 13650706059 | 0 | 0 | 46 | WEB-ONLY rollout candidate |
| worker | 128986039441 | 13651060397 | 0 | 0 | 301 | Build evidence only |

Downloaded public evidence is under `/private/tmp/jayburd-479-build-42561705439/`.
Each `image-COMPONENT-9b2a6568ffb05ed4e66c77857caa0ad07cc5509b-42561705439-1/`
contains candidate.json, build.json, repo-digests.json, vulnerabilities.json and
sbom.cdx.json. Independently authored verify.mjs imports the existing receipt
function, uses assert.deepStrictEqual on each complete receipt, requires all four
components, and separately verifies scanned/tested identity, architecture, source,
BuildKit entrypoint and SBOM source/identity/tool. verified-summary.json records
numerical results. trusted-run.json, jobs.json and artifacts.json retain selected
public metadata; initial pending metadata is separately preserved.

Verified immutable candidates:

| Component | Registry image |
| --- | --- |
| api | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:21f7a7ac0e64630090f885c9dba7cfa6f4ec82f182f8ac7ce07ab06b7ef96e6a` |
| migrate | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/migrate@sha256:bf1b7dda8f4bfc5a340da82890af3f1bb7897968ae43094cc9f16948f44fe6b0` |
| web | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:e4521574edecc6a5901c7f590341112484370e6cf93d63a3f13e30c79f3ba16d` |
| worker | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:19e200e64ff7cdf7c6fb8bae8bed426595667311cecae884d9e1b2dcab104b9d` |

Exact tested image identities:

| Component | Tested image ID |
| --- | --- |
| api | `sha256:dd62b6e0038387538d1fa5df209a48a211790950acdaff7dde461ac0c51b79af` |
| migrate | `sha256:7ed55f542aa7623a3452eaa196c18db08b938f8844d50b3624a6bc15610b6d08` |
| web | `sha256:f420d28c5252334879c07a4e078df0871232f03460b71c4c9ab653a6b05f2c7c` |
| worker | `sha256:edc2833419758f4f5048017a898b071ad807bac37b6121b5c94d14b12bc597c0` |

Published repository digests can differ from BuildKit buildDigest values. The
existing receipt binds the tested config/manifest identity to a single published
repository digest; no tag substitution occurs. These are unsigned candidate
metadata, not a SLSA attestation. Zero HIGH/CRITICAL describes the retained build
reports, not an unlimited future vulnerability guarantee.

## Remaining work

Candidate acceptance is complete for this exact source/run. Root separately owns
WEB-ONLY image rollout, unchanged selected settings, sole-traffic/health and
transport checks, while preserving the accepted artifact-policy API. This file
does not claim deployment acceptance, owner visual approval, live bookkeeping
approval, document ingestion, new bank consent, or deployment of new API/migrator/
worker images. No migration execution is required by this presentation-only
change.

## PR handoff

Independent verification complete. Only this allocated new marked evidence file
is edited by this lane. Root owns staging, commit, publication and cloud writes.
No build dispatch, image publication or deployment was performed by this lane.
Stage-runbook structure and git diff --check PASS for this completed evidence.
