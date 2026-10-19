<!-- stage-runbook: true -->
# Connection release delivery

## Hosted completion recovery migration 19 candidate (#187)

PR #188 merged after all five checks passed. Owner-dispatched image build
[39199242284](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/39199242284)
passed the repository's validation and image jobs. Its matching API, migrator,
and web candidate receipts are pinned in `deploy/dev-release.json`. This
migration-only operation applies forward migration 19, adding an immutable
encrypted Hosted Link token handoff and bounded completion window. It does not
replace the running API or web services or activate the acquisition worker.
The manifest's SHA-256 is
`8189e57e0f7dfd3dd9d7df93afa45697ffc08600738c287e513f55fcf00b0a2c`.
GitHub run metadata and all three candidate receipts passed the exact release
validator. `make release-test` passed its 32 release checks, deployment-script
proof, and 11 worker-release checks.

Verify the migration receipt and schema status before preparing a separate
`sandbox-promote` release with the same build digests and existing connection
metadata. Only after that runtime release should the owner start a fresh Plaid
Sandbox enrollment. Historical attempts created before migration 19 lack a
recoverable token, and a successful release alone does not establish a
completed Plaid exchange or connected Item.

## Hosted completion recovery runtime candidate (#187)

After migration 19's deployment receipt confirms the schema version, promote the
matching API and web images from build 39199242284 with `sandbox-promote`.
The manifest preserves the previously reviewed public identity, Infisical
metadata, static Sandbox client pins, and explicit `link_events_qualified=false`.
Its migration job performs status verification only; the acquisition Scheduler
stays paused. Check the ready API/web revisions and public transport receipt
before asking the owner to begin a fresh Hosted Sandbox attempt. A deployed
service is not evidence that Plaid completion or token exchange succeeded.
The runtime candidate manifest SHA-256 is
`878e4d86a6740faa20c4945cbf2e9f757c7040624d57fd934d13c45f7486ac6e`.
Exact build receipts passed the release validator, and `make release-test`
passed locally. Promotion remains contingent on the migration receipt.

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

## Failed-connection retry release candidate (#160)

PR #184 merged as `c0cb29efe0d71c26293a1b9b9c4ccdc7ce73cbbc` after all
five checks passed. Owner-dispatched [image build 39152439769](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/39152439769)
passed its application, browser, SQL and container checks and published matching
candidate API, migrator and web images. This release pins their exact digests.
It retains the reviewed Sandbox connection configuration, uses schema-status-only
`sandbox-promote`, and leaves the acquisition Scheduler paused.

The browser offers **Try connecting again** for a failed new connection. It
reconciles the exact old attempt and asks the server to close it only after a
confirmed missing-credential result. A stored credential, another unfinished
attempt, or uncertain response stops the restart. A fresh Link session begins
only after the server confirms the old attempt is safely closed. This release
does not claim a successful fresh enrollment, encrypted credential write,
binding review, transaction acquisition or posted journal. Those require the
owner's next hosted Sandbox test and separate review.

Release metadata validation passed against the successful build run and all
three retained candidate receipts. The exact manifest SHA-256 is
`767ce3e126bbaf0f35b86c05baab2da8533ad50c5de52a044905bea7c53eca98`.
`make release-test` passed (32 release/receipt tests, 11 worker release tests,
and deploy orchestration checks). Release PR CI and owner-dispatched deployment
must still pass before the URL is offered for a fresh test.
