<!-- stage-runbook: true -->
# Plaid decoder delivery

## Status and boundary

Issue #102, lease PLAID-DECODE-102. Synthetic-only preparatory page decoder in
`internal/ingestion/plaid`; no network, SDK, secret handling, SQL or runtime sync.
Coordinator approved interface/unknown-field policy before implementation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Sources | Refreshed official Plaid Transactions sync and API versioning selected sections; precise reviewed scope and interpretations in decoder contract. |
| Contract | Explicit same-account expectation, installed ISO references, exact lexemes, bounded schema policy, zero Page on all failures. SQL author received exact tagged Go types. |
| Implementation | Deep duplicate-key/Unicode validation, normalized signed amounts with decimal-position arithmetic, preserved dates/lineage/raw bytes, pending/zero readiness separation. |
| Focused tests | PASS race suite and deterministic fuzz seeds; independent math/big.Rat oracle plus fixed expected quantities. |
| Fuzz/default gates | PASS make check test build, focused race tests and both 60-second fuzz targets (parallel=2). |

## Current evidence

[Decoder contract](../contracts/plaid-decoder.md) owns API, limits, ignored metadata,
standards applicability and source review. `TestPlaidExactMoney` covers fixed USD,
JPY, KWD, exponent, zero, large values and insignificant-zero representations.
Other tests cover account/schema mismatch, source identity and pending lineage,
source date/timestamp/raw preservation, deep JSON/Unicode restrictions, 500-row and
2 MiB boundaries, installed reference validity and empty initialization.
Fuzz targets independently compare exact rational quantities and minor-unit
integrality, ensure rejected pages expose no partial data, and check preserved raw.
These properties do not prove acceptance of every valid Plaid schema variant.

Final logs: `/private/tmp/jayburd-plaid-decoder-default.log` and
`/private/tmp/jayburd-plaid-decoder-tests.log`. Money fuzz:
`/private/tmp/jayburd-plaid-money-fuzz.log`, 4,353,639 executions, 296 new interesting
inputs (307 total), 60.482 seconds package duration. Page fuzz:
`/private/tmp/jayburd-plaid-page-fuzz.log`, 2,483,899 executions, 117 new interesting
inputs (121 total), 61.814 seconds. Fuzz targets used a fresh task-specific Go cache;
counts include harness skips and do not measure provider coverage or throughput.
No failing fuzz corpus was produced. Coordinator peer-reviewed numeric mapping,
strict JSON, atomic error return and account rejection with no blocking finding.
Self-review added explicit offset-hour/minute bounds because Go's timestamp parser
accepts some offsets outside the selected strict profile; fixed tests cover both.

## Remaining work

Complete-set cross-page validation, durable fencing/staging/cursors, raw retention,
source correction behavior, HTTP acquisition and actual provider qualification
remain independent work. Source-ready is not authorized, draftable or postable.
The fixed allowlist deliberately quarantines unsupported material fields including
running_balance. No browser/API/SQL files changed in this delivery.

## PR handoff

Issue #102. Integrate decoder independently; caller must supply trusted account
binding and installed currency metadata. Root owns shared index/bibliography links,
combined review and PR. No migration or dependency changes, no push by this agent.
