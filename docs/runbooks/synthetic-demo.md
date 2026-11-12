# Local generated-data review

<!-- stage-runbook: true -->

## Current authority and supported review

Reviewed2026-11-11 under486. The accepted
[artifact policy](../contracts/artifact-backed-accounting.md) governs this harness:
new entries require retained source evidence and explicit `source_post` review,
save, confirmation and acknowledgement. The old manual Flow/Volume runners are
retired; the dated PR23 results below preserve historical proof only. They are
not current buttons, supported customer workflows or current load acceptance.
Historical seeded journals remain readable synthetic evidence, not a manual-entry
capability. No application authentication bypass or provider connection is enabled.

Current `web/demo/main.tsx` mounts `LoadDemo` and the production bookkeeping
components. Load demo books imports3,000 source observations across three fictional
LLCs; loading evidence never authorizes accounting. Review an imported transaction,
categorize it, save its source-linked draft/command and explicitly execute/ack.
The current four-journey demo proof independently requires3,000 observations,
one source-linked review posting and one fictional Harbor journal, with no
Volume Lab journals. This proves the named journey, not3,000 journal postings.

Latest [482 acceptance](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/482)
on tested headf84a359070a116f863b49de6cd66506228b2ebe5, Seed43082561882,
passed all five gates. Four demo journeys took10.4s. Separate restricted proofs
passed source-backed restore for prepared/committed work and1,000 sequential
0.10USD source postings:1,000 journals/links/receipts/outbox,2,000 lines and
100.00USD per side; total24.34s on that hosted runner. These are distinct workloads,
not a demo throughput, concurrent scaling, managed backup or runtime capacity
claim. See [implementation-stage evidence and limits](synthetic-readiness.md).
Synthetic identity/provider adapters do not qualify real sign-in, bank feeds or
real books. No deployment follows from these tests.

## Status and boundary

The following status describes the historical PR23 scope.

Implemented for #9, based on PR23. Owner requested a usable local synthetic demo
and cent checks. This is an isolated review harness, never a production auth mode.

## Integration into main — 2026-10-12

PR #23 merged to main at `42ee7a8`; #24 merged to the old recovery branch at
`5e03aed`, not main. This branch merges that ancestry without duplicating SQL.
Updated the current API bridge to carry the already-reviewed recovery routes and
respect registration versus transition idempotency. The test-only identity uses
an unsigned three-segment synthetic marker to exercise normal transport syntax;
production authentication remains unchanged. The harness selects explicit loopback
mode and waits for PostgreSQL TCP readiness. Deployment now checks predecessor
merge commits are ancestors of the release, not merely marked merged in GitHub.
PASS: Go check/race tests/build, 94 frontend units, six browser journeys,
production build and workflow lint. The existing demo occupies ports 39081/39083;
left it running. Real integration uses a disposable tracked-source copy with only
fixed port substitutions to 39181–39183. Recovery through the production bridge
and all three guided/cents/volume demos passed. The 1,000-posting browser test
completed in 3.1 minutes, verified USD 495.55, and the shell independently confirmed
exactly 1,000 PostgreSQL journals. No cloud or real financial data access.

## Stage ledger

Historical PR23 delivery stages:

| Owner | Work | Evidence |
| --- | --- | --- |
| Coordinator | Loopback demo lifecycle and browser review | Real demo cent posting, reload, guided flow and SQL count PASS |
| Data agent | Generated fixtures, exact cent audit and SQL seeding | Four reviewed generator journals + three original cents journals; stored balances verified |
| UI agent | Decimal-string currency formatting | Currency display and six mocked browser journeys PASS |
| Flow agent | Watched prepare/execute/acknowledge, rejection and replay | Guided real browser proof PASS |
| Volume agent | Bounded sequential 1,000/5,000/10,000 runs | Real 1,000 run PASS in approximately 3.2 minutes |

## Current evidence

The entries in this section preserve historical PR23 evidence; current482
acceptance and its separate limits are recorded above.

Reviewed [generator source audit](../research/testing-sources.md#t04--synthetic-accounting-data-generator):
15 upstream journals have one-cent residuals. Invalid journals must not enter
posted books or receive fabricated balancing lines. Existing accounting policy,
ISO4217 references and SQL authority remain unchanged. Demo data is disposable.

PASS: `TestDemoCentsOracle` rejects a one-cent imbalance, fractional cents and a
wrong expected balance. Startup checks original EUR and functional USD balances
against independent fixed oracles. The [source mapping and license](../../internal/server/testdata/demo/README.md)
record the four-journal subset and explicit fictional parity rate.

PASS: Go `make check test build`, generated-type drift and TypeScript checks.
PASS: `make web-test web-browser` (53 units including volume checks, six Chromium journeys).
PASS: standalone demo Chromium test: generator bank5432.29 and totals5749.26;
cents bank0.29 and totals0.30; posting0.01 then reload gives bank0.30/totals0.31.
PASS: `make app-integration demo-test web-build` for the seeded demo.
After adding watched flows, `make demo-test` passed all three real browser tests:
guided flow 21.9s, manual review 1.4s, bulk 1,000 approximately 3.2 minutes. The bulk
test verified bank debit 495.55/capital credit 495.55 and 1,000 unique response IDs;
the shell independently confirmed exactly 1,000 PostgreSQL journal rows.
The 5,000/10,000 options have fixed-oracle unit checks; those full browser runs
have not been executed. Final `make check test build web-check web-test web-build
web-browser` passed against the complete implementation (53 units, six mocked
journeys, production build and Go race/vet checks).

Startup initially rejected duplicate synthetic rate references; fixed with
date-specific references and reran successfully. Type checks caught a Vite option
and Playwright import mismatch; corrected and TypeScript passed.

## Current review commands

Requires Docker, tracked Go1.26.11, Node24.18.0, locked
`npm --prefix web ci` dependencies and free loopback ports39081–39083.
Browser tests additionally require the locked Playwright Chromium installation.

```sh
make demo
# Open http://localhost:39083; Ctrl+C stops and cleans the owned database.
# Run make demo again for a clean reset.
make demo-test
```

The synthetic Vite entrypoint and identity remain outside Next application routes.
No environment files are loaded. The harness uses real Go/restricted SQL and
production bookkeeping components; ordinary application sign-in is unchanged.
Original currency values remain exact strings with versioned minor units.
`LoadDemo` discovers progress on mount; mutations require an explicit click.
Pausing stops new requests after the current operation, and unknown replies retain
same-identity recovery rather than silently creating another effect. Actual posting
requires the separate reviewed source workflow; no manual/bulk-journal runner is
available. Reset affects only the harness's owned disposable resources.

The old “Watch a few transactions” and “Run thousands”/Volume Lab descriptions
belong to the PR23 history above. Their1,000/5,000/10,000 manual deposit options
are retired. Do not use their historical count/timing claims as current proof.

## Remaining work

482 supplies the named current synthetic acceptance above; the historical PR23
handoff below does not mean those checks remain pending. Broader ingestion,
concurrency/noisy-neighbor benchmarks, hosted provider qualification, real-book
reconciliation and document/matching/correction workflows remain separate work.
No fixed sequential fixture establishes production capacity or availability.

## PR handoff

Historical PR23 handoff, preserved for traceability:

No migrations, production authentication bypass or live provider connections.
Reset stops and recreates only the demo's owned disposable database.
