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
PR must pass its own checks before merge. No application deployment has run.
