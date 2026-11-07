<!-- stage-runbook: true -->
# Private collection label API stages

## Status and boundary

Refs304/305; branch feat/304-label-api from460bad4. This lane owns only the
new collectionlabels Go transport/service, focused tests and its contract appendix.
The coordinator owns shared server composition, OpenAPI, generated types, browser
bridge and CI. The registry lane owns SQL and real restricted-role integration.
No provider/cloud operations, private records, credentials or inherited DSNs.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | AGENTS, index, issue304, source registry and private-categorization contracts reviewed; owner authorized implementation after447 merge |
| Contract review | Coordinator froze8routes/6functions, per-operation caps, profiled command wrapper, serializable persisted preview and unchanged raw selectors |
| Source review | Existing collectionread adapter inspected; PostgreSQL17 isolation13.2.2–13.2.3 and RFC9457 security section5 reviewed2026-11-06 |
| Implementation | New isolated collectionlabels service/handler; no server, SQL, OpenAPI, browser, dependency or CI edits |
| Focused proof | go test -race -cover ./internal/collectionlabels -count=1 PASS94.7%; all fixed signatures/isolation, HTTP caps/auth framing, safe errors, known abort recovery and unknown commit suppression |
| Core proof | make check/build PASS with local adapter; full make test attempted,18 release tests PASS and new package race tests PASS; existing acquisition/webhookhost/workerhost/workerruntime listeners fail EPERM |
| Restricted SQL dependency | Imported SQL lane491cea6 locally (cherry-pick6aa176b) solely for actual43 and integration fixture helpers; source remains SQL-lane owned |
| Real HTTP lifecycle | TestCollectionLabelsHTTPPostgres PASS0.27s: handler→Service→actual synthetic_runtime/SQL43, complete catalog, hugeUSD9007199254740993.01, nullableEURnumericcode, two explicit holds, principal denial, duplicate/unknown keys, path/body caps, taxonomy preview/prepare/execute/ack, immutable replay, exact4-row decisions, stale source/taxonomy409, stale prepared execute atomic refusal followed by cancel/ack, current-grant replay denial and zero books/journals/outbox |
| Result fault proof | TestCollectionLabelsHTTPResultCapPostgres PASS0.04s: oversized SQL read result in a separately cloned disposable fixture produces bounded503 without private padding; fault function replacement exists only in that owned clone and does not establish production SQL semantics |
| Disposal | Each new HTTP test clones labels_template into its own labels_http_timestamp database, closes its own pool/connections and drops only that clone; shared template/container/network/volume untouched |
| Shared gate review | Independently reviewed both-engine principal-first admission, exact reservation ownership/release, backfill/down/grants and browser fixture framing. Identified pre43 SERIALIZABLE acknowledgement snapshot missing the new backfilled reservation; SQL lane fixed release to require exact DELETE RETURNING or40001 |
| Upgrade acknowledgement proof | TestCollectionLabelsUpgradeAcknowledgementRace PASS0.16s with current SQL437dfb3: own empty43→42 clone, old cancelled/unacknowledged command, primed runtime SERIALIZABLE snapshot, uncommitted real43 backfill, observed blocked ACK, DDL commit, stale40001, then fresh acknowledgement with exact claim removal/one audit and retained cancelled history |
| Independent negative control | An external temporary Go overlay removed only the new missing-claim40001 guard from the migration string installed in the test's own clone. The identical race test FAILED0.09s with wanted40001/gotnil. No tracked SQL, shared migration artifact, template or product source changed; demonstrates the guard distinguishes the silent-ack defect |

## Current evidence

Focused synthetic adapter/HTTP proofs passed using installed Go1.26.5 with
GOTOOLCHAIN=local, external /private/tmp/jayburd-436-local.mod and build cache.
The tracked Go1.26.10 pin remains unchanged and canonical CI is required. Full
make test is not green locally: existing listener tests cannot bind loopback in
the sandbox; no test skips or listener bypasses were added. Fake transaction
tests alone cannot establish SQL authority or durable label semantics. The added
HTTP integration tests ran against the SQL lane's actual491cea6 schema43 in its
owned internal-network PostgreSQL17.11 container on Linuxarm64, using an
integration-tagged test-only authenticator and a real synthetic_runtime pool.
The binaries were cross-compiled locally with the adapter above; no listener,
provider/cloud connection, private record or inherited DSN was used. The shared
runner's later pending-capability/shared-reservation follow-ups were not present
in the original two HTTP lifecycle tests. The subsequent upgrade-race test uses
the SQL lane's exact437dfb3 up/down files from fixed runner /proof/migrations,
including the shared reservation and missing-claim retry. Coordinator's latest
combined lifecycle rerun remains required. Browser/API and full canonical CI
evidence remain separate. Existing unit checks were not repeated solely for
this integration-only regression; Linux integration binary compilation and the
actual positive/negative PostgreSQL schedules supply its focused evidence.

## Remaining work

Coordinator source review of the new HTTP integration tests, latest combined SQL
rerun, managed server composition/OpenAPI/browser bridge and app/web/canonical
checks remain. The HTTP recorder proof is not a browser or verified-provider
identity acceptance claim.
No standalone transport result completes304 or305 or enables hosted operation.

## PR handoff

Ready for coordinator review and integration. No SQL migration, application
activation, deployment or issue completion is claimed. Coordinator records
cross-lane integration and GitHub scheduling. The transport appendix records
frozen calls, standards applicability, selected source depth, alternatives,
threat/control boundaries and named tests.
