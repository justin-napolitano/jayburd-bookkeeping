<!-- stage-runbook: true -->
# Limited Sandbox release delivery

## Status and boundary

Issue #132; branch issue/132-sandbox-release. Implement the reviewed-for-PR
[Sandbox contract](../contracts/sandbox-release.md). No deployment is performed by
this branch; owner review/merge remains required. Full #11/#26 acceptance stays open.

## Stage ledger

- Audited current issues and hosted receipts before coding; updated #11/#26/#27/#88/#104.
- Created #130 database probe, #131 dynamic delivery, #132 release policy and #133 hosted activation.
- Split implementation from hosted qualification to avoid requiring an undeployed UI to prove its own release readiness.
- Selected separate Sandbox operations with fixed project, status-before-runtime,
  scoped rollback, unchanged full gates and paused workers.

## Current evidence

PASS: six new Sandbox policy tests plus existing release tests and both fake-command
orchestration suites; Sandbox wrong-action blocks runtime, rollback skips SQL,
worker rollouts stay paused. `make check test build release-test`, document checks
and pinned actionlint1.7.7 pass. Independent agent review found no blocking issue;
its status-only wording suggestion was applied. Existing hosted schema status is clean15;
actual API IAM denial and dynamic delivery prerequisites are not yet complete.
No secret values, protected payloads or provider/state logs are inspected.

## Remaining work

Run focused policy/orchestration checks and required Go/build checks. Review and
merge #130/#131 implementations, build exact images, collect hosted probe/replay
receipts, then prepare separate application/worker manifests. Owner configures the
Infisical API writer and performs password/MFA. #133 records integrated acceptance.

## PR handoff

No dispatchable application manifest, SQL mutation, scheduling activation or real
bank connection. Full release closure gates remain intact; Sandbox is a qualification
scope with outstanding restore/load/rotation work documented in its parent issues.
