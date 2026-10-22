# Provider webhook delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #103, lease WEBHOOK-GO-103. Implements the reviewed
[webhook contract](../contracts/provider-webhooks.md) with synthetic inputs only.
No application API mount, provider call, real credentials or cloud deployment.

## Stage ledger

- Read project instructions, source register and provider/runtime contracts.
- Refreshed official Plaid verification and Transactions notification sections.
- Coordinator approved64KiB/future60second profile, maintained JWT verifier,
  bounded key cache, separate ingress and narrow SQL receipt/catch-up adapters.
- Sole SQL author froze migration14 interfaces and role ownership.
- Implemented explicit dependencies, fixed Sandbox key endpoint, exact signed-byte
  verification, body schema, short transactions and safe acknowledgement behavior.

## Current evidence

- `make check test build` passed: vet, race tests, migration file checks and builds.
- `go test -race ./internal/webhook -count=1` passed.
- `bash scripts/test-webhook.sh` passed15 groups and16 subtests, including three
  actual PostgreSQL groups against final migration14 fromc00325c. Tested up-file
  SHA256:e5e19f843358b17b425db265a2b4b914064aefc8053c03371d68be08e7beee33.
- Real restricted sessions commit a receipt/wakeup before injected response loss;
  HTTP returns503, then identical signed delivery resolves204 with exactly one
  receipt and intent. Valid high/low-S equivalents also share that same receipt.
- Revoked replay, unknown Item, API/sync/mixed-role denial create no new intent.
  Catch-up lost-commit replay preserves the same tick, bounded cursor and one intent;
  revoked streams are skipped. Source observations, journals and financial outbox
  remain unchanged. Key/credential callbacks assert no open SQL transaction.
- Signed synthetic tests cover exact original bytes, strict duplicates/claims/body,
  stale/future age, malformed metadata, cached expiry, bounded rotation/misses,
 16-request admission, single-lookup cancellation and fixed key transport bounds.
- Independent peer review identified ECDSA equivalent-signature replay identity.
  Coordinator-approved canonical-S fingerprinting fixed it; independent re-review
  found no remaining blocker. Original signatures still use maintained JWT verify.

Tests generate P256 keys in memory and inject HTTP; no real credentials or provider
network is contacted. Temporary SQL/harness copies were used only for proof and
removed before commit. The coordinator owns their final integration.

## Remaining work

Coordinator integrates migration/harness/shared dependency promotion and runs the
combined status/UI/transport gates. Hosted endpoint,
credentials, scheduler cadence, provider qualification and operational load remain
separate approval gates; no claim of deployed or timely transaction acquisition.

## PR handoff

Only internal/webhook and its contract/runbook are owned by this lane. No SQL or
shared server/deployment/API modifications. Local proof is complete; hosted qualification is not claimed.

## 2026-10-22 bounded host stage — #103 / #88

Lease `PLAID-88-SWARM-20261022T1540Z/WEBHOOK103`, dedicated branch
`issue/103-webhook-runtime`. Coordinator approved an injected listener composition
and reviewed constructor checkpoint; SQL, application API, IAM, deployment and
provider calls remain excluded. Read fresh #103/#87 exports and W01–W07; refreshed
P88-14/P88-15 and official Go server field descriptions. Added contract alternatives,
applicability and STRIDE review. No new dependency or financial message.

Added `internal/webhookhost` with exact verified route, restricted-pool constructor,
fixed transport/context bounds, no general OPTIONS bypass, silent HTTP diagnostics,
safe headers and single-use lifecycle/cleanup. It composes existing verifier and
SQL business logic without mounting the private API. Real loopback tests use
original synthetic notifications, ephemeral in-memory P256 keys and fake committed
receipt outcomes, not provider data or a hosted service.

Validation: `make check test build` PASS, using local Go1.26.5 with an external
modfile (tracked pinned1.26.10 unchanged). Final
`go test -race -count=1 ./internal/webhookhost ./internal/webhook` PASS including
all five new host groups, ordinary stalled-body timeout and actual `OPTIONS *`
denial. `make webhook-test` PASS against disposable synthetic PostgreSQL using
the cached-image adapter. Existing actual-session SQL receipt/replay/revocation
tests remain unchanged. Logs `/private/tmp/jayburd-103-core.log` and
`/private/tmp/jayburd-103-webhook-sql.log` are local synthetic command outcomes.
Sandbox loopback denial was retried with approved escalation, not skipped.

Remaining: managed Cloud SQL currently has no webhook component; dedicated
`jayburd-dev-webhook` IAM/bootstrap/role review, exact-version secret composition,
HTTPS ingress and enrollment configuration are required. Reusing API or sync
identity is prohibited. W01–W05 and hosted W07 remain OPEN. Existing synthetic
verification cannot clear actual delivery, retry timing, catch-up scheduling or
receipt→task→worker→source publication. No Item-event extension, real provider call,
cloud mutation or production activation occurred. Root owns next-stage integration.

## 2026-10-22 managed runtime stage — #103 / #88

Coordinator reviewed `docs/contracts/webhook-managed-runtime.md` and dedicated
Cloud SQL component capability in root commit2422e68 (cherry-picked5926111) before
this stage. That component support is not an IAM/database grant. This lane adds
`webhookhost.Config`, `Managed`, controlled `Runtime.Run` and `cmd/webhook`, reusing
existing metadata-only direct HTTP, integrity-checked exact-version GCP secret
reader, fixed Sandbox key client, verifier and SQLReceiver. Cloud fields have no
defaults; listener defaults only to127.0.0.1:8080. No DSN, role, plaintext-secret,
provider environment or caller-supplied URL override is accepted.

Config validates before factories or I/O: reviewed instance/region, canonical
numeric project, fixed client ID/secret names and numeric versions, bounded
literal listen host/port. Managed construction does no credential/key read; it
opens only the dedicated webhook Cloud SQL connector/pool and a20s startup Ping.
Ping proves transport only. Existing SQLReceiver still checks actual session
capability per delivery before atomic SQL work. Runtime and Host reject reuse
and post-close execution. Failures unwind partial dependencies; successful
shutdown closes pool/connector and idle HTTP connections once. CLI emits one fixed
failure message, never arguments, infrastructure diagnostics, payloads or tokens.

Named synthetic tests: `TestManagedInvalidConfigurationHasNoIO`,
`TestManagedOwnershipAndDedicatedPool`, `TestManagedFailureCleanup`,
`TestPinnedCredentialResolution`, and `cmd/webhook` `TestParse`/`TestExecute`.
They cover inert configuration, exact secret resources, no construction-time
SQL/key lookup, dedicated component, startup cancellation, partial construction
cleanup, command failure cleanup and redacted outcomes using generated keys and
injected capabilities. Targeted managed/CLI race tests PASS. Final
`make check test build` PASS; explicit `go build -trimpath -o bin/webhook
./cmd/webhook` PASS (root coordinates addition to the shared build target).
Local log `/private/tmp/jayburd-103-managed-core.log`; Go1.26.5/external modfile,
tracked pinned toolchain unchanged. Staged runbook guard and diff check PASS.
No new dependency or migration.

Remaining hosted prerequisites from the prior stage still apply. The component
constructor now exists, but no dedicated IAM user/grants, secret permissions,
public routing, image release, enrollment update or actual provider delivery has
been provisioned. Root coordinates separate real-socket→restricted-SQL integration
proof and exact trusted-image delivery review. No source-provider/cloud call,
financial command, environment/credential inspection or account activation was
performed by this implementation/testing lane. W01–W05 and hosted W07 remain open.
