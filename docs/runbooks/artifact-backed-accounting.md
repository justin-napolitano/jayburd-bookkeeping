<!-- stage-runbook: true -->
# Artifact-backed accounting delivery

## Status and boundary

Owner explicitly approved recording and implementing artifact-backed accounting,
with no manual entries. Build on frontend PR472/36eebfa and retain its review
evidence. Source-only first restriction: public posting must use existing bank/
statement source evidence. No cloud/provider/Production/private-record read,
new SQL migration or dependency. Existing1–47 immutable;48 unallocated.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Owner decision | Artifact→review→confirmed accounting→linked posting. Receipt+bank evidence must not double count. Documents/corrections need separate implemented evidence support. |
| Contract | docs/contracts/artifact-backed-accounting.md freezes current source-only public boundary, legacy cancellation/ack safety, standards applicability and residual privileged SQL scope before implementation. |
| Dispatch |473 governing policy,474 public API,475 artifact-first UI,476 future documents/matching/corrections,477 actual source proof. Root owns integration/OpenAPI/AGENTS/research; three disjoint source lanes. No migration or cloud writer. |
| API |Public Handler always artifact-only; direct journal/reversal and new post registration fixed422 after authentication. Missing ArtifactRecovery port fails503. Service checks current pending plus PostgreSQL-canonical target UUID inside the same SERIALIZABLE execute attempt. Prepared matching manual post cannot transition; committed command receipt replay and safe cancel/ack retained. Closed operation profile preserves raw known requests for SQL validation. |
| UI |Removed blank ManualJournal/LineDraft and New journal from default/standalone nav and view type; Ledger label becomes Posted entries. Prepared legacy manual request offers explanation/cancel only; historical receipt read/ack and explicit source_post confirmation remain. No fake Documents view. |
| Local source proof |Ledger package/race and new authenticated deny/no-fallback/raw request/same-transaction/stale authority/retry/unknown-commit/UUID tests PASS. make check/core build, generated API/TypeScript and Next production build PASS.662 frontend units/full95 component browsers PASS, including retained-source draft and old saved manual cancellation/history scenarios. Rebuilt isolated review PASS. |
| Existing CI defect |Root read public job128891593105 selected failure: old demo acquisition line27 and review line15 waited on removed Tools popup (120s/30s). Both now use actual navigation helper. Initial source-post connection/Production journeys had passed; no unverified claim that old demo was green. |
| Actual proof adaptation |Default real app now asserts zero journals/outbox and exactly one setup receipt/execute/ack. Nonzero0.10 four-report/CSV/comparison/posted-journal/account drill checks move onto the actual source_post lost-execute/remount/ack/replay journey; SQL additionally requires one journal/review_posting/outbox and two observations. Demo retains3000 observations/one review_posting/one Harbor journal and historical cent oracles; retired manual Volume Lab effects must be zero. Independent review found no lost/vacuous financial assertions. |
| Limits |Actual composed --connections attempted standard runner: upfront Synthetic proof port unavailable, before fixture/Docker startup. Canonical exact-head composed SQL/browser gates remain required. Full Go suite attempted: ledger/server and other offline packages pass, four existing socket suites fail EPERM; no listener/timeout/guard workaround. No managed/private/provider call or database change. |
| Independent review |Public handler/service current-authority/atomic policy and raw/retry/unknown outcome boundary reviewed independently: no findings. Actual source report/replay/CSV/drill and exact-count proof relocation reviewed independently: no findings. Low-level primitive HTTP/math proof remains explicitly isolated, not an alternative production Handler. |

## Current evidence

Existing banking.execute_source_post atomically commits journal+review_posting
link+source projection+receipt after current authority/revision checks. Source now
closes unsupported public manual paths while preserving that boundary and safe
legacy recovery. OpenAPI0.11 separates new ArtifactRegisterCommand from historical
RegisterCommand decoding. Runtime deployment and actual composed acceptance are
not inferred from unit/component success. Earlier472 demo failure was reproduced
in public log evidence and its exact obsolete navigation selectors corrected.

## Remaining work

Publish the combined source and require exact-head canonical checks, particularly
real source-post/accounting/report/demo proofs. Existing Sandbox release follows
only accepted trusted images and scoped runtime verification; no live use or
document completeness inferred.476 remains the retained document/matching/
correction capability, not a hidden manual fallback. Preserve owner review.

## PR handoff

Source implementation and local review ready. Existing deployment remains unchanged. Artifact evidence
policy is governing; document upload/matching/correction children remain explicit
until implemented, and no unsupported Documents screen is advertised.

## Tagged integration compilation repair

Canonical cd2bc01 validation exposed a test-only constant name collision between
the new artifact policy unit fixtures and the integration-tag ledger fixtures.
Renamed the new fixture constant to artifactPolicyCommandID; no runtime behavior,
assertion, migration or financial fixture changed. Explicit integration-tag
compilation is required alongside the ordinary ledger package checks.
