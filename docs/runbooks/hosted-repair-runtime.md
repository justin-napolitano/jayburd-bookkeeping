<!-- stage-runbook: true -->
# Explicit Hosted repair runtime — #328

## Status and boundary

`RUNTIME328-HOSTED-REPAIR`, dedicated `feat/328-hosted-repair-runtime`,
base3763b579 (329 merged after all five checks;32/33/324/325 accepted).
Source-only configuration/release delivery. No live manifest, workflow dispatch,
schema apply, cloud/provider/secret access, consent renewal or personal account.
This initial documentation-only draft is not ready for merge or activation.

## Stage ledger

| Stage | Evidence / limit |
| --- | --- |
| Resume | Current65/328 and merged329/326 checked; no competing writer; exact source path lease recorded |
| Review | Index/bibliography/LR07, standards/decision/database, runtime/release/status/webhook contracts and actual deployment/status source reviewed |
| Design | Separate default-false capability; unchanged legacy digest/behavior; exact source/receiver/schema gates; status-only repair rollback exception |
| Draft identity | Pending actual PR number before binding its runtime source ancestry gate; no guessed source gate |
| Implementation | Pending runtime/CLI, manifest/infra, status receipt/orchestration and independent negative tests |
| Validation | Pending required core/race/build, release/infra/runtime and synthetic recovery gates |

## Current evidence

Current service takes `HTTPConfig.HostedRepair` default false but managed
composition does not pass it. Existing webhook-only approvals stay disabled.
The current deploy workflow checks required PR ancestry and all three same-build
receipts before cloud authentication; sandbox-promote then runs exact-image
status/hash verification before API Terraform. Existing source readiness is not
actual signed delivery or live SQL authority.328 will reuse these mechanisms,
not add direct API schema grants or assume compatibility from a green build.

New profile must explicitly require302/303 dedicated readiness and owner review
of exact API/receiver image, schema33 and restricted bootstrap. Automated issue/
ancestry gates and structural evidence references do not verify prose, a current
receiver revision, ordinary update completion, identity/MFA or transaction equality.
Keep actual275/320 and Production/personal pilot gates open.

Existing legacy rollback never touches job/database. New repair-enabled Sandbox
rollback needs a narrowly explicit read-only status/hash execution first; selected
old/mismatched/dirty image/schema fails before API rollback, never auto-up/down.
This extends validation, not migration authority or data changes. The same config
digest and prior Sandbox receipt stay required; no expired consent is renewed.

Named planned evidence: `TestHostedRepairRuntimeOptIn` plus malformed/disabled/
foreign/expired/only-webhook no-I/O negatives; exact CLI/TF propagation; legacy
digest stability, wrong types/approval/hash/gates, same-profile rollback; actual
mocked orchestration proves status before runtime and stop-on-failure. Existing
global expired optional enrollment startup/SQL recovery must continue to pass.
Expected outcomes are independently fixed, not generated from the implementation.

## Remaining work

Publish a documentation-only draft to obtain its actual PR identity, implement
and bind that exact source gate, complete required tests and mark ready only then.
No source activation from draft/merge alone. Human security review and fresh
owner-approved exact finite release remain necessary.328 blocks parent319 until
its source boundary is accepted;320/303 actual flow and275 qualification remain.
Production308–316 and owner-private statement reconciliation under90 are separate.

## PR handoff

Draft only; refs328/319/275. No migration or public API/OpenAPI/UI change; schema33
and owned service are prerequisites. Tracked Go1.26.10/dependencies unchanged.
Every implementation commit retains this marked stage runbook and staged check.
Final PR must record exact tests, source ancestry, rollback/migration impact and
remaining actual-world limitations before ready review. No hosted gate is cleared.
