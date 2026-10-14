# Acquisition worker infrastructure

This root is **owner-operated infrastructure**, not a worker deployment workflow.
It combines bootstrap IAM, precreated secret containers, private buckets, queue,
SQL IAM users, and optional worker services. The existing application deployer is
not authorized to apply it. No deployer permission is broadened here; image build
receipts do not authorize promotion. A separately reviewed bootstrap/runtime split
and state/IAM workflow is required before automated worker deployment.

Defaults provision no Cloud Run service or Scheduler job: `runtime = null` and
`scheduler_enabled = false`. Supplying a digest-pinned runtime still leaves the
scheduler paused. Activation additionally requires completed SQL role/bootstrap and
publisher binding enrollment, exact credential versions/mappings, caller identity
qualification, Infisical sync permissions/freshness, and Sandbox qualification.
No secret values or service-account keys are Terraform inputs; only exact numeric
version identifiers are supplied.

The owner may install `backend.tf.example` only after the reviewed foundation
addition provisions `${project}-jayburd-dev-workers-state`, using prefix `workers`.
This dedicated state bucket receives no existing application deployer grant. A
prefix within the runtime-state bucket would not isolate bootstrap state because
the deployer has bucket-wide ObjectAdmin there. Never initialize a backend or use
real state for local tests. No apply was performed in this implementation.

## Identity and storage boundaries

- Existing foundation identities own API, publisher, scheduler and task invocation;
  this root adds a separate acquisition sync identity and SQL IAM user.
- Scheduler can invoke publisher only; task-invoker can invoke consumer only.
  Publisher can enqueue only in the acquisition queue and act as the task identity.
  Service IAM remains mandatory even though HTTPS ingress is reachable.
- Raw evidence create/get/delete applies only to `raw/`. Immutable tombstones use
  the adapter's `deleted/` prefix and receive create/get **without delete**.
  Only raw objects expire after seven days; tombstones have no lifecycle expiry.
- Credential registry API create/get and sync get are limited to
  `provider-credentials-v1/`. Neither principal gets registry delete or object list.
- Secret access is bound to the ten precreated containers only. The optional
  Infisical sync principal gets metadata/version-add authority on those containers,
  without secret create/delete or project secret administrator access. Actual
  Infisical sync compatibility with this narrow role remains a hosted qualification
  gate; do not compensate with broad permissions.
- Consumer receives explicit numeric `-project-number`, exact numeric Google
  version resources, and the fixed registry prefix. Logical provider versions are
  not Google version ordinals. The eight credential slots are never reused.

Cloud SQL connector/login roles are project-level standard roles; actual SQL grants
remain independently scoped by owner bootstrap. No runtime receives SQL ownership
or provider secret write capability. Queue retries do not replace SQL completion,
retry budgets or current authorization.

## Offline validation

```sh
terraform -chdir=infra/gcp/workers init -backend=false
terraform -chdir=infra/gcp/workers validate
terraform -chdir=infra/gcp/workers test
```

Tests use `mock_provider "google"` and fictional project/service identities. They
check infrastructure-only defaults, exact prefix IAM, immutable registry/tombstone
permissions, finite secret containers, digest/version validation, private invocation,
capacity/deadlines and paused scheduling. They prove configuration behavior, not
hosted IAM, actual storage deletion, Infisical behavior or provider access.
