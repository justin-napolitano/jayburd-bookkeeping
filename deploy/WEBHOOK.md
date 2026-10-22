# Review the dedicated Sandbox webhook release

Issue #271 remains open. Offline tools and inert infrastructure now have a
[dedicated existing-service updater](../docs/contracts/webhook-rollout.md).
Its new federation root is inactive by default; nothing has been deployed or
activated. Current source consent and old release manifests are expired.
Production is excluded.

## Executable updater after first owner bootstrap

First private creation, isolated runtime-state handoff, SQL mandate and separately
approved public invocation remain owner operations below. Owner then reviews
`infra/gcp/webhook-delivery` with exact main workflow SHA and independent federation
activation. This grants update/read on the one existing service, not create/delete,
public IAM, SQL or secret administration. The actual custom-role definition is
verified during selected readback; a matching role name alone is insufficient.

For a fresh reviewed tracked `deploy/webhook-release.json`, dispatch the new
`deploy-webhook.yml` on that exact owner-approved main commit with its SHA256.
No enabled manifest is supplied here. It verifies build provenance, merged270/273
ancestry, unchanged binary inputs, latest main and expiry before explicit WIF,
then preflights the existing host, updates only the fixed runtime root/backend,
and retains only a strict public receipt. No raw state/plan or provider response
is inspected/uploaded. Source consent and actual signed delivery need separate
finite approval/evidence. See the updater contract for failures, token expiry,
owner deactivation and rollback; do not widen privileges when a permission fails.

## Prepare a concrete owner review

1. Merge reviewed code after required synthetic gates and human security review.
   Build the exact merged source through the existing owner/main
   `build-images.yml` path. Retain its successful run/attempt metadata, scanned
   worker candidate, SBOM and exact-image startup evidence. The build must include
   merged PR270 and this release implementation; source ancestry needs a separate
   explicit check because a structural candidate receipt cannot prove it.
2. Review the [bootstrap/runtime ownership](../docs/contracts/webhook-release.md),
   [SQL bootstrap](../docs/contracts/webhook-bootstrap.md) and
   [release evidence schema](../docs/contracts/webhook-release-evidence.md).
   Record owner review in issue comments for security, bootstrap, identity,
   secrets, scope and compatibility. Evidence URLs are references, not automatic
   interpretation of acceptance or permission.
3. Prepare `deploy/webhook-release.json` with the exact schema in the release
   evidence contract. No enabled example is committed. `image` comes from that
   worker candidate; versions are canonical decimal strings. Set fresh expiry
   and qualification end; scope evidence binds finite Item/book/source, source
   consent, ingress keys, cleanup owner and interval. Retain only public metadata.
4. Owner retrieves explicit **public** build evidence into a private staging
   directory: `build-run.json` and `worker/candidate.json`. The directory must
   not contain credentials, provider payloads or financial records. Validate:

   ```sh
   node scripts/webhook-release-manifest.mjs deploy/webhook-release.json REVIEWED_SHA256 PREPARED_DIRECTORY PUBLIC_EVIDENCE_DIRECTORY
   ```

   This command is offline and requires all four arguments. It writes only
   `approved.json` and `webhook-runtime.tfvars.json`; it never authenticates/applies.
   Re-run immediately before each separately approved operation to check expiry.

## Proposed operator execution order (separate approval required)

- Owner creates separately reviewed backend buckets and lock/version/retention
  settings; no existing state is reused. Operator identity and exact current
  infrastructure diff require review. No new deployer permission is implied.
- Initialize bootstrap with `enabled=true`, `public_invocation_approved=false`.
  Apply reviewed bootstrap metadata; owner verifies effective dedicated identity,
  only two secret-level reads, SQL IAM user/client/login and no inherited broader
  authority. Terraform assertions prove requested grants, not effective IAM.
- Owner supplies a local Unix socket to the reviewed SQL bootstrap wrapper,
  runs its explicit check then approved enable operation with a stable key.
  Role membership alone does not enable ingress. No token or password CLI input.
- Apply the separate runtime root with the exact generated runtime variables.
  No route, caller, worker or Scheduler root is touched. Read ready revision/config
  and IAM before separately approving public invocation on this one service.
- Apply only the reviewed public binding. Retrieve **selected** Cloud Run Admin
  API **v2** Service fields (the offline receipt's allowlisted configuration,
  reconciliation and traffic fields) and this service's IAM bindings into
  private staging files. Do not substitute default gcloud v1 output, a plan, full
  state, logs or a claimed readiness boolean. No token should appear in a file
  or command argument. The new updater performs selected retrieval in memory after
  explicit WIF injection; manual offline receipt validation below remains available.

  ```sh
  node scripts/webhook-deployment-receipt.mjs PREPARED_DIRECTORY/approved.json SELECTED_V2_SERVICE_JSON SELECTED_SERVICE_IAM_JSON PUBLIC_RECEIPT_JSON
  ```

- Only after effective SQL/secret/config acceptance, review a separate application
  release enabling `connections.webhook_url`. Existing Item update remains a
  contract proposal; do not assume new enrollment settings update old Items.
- Execute separately scoped W01–W07 qualification with current consent and
  independent exact/no-journal evidence. No provider operation is part of this
  offline tooling. A signed synthetic notification or webhook-update ACK cannot
  clear actual Transactions delivery.

## Failure, rollback and cleanup

Stop on any mismatch. An apply may have partially succeeded: inspect through
owner access without copying plans/state/provider diagnostics into issues.
Disable the ingress mandate with a **new** operation key; same-key changed JSON
is correctly rejected. Revoke this service's public binding through bootstrap
`public_invocation_approved=false` while keeping `enabled=true`.
`prevent_destroy` protects the service/identity; setting runtime null is not a
safe live stop/cleanup procedure. Do not disable the original workers/Scheduler.

If runtime rollback is required, review a fresh finite forward release pointing
to a retained known compatible worker digest and matching trusted evidence, then
validate/read back again. No SQL rollback or secret-version substitution.
Before qualification end, owner revokes the scoped mandate/consent and public
binding, records fixed redacted outcomes, and reviews any retained resources.
Approval timestamps do not stop Cloud Run automatically. No broad project
privilege, unbounded provider consent or unattended cleanup is authorized here.
