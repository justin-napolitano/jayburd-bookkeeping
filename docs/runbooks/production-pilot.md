# Production pilot profile review
<!-- stage-runbook: true -->

## Status and boundary

#308/#90; PROFILE308-20261024T0550Z, design/308-production-pilot from74dd4b1.
Source/contract planning only; no Production provider, SQL, cloud, secret or real
data action. Proposed architecture/profile requires explicit acceptance before
implementation/activation; no live gate cleared by this document.

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

## Current evidence

[Profile proposal](../contracts/production-pilot.md),
[sources and code inventory](../research/production-pilot-sources.md).
Items cannot be transferred between provider environments. Current source does
not support Production; Dashboard approval metadata and Sandbox success cannot
change that. No paid plan/fee/retention/legal attestation inferred from public docs.

## Remaining work

Staged runbook and PR review; owner/security review
of concrete isolation/profile proposal; exact SQL/transport contracts before code.
Continue independent Sandbox/recovery prerequisites while those decisions remain
pending. Actual personal-account authentication/statement review stays private.

## PR handoff

Refs #308/#90/#88/#274. Documentation/issue work only; no migration, runtime,
provider product, secret reference or deployed resource changed. All listed
acceptance names remain proposed until their execution evidence is recorded.
