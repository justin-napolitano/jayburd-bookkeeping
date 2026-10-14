# Acquisition worker bootstrap

This root is owner-operated infrastructure: IAM, SQL identities, secret containers,
private storage, queue, invocation bindings and always-paused Scheduler. Runtime
services now belong to `../worker-runtime`, with separate state and deployment
identity. Application and worker CI cannot apply this bootstrap state.

Fresh bootstrap uses `worker_services_exist=false`. After runtime creates both
services and the owner verifies Scheduler PAUSED, owner applies with that flag
true to install exact invocation bindings. There is no scheduling activation flag.
For existing deployments, retain true during the non-destructive state transfer.

Read [worker release migration](../../../docs/contracts/worker-release.md) before
applying this split. The removed block preserves existing services; owner imports
them into dedicated worker-runtime state. Scheduler and invocation IAM remain at
the same bootstrap addresses. Never perform automatic destroy/recreate adoption.
Bootstrap backend remains `${project}-jayburd-dev-workers-state`, prefix `workers`.

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
permissions, finite secret containers, private invocation, narrow deployment grants and paused scheduling. They prove configuration behavior, not
hosted IAM, actual storage deletion, Infisical behavior or provider access.
