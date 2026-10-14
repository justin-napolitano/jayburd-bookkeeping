<!-- stage-runbook: true -->
# Link and controlled worker delivery

## Status and boundary

Implemented #104 browser Link and #7/#27 controlled worker promotion on open
PR #111. No deployment, provider calls, secret retrieval or live-account approval.

## Stage ledger

| Stage | Work | Evidence |
| --- | --- | --- |
| Scope | Separate browser adapter, Terraform and release/review lanes | Lease LINK-WORKER-112 in #65; shared interfaces frozen |
| Design | Fixed SDK, transient tokens; separate bootstrap/runtime; paused release | [Browser contract](../contracts/plaid-link-browser.md), [worker contract](../contracts/worker-release.md), bibliography |
| Browser | Lazy real SDK adapter, timeout/abort cleanup and stale callback fencing | 253 unit tests; 15 browser journeys, including two intercepted-CDN tests |
| Infrastructure | Dedicated identity/WIF/state; owner-only IAM and Scheduler | 56 mocked Terraform tests across seven roots |
| Release | Exact-byte manifest, build/rollback pins, isolated apply and observed receipt | 34 Node release/receipt tests; both fake-executable orchestration suites; actionlint |
| Integration | Independent review fixed manifest/TF int64 mismatch | Default Go race/check/build and both browser/API/PostgreSQL journeys PASS |

## Current evidence

PASS: `make check test build`, `make web-check web-test web-build web-browser`,
`make app-integration`, `make connection-browser-test`, `make release-test`,
`bash scripts/test-infra.sh`, and actionlint for the worker workflow. The final
worker manifest/receipt rerun includes the CLI mapping regression (11 tests plus
23 existing release tests). Infrastructure uses google8.6.0 mocks without cloud
authentication. Integration used temporary ports39481–39483 to preserve the
existing local demo; all temporary source substitutions were restored.

No SQL changes. No actual owner-state migration, deployment, SDK/provider session,
secret propagation, managed OIDC, queue delivery or account readiness is proven.

Initial GitHub runs were blocked by account billing. After the owner made the
repository public, reruns started successfully. Infrastructure attempt2 of
run38581323364 exposed an undeclared ripgrep dependency in the shell release
tests. Replaced those searches with portable extended grep, including older
negative assertions that had silently skipped checks when ripgrep was missing.
Local `make release-test` rerun passes; new GitHub results remain pending.

## Remaining work

Owner bootstrap/adoption follows the [operator guide](../../deploy/WORKERS.md).
Infisical setup, real Link Events association and hosted Sandbox proof remain
separate acceptance gates. Reconnect/OAuth return remain excluded here. The first
release stays paused; this is not queue-drain or measured scale evidence.

## PR handoff

PR #112 is stacked against #111; merge #111 first, then retarget to main.
Preserve all existing issue gates. No SQL migration or automatic scheduling
activation is part of this change. Existing worker deployments require the owner
state handoff documented in the worker contract before adopting this split.
