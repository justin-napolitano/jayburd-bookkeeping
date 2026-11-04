<!-- stage-runbook: true -->
# Sandbox scanner installer repair —437

## Status and boundary

CI437 frommain46e67ea, child/blocker436 deployment. Same Trivy0.75.0, pinned
official GitHub binary SHA; unchanged actual image scans, severity/report, runtime,
owner/main/WIF/publication. No product dependency, SQL/API, Production workflow,
provider/private/secret/IAM/consent change.42+unclaimed.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |436/437/65 and build41765752855 read; dedicated branch/worktree, prior source/release boundaries and exact public action/asset source reviewed |
| Reproduce | All four image runtime tests passed, then installer exit6 before scanning; retry reproduced; get.trivy.dev independently fails DNS. No images published or Sandbox schema/runtime changed |
| Source decision | Fixed same-version GitHub release API digest matches independently downloaded bytes; separate verified installation supported by existing pinned action |
| Implementation | Hosted-only checksum-first installer; default/corrupt-byte negatives; Sandbox build steps before cloud authentication; scanner action installation only omitted after verified install |
| Validation | Core make check/test/build/release-test PASS with documented installed Go1.26.5 external modfile; synthetic inert/corrupt-byte negatives, shell syntax/diff/size and pinned actionlint1.7.7 PASS. Actual hosted positive proof remains required |

## Current evidence

[Source record](../research/scan.md) distinguishes DNS/setup failure, integrity
verification and actual vulnerability scanning. No scan pass inferred yet.

## Remaining work

Validate source, publish/merge bounded repair, then rerun trusted main build for436.
No deployment before matching successful runtime/scan/publication receipts.

## PR handoff

Refs437/436/412. No migration impact or product runtime behavior change. Same
already-approved scanner dependency, no scanner/security gate weakening. Preserve
the intended Sandbox-only rollout; no Production or provider action follows here.
