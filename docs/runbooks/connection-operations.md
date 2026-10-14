<!-- stage-runbook: true -->
# Connection operations delivery

## Status and boundary

Implemented synthetic #103/#104 slice on merged #107 (`fe9a43a`). Durable attempt
recovery, account sync controls and verified webhook/catch-up intent are integrated.
No live provider calls, credentials, cloud changes or live bookkeeping approval.

## Stage ledger

- Three parallel lanes: SQL c00325c/6d74694, webhook d75fbea, UI1490175/a0700da.
  Root owns service/HTTP, bridge, OpenAPI, role bootstrap, CI and combined proof.
- Contract review chose independent stream revision for pause/resume, authoritative
  unfinished counts, current-grant replay and atomic generation intent.
- Peer review corrected equivalent ECDSA signature replay identity. Expired known
  binding review no longer blocks fresh enrollment; unknown exchange still blocks.
- Integration corrected schema-version fixtures and ordered an old expiry fixture
  before its deliberately unresolved operations; no product assertion was weakened.

## Current evidence

All commands use pinned tools and synthetic disposable data. PASS:

| Gate | Evidence |
| --- | --- |
| Default | `make check test build`: vet, race, checksum/history checks and builds. |
| SQL | `make migration-test cloud-sql-test ledger-test observer-test source-registry-test sync-state-test connection-state-test connection-service-test acquisition-test provider-flow-test`. |
| New SQL/signatures | `make connection-ops-test webhook-test`: eight SQL groups; 15 webhook groups including three actual PostgreSQL groups. |
| Frontend | Generated API check, TypeScript, 244 unit tests, production Next build and 13 browser regressions. |
| Full application | `make app-integration`: real browser/Next/Go/PostgreSQL posting recovery. |
| Connections | `make connection-browser-test`: lost exchange reply→remount→discovery→binding; pause/resume; lost manual-request reply→same-key resolution; acquisition→explicit 0.10 posting→lost-post recovery. |
| Workflow | Pinned actionlint passed; migration1–13 remain unchanged. |

Connections proof also checks mobile overflow, empty browser storage, zero journals
before explicit posting and one exchange/fetch/journal across completed-claim replay.
That replay reuses its completed claim; it is not a new independent provider pull.
SQL separately proves concurrent completion/signal keeps later work pending.

Local logs: `/private/tmp/jayburd-ops-default.log`, `-sql-gates.log`,
`-regression.log`, `-final-sql.log`, `-webtest.log`, `-browser.log`,
`-browser-integration.log`, `-app-integration.log`, `-workflow.log`.
Old failed schema/fixture attempts remain in logs; successful reruns supersede them.
Legacy connection rerun: `/private/tmp/jayburd-connection-ops-legacy-proof.log`.
Alternate proof ports39881–39883 were restored to committed39081–39083; existing
owner previews were preserved. Sandbox listener/Docker restrictions required
escalated synthetic execution; they were not product failures.

## Remaining work

#7 must compose private acquisition dispatch, stable claim identity, scheduled
catch-up/backoff, wakeup consumption and bounded server lifecycle. Migration14's
outbox records intent only. Owner bootstrap must precreate NOLOGIN
bookkeeping_webhook before restricted hosted migration; no login was provisioned.

Scoped Infisical writer/version propagation, hosted Link and qualified reconnect,
existing-source overlap UI, #88 Sandbox and #90 live pilot remain. Status cannot
invent provider-update time; requests do not prove publication. #85/#87 fault and
security acceptance and #89 measured capacity remain open.

## PR handoff

Review against main. Forward migration14 only; no automatic rollback or deployment.
Shared contract changes and validation are complete; GitHub CI results are separate.
Leases OPS-SQL-103-104, WEBHOOK-GO-103, STATUS-UI-104 and root are released into
review at PR creation. No live activation follows from merge.
