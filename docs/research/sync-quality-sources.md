# Sync quality source review

## Decision and scope

Review date: 2026-10-13. Coordinator review supports the
[quality plan](../plans/sync-quality.md); no accounting-policy, cloud-capacity or
live-data approval follows from vendor documentation. Sources below were read
at the named sections of mutable official documentation; recheck during adapter
implementation. Existing [accounting sources](database-sources.md) and
[test corpus sources](testing-sources.md) remain authoritative for those topics.

## Reviewed sources

| Resource and reviewed sections | Established behavior | Project consequence |
| --- | --- | --- |
| [Plaid transaction states](https://plaid.com/docs/transactions/transactions-data/), Pending and posted transactions; Testing pending and posted transactions | A pending replacement has a new posted identity; removal/addition can span pages. Details can change and matching links can be absent. Dynamic Sandbox users support lifecycle testing. | #85 must test the complete update; #88 exercises the actual adapter. Local posted journals stay immutable despite source revisions. |
| [Plaid transaction errors](https://plaid.com/docs/errors/transactions/), TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION | Restart the pagination loop at the original cursor when its underlying data changes. | Inject failure after intermediate pages; no partial cursor publication. |
| [Plaid Sandbox](https://plaid.com/docs/sandbox/), Webhooks; Differences between Sandbox and Production | Test webhooks can be triggered, but institution history and pending behavior are not fully representative. | Sandbox acceptance precedes a separately authorized statement-reconciled pilot; it cannot certify production behavior. |
| [Cloud Run concurrency](https://docs.cloud.google.com/run/docs/about-concurrency), Maximum concurrent requests per instance; Cost considerations | Concurrency limits and instance allocation affect resource consumption, scaling and cost. Application parallelism matters. | Measure concurrency, throughput, latency and resources together; replica count alone proves no capacity. |
| [Cloud Tasks pitfalls](https://docs.cloud.google.com/tasks/docs/common-pitfalls), Execution order; Duplicate execution; Resource limitations | Delivery can duplicate or reorder; target capacity, throttling and concurrency affect backlog and delay. | Test duplicate effects, fairness, backpressure and recovery with exact final invariants. |

## Interpretation, alternatives and uncertainty

The 70% efficiency and latency targets in #89 are proposed engineering thresholds,
not vendor guarantees. Real results depend on contention, database capacity,
provider limits and workload distribution. Independent books and one hot book
need separate measurements. Finite local workloads establish no hosted cost SLA.

Alternative: only increase Cloud Run instances and watch request success. That
is cheaper to instrument but misses duplicate financial effects, stalled queues
and database saturation. Keep the existing transaction/outbox architecture and
measure it before considering partitioning or another database. #64 owns the
operational target review; #89 owns the executable evidence.

No financial wire format changes in this planning slice. Existing
[standards applicability](../contracts/standards.md) and
[provider mappings](../contracts/provider-ingestion.md) remain required. Plaid
JSON is proprietary; ISO identifiers or schema checks alone do not establish
bank reconciliation or ISO conformance.
