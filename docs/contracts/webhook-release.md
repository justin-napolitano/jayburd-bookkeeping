# Isolated webhook infrastructure and release boundary

## Decision, owner and scope

#271 engineering preparation, coordinator-reviewed2026-10-22 under
`PLAID-271-PREP-20261022T1725Z`; implementation and synthetic tests only. Extends
[managed runtime](webhook-managed-runtime.md) with separate bootstrap/runtime roots.
No cloud apply, provider call, secret access, source consent renewal or public
exposure is authorized by merging this contract. Human/security review remains
required under the applied Citizen Developer security skill. Repository
Infisical/GCP authority governs over generic enterprise Azure examples.

## Ownership and inert defaults

`infra/gcp/webhook-bootstrap` owns only `jayburd-dev-webhook` service account,
Cloud SQL client/login project bindings, a `CLOUD_IAM_SERVICE_ACCOUNT` user on
existing instance `jayburd-dev`, two static secret-level reader bindings, and the
optional Invoker binding on the fixed `jayburd-dev-webhook` service. It creates
no secret/value/version, database, bucket, scheduler, queue, KMS, WIF, deployer
capability or private API/worker resource. `enabled=false` creates no resources.
`public_invocation_approved=false` is independent: identity preparation does not
expose ingress. The public binding requires explicit enabled+approval and an
already-existing service. It retains IAM invocation checks and grants allUsers
only on that service. Organization policy failure is a stop, never an automatic
IAM-check bypass or private-API exposure.

`infra/gcp/webhook-runtime` owns exactly one fixed service when `runtime` is
non-null; default null creates no service. Inputs are explicit project ID/number,
reviewed regionus-central1 and runtime `{image,client_id_version,client_secret_version}`.
The image must be that project's `jayburd-dev/worker@sha256:<64 lowercase hex>`;
versions and project number are canonical positive signed64 decimal identifiers,
never aliases. No arbitrary identity, service name, executable, secret path,
provider URL, environment value or volume is configurable. Runtime owns no IAM.

Owner-created backend buckets must be distinct: `PROJECT-jayburd-dev-webhook-bootstrap-state`
(prefix `webhook-bootstrap`) and `PROJECT-jayburd-dev-webhook-runtime-state`
(prefix `webhook-runtime`). Backend examples are not loaded by tests. Existing
application/workers/worker-runtime state and `deploy-workers` manifest remain
unchanged. Existing SQL is immutable; root coordinates separate owner bootstrap
proof, release tooling and shared source/index integration.

## Exact service profile

Service `jayburd-dev-webhook` selects command `/webhook` from the reviewed worker
digest and passes exactly five public arguments: listen0.0.0.0:8080, explicit
Cloud SQL instance, numeric project, and full fixed-name numeric client-ID and
client-secret version resources. It uses only the dedicated webhook identity.
No secret values or bank-token slot reads are injected. Secret IAM grants cover
only `jayburd-dev-bank-sync-client-id` and `jayburd-dev-bank-sync-client-secret`;
version pinning is enforced by configuration, not claimed as version-level IAM.

Targets: request timeout10s, concurrency16, service/revision max1 instance/min0,
1CPU/512Mi, idle CPU enabled, startup boost disabled, bounded TCP startup probe
on8080, deletion protection and single100% latest traffic. TCP startup proves
listener availability, not signature verification or SQL authority. The Go pool
maximum is2; planned old/new revision overlap needs up to4 SQL connections.
Scaling settings are requested capacity policy, not an absolute billing/load
ceiling. Hosted latency/retries/cold starts still need measurement.

Reviewed Sandbox enrollment constant is
`https://jayburd-dev-webhook-9088794002.us-central1.run.app/webhooks/plaid` for
`jayburd-bookkeeping-dev`; the generic Terraform root computes the corresponding
project-number URL for isolated synthetic tests. Exact release orchestration must
bind the approved project/number and confirm observed URL/revision/configuration
before enrollment. No deploy-by-tag, source rebuild between stages or automatic
fallback to old worker command is accepted.

## Operator sequence and revocation

