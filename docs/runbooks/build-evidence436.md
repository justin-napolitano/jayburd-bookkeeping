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

## Current evidence

[Trusted build run](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/42341240949).
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

## Remaining work

The coordinator must resolve the scan findings through reviewed source changes,
rerun the trusted build and require four successful runtime/scan/publication
receipts. Recompute each receipt using scripts/image-receipt.mjs, compare the
tested image ID with build metadata and published repository digest, and verify
the scan reports contain zero HIGH and zero CRITICAL findings. Preserve the
existing gate. This run cannot authorize deployment.

## PR handoff

Read-only build review complete; no production source or cloud changes authored.
The coordinator owns issue updates, source fixes and subsequent rollout evidence.
