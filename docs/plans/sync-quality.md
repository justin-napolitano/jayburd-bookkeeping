# Sync correctness and scaling gates

## Decision

Prove complete source-to-book correctness before live use. Existing ledger,
observer and staging tests are component evidence, not a qualified bank sync.
GitHub owns scheduling and acceptance state; [#65](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/65)
owns bounded agent leases. [Sources](../research/sync-quality-sources.md) explain
provider behavior and the limits of horizontal scaling.

## Work

| Issue | Required proof | Prerequisites |
| --- | --- | --- |
| [#84](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/84) | Independent golden statement through import, review, posting and reconciled reports | #8, #10 |
| [#85](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/85) | Provider pagination, revisions, atomic cursor publication, replay and crash recovery | #7, #8, #70 |
| [#86](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/86) | Exact money, independent property oracles and cross-LLC denial | Existing boundary tests now; new adapter properties after implementation |
| [#87](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/87) | Webhook verification and revoked/wrong-source isolation | #7, #70 |
| [#88](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/88) | Opt-in real Plaid Sandbox qualification | #8, #85, #87; owner-injected secrets |
| [#89](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/89) | Scaling, overload, noisy tenants and recovery | #64 design; #7/#8 runtime; #26/#27 hosted gates |
| [#90](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/90) | One authorized read-only account reconciled to a complete statement | Prior quality gates, #1/#10/#27 and explicit live scope |

Extend the existing golden corpus; do not regenerate expectations from production
code. Imports create no journals until explicit posting. A balanced duplicate
journal is still wrong. Reconciliation explains pending/timing differences and
compares compatible dates, currencies and balance meanings.

## Scaling acceptance proposal

Measure 1k/10k/100k events, 1/10/100 books and 1/2/4/8 workers. Hold fixture shape
and database resources fixed within each comparison. Record throughput of unique
committed effects, p50/p95/p99 lag, backlog age, retries, resource pressure and
cost per 1k events; separate warmup and repeated measurements.

Proposed review target: at least 70% efficiency `T(n)/(n*T(1))` for independent
books before measured saturation, with p95 at most twice baseline at matched
per-worker offered load and zero lost/duplicate effects. This is a project target,
not an observed result or unlimited scaling guarantee. Measure hot-book contention
separately. Preserve serializable correctness; do not weaken isolation to pass.
[#64](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/64) retains
budget, RTO/RPO, cleanup and hosted-run acceptance ownership.

## Done when

Each issue records named tests, immutable code SHA, inputs, independent expected
results and failures/limits. Default CI stays synthetic and credential-free.
Sandbox and live qualification are separate gates. The
[runbook](../runbooks/sync-quality.md) records what actually ran.
