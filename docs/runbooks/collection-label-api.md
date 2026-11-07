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

## Current evidence

Focused synthetic adapter/HTTP proofs passed using installed Go1.26.5 with
GOTOOLCHAIN=local, external /private/tmp/jayburd-436-local.mod and build cache.
The tracked Go1.26.10 pin remains unchanged and canonical CI is required. Full
make test is not green locally: existing listener tests cannot bind loopback in
the sandbox; no test skips or listener bypasses were added. Fake transaction
tests cannot establish SQL authority or durable label semantics. This branch has
only migration42; real label SQL/app integration and shared browser/API validation
must run on the coordinator's combined tree with the registry lane implementation.

## Remaining work

Coordinator source review, real restricted-role label SQL proof, managed server
composition/OpenAPI/browser bridge and combined app/web/canonical checks remain.
No standalone transport result completes304 or305 or enables hosted operation.

## PR handoff

Ready for coordinator review and integration. No SQL migration, application
activation, deployment or issue completion is claimed. Coordinator records
cross-lane integration and GitHub scheduling. The transport appendix records
frozen calls, standards applicability, selected source depth, alternatives,
threat/control boundaries and named tests.
