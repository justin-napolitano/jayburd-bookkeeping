# Provider ingestion contract evidence

<!-- stage-runbook: true -->

## Status and boundary

Research/design only for [#70](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/70).
No runtime, SQL, provider requests, credentials or live accounts. Lease
P70-20261013T143600Z; coordinator owns shared bibliography/integration and GitHub.

## Stage ledger

| Stage | Evidence | Result |
| --- | --- | --- |
| Official source review | [PI01–PI08](../research/plaid-ingestion-sources.md) | Relevant public sections reviewed; no provider execution |
| Contract and worker alignment | [Provider contract](../contracts/provider-ingestion.md), coordinated #63 generation/fence interface | Proposed; source revision separate from ledger revision; filtered account streams; Item scope cannot span books |
| Repository checks | Commands below | PASS: links, vet, race tests, migration-file checks and builds |

## Current evidence

PASS: `PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache make check test build`.
PASS: `bash scripts/check-stage-runbook.sh --staged` (only the three owned documentation files).
Named provider tests and original money/lineage examples are future acceptance,
not evidence of a running ingestion adapter.

## Remaining work

Accept/revise profile, complete-update bounds, Item/account policy, consent and
retention. Review source/worker/secret lifecycle SQL/API contracts before runtime.
#17 ISO artifact/profile review, #8 ingestion and #13 provider/cutover remain open.
SDK money decoding and uncertain credential-operation recovery need explicit proof.

## PR handoff

Three owned documents only. Coordinator links annotated sources and reconciles #63
before a combined PR. No independent push/PR/GitHub mutations. Next slice is reviewed
core interfaces and synthetic proofs, not authorized provider onboarding.
