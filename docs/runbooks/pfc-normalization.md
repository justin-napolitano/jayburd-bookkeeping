<!-- stage-runbook: true -->
# PFC attribution normalization —413

## Status and boundary

PFC413-GO, feat/413-pfc-normalization from2315cc4; one focused Go writer.
Explicit unintegrated current-response sidecar and versioned map helper only.
No migration, default wire/runner/HTTP option/API/UI/provider/private-history or
cloud/consent change. Registry42 belongs to239; no other migration reserved here.
Owner design approval is recorded in238/242 and is not requested again.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |AGENTS/live413/238/242/65, private/source contracts and security policy read; dedicated worktree |
| Compatibility review |Actual HTTP→decoder→pageWire→SQL25/29 path inspected. Root reviewed sidecar decision; default financial JSON must remain exact |
| Source review |Official API PFC/options/version/confidence and migration/backfill public sections reviewed; scoped record linked below |
| Implementation |Explicit atomic DecodePageWithCategories sidecar, page-hash/account/transaction/change binding, bounded metadata and unavailable reasons, immutable explicit versioned map; default decoder/types/runner unchanged |
| Verification |8 named synthetic PFC unit groups plus fuzz seeds PASS under race detector; final plaid package90.0percent statement coverage.10second sidecar fuzz PASS266923 executions. make check/build PASS using installed Go1.26.5/external modfile/cache; default make test attempted, existing socket-binding packages blocked by local sandbox. No skip/waiver or tracked toolchain change |

## Current evidence

[Contract](../contracts/pfc-normalization.md) and
[source record](../research/pfc-normalization-sources.md) distinguish provider facts
from project decisions. This is not normalized PFC database publication or an
authorized taxonomy API. No historical record or provider was read.

The exact default path remains DecodePage→Page/Transaction JSON→SQL25/29.
No new sidecar consumer, request option, schema field or observation-revision
policy is enabled. Parent coordinator retains shared index/bibliography and
runner/HTTP/OpenAPI changes; those are not edited in this branch.

## Remaining work

Publish exact-head draft PR and actual canonical CI/source review.
Integrated413 still needs additive SQL wire/profile and category-only metadata
revision semantics,239/304 current private authority/taxonomy, API/UI and real
synthetic no-journal/revocation/recovery proof. No activation by default.

## PR handoff

Refs413/238/242/304/412. No migration or current-runtime wire impact. Human source/
security review and canonical tracked-toolchain CI precede merge; later SQL/API
activation remains separately evidenced. Keep413 open for its actual full outcome.
