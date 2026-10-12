# Local generated-data review

<!-- stage-runbook: true -->

## Status and boundary

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

| Owner | Work | Evidence |
| --- | --- | --- |
| Coordinator | Loopback demo lifecycle and browser review | Real demo cent posting, reload, guided flow and SQL count PASS |
| Data agent | Generated fixtures, exact cent audit and SQL seeding | Four reviewed generator journals + three original cents journals; stored balances verified |
| UI agent | Decimal-string currency formatting | Currency display and six mocked browser journeys PASS |
| Flow agent | Watched prepare/execute/acknowledge, rejection and replay | Guided real browser proof PASS |
| Volume agent | Bounded sequential 1,000/5,000/10,000 runs | Real 1,000 run PASS in approximately 3.2 minutes |

## Current evidence

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

## Review commands

Requires Docker, Go1.26.9, Node24.18.0, locked `npm --prefix web ci` dependencies,
and free loopback ports39081–39083. Browser test additionally needs Chromium.

```sh
make demo
# Open http://localhost:39083; Ctrl+C stops and cleans the owned database.
# Run make demo again for a clean reset.
make demo-test
```

The review page uses the production bookkeeping components against real Go/SQL;
its synthetic identity and Vite entrypoint live outside Next application routes.
No environment files are loaded. Production mode still requires managed sign-in.
Currency display uses exact strings and versioned minor units; no float
conversion or rounding hides nonzero extra digits.

**Watch a few transactions** offers manual stepping or a 19-step continuous suite:
setup, deposits 0.10/0.20, fee 0.01, same-command replay, rejected imbalance and
trial balance. The timeline records observed API boundaries, not invented SQL
tracing. **Run thousands** creates a separate Volume Lab and posts 1,000/5,000/
10,000 real deposits sequentially. Each request is prepared, executed, receipt-
checked and acknowledged; recent receipts are bounded to 20 rows. Pause finishes
the current transaction. Retry retains the same request/key/command identity.
No execution occurs merely by opening the page. Start explicitly authorizes this
fixed synthetic sequence and its acknowledgements.

Each volume receipt must have a unique journal ID, expected revision and exact
cents. A separate closed-form oracle checks the final API trial balance;
the 1,000 case expects bank 495.55 and capital credit 495.55. The test runner also
queries PostgreSQL for exactly 1,000 journal rows in the created Volume Lab.
Count and elapsed time describe actual confirmed responses. They include this
local HTTP bridge and sequential browser overhead; no production capacity claim.

## Remaining work

Local checks complete; PR review and hosted CI remain pending.
This is local workflow evidence, not production capacity. Larger browser runs
and multi-user concurrency/load benchmarks remain separate evidence.

## PR handoff

No migrations, production authentication bypass or live provider connections.
Reset stops and recreates only the demo's owned disposable database.
