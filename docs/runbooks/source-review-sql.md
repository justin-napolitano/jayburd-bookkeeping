# Source review SQL delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #96, bounded migration11 review/category/draft/source_post implementation.
Contract: [source review SQL](../contracts/source-review-sql.md). No provider/cloud
access or automatic posting. Existing migrations1–10 remain unchanged.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Review | Read AGENTS, index/bibliography, durable import, posting and recovery contracts. Coordinator approved exact signatures, two-line category drafts, explicit FX gate and full posting body in recoverable source_post before DDL. |
| Implementation | Scoped readable inbox, bounded source/filter snapshot cursors, immutable drafts and source journal links, source review provisioning, same-transaction posting and recovery. |
| Review | Coordinator approved source revision changes for review projection, exact registry function allowlist extension and compatibility P2401 for existing detail reads. Independent agent reviewed SQL/contract with no blocking finding. |
| Verification | `bash scripts/test-source-registry.sh` PASS: 28 top-level groups (7 new review), 56 subtests; `make check test build` PASS, 2026-10-13. |

## Current evidence

`bash scripts/test-source-registry.sh` uses pinned PostgreSQL17.11/golang-migrate4.20.1
images, pinned local Go1.26.9, disposable private Docker networking and fictional
operator/runtime identities. Logs `/private/tmp/jayburd-review-sql-proof.log` and
`/private/tmp/jayburd-review-sql-default.log`; both commands exited 0.

New tests cover explicit outflow/inflow category posting, no journal on draft,
immutable replay and remount posting lines, exact source legs, filtered pagination
and stale snapshots, current prepare/post/review grants, denied roles/scopes,
late link/source/projection/command fault rollback, concurrent execute and another
key for an already-linked draft. Additional fixed JPY/KWD cases prove visible
original units and explicit FX denial; overlap holds and closed-period checks
cannot be bypassed. Existing import/registry proofs run alongside them.

Initial syntax proof applied migration11 successfully. Its old exhaustive registry
EXECUTE expectation failed until the coordinator-authorized four-function allowlist
extension; observer/publisher permissions remain empty. No blanket grant or weakened
role assertion was introduced. A closed-period test first omitted required close
metadata, then observed the expected stale-book gate when closure followed draft;
the final fixture closes before drafting to exercise the ledger period gate itself.

## Remaining work

Coordinator owns integrated migration/ledger/Cloud SQL/Go/HTTP/browser/demo checks
and version11 shared expectations. This lane proves SQL behavior, not hosted
identity, source acquisition or accounting policy approval. Full FX/splits, overlap
matching, corrections/reversals, reconciliation and live cutover remain gated.

## PR handoff

Owned files: migration11 pair/checksum, new source review integration tests,
coordinator-approved exact registry privilege fixture extension and contract/runbook.
Required make check/test/build and staged runbook guard before commit. Root integrates
and publishes; no lane push or deployment.
