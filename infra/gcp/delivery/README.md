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
- `source_id`, `source_version`: exact canary source UUID and numeric version.

Then explicitly enable `enable_github_canary` to permit owner-only workflow invocation.
Terraform never executes the job. Manual workflow **Run development delivery canary**
submits it without overrides and does not claim it passed. Do not dispatch while
an earlier execution is running; do not delete intents to retry uncertain delivery.
The existing Plaid containers and their pending Terraform adoption are separate.
