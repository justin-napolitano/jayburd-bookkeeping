# Development canary delivery job

## Decision

Owner requested job/identity wiring on 2026-10-14 while PR #113 is reviewed.
Compose its adapter as a one-shot Cloud Run Job, initially restricted to the
root key `JAYBURD_DELIVERY_CANARY` and destination `jayburd-dev-bank-sync-canary`.
The command cannot select Plaid keys, custom URLs or credential values. This
canary must pass before widening job scope to the adapter's existing mappings.
It is not a financial message; ISO 20022 is not applicable.

`cmd/deliver` shares the existing digest-pinned worker image, with `/deliver` as
an explicit entrypoint override. The source is the owner-selected isolated canary project, keeping Viewer scope
away from Plaid. Missing configuration rejects before authentication;
`-check` validates metadata offline. `-lookup` authenticates for an exact-key,
exact-version request with `viewSecretValue=false`, imports and expansion disabled;
it prints only validated ID/version fields and makes no registry/destination calls.
Only `-execute` permits delivery and requires an explicitly supplied source ID.
SIGTERM cancels the bounded operation. Logs contain fixed status text only. An
acknowledgement is not independent payload verification or current authorization.

The isolated `infra/gcp/delivery` root owns the delivery service account, a canary
container, private receipt bucket, optional job and separate invocation identity.
It owns no Plaid container or workers state. Registry IAM is create/get only within
`delivery-v1/`; destination IAM is versions.add only on the canary. The task has
no SQL, source write, destination read, metadata list or secret administration role.
One task, parallelism one, retry count zero and a 60-second platform timeout bound
execution. Concurrent executions still rely on the adapter's durable intent fence.

The Infisical identity uses GCP ID Token Auth with the exact delivery service-account
email, Viewer access only to the isolated canary project, and token TTL/max TTL at most 900 seconds.
It is separate from both the native App Connection and the application writer.
No static keys, personal token or GitHub secret retrieval is needed.

## Controlled invocation

Default `canary=null` creates no executable job. A configured job defaults to lookup
mode with no destination write grant; execute mode adds only the canary version-adder
grant after an explicit metadata review. GitHub federation is disabled by
default and cannot be enabled without reviewed canary metadata. The manual workflow
is bound to repository/owner/actor IDs, main and its exact workflow_ref. Its separate
runner can get/run only the canary job; it has no overrides, job update, actAs,
secret or registry access. There is no scheduler. Workflow success means submission
only; the owner checks Cloud Run execution and redacted receipt evidence separately.

Owner-operated Terraform bootstrap requires an isolated durable backend and reviewed
image receipt. This slice adds invocation through GitHub, not an IAM-capable GitHub
bootstrap deployer. Existing application/worker release workflows remain separate.

## Sources, alternatives and evidence

Reviewed relevant official sections on 2026-10-14:

- [Google provider v8.6.0 job reference](https://raw.githubusercontent.com/hashicorp/terraform-provider-google/v8.6.0/website/docs/r/cloud_run_v2_job.html.markdown):
  task template, service account, command, task count, retries and timeout fields.
  These define configuration, not measured runtime behavior.
- [Infisical GCP ID Token Auth guide](https://infisical.com/docs/documentation/platform/identities/gcp-auth):
  machine identity creation, removal of default Universal Auth, exact allowed
  service-account emails and project membership. GCE project/zone restrictions
  do not establish Cloud Run identity scope.
- Existing [delivery source review](scoped-delivery.md#sources-and-alternatives)
  governs exact source versions and unknown acknowledgements.

An always-on service or scheduled poller adds unnecessary lifecycle/retry surface
for the first canary. Reusing API identity would combine source-write and delivery
capabilities. Neither is selected. Runtime IAM propagation and Infisical plan scope
remain uncertain until actual qualification.

Acceptance: `TestExecutionGate`, `TestExecuteRedaction`, providersecrets canary tests,
eight mocked `delivery.tftest.hcl` runs, and candidate container rejection checks.
See the [living runbook](../runbooks/delivery-job.md) for executed evidence and limits.

Metadata source review (2026-10-14): [Infisical Retrieve query parameters](https://infisical.com/docs/api-reference/endpoints/secrets/read)
document `viewSecretValue=false`, exact `version`, disabled reference expansion and
imports. Reviewed those sections, not a hosted response. Unknown response fields
are discarded and never printed; this does not claim the provider cannot return
unexpected fields. Lookup success establishes metadata/auth access only.

## Authorized Sandbox extension

The [hosted Sandbox probe](hosted-sandbox-probe.md) extends the delivery command
allowlist to the two original root Plaid keys. Explicit `key` selects the only
destination granted write access; lookup stays value-disabled. A separate optional
probe identity gets read-only access to the two destination containers. The manual
workflow selects one of two fixed jobs, without overrides or broader invocation IAM.
This supersedes the initial canary-only command restriction above.

## Fixed dynamic-slot extension (#131)

Owner authorized completion of the hosted Sandbox journey on 2026-10-15. The same
explicit command now also accepts exactly `slot-00` through `slot-07`; adapter
mapping fixes their folder to `/bank-sync` and destinations to
`jayburd-dev-bank-sync-slot-00` through `slot-07`. No folder flag, arbitrary key,
credential argument or source discovery/listing is added. `LookupSelected` reads
only metadata for an explicit key/version with values, imports and expansion
disabled; `LookupRoot` retains its root-only compatibility boundary.

Terraform retains the existing `canary` input/job name for compatibility and adds
one selected slot's `secretVersionAdder` grant in execute mode only. Workers own
slot containers; the delivery root does not create/adopt them. Lookup mode removes
all delivery destination-write grants. The job stays one task, zero retries and
60 seconds, with the distinct delivery identity and existing immutable registry.
The adapter rejects returned wrong folder metadata when present; the request
always selects the fixed folder even when the provider omits that response field.

Use a dedicated API-writer Infisical identity and an approved development project
with `/bank-sync`. Writer create/read permission and delivery read permission are
separate from static Plaid source access. On plans without folder/custom-role
support, project isolation alone does not prove create/read-only permissions;
review the actual grant before activation. Do not reuse the delivery identity for
API writes or silently widen it to Admin.

Evidence: `TestSelectedSlots`, `TestSlotMetadataLookup`, `TestDynamicSlots`, existing
unknown-acknowledgement/slot-reuse tests and twelve new mocked Terraform runs.
These are synthetic implementation evidence; actual token propagation, readback
and SQL activation remain a separate hosted acceptance. The prior Infisical and
Google source review above and [adapter sources](scoped-delivery.md#sources-and-alternatives)
apply unchanged. This internal credential protocol has no ISO 20022 mapping.
