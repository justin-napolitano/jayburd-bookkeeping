# Development migration infrastructure

This root owns the single-task, forward-only migration job.
Foundation owns identities and API enablement; application owns Cloud SQL.
The deployment workflow selects immutable images from a reviewed release, using
`PROJECT-jayburd-dev-runtime-state` with prefix `migration`. No local backend is enabled.
Use `make infra-test` for credential-free mocked validation.

Read the [rollout contract](../../../docs/contracts/cloud-run-rollout.md) and
[release instructions](../../../deploy/README.md) before an owner-controlled apply.
Do not read or publish Terraform plans/state. This source is not live-cloud evidence.
