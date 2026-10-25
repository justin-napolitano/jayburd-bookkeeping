# Production pilot profile review
<!-- stage-runbook: true -->

## Status and boundary

#308/#90; PROFILE308-20261024T0550Z, design/308-production-pilot initially74dd4b1,
now rebased onto main1bacecda under PROFILE308-REBASE.
Source/contract planning only; no Production provider, SQL, cloud, secret or real
data action. Proposed architecture/profile requires explicit acceptance before
implementation/activation; no live gate cleared by this document.

Follow-up PROFILE308-READONLY lease2026-10-25T00:36Z–02:36Z reserves only this
runbook and the proposed contract; released into317 review after2c0122f.
Subsequent PROFILE308-REBASE through02:43Z reserves the four original proposal
paths to resolve actual DIRTY/CONFLICTING state. No implementation or approval change.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Current issue90/308, current main and released predecessor leases, database foundations/standards/provider/identity/release contracts reviewed |
| Source review | PP01–05 selected official sections recorded with depth/limits; deprecated checklist distinguished from personalized Launch Center |
| Code inventory | Sandbox gates confirmed in API/worker transports, SQL namespace/Item/credential functions, environment-bound AAD and signed webhooks |
| Proposal | Isolated Production deployment recommended for review; exact authority/threat/acceptance matrix and unresolved choices documented |
| Scheduling | Created #311–#316 under #90 with native hard dependencies; #308 remains the design gate; no SQL migration or runtime writer reserved |
| GitHub readback | All six issues' native pilot parent/blocker and prerequisite relationships verified independently after publication |
| Validation | make check/test/build PASS using cached local Go1.26.5 temporary modfile; tracked1.26.10 unchanged. Documentation/links/structure/whitespace PASS; canonical CI remains separate |
| Read-only clarification | Owner's pilot objective and prior308/311 acceptance comments require manual as well as automatic posting denial. Proposed contract now names API/restricted SQL/runtime/worker and saved-command paths, current permission changes, unchanged history/journal.posted evidence and permitted acquisition/review metadata; no existing guarantee or test execution claimed |
| Follow-up validation | Required local make check/test/build PASS for documentation branch; final contract/whitespace and staged runbook check before commit. No proposed Production-profile test was executed or implied by these source gates |
| Rebase | Actual317 conflict resolved onto main1bacecda; preserved both Production PP01–05 and current LR01–07 recovery source records. Compressed redundant PI13 annotation wording without deleting sources; bibliography32747bytes, within32KiB. Rebased make check/test/build PASS; no implementation/architecture/authority change |

## Current evidence

[Profile proposal](../contracts/production-pilot.md),
[sources and code inventory](../research/production-pilot-sources.md).
Items cannot be transferred between provider environments. Current source does
not support Production; Dashboard approval metadata and Sandbox success cannot
change that. No paid plan/fee/retention/legal attestation inferred from public docs.
No-automatic-journal tests alone cannot prove the requested read-only pilot.
TestProductionPilotPostingDenied is an explicit implementation/qualification
obligation in311/315/316; no DDL or additional privilege is authorized by this plan.

## Remaining work

Staged runbook and PR review; owner/security review
of concrete isolation/profile proposal; exact SQL/transport contracts before code.
Continue independent Sandbox/recovery prerequisites while those decisions remain
pending. Actual personal-account authentication/statement review stays private.

## PR handoff

Refs #308/#90/#88/#274. Documentation/issue work only; no migration, runtime,
provider product, secret reference or deployed resource changed. All listed
acceptance names remain proposed until their execution evidence is recorded.
