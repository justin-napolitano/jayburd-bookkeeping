# Development Cloud Run rollout

## Decision

Implement the owner-requested #27 deployment slice, 2026-10-12. Terraform owns
services and the migration job; GitHub only invokes the configured job. A separate
runtime state bucket holds `runtime` and `migration` prefixes. Existing
`application` state remains owner-operated Cloud SQL. This avoids giving the
workflow database administrator rights or using partial `-target` applies.
Sources G19–G23 in the [GCP register](../research/gcp-sources.md) establish the
mechanisms, not successful cloud acceptance. Alternative: one state with targeted
migration applies; rejected because it conflates database/runtime authority and
makes staged plans harder to reason about.

## Authority and runtime

Foundation adds a separate `jayburd-dev-deploy` federation pool, disabled by default,
with exact owner/repository IDs, main, workflow_dispatch and deploy-dev workflow.
A separate pool prevents the existing repository-only builder principal binding
from accepting deployment tokens, or vice versa. Owner dispatch and an exact
release-file hash are the promotion boundary; an unprotected GitHub environment
name is not approval. The triggering actor must also be the owner.

Deployer gets Cloud Run admin, repository image reader, runtime bucket object
admin and ActAs on only API/frontend/migrator. The former application-state grant
moves to runtime state. No database-state, foundation-state, project IAM-admin,
image-writer or secret-reader grant is added. Cloud Run admin is project-scoped:
it can change service IAM and launch code as those three identities, including
migrator SQL privileges. It is a privileged release identity, not a sandbox.
Only owner-reviewed source may receive it. No runtime Token Creator role is needed.
API gets a custom `firebaseauth.users.get` role for revocation lookup.

API ingress is network-reachable but IAM-restricted to frontend Invoker; this is
not an internal-only VPC service. Cloud Run-to-internal ingress requires additional
network routing, deferred to avoid VPC/NAT cost and complexity. Frontend exposes
sign-in publicly; finance still requires the managed Go authentication boundary.
Deterministic service origins remove circular frontend/API configuration. The
API explicitly accepts its deterministic origin as a custom ID-token audience.

Both services use request billing, min zero/max two at service and revision levels,
1 CPU/512 MiB. API concurrency 16 and a five-connection pool; web concurrency 32.
Old/new API revisions can use 20 connections, bounded again by the SQL login cap.
Cloud Run instance maxima are scaling controls, not hard spending guarantees.
API startup probes `/readyz`, web probes `/api/auth-config`; only a ready revision
receives latest traffic. API liveness uses `/healthz`, avoiding database-triggered
restart loops. Migration has one task, zero retries, dedicated identity and a
660-second task bound around its existing ten-minute process deadline.

## Release and rollback

The workflow requires reviewed `deploy/dev-release.json` and its exact SHA256
provided by the owner at manual dispatch. There is deliberately no enabled/sample
release file. [Release instructions](../../deploy/README.md) define fields.
Missing, expired, foreign-project, mutable-image, mixed-build and incomplete
metadata fail before cloud authentication. Unsigned candidate receipts alone
cannot approve release. Successful owner/main build metadata and all three
same-run/attempt receipts must match the manifest. Recheck #23/#24 merged and verify their merge commits are ancestors of the release; require
#6/#11/#26 closed; owner evidence links supplement, not replace, these checks.

Forward order: migration-root plan/apply, execute-and-wait, runtime-root plan/apply,
anonymous transport probes, minimal receipt. One GitHub concurrency group and
GCS locking serialize workflow deployments; the SQL advisory lock protects other
migration runners. Do not run owner applies concurrently. A migration failure
leaves both service configurations untouched. Failure after API update can leave
mixed API/web versions: this is not an atomic two-service rollout. Require
backward-compatible contracts, investigate and dispatch explicit rollback.

Rollback requires a successful earlier deploy run, matching retained API/web
receipt and reviewed compatibility with the current schema. It skips both job
configuration and migration execution, redeploying known digests via Terraform.
Never down-migrate automatically. Expired build/deployment artifacts block this
workflow; preserve approved evidence and review a durable retention path before
long-term operation. No arbitrary image override bypass is offered.

Provider logs and saved plans stay runner-private and are never artifact uploads.
After all offline gates, WIF produces one masked, one-hour deployer access token
passed through step-scoped environment settings to Terraform and gcloud. No
credential file is created; the job timeout is 50 minutes. Tokens are not renewed
or logged. Receipts contain public image/release identity and ready revision IDs. No real token acquisition,
state/plan inspection or cloud operation occurs in default tests.

## Acceptance and limits

`make release-test`: immutable/hash/expiry/build/rollback rejection, anonymous
transport denial and actual shell orchestration under fake executables, including
migration failure and rollback-without-SQL. `make infra-test`: pinned-provider
validation with synthetic mocks for IAM, state separation, service capacity,
managed configuration and job bounds. Existing Go checks remain required; no SQL,
API financial semantics or new ISO 20022 message changes.

Real deployment still requires owner foundation update, reviewed Cloud SQL apply
and role bootstrap, Identity Platform/MFA/domain setup, principal enrollment and
Infisical evidence. The first runtime uses IAM DB login and public Firebase
identifiers, so it has no application secret to invent or sync. Infisical remains
the selected authority when secrets are needed. Transport smoke proves only API
anonymous denial, web unauthenticated denial and available shell/config. Real MFA,
revocation, cross-entity denial, synthetic posting/recovery, load, restore and cost
remain separate #27/#12 acceptance; passing deployment must not close those issues.
