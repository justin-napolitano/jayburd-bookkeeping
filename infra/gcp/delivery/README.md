# Development delivery canary

This owner-operated Terraform root provisions identity and empty storage without
creating a job by default. Read the [contract](../../../docs/contracts/delivery-job.md)
and [setup runbook](../../../docs/runbooks/delivery-job.md) first.

Use the owner-administered workers bootstrap state bucket with separate prefix
`delivery`, following `backend.tf.example`. Do not reuse another root's state. Backend/state files remain local
and must never be committed or shown to agents. No secret values are Terraform inputs.
Foundation must already enable IAM, IAM Credentials, STS, Run, Storage and Secret
Manager APIs. This root does not alter shared API lifecycle.

Inputs: `project_id`, `project_number`; region is fixed to `us-central1`. For the
first bootstrap leave `canary=null` and `enable_github_canary=false`.
After Infisical setup and a reviewed worker build, supply `canary` containing:

- `image`: full worker image digest from the successful reviewed image build.
- `infisical_project`, `infisical_identity`: actual UUID metadata.
- `source_version`: exact numeric canary version; owner confirmed `1`.
- `mode`: defaults to `lookup`, requires no `source_id` and installs no destination
  write permission. Review lookup ID/version output before setting `mode="execute"`
  and the returned exact `source_id`. Execution remains a separate manual action.

Then explicitly enable `enable_github_canary` to permit owner-only workflow invocation.
Terraform never executes the job. Manual workflow **Run development delivery canary**
submits it without overrides and does not claim it passed. Do not dispatch while
an earlier execution is running; do not delete intents to retry uncertain delivery.
The existing Plaid containers and their pending Terraform adoption are separate.

## Sandbox credentials and probe

`canary.key` defaults to `JAYBURD_DELIVERY_CANARY`; it also accepts only
`PLAID_CLIENT_ID` or `PLAID_SECRET`. Lookup has no destination write grant.
Execute grants version-adder only on the selected destination; existing provider
containers remain owned by their original Terraform root. Supply reviewed source
ID/version separately for each key. The resource/job name is retained to preserve
state ownership; it no longer implies canary-only arguments.

Optional `sandbox_probe={image,client_id_version,client_secret_version}` creates a
separate managed identity and bounded job reading those exact GCP numeric versions.
After apply, the owner workflow input `job=sandbox-probe` invokes that job without
argument overrides. No source-project access is granted automatically.
See [probe runbook](../../../docs/runbooks/hosted-sandbox-probe.md).
