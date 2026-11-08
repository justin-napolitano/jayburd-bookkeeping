<!-- stage-runbook: true -->
# Authenticated collection onboarding SQL47

## Status and boundary

Issue460, dedicated feat/460-onboarding-sql. Source-only application flow for
already enrolled, active verified principals. New47 only; migrations1–46 remain
byte-for-byte unchanged. No managed activation, cloud/provider calls, private
records, credentials, real identity lookup, enrollment or operator SQL delivery.
The normal application uses trusted existing authentication and the actual API
login; no caller actor, inferred email/book ownership or API provision grant.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Review | AGENTS, index, database foundations/bibliography, identity/setup, private registry, recovery and frozen root collection-onboarding contract reviewed. Root approved six signatures and exact review profiles before DDL. |
| Storage | Immutable previews, recoverable commands and audit; third typed shared reservation; engine-specific immutable closure proof; persistent workspace/recovery fences. |
| Authority | Fixed own capabilities only on explicit execute; existing workspace never restored. Exact legacy candidate, full source set/generation/binding/book/workspace snapshot; original42 adoption uses actual API session. |
| Review fixes | Normalize existing source/book authorization SQLSTATEs. Lock current original Item, canonical Item and reviewed adopted aliases before replay disclosure. Historical same-key replay precedes expiry/pin checks, retaining current authority. |
| Validation | Twelve actual restricted PostgreSQL groups pass, including four blocked primed Item/alias/consent/generation schedules, both book/label shared-gate winner orders, late atomic aborts, current revoked recovery and immutable legacy anchors. |

## Current evidence

Executed TestOnboarding47SetupLifecycle, TestOnboarding47AdoptionAndFences,
TestOnboarding47RevokedRecovery (prepared/committed crossed with book read,
source read and import revocation), TestOnboarding47ClosedAuthorityAndNoRestore,
TestOnboarding47PrimedWorkspace, TestOnboarding47SharedGateRaces,
TestOnboarding47HistoricalPrepareReplay, TestOnboarding47CurrentItemReplayLocks,
TestOnboarding47Privileges, TestOnboarding47LabelGateRaces and
TestOnboarding47AtomicFailure and TestOnboarding47LostAcknowledgementRecovery. All pass on pinned PostgreSQL17.11 restricted
synthetic_runtime sessions. Observed pg_locks waiting precedes writer commit;
primed readers fail40001, fresh readers enforce current authority. Legacy
consent change is permitted to the fixture owner by existing SQL; the test proves
its real race rather than assuming identity immutability. Generation-only drift
stales execution without stranding cancellation/acknowledgement.

Fresh migrate1–47, latest empty47→46→47 and retained-review down refusal pass;
the retained preview count remains1. Original legacy source, Item/cursor and
posted journal anchors are hashed before
adoption and remain unchanged afterward. Explicit adoption pauses old acquisition
and retains the existing source path; it does not copy facts into native storage.
Late test-only terminal-write rejection rolls back setup grants/adoption and
leaves the prepared command and original history intact. Exact closure replay
cannot release newer label work or be mistaken for another engine's receipt.
All six new API signatures are checked across seven service roles; helper/base
aliases, tables and general provision remain inaccessible to runtime.

A fresh migrate1–47 and latest47→46→47 roundtrip pass. The command stores a
unique immutable recovery_id, echoed on every receipt and bound by composite
principal/recovery/command foreign keys to both reservation and closure proof.
Actual committed acknowledgement followed by book revocation and a new label
claim permits only the exact payload-free old closure replay, preserving new work.

make check/build pass using local Go1.26.5 and the existing external modfile/cache
adapter; tracked Go1.26.10 and dependency pins are unchanged. Synthetic Docker
uses only existing digest-pinned public PostgreSQL/migrate images, an internal
network, tmpfs data and disposable owned volume. No inherited DSN is accepted.
The runner compiles only the dedicated TestOnboarding47 prefix and also verifies
empty migration rollback/reapply and retained review refusal. make test was attempted; existing acquisition/webhookhost/workerhost/workerruntime
listener tests fail because the sandbox denies binding loopback sockets. No test
was skipped or gate weakened. No latency SLA is claimed by local timings.

Standards applicability: this is internal authorization/setup/recovery metadata,
not a financial exchange or new ISO20022 message. Existing currency, original
units and journal contracts remain intact. Existing database/identity/recovery
sources and the root onboarding contract supply the design basis; no new external
standard or accounting policy is introduced.

## Remaining work

Root integration owns actual HTTP/UI composition, all historical CI gates and
OpenAPI. These SQL tests alone do not claim a deployed or usable managed flow.
Source review and canonical combined CI must complete before merge or rollout.

## PR handoff

Deliver only47 pair/seals, dedicated integration test/runner and this runbook.
Root owns shared contract/server/OpenAPI/Makefile/CI and GitHub publication.
Keep jayburd-onboarding47-proof owned container/volume/internal network until root
finishes combined proofs, then remove only those resources on explicit release.
