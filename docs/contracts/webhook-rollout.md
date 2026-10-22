# Existing webhook updater

## Decision and owner boundary

#271 wave2, coordinator-reviewed2026-10-22 under lease
`PLAID-88-WAVE2-20261022T1930Z`. Implements the missing executable updater after
merged preparation in PR273. No actual federation, deployment, provider operation,
source-consent renewal or new enabled release manifest is authorized here.

First creation remains owner-operated: dedicated bootstrap identity/SQL login,
restricted SQL grants and mandate, first **private** runtime creation in the fixed
isolated backend, separate security/public-Invoker approval and effective identity
review. Only then may owner prepare updater federation and a fresh exact release.
This avoids giving the routine updater create/delete/IAM/bootstrap authority.
Existing deploy-workers, private application IAM, workers and Scheduler are untouched.

## New owner bootstrap capability

`infra/gcp/webhook-delivery` defaults `enabled=false` and creates nothing.
With explicit owner review it creates a separate `jayburd-dev-webhook-deployer`,
custom roles and narrowly bound permissions. Separate `federation_active=false`
disables the pool/provider while preserving protected identity/grants. Activation
requires enabled resources and an exact reviewed40-hex `workflow_sha`. Owner must
review and update the pin for each new main workflow commit; dispatch inputs do
not select cloud trust. Invalid contradictory activation is rejected.

The pool/provider bind the canonical numeric repository/owner/actor IDs,
workflow_dispatch, refs/heads/main, exact deploy-webhook workflow_ref and SHA.
The workflow also checks triggering actor, latest main and tracked manifest bytes.
Short-lived tokens are minted only by the explicit pinned auth step; there are no
credential files or service-account keys. Disabling federation stops future
exchanges, **not already issued tokens**. Those expire separately; emergency
runtime/SQL mandate/public-binding controls remain owner operations.

Capabilities:

- Exact service `jayburd-dev-webhook`: get, update, getIamPolicy only. No service
  creation/deletion, setIamPolicy, public binding change, role administration,
  secret access/write, SQL grant, source fetch or worker/Scheduler capability.
- Project read-only `run.operations.get` for provider polling and `iam.roles.get`
  for actual custom-role definition verification. These are broader metadata reads,
  not project-wide service mutation.
- ActAs only the dedicated webhook runtime service account.
- ObjectAdmin only the existing `PROJECT-jayburd-dev-webhook-runtime-state` bucket;
  no bootstrap/delivery/application/worker state bucket. Backend must be owner-created
  and handed off explicitly. Tooling never reads state or plan contents; Terraform
  privately creates/applies its opaque plan using the existing approved pattern.
- Artifact Registry reader only on existing `jayburd-dev` repository, required for
  image deployment. This is read access to that repository's images, not image-path
  IAM or registry writes.

Owner-only backend for this bootstrap is
`PROJECT-jayburd-dev-webhook-delivery-state`, prefix `webhook-delivery`. The updater
has no access to it. No backend, public service or IAM grant is applied by tests.

## Executable release

The new `deploy-webhook.yml` workflow has only workflow_dispatch. Missing
`deploy/webhook-release.json` or invalid hash fails before authentication. No
fresh authorization dates or scope have been invented in a committed example.

Before explicit WIF authentication, it validates exact manifest SHA256/expiry and
fixed Sandbox project, downloads the exact trusted successful owner/main worker
run/attempt artifact, verifies tracked manifest bytes and current HEAD/main, and
checks immutable PR270/273 ancestry. Binary inputs in Dockerfile.worker, go.mod,
go.sum, internal and the four compiled command directories must match the trusted
build source even when release tooling/docs changed afterward. Tests run before
cloud authentication. Candidate-only receipt is not deployment permission.

`scripts/deploy-webhook.sh` rederives the runtime variables from approved tracked
bytes, requires explicit injected tokens and verifies an existing, reconciled,
correctly configured public webhook service before touching only the fixed runtime
root/backend. It prepares/applies the opaque private plan. Immediately before
apply it rechecks actual latest main, provenance and authorization expiry. It
cannot inspect/import/repair state: owner evidence must attest the first private
creation and isolated state handoff. Missing/wrong state must fail through narrow
IAM/protected resource semantics; there is no broad permission fallback.

The collector calls only fixed v2 Service GET, service getIamPolicy and the fixed
custom-role GET, with explicit WIF token held in process memory. No token is placed
in argv, files or logs; no ADC/token discovery occurs. HTTPS reads have10s timeout,
128KiB maximum and no redirects. Server-side field mask excludes raw environment
values. Raw selected responses remain in memory and are rejected without logging;
only strict validated public receipt is written. Preflight permits an old valid
worker image/version policy but requires current fixed identity, routing and bounds.
Post-apply receipt requires the exact newly approved image/versions/configuration.

Public service policy permits the existing sole allUsers Invoker plus optionally
one **exact** updater binding to the fixed deployer. The optional role must also
be read from actual IAM: fixed name, GA, not deleted, exactly get/update/getIamPolicy.
No role-name-only trust or extra conditional/public/admin binding is accepted.
Direct service policy does not establish absence of inherited broad IAM; owner
identity evidence must independently verify that before enabling federation.

Only the public receipt artifact is retained. Provider diagnostics, raw metadata,
opaque plans and state are never uploaded or inspected. Any failure stops; partial
apply is possible and needs owner review. No automatic rollback, paid refresh,
Item mutation or posting occurs. A successful receipt remains control-plane
configuration evidence, not actual signed receipt→wakeup→worker publication,
financial equality, zero-journal proof or Production approval.

## Cleanup and rollback

Before qualification end, the named owner disables the scoped SQL ingress mandate
with a new stable key and removes the dedicated public binding through its original
bootstrap, preserving receipts. Owner deactivates updater federation while keeping
enabled identity/grants; existing short-lived token expiry is a separate bound.
Do not toggle protected resource enabled flags/runtime null to simulate rollback.
A compatible rollback uses a newly reviewed finite forward manifest selecting a
retained trusted compatible worker digest, exact evidence and readback again.
Authorization dates do not stop Cloud Run or renew source consent automatically.

## Evidence and security review

[WU01–WU07](../research/webhook-rollout-sources.md) distinguish platform facts from
project choices and actual hosted gaps. Alternative of automatically creating a
public host was rejected because it requires broader IAM and mixes first bootstrap
with routine promotion. Alternative of accepting a role name without its definition
was rejected because an administrator could widen that role after initial setup.

| Threat / risk | Control / remaining evidence |
| --- | --- |
| Untrusted dispatch/source, high | Owner/repo/main/event/workflow SHA federation, tracked hash, trusted build provenance and immutable ancestry; actual federation remains untested. |
| Privilege elevation, high | Service-only update, actual exact role readback, one ActAs/state bucket; inherited IAM needs owner verification. |
| Disclosure, high | Explicit in-memory token, no credential files/argv, bounded selected response and receipt-only upload; fake failure/redaction tests. |
| Lost/partial promotion, high | Current-main/expiry rechecks, strict preflight/post-readback and owner cleanup; no blind retry/automatic rollback. |
| Resource abuse, high | Existing finite service profile, short-lived token, separate inactive default; hosted timing/cost/load remains open. |

The Citizen Developer security skill requires human/security review before cloud
exposure or IAM/federation activation. Repository Infisical/GCP authority remains
in force. This operational interface is not an ISO20022 financial exchange; no
money/currency/SQL or accounting semantics change, and no ISO conformance is claimed.
