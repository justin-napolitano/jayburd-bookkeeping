<!-- stage-runbook: true -->
# Sandbox 436 independent build evidence

## Status and boundary

Issue 436 controlled private rollout. This review uses only GitHub build metadata
and generated vulnerability artifacts. The coordinator owns deployment and cloud
actions. No credential, environment, state, plan, provider or financial data was
accessed. A candidate build is not rollout acceptance.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Trusted identity | Run 42341240949, attempt 1, workflow_dispatch on main at f8d9e06b4c4a39fb152fed58ebde416abc854149; actor and triggering actor justin-napolitano (55423093), repository justin-napolitano/jayburd-bookkeeping (1414515318), expected build-images.yml. |
| Prerequisites | Reusable validate, frontend and application-integration jobs all succeeded. |
| Candidate runtime | Exact candidate runtime proof succeeded independently for api, migrate, web and worker. |
| Scan gate | All four jobs failed the pinned HIGH/CRITICAL scan; cloud authentication and publication were skipped in every job. Overall run completed failure. |
| Remediation | PR 456 merged as a4ad6c1dd2a6b257981c4d19513ad0aebbeb0daf after all five canonical checks passed; x/net and three web runtime library bounds patched without weakening gates. |
| Corrected trusted build | Run 42343986400, attempt 1, manual main build of exact a4ad6c1. Numeric actor and triggering actor 55423093, repository 1414515318, owner 55423093, expected build-images.yml. Overall SUCCESS; all three prerequisites and four image jobs SUCCESS. |
| Independent recomputation | All four candidate receipts deep-equal scripts/image-receipt.mjs recomputation from build metadata, tested IDs and repository digests. Each scan Metadata.ImageID equals testedImageID; each OCI revision label equals exact source a4ad6c1. All four have 0 HIGH/0 CRITICAL and nonempty CycloneDX SBOMs. |

## Current evidence

[Rejected predecessor build](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/42341240949).
Counts below are scanner findings per installed target/package, not unique CVEs.

| Component | Image job | HIGH | CRITICAL | Rejected artifact ID |
| --- | --- | ---: | ---: | --- |
| api | 128312535410 | 2 | 0 | 13562466288 |
| migrate | 128312535396 | 1 | 0 | 13562396287 |
| web | 128312535401 | 3 | 0 | 13562536146 |
| worker | 128312535371 | 7 | 0 | 13561873890 |

The Go findings all report CVE-2026-78661 in golang.org/x/net v0.61.0,
fixed version 0.62.0, repeated across two API, one migration and seven worker
binaries. The web report lists CVE-2026-107891 in musl and musl-utils
1.2.6-r2 (fixed 1.2.6-r3), and CVE-2026-85091 in zlib 1.3.2-r0
(fixed 1.3.2-r1). These are retained scanner assertions; this artifact review
does not establish exploitability or independently validate the advisory text.

The four downloaded reports are under
`/private/tmp/jayburd-436-build-42341240949/rejected-scan-COMPONENT-f8d9e06b4c4a39fb152fed58ebde416abc854149-42341240949-1/vulnerabilities.json`.
Artifact IDs above identify the remote evidence independently of the temporary
local path. No successful image artifact, published digest or candidate receipt
exists for this failed run, so published receipt recomputation cannot proceed.

The [corrected build](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/42343986400)
completed SUCCESS. All four jobs separately passed exact candidate runtime,
pinned vulnerability scan, dependency inventory and publication without rebuild.

| Component | Image job | Artifact ID | HIGH | CRITICAL | SBOM components |
| --- | --- | --- | ---: | ---: | ---: |
| api | 128320971300 | 13563267112 | 0 | 0 | 106 |
| migrate | 128320971383 | 13562724019 | 0 | 0 | 44 |
| web | 128320971435 | 13563441649 | 0 | 0 | 46 |
| worker | 128320971439 | 13562991804 | 0 | 0 | 301 |

All four receipts, build metadata, repository digests, scan reports and SBOMs are
downloaded under
`/private/tmp/jayburd-436-build-42343986400/image-COMPONENT-a4ad6c1dd2a6b257981c4d19513ad0aebbeb0daf-42343986400-1/`.
The independent Node assertion imported the existing receipt function and used
fixed expected project, component, commit, run and attempt inputs, comparing its
full object against candidate.json with assert.deepStrictEqual. Scan identity,
zero HIGH/CRITICAL counts and nonempty CycloneDX components were separately
asserted. OCI revision labels match the exact build source in every scan.

Verified immutable candidates:

| Component | Registry image |
| --- | --- |
| api | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:45bc930c4853edf171e8b4c1a7f1684524248c4d03a59f1f4b48e8b1094dfd7c` |
| migrate | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/migrate@sha256:a2bdc64d39a852237d01d2dde5e884cba575fed96c64b30dfe6b074a63c7bf12` |
| web | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:e8f2a258a8454a1c455044691ea89cf5b84320e967e04c7c521e69969847796c` |
| worker | `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:ce23dad282ec8f88ab0cc4affa085161355d26c0d761b1770ea643c2f194a730` |

The tested image IDs are respectively API
`sha256:be5659af0562375e3a9460affc7338676de55d00e24700d972559f017d4bd306`,
migrate `sha256:209a431d6a188cfc4ce38896484f487cdead2765be7e535a039388ed7dc4eb23`,
web `sha256:39092651ce82fdea32943b5abfff1e3b8233cda7fdf1d254490fa9953956f393`,
worker `sha256:a69f268a310a77367acd87a956635451ee288a4e5e6d1f5d25df718918cb81dd`.
SBOMs confirm x/net v0.62.0 in every Go binary, and the web runtime resolves
musl/musl-utils 1.2.6-r3 and zlib 1.4-r0. APK minimums allow subsequent patched
versions; only these final tested/scanned digests are the verified candidates.

## Remaining work

The build gate is satisfied for corrected run 42343986400. The coordinator must
separately verify managed migration predecessor/status, approved image-only
rollout scope, service revisions and traffic, preserved operational settings and
post-rollout public/anonymous probes. These candidate-only receipts retain
unsigned metadata and are not a SLSA attestation or rollout acceptance.

## PR handoff

Independent artifact review complete; root owns this evidence commit, issue
updates and all rollout actions. No cloud action was performed in this review.