1. Review source, trusted same-source worker build/exact-image tests/SBOM/scans,
   finite Sandbox Item/book scope, current consent, exact secret versions and
   security approval. Expired releases/consent are not renewed implicitly.
2. Owner explicitly bootstraps identity/Cloud SQL IAM user/two secret reads with
   public approvalfalse. Existing SQL IAM authentication/secret containers must
   already exist. No provider tokens are retrieved by an agent.
3. Separately reviewed owner SQL bootstrap grants only `bookkeeping_webhook`
   with INHERIT true, ADMIN/SET false, database CONNECT, search_pathpg_catalog and
   bounded connections; reject privileged/mixed/direct-table authority. Owner
   binds/enables actual login using existing `integration.set_webhook_ingress`
   in a serializable owner transaction with stable idempotency key. IAM login
   creation and successful transport Ping are insufficient authority evidence.
4. Owner-operated guarded runtime release validates exact approved metadata,
   applies only dedicated runtime state, then confirms Ready/generation, image,
   command, exact args, identity, traffic, URL and private IAM unchanged.
5. After independent public-exposure approval, bootstrap public flag grants
   invocation only to this service. Fixed unsigned rejection and private-route
   denial are transport checks, not W01 success.
6. Configure approved enrollment URL / scoped existing-Item webhook update through
   reviewed credential-owning adapters. Actual signed SYNC_UPDATES_AVAILABLE must
   reach SQL receipt/wakeup and existing publisher/task/worker/source review.
   Update acknowledgement and fire_webhook response alone do not qualify it.

Rollback uses a previously qualified worker digest with identical webhook
configuration and identity; it must still contain `/webhook`. Root owns exact
release/readback/rollback tooling; existing worker receipt cannot be reused as
webhook deployment proof. To stop ingress, owner disables the SQL ingress binding
and removes the dedicated public grant with separately reviewed operations; do
not change private-service IAM or delete durable receipts. Current consent and
stream authority remain checked. `prevent_destroy` deliberately blocks toggling
bootstrap off/runtime null after creation as a destructive shortcut. Revocation,
retention and cleanup must preserve evidence; no automatic Item removal or consent
renewal occurs. New root deployment and W01–W05/W07 qualification remain open.

## Evidence, alternatives and threat review

[WR01–WR06](../research/webhook-release-sources.md) record authoritative sections,
review depth and access limits. Standards assessment: internal operational
infrastructure and existing proprietary notification; no financial message,
amount/currency/SQL contract or accounting policy changes. Existing ISO applicability
is unchanged; no ISO20022 designation or conformance claim.

Alternative of adding a third mode to private worker release was rejected because
its strict manifest and Scheduler-paused receipt govern different caller authority.
Two roots cost separate bootstrap/state ownership but avoid coupling public
invocation to private API/workers. Bootstrap public approval remains explicit;
merely attaching an evidence URL is not authorization.

| Threat / initial risk | Mitigation / remaining acceptance |
| --- | --- |
| Spoofing/tampering/replay, high | Existing ES256/body/hash/age and SQL receipt atomicity; real signed provider chain still required. |
| Privilege elevation, high | Fixed dedicated identity, two secret-level readers, no additional service capabilities; SQL actual-session checks and owner bootstrap proof. |
| Disclosure, high | No Terraform secret values, secret creation, env/mount injection or provider payloads; exact pinned paths and redacted evidence. |
| Resource exhaustion, high | Fixed request/compute/concurrency/scaling targets, Go socket/admission/key/pool bounds; real timing/overload and cost monitoring still open. |
| Repudiation/lost work, medium/high | Trusted candidate and exact readback receipt, durable notification/wakeup, separate redacted operational audit; hosted retry/catch-up proof pending. |

Mocked tests name `default_is_inert`, `dedicated_identity_and_two_static_reads`,
`public_approval_is_one_fixed_service`, `isolated_route_identity_and_budget`, and
negative project/region/image/version scenarios. They prove requested configuration,
not effective inherited IAM, a created resource, transport, delivery or posting safety.
