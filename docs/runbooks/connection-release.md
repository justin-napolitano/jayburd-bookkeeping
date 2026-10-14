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
