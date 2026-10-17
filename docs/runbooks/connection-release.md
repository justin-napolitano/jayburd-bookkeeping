<!-- stage-runbook: true -->
# Connection release delivery

## Status and boundary

Build on merged PR110 (60c0e3e). Fix explicit connection configuration through the
existing reviewed release pipeline; no deployment or provider call.

## Stage ledger

Reviewed release/hosted contracts and official variable/revision sections. Selected
runtime manifest v2 with mandatory explicit connection state and unchanged migration
v1. Preserve existing gate authority; record approved configuration separately from
provider/runtime acceptance.

## Current evidence

`make release-test`:23 Node tests and fake-executable deployment orchestration PASS.
Covers explicit null/enabled metadata, omissions, wrong project/aliases/overflow,
policy expiry, exact CLI input generation, migration isolation and rollback digest.
Independent review caught project-number int64 mismatch; fixed before handoff.
Default `make check test build`:PASS, including all-package race checks. No
browser/API or SQL behavior changed; their extra integration suites were not rerun.

## Remaining work

Actual hosted activation, Link/reconnect, owner Infisical configuration and worker
promotion remain separately gated. No enabled release manifest was created.

## PR handoff

Ready for review against main after PR110. Runtime manifests require v2; migration
v1 stays compatible. Legacy rollback receipts lack configuration evidence and fail
closed. No product SQL/API/browser changes, cloud mutation or credential retrieval.

## Historical attempt review release candidate (#160)

PR #176 merged with all five checks passing. Its browser change keeps an authorized
book selected after an attempt-only denial and redirects Cloud Run's alternate
service URL to the configured public origin. Successful image build
[39067510528](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/39067510528)
published matching API, migrator and web candidates from merge commit
`d1913796ad51699c361413b7915d4eb0e81ddc4d`. The next
`deploy/dev-release.json` candidate pins those exact image digests and preserves
the reviewed Sandbox connection policy, public identity configuration, and
schema-status-only `sandbox-promote` operation. It does not migrate SQL or alter
the paused acquisition worker.

Release checks and owner review precede dispatch. After promotion, verify the
canonical web URL, inspect the historical attempt without losing the book, and
only then begin a fresh Sandbox Hosted Link session if the old credential is
confirmed unavailable and explicitly abandoned. Binding review, encrypted
credential persistence and authorized worker read remain hosted acceptance
evidence, not claims of this image build.

Local release validation passed: exact manifest hash
`05803d7210c8fa40cade63cd196dad65f407b922ff31fbbc36683e23cd99512e`,
GitHub build metadata and all three retained candidate receipts match, and
`make release-test` passed. The source PR's five CI checks passed; this release
PR passed its five checks before merge.

PR #177 merged as `700fdbca45e1e51559612cf7a9f0f45d6ba47ced`.
[Deployment 39072895377](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/39072895377)
completed with `transport-smoke-passed` against the exact approved manifest.
The receipt names ready API revision `jayburd-dev-api-00008-f8b` and web revision
`jayburd-dev-web-00008-k5t`. Public HTTP readback returned 200 for the configured
URL and 307 from Cloud Run's alternate service URL to that configured URL.
Scheduler readback remains PAUSED. Issue comments
[#160](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/160#issuecomment-6241885269)
and [#164](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/164#issuecomment-6241889935)
retain the hosted acceptance limits. No fresh Hosted Link completion, ciphertext
persistence or authorized worker read has been observed from this release.
