# Owned Hosted repair delivery

<!-- stage-runbook: true -->

## Status and boundary

#275 contract phase only. [Proposed protocol](../contracts/hosted-reconnect.md)
defines modern signed session proof and same-Item recovery before implementation.
Reconnect remains rejected in the concrete HTTP provider. No migration number,
source consent, provider/cloud call, Production profile or hosted clearance.

Lease `REPAIR275-DESIGN-20261024T0620Z`; branch
`design/275-owned-hosted-reconnect`; base
`5f803c8325e6397cde80878e4daa7cb0e75d8345`. The implementation needs a new bounded
lease and sole SQL writer. GitHub #65 is live scheduling authority.

## Stage ledger

| Stage | Outcome |
| --- | --- |
| Resume | Read #275/dependencies, current index, recovery/source and webhook/bootstrap contracts; current main and lease checked |
| Source review | Refreshed official modern SESSION_FINISHED and Hosted/update sections; no provider payloads or credentials read |
| Design | Defined owned hash/proof/claim association, current scope/expiry, closed receiver ACL, ambiguity, replay and lost-response recovery |
| Validation | Core check/race test/build and local-link/migration-integrity checks PASS; no new behavior tested |
| Handoff | Pending reviewable PR and bounded implementation issues |

## Current evidence

LR07 records exact public sections and scope. Pinned generated SDK lacks ordinary
Item-update result evidence; modern signed Link receipt is the proposed owned
association. No actual Sandbox update shape or webhook delivery is demonstrated.
All new test names in the protocol describe future acceptance, not passing tests.

Local validation used cached Go1.26.5 via a temporary modfile; tracked toolchain
remains1.26.10. No dependency/package or financial-data access. `make check test
build` passed with that explicit local override. Exact-head CI remains separate.

## Remaining work

Review protocol and exact SQL constraints under a sole-writer implementation
lease; implement SQL/ACL, then provider/receiver/service, then UI and actual socket
integration. Human security review and #271/#303 dedicated signed delivery precede
finite hosted reset_login/recovery qualification. Production #308 is separate.
Do not close #275 or #88 rows from this design or a passing build.

## PR handoff

Refs #275; docs only, no migration or runtime behavior. Compare retained Hosted
with SDK transition and reject health/callback/new-exchange shortcuts. Remaining
risks are actual event availability, same-session ambiguity, SQL lock/proof bypass,
permission revocation, managed role compatibility and exact release qualification.
Staged runbook validation is required before commit; exact-head CI before merge.
