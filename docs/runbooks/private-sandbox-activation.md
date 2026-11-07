<!-- stage-runbook: true -->
# Private Sandbox activation — 453

## Status and boundary

Branch feat/453-private-activation from ca1b37e. Owner requested source
implementation; root coordinates GitHub, shared API/server/web composition and
all managed actions. Only dedicated activation files, contracts and tests belong
to this lane. No merged SQL edit, migration allocation, cloud/provider call,
credential/financial-record read or live activation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Read live453, AGENTS, index, bibliography, decision evidence, standards, identity/setup/bootstrap and private registry contracts/source. Created isolated worktree. |
| Proposal | Sent coordinator the two-stage owner-only provisioning and verified-actor finite adoption plan; documented deadline/unknown-outcome and missing managed operator mechanism explicitly. Executable changes await profile review. |
| Profile freeze | Coordinator approved finite two-stage source mechanism, no new SQL/IAM/job. Root owns default-absent Sandbox-only flag, shared server/OpenAPI and real HTTP/SQL integration fixture. |
| Initial implementation | Immutable strict hash-bound plan loader, authenticated plan-reference-only handler and fixed adoption SQL adapter; separate owner-only psql provisioning stage with exact enrolled identity comparison and precommit expiry check. |
| Unit evidence | go test -race ./internal/collectionactivation PASS with installed Go1.26.5/external temporary modfile; loader/identity rejection, strict JSON, exact SQL tuple, known abort bounds and lost-commit explicit replay. No tracked toolchain change. |
| Operator SQL proof | Actual pinned PostgreSQL17 with unchanged migrations1–44 and real synthetic operator/API logins: TestActivationOperatorInitialReplay, TestActivationOperatorIdentityScopeAndAtomicDeny, TestActivationOperatorPrecommitExpiryRollback and TestActivationOperatorRuntimeProvisionDenied all PASS. One original workspace/provision receipt retained; delayed new grant fully rolled back after approval expiry. |
| Transport followup | Six activation unit/race groups PASS, adding precommit-expiry discard and closed HTTP header/query/oversize/current-expiry denials before DB work. |
| Source checks | make check/build PASS using installed Go1.26.5 and external temporary modfile/cache. make test attempted:18 offline release tests and non-listening packages PASS; preexisting acquisition/webhookhost/workerhost/workerruntime listener tests fail because local sandbox denies TCP bind. No skips or test weakening. |
| Restricted HTTP/SQL | New integration-tagged activation_http_integration_test.go uses existing legacyFixture, actual restricted synthetic_runtime pool and real Handler→Service→SQL. All5 groups PASS on source44 and again on source43 after empty44 down in the owned unused template. TestActivationHTTPPostgres fixes exact replay/changed-intent denial/one adoption receipt/paused generation2 and unchanged original anchors; TestActivationHTTPCurrentAuthority covers principal/workspace/book/source revocations before execution and receipt replay (8 subcases); TestActivationHTTPPreconditions covers generation, omitted active second source, pending connection/staging and injected late receipt failure with atomic rollback (5 subcases); TestActivationHTTPLostReplyPostgres loses one real commit reply then explicitly replays and rechecks revoked source authority; TestActivationIdentityClaimRaces observes actual blocked legacy/native identity contenders in both winner orders. |

## Current evidence

[Contract](../contracts/private-sandbox-activation.md) is approved for source
implementation. Unit proof is synthetic transport evidence; the four operator
SQL groups ran against a network-none disposable fixture. Host nested shell
Docker access was denied, so equivalent direct Docker commands copied the exact
SQL and fixed fixture assertions into the owned container and executed there.
scripts/test-private-activation.sh preserves the same proof as a normal CI gate;
bash syntax validation PASS. Root reassigned its previously uncreated dedicated
HTTP/SQL test file to this lane; its actual restricted PostgreSQL evidence is
recorded above. The authenticator in this integration binary remains a synthetic
test double, not managed Firebase identity proof. Existing managed
operator authority is unresolved and no managed activation is claimed.

## Remaining work

Run canonical full checks with root-owned shared wiring. No managed
operator profile or live activation is implied.

Before managed use, the owner must review the fixed Sandbox database, active
issuer/subject and matching principal UUID, exact capability set, expected workspace
revision/null, evidence UUID, SQL key and finite expiry. An existing authorized
human operator with SET bookkeeping_owner is required. Source review found only
the earlier owner-operated bootstrap prerequisite; no retained interactive owner
session or managed activation job is established. Do not repurpose the restricted
migrator, grant API owner/provision rights or invent an impersonated actor. The
returned workspace then becomes an explicit adoption-plan target. Independently
review its hash and every selected entity/book/Item/source UUID, generation,
display name, evidence, key and expiry before enabling the source-only plan flag.

## PR handoff

Refs453/65. Root coordinates integration/publication. No migration, IAM/job or
cloud change. Preserve actual verified identity and current SQL authorization.
