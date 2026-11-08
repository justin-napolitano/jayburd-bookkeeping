<!-- stage-runbook: true -->
# Private onboarding API

## Status and boundary

Issue460, branch feat/460-onboarding-api from b2e6621. This lane owns only the new
internal/collectiononboarding Go package and this marked runbook. Root owns the
contract, OpenAPI, managed/server/main wiring and combined publication. The SQL
lane owns forward47 and actual authority/storage proofs. Existing Firebase/TOTP
identity remains unchanged: enrolled active verified users only, no public signup.
No operator plan, caller actor/capabilities/evidence, cloud, provider, credential,
private financial record, state or plan access. Merged1–46 remain immutable.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Contract | Read root frozen collection-onboarding.md at1e3a4cf and coordinated directly with SQL lane on all six banking functions, direct Command profile, complete adopt review, pending states and exact cancelled result. Incorporated root's required persisted prepare_key echo addition. |
| Transport | All eight mounted routes authenticate before private path/body/framing; raw JSON remains unchanged for SQL duplicate/unknown/closed-key authority. Explicit GET/HEAD/body/query/content-type/encoding/key framing, size and UTF8 bounds. No-store, nosniff, fixed RFC9457 diagnostic, fresh UUID correlation and401 Bearer challenge. |
| Transactions | Every operation uses SERIALIZABLE with4s context and bounded SQL timeouts. Only known40001/40P01 whole aborts retry, at most3. Unknown commit503 has no automatic replay; results remain private until commit. Rollback has an independent1s cleanup context. |
| Response fence | Complete envelopes capped1MiB; recursive duplicate-key/nesting rejection, exact closed profile/fields, UUIDs and safe DTO bounds; current request command/preview/key/recovery identity checked. Prepared/terminal/acknowledged result coherence checked without reproducing SQL authority or eligibility. Capability set accepts canonical SQL order; no capability is supplied to SQL from HTTP. |
| Focused proof | All eight HTTP/service/DTO groups PASS: exact fixed statements/trusted actor/raw bytes; all routes; auth-before-parse; advertised/chunked oversize; strict headers/query/UTF8; safe failures; known abort replay versus lost commit; result cap/profile/duplicate/foreign identity rejection before commit; explicit prepare replay and cancelled cleanup. These mocks prove adapter behavior, not SQL authority or managed identity. |

## Current evidence

The package is implemented and its focused unit tests pass. New read-only catalog
and pending use the same SERIALIZABLE current fence as mutations. Root concrete review corrected prepare HTTP status to201 (all other successes200),
and added catalog cross-field consistency: setup only absent, adoption only ready
with acquire and candidates, and no candidates without acquire. Focused regressions
and final targeted race PASS1.64s. Same-key prepare
can return an authorized historical terminal command, including acknowledged,
without imposing expiry checks in Go. Adoption keeps the existing42 result shape;
Go checks its immutable response envelope and leaves grant/source/consent/full-set
validation to SQL. No financial arithmetic or provider call occurs in this package.

Required stable recovery identity followup: root reviewed the lost-ack plus
subsequent-revocation edge and froze Command.recovery_id as a required immutable
UUID before migration47 publication. The response guard now requires it on every
command including acknowledged replay, and available pending must echo that
exact same ID. This lets the UI request typed payload-free closure proof after
losing a successful acknowledgement and disclosure authority; pending:none
alone is still insufficient. Focused missing/zero/mismatched/acknowledged identity
regressions PASS, with all nine focused groups/race PASS1.74s and make
check/build PASS after this followup.

## Remaining work

make check/build PASS. Targeted race PASS. Full make test passes18 Node tests
and the new onboarding package, but remains non-green in four existing
acquisition/webhookhost/workerhost/workerruntime packages because this sandbox
denies TCP listener binds (EPERM). Installed Go1.26.5 uses an untracked modfile
resolving only the previously checksum-verified local x/net0.62.0 source;
tracked Go1.26.10 and dependency files remain unchanged. Canonical CI is required. Actual restricted API-role SQL47, current grant/recovery/atomicity/race
proofs, managed readiness, server composition, real browser journeys and canonical
CI belong to combined acceptance; no mocked success is substituted for those.
Root performs concrete source/security review before merge and separately owns
trusted scanned rollout/owner acceptance.

## PR handoff

Every own implementation commit includes this marked runbook and passes the
staged runbook gate. Root owns issue/PR/publication and cloud actions. This is an
ordinary explicit application flow; legacy finite operator activation is not
silently broadened or reused as authorization.
