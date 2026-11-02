<!-- stage-runbook: true -->
# Selection registration API delivery —393

## Status and boundary

REG393-API, feat/393-registration-api from399/44b8a53. Owner reviewed and merged
399 after all five checks PASS. Source/tests only; no managed apply, provider I/O,
source-limit increase, second live account or Sandbox renewal.36 immutable.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live393/290/65 and reviewed391/392 contracts read; dedicated worktree and finite API lease |
| Contract | registration-http freezes selection transport/recovery and290 no-new-metadata assessment;290 remains open for394 presentation |
| Implementation | Closed16KiB selection route, finite/profile preflight, server audit UUID, restricted SQL preparer; typed recovery/preview/explicit terminal Done; managed readiness requires36 |
| Focused proof | Service/HTTP denial tests PASS;33 restricted connection-state cases PASS including actual HTTP→Go→SQL initial/append and lost execute response; zero provider/crypto calls and no journals |
| Browser recovery | New component browser case PASS: dropped prepare/execute responses, remount before ack, mobile layout, no browser persistence; separate from real SQL proof |
| Broad validation | `make check test build`, `make ledger-test`, locked `make web-check web-test web-build web-browser` PASS (350 unit/27 browser); `make app-integration` PASS through production Next bridge→Go→synthetic SQL; no hosted proof claimed |

Local tools: tracked Go1.26.10 unchanged; temporary modfile uses installed1.26.5.
Official checksum-verified Node24.18.0, locked npm install. Initial ledger invocation
stopped before SQL because Docker was absent from PATH; corrected invocation PASS.
CI must verify the unchanged tracked Go toolchain at the exact PR head.

## Current evidence

SQL36 source merged, not applied. Initial Production checking's187-row evidence is
separate; no real records/credentials read during implementation. Shared recovery
must preserve immutable selected mappings, current authorization and terminal ack.

## Remaining work

Route/service/types/recovered command panel and local gates complete. Exact-head
CI/human security review remain.393 stays open: this transport increment does not
yet compose picker-facing eligibility/reason/already-registered/history summaries
or optional advanced mapping in the selection preparer.394 picker,290 private
labels and395 release/hosted qualification remain. Existing advanced binding is
retained separately; no feature completion claim from source merge.

## PR handoff

Refs393/391/290/392. No new migration or dependency. Deploy reviewed36 before this
runtime; rollback must understand saved register_accounts commands.
