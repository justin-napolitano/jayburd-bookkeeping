<!-- stage-runbook: true -->
# Image dependency security patch

## Status and boundary

Issue 454 blocks controlled Sandbox rollout 436. Branch
fix/436-image-vulnerabilities starts at merged f8d9e06. Only shared x/net
dependency metadata and existing web runtime library bounds change. No SQL,
product behavior, provider call, cloud action, scanner waiver or credential use.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Diagnosis | Trusted build 42341240949 passed all source/application/runtime checks, then rejected all four images: API 2 HIGH, migrate 1 HIGH, worker 7 HIGH, web 3 HIGH; all 0 CRITICAL. Authentication/publication skipped. |
| Source review | IS01/IS02 in the linked source record distinguish full official Go advisory review from unavailable Alpine advisory pages and retained scanner repair bounds. |
| Patch boundary | Shared x/net v0.62.0 updates all Go binaries. Extend existing runtime APK bounds for musl/musl-utils 1.2.6-r3 and zlib 1.3.2-r1 on unchanged pinned Node base. Existing OpenSSL bounds and nonroot runtime remain. |
| Independent review | Root and private_registry independently reviewed the concrete patch and found no actionable source issue or gate change. The latter independently confirmed the official Go fixed version and Alpine access limits. No direct x/net/html caller is established; the installed-package gate remains mandatory. |

## Current evidence

Go's official module download verified v0.62.0 checksums and source tag commit.
Source checks are running with a temporary local adapter for installed Go 1.26.5;
tracked Go 1.26.10 is unchanged. The first host run could not write the restricted
module cache; the adapter now points only x/net at its verified temporary source.
The BuildKit web build is blocked by local Alpine CDN certificate validation,
including the unchanged OpenSSL bounds. No TLS bypass or source workaround was
introduced; the canonical exact runtime/image proof remains required. An earlier
legacy builder ignored Dockerfile-specific ignore rules and failed on excluded
test fixtures; this is not a canonical BuildKit failure. See
[source register](../research/image-security-sources.md) for evidence and limits.

## Remaining work

Complete local source/build/runtime checks, record sandbox limits, obtain
coordinator source review and full canonical PR checks. After merge, require a
new trusted main four-image runtime/scan/publication run and independently
recomputed receipts. The failed predecessor run cannot authorize deployment.

## PR handoff

No merged migration changes. Root owns merge, trusted build dispatch and every
cloud/deployment operation. Every implementation commit includes this marked
runbook and passes the staged runbook check.
