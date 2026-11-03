<!-- stage-runbook: true -->
# Spending metric contract and oracle —428

## Status and boundary

MET417-CONTRACT, child428 of417; issue/428-metrics-contract fromd2215a8/PR427.
One contract/oracle author under65, no DDL/API/UI implementation or migration
allocation. Parent427/d2215a8 has five successful exact-head CI checks;
human reviews and merges remain pending. No private/provider/cloud/consent/
deployment actions or accounting-policy approval.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |417/418/65/native children rechecked;428 created as contract child; dedicated worktree/JB-001 packet |
| Source review | Selected SM01–06 public/source sections reviewed; existing corpus/standards/decision boundaries retained |
| Contract | Phase A source activity versus Phase B reviewed spending, contribution drills, exact units/current facts, unknowns and source/annotation snapshots |
| Independent oracle | Four separate hand-authored JSON artifacts;30 logical observations/33 retained revisions; manually fixed expectations for original units, source corrections, contributions and merchant grouping |
| Validation | PASS JSON/reference/BigInt arithmetic audit and make check/test/build with local Go1.26.5 modfile adapter; no runtime metric tests claimed |
| Issue graph |429 implementation is a native child of417, blocked by428; native dependency verified.418 still requires an implemented reviewed metric wire |

## Current evidence

[Contract](../contracts/spending-metrics.md) and
[source register](../research/spending-metrics-sources.md) record facts versus
project proposals. Snapshot contribution equality catches220 payment/20 interest
and card repayment double-counting before implementation. No current metric wire
or frontend is delivered by this branch. The temporary independent arithmetic
audit checks2017.50 incoming/940.00 outgoing in the first USD version,
157.50 known spending/165.00 unallocated outflow, and the separate3000-row
target60.00 versus first50 rows0.99. It does not exercise SQL, authorization,
classification, ISO conformance or a metric endpoint. The existing core suite
passes; tracked Go versions remain unchanged.

## Remaining work

Human contract review and merge of428 precede
[429](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/429).
429 must refine the closed wire and claim its SQL allocation before runtime work.
Runtime evidence/authority checks and category/merchant-dependent features remain
417 and its implementation follow-up;418 depends on implemented wire.

## PR handoff

Refs428/417/412. Contract/fixtures only; no migration impact.40 reserved426 and
41+unclaimed. No closed predecessor, approved classification or live-use inference.
