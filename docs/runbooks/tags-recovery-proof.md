<!-- stage-runbook: true -->
# Strategic tags and payload-free recovery proof

## Status and boundary

Issues 450/451; branch test/450451-proof from merged 1a1491a. The coordinator
owns task state and wire freeze. This lane owns independent new SQL/HTTP proofs,
the synthetic label browser fixture/journey extensions and this runbook. The
SQL lane owns migration 44; API/server/bridge and production UI remain separate.
No provider/cloud operation, private financial record or inherited DSN is used.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Read live issues 450/451, AGENTS, standards and the existing migration 43 command/browser proof harness. Worktree was precreated by the coordinator at 1a1491a and was clean. |
| Contract review | Reviewed the coordinator's frozen v2 tag profiles and v1 global recovery profiles against the collection-labels contract. Independent expected cases distinguish tag sets from primary topics, current authority from stale pins, and confirmed acknowledgement from absence. |
| Source assembly | Imported root API 39d6c6b, runner bootstrap 99e7d04, composition/UI 72755b5 and final SQL 173e4a2. This lane authors proof source only; imported production commits remain separately owned. |
| Independent SQL/HTTP | Nine TestIndependent groups passed against final migration 44 in disposable clones of tagrec_template, using actual restricted synthetic_runtime login, Handler, Service and PostgreSQL. |
| Recovery fence negative control | In an owned disposable clone only, replaced the status function with the same body minus its collection_recovery_lock call. The identical primed private-admission schedule incorrectly returned none instead of SQLSTATE 40001. The expected test failure demonstrates that SSI alone did not protect this schedule. No overlay, weakened SQL or shared template change is tracked. |
| Browser fixture | Listener-free TestLabelBrowserBootstrapPostgres passed against PostgreSQL. The extended actual browser journey and post-run SQL assertions are prepared for the coordinator's combined canonical run. |
| Legacy compatibility | TestCollectionLabelsUpgradeAcknowledgementRace passed with the empty migration 44 downgrade preceding its original 43-to-42-to-43 race fixture. The ordinary migration 43 runner remains pinned separately. |
| Canonical browser correction | PR 452 head 497ac9e, application job 128299317574: the real topic/tag response-loss journey passed in 3.0 seconds. Both inaccessible-clear journeys stopped before application execution because Playwright cannot mount a test-local component. Moved RecoveryHarness unchanged into an imported test story; all clear assertions and delivery fault injection remain intact. This correction requires the next canonical run. |

## Current evidence

The nine independent groups cover tag add/remove/replace, distinct topic and tag
decisions, exact result sets, foreign and retired IDs, duplicate and size bounds,
stale tag/taxonomy/source pins, atomic rollback after a fixture-only failure,
all v2 HTTP operations and unchanged historical v1 preview recovery. They compare
source, original currency evidence, primary topic and accounting-state hashes
across tag-only changes. Financial values remain decimal strings, including USD
9007199254740993.01 and unchanged null/held projections.

Recovery cases verify exact minimal response keys; foreign and disabled actor
denial; currently authorized clear denial; prepared cancellation plus explicit
acknowledgement; committed receipt preservation; actual committed-but-lost clear
response; old clear replay preserving a newer book claim; ordinary acknowledgement
resolving an earlier failed clear while preserving a newer private claim; and
primed SERIALIZABLE status races for private and book admissions. Absence is never
accepted as proof that an unknown clear succeeded.

The actual HTTP 100-of-3,000 case passed with query 92 ms, preview 109 ms,
prepare 18 ms, execute 27 ms and acknowledgement 16 ms, each inside the real
4-second Service context. The complete nine-group final-source run passed;
the largest fixture took 1.75 seconds. These are local synthetic measurements,
not production capacity claims.

The browser fixture bootstrap passed in 0.10 seconds and independently asserted
four source observations, zero books/journals, and prepared versus committed
inaccessible commands. The browser journey adds tag catalog creation, committed
tag response loss and recovery, opaque inaccessible clear response loss, explicit
same-identity retry after a none status, and committed clear. Post-run SQL checks
assert exact command states, closure receipts, unchanged source revision count,
retained committed tag catalog and no reservation, book or journal effects.

Local make check/build and integration-tagged Go vet passed after final source
assembly. The local adapter uses installed Go 1.26.5 and a temporary modfile;
tracked Go 1.26.10 remains unchanged and must pass canonical CI. Browser source
was formatted with the locked Prettier version; shell syntax checks passed.
Local make test ran: the 18 Node tooling tests and all other Go packages passed,
including collectionlabels and server race tests, except existing socket-binding
tests in acquisition, webhookhost, workerhost and workerruntime, which failed
with sandbox bind operation not permitted. This is not recorded as a green full
unit run.

## Remaining work

The coordinator must run the combined full browser/API/SQL journey and canonical
Go/frontend gates before merge. This worktree has no installed frontend
dependencies; the coordinator owns the assembled frontend gate. Local host
listeners have previously been denied by the sandbox, so the listener-free
fixture check is not represented as a full browser pass. No hosted rollout,
provider/rule-map integration or spending/classification deployment is claimed.

## PR handoff

Pending coordinator source review and combined gates. Each implementation commit
includes this marked runbook and passes the staged runbook check. No production
SQL, UI, API or CI source was authored in this lane. Owned fixture databases are
dropped by cleanup; shared Docker resources remain owned by the SQL coordinator
until its explicit release.
