# Controlled worker release

This contract separates release configuration from infrastructure ownership. It
introduces no financial wire format and makes no new ISO conformance claim; the
existing acquisition message and accounting contracts remain authoritative.

## State and authority

`infra/gcp/foundation` owns the dedicated `jayburd-dev-worker-deployer` identity,
`jayburd-dev-worker-deploy` workload identity pool, `owner-worker-deploy` provider,
and `${project}-jayburd-dev-worker-runtime-state` bucket. Federation is disabled
by default. The only accepted workflow is owner-dispatched
`.github/workflows/deploy-workers.yml@refs/heads/main`, with immutable repository,
owner and actor IDs. Application deployment trust remains separate.

`infra/gcp/workers` remains owner-operated: SQL IAM users, queue, secret containers,
storage, runtime and release IAM, invocation bindings and Scheduler. Scheduler is
always configured paused. `worker_services_exist=false` initially withholds the
two invocation bindings; the owner sets it true only after both services exist and
pause is verified. This flag is not a CI input.

`infra/gcp/worker-runtime` owns only two Cloud Run services. Its isolated backend
uses prefix `worker-runtime`. It never reads bootstrap remote state. Inputs are
`project_id`, positive canonical int64 `project_number`, fixed `region=us-central1`,
and required `runtime={image,binding_id,client_id_version,client_secret_version}`.
Image must be a worker digest in this project's repository. Secret versions are
positive canonical int64 identifiers, never aliases or values. No activation flag
exists. Runtime capacity remains one instance and one concurrent request per
service, 90-second request timeout, private IAM invocation, fixed service accounts,
exact caller subject lookups, and the existing bounded worker arguments.

Release authority is the custom permission set `run.services.create/get/update`,
`run.operations.get`, and read-only `cloudscheduler.jobs.get`. It also has repository
reader, ObjectAdmin on its own state bucket, serviceAccountUser on publisher/sync
only, and `iam.serviceAccounts.get` on scheduler/task-invoker only. It receives no
IAM policy mutation, secret access/admin, SQL ownership, service deletion, queue
publication, Scheduler creation/update/run/enable or token creation permission.
Create/update permissions are project-scoped: Terraform and reviewed release code
constrain resource names; this is not an IAM guarantee of only two service names.
A deployer can alter code running as its allowed identities, so digest review and
protected workflow trust are material security boundaries.

## Owner migration and first deployment

No agent reads state, plans, secret values or live provider data. The following is
an owner procedure, not an executed migration:

1. Freeze application/worker releases; pause and verify any existing Scheduler
   job. Resolve queued/in-flight work separately: a paused schedule does not cancel
   tasks already queued. Do not claim a provider-free maintenance window from pause
   alone. Apply foundation additions with owner authority.
2. For existing deployments, retain `worker_services_exist=true` so invocation IAM
   stays under the same bootstrap addresses. Remove obsolete `runtime` and
   `scheduler_enabled` owner variables. Review the bootstrap plan: the included
   `removed { destroy=false }` block forgets only the two service instances; it must
   not delete or replace them. Existing Scheduler address `[0]` is unchanged.
   Apply only the reviewed owner migration. Never apply both states as owners of
   the same services.
3. Initialize the new dedicated runtime backend. The owner imports the existing
   service IDs into `google_cloud_run_v2_service.worker["publisher"]` and
   `["sync"]`: `projects/PROJECT/locations/us-central1/services/jayburd-dev-acquisition-publisher`
   and `.../jayburd-dev-acquisition-sync`. Use the currently approved image and
   metadata for initial adoption. Review a no-destroy/no-replacement reconciliation
   before authorizing CI. Preserve private owner backups; do not upload state/plan.
4. For a fresh deployment instead, apply workers with `worker_services_exist=false`
   first. It creates the paused job before service invocation permission exists.
   The owner may create the two runtime services directly with the reviewed
   runtime root, then verify Scheduler remains paused; owner subsequently
   applies workers with `worker_services_exist=true`. The provider creates a job
   before issuing pause, so invocation isolation during initial creation matters.
5. Alternatively, first creation may use CI after the separate worker federation
   is enabled based on pre-deployment evidence (successful runtime creation is not
   a prerequisite). Enable federation only after ownership, SQL role/binding,
   credential-version/mapping and identity evidence is reviewed. CI checks PAUSED
   before and after applying runtime. It does not invoke workers or activate jobs.
   A separate explicit activation review is required for provider traffic.

On partial migration failure, keep releases frozen and finish import using known
resource identifiers. Never recreate services or restore dual state ownership as
an automated retry. The owner reviews any rollback and IAM drift separately.

## Reviewed evidence and limits

Reviewed 2026-10-14, primary documentation sections and pinned provider code:

- [Cloud Run permissions](https://docs.cloud.google.com/run/docs/reference/iam/permissions),
  Services table: get/update/create are separate from setIamPolicy/delete.
- [Cloud Run image deployment](https://docs.cloud.google.com/run/docs/deploying),
  Required roles: service identity actAs and repository read are separate grants.
- [Operation polling](https://docs.cloud.google.com/run/docs/reference/rest/v2/projects.locations.operations/get):
  `run.operations.get` authorizes asynchronous deployment polling.
- [Scheduler IAM](https://docs.cloud.google.com/scheduler/docs/access-control),
  role permission table: job configuration and invocation permissions differ;
  job create can also target project Pub/Sub, supporting owner-only ownership.
- [Google provider v8.6.0 Run service implementation](https://github.com/hashicorp/terraform-provider-google/blob/v8.6.0/google/services/cloudrunv2/resource_cloud_run_v2_service.go),
  Create/Read/Update: reads service resource and polls operation; this configuration
  does not require a separate revisions.get grant. No quota billing override is
  configured; additional serviceusage permissions are not assumed necessary.
- [Google provider v8.6.0 Scheduler implementation](https://github.com/hashicorp/terraform-provider-google/blob/v8.6.0/google/services/cloudscheduler/resource_cloud_scheduler_job.go),
  Create/paused handling: pause follows creation, not an atomic create-paused API.
- [HashiCorp state refactor](https://developer.hashicorp.com/terraform/language/state/refactor),
  cross-state removed/import procedure: `destroy=false` preserves infrastructure.

These facts support the local design; mocked Terraform tests validate shape,
separation, default pause, exact trust and bounds. They do not establish deployed
IAM correctness, workload identity authentication, a completed owner state
migration, credential freshness, or successful provider acquisition. Hosted
permission gaps must fail deployment and be reviewed, never trigger broad grants.

## Limited Sandbox qualification

The separately reviewed [Sandbox release contract](sandbox-release.md) adds explicit
`sandbox-promote`/`sandbox-rollback` operations for the fixed development project.
Full promote/rollback gates above remain unchanged. Sandbox releases require the
bounded database proof (#130), and connection-enabled/worker releases also require
delivery implementation (#131). Hosted activation is separately tracked in #133.
Scheduling remains paused; no production or full-readiness claim follows.
