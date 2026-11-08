<!-- stage-runbook: true -->
# Onboarding rollout independent build evidence

## Status and boundary

Issue 460 onboarding delivery and Issue 436 controlled existing Sandbox rollout.
This lane independently reviewed public GitHub metadata and downloaded build
artifacts. Root owns the evidence commit and all cloud/deployment actions.
No credential, private environment, state, plan, provider or financial record
access occurred. SQL 1–47 remain immutable; migration 48 is unallocated. A
successful candidate build does not prove managed rollout acceptance. The
worker candidate was verified; no worker rollout is authorized by this proof.

## Stage ledger

| Stage | Observed evidence |
| --- | --- |
| Trusted identity | Run 42498017914 attempt 1, workflow_dispatch on main, source 950c26d9e3851a919c8ce75ba7728a4a4a314870. Repository and head repository 1414515318; owner, actor and triggering actor 55423093, justin-napolitano. Workflow 381644107 at .github/workflows/build-images.yml. Created/started 2026-11-08T16:04:38Z. |
| Prerequisites | Frontend 128784463558 SUCCESS 16:08:13Z; full application-integration 128784463780 SUCCESS 16:14:19Z, including actual onboarding; backend/SQL 128784463757 SUCCESS 16:26:09Z. All dates 2026-11-08. |
| Runtime, scan and publication | All four image jobs SUCCESS. Each exact candidate runtime, HIGH/CRITICAL scanner gate, dependency inventory, publication without rebuilding and candidate artifact retention step SUCCESS. Last image and overall run completed 16:28:14Z. |
| Independent receipt recomputation | All four complete candidate.json objects deep-equal the existing scripts/image-receipt.mjs result using fixed expected project, component, source, run and attempt, downloaded build metadata and published repository digests. |
| Source and inventory binding | Every scan image ID equals its tested image ID; OCI revision and BuildKit provenance revision/source match exact merged 950c26d9. Every nonempty CycloneDX SBOM separately matches the tested image ID and source revision; scanner tool identity is Trivy 0.75.0. Each report has 0 HIGH and 0 CRITICAL. |

## Current evidence

[Trusted build 42498017914](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/42498017914)
completed SUCCESS. The workflow builds once before cloud authentication,
proves the exact local runtime, blocks HIGH/CRITICAL vulnerabilities using the
pinned scanner, records inventory, and publishes the tested image without
rebuilding. No rejected-scan artifacts were listed for this run.

| Component | Image job | Artifact ID | HIGH | CRITICAL | SBOM components |
| --- | --- | --- | ---: | ---: | ---: |
| api | 128790077574 | 13625017370 | 0 | 0 | 106 |
| migrate | 128790077625 | 13625450903 | 0 | 0 | 44 |
| web | 128790077533 | 13624891786 | 0 | 0 | 46 |
| worker | 128790077526 | 13624942715 | 0 | 0 | 301 |

Downloaded evidence is under `/private/tmp/jayburd-460-build-42498017914/`.
Each `image-COMPONENT-950c26d9e3851a919c8ce75ba7728a4a4a314870-42498017914-1/`
contains candidate.json, build.json, repo-digests.json, vulnerabilities.json and
sbom.cdx.json. The independently authored verify.mjs imports the existing receipt
function, uses assert.deepStrictEqual on every full receipt, checks all four
components are present, and separately verifies scan, provenance and SBOM
identities. Its verified-summary.json records numerical results; trusted-run.json
and artifacts.json retain selected public identity metadata.

Verified immutable candidates:

| Component | Registry image |
| --- | --- |
| api | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:971a9381b4e82c69a0398623a4fce85f679dab6e7a274f1fa9619b98fee2b74e` |
| migrate | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/migrate@sha256:199ed337de73e622c7398516a90ef196620b100701cb3f07924e6cd86df5f31a` |
| web | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:def2c3f5f23a4a509f9af4e2ed5b662e2747316b2c2c17db24ae920c7de9ccfa` |
| worker | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:dae08d1d38145a3c326c8bf09b4a73bc855e4ea4e7d2dd9671607fef74be7e0f` |

Exact tested image identities:

| Component | Tested image ID |
| --- | --- |
| api | `sha256:4f7f58b11b74076ce8e16983b355137eea708873d94986dfb61caf92193aa988` |
| migrate | `sha256:014c29b081dfb3c30dc7d94e55314c623b115b8cd100e5e0b1ff58e337a1610e` |
| web | `sha256:f6af7f71b015e5243cdb7e5d06740251cc04c251dfa52103cfcca7760ea72d3f` |
| worker | `sha256:99993b97acd79555e357bca97bce98c350c37ce4c94190ca47e2fc8e636de334` |

Published repository digests may differ from BuildKit buildDigest values. The
existing receipt binds the tested image to the build's config/manifest digest
and the single published repository digest; no tag-based substitution occurs.
These are unsigned candidate metadata, not a SLSA attestation or a new scanner
assessment beyond the retained build reports.

## Remaining work

Build, runtime, scan, inventory and publication evidence is satisfied for this
exact source/run. Root separately owns managed migration predecessor/status,
service rollout, traffic/settings preservation and post-rollout probes. This
file does not claim those acceptance steps, user onboarding completion, live
bookkeeping approval or any worker deployment.

## PR handoff

Independent verification complete. Only this allocated marked evidence file is
edited by this lane; root owns commit/publication and all cloud actions.
