# Development runtime infrastructure

This root owns the API/frontend services, IAM bindings and public identity configuration.
Foundation owns identities and API enablement; application owns Cloud SQL.
The deployment workflow selects immutable images from a reviewed release, using
`PROJECT-jayburd-dev-runtime-state` with prefix `runtime`. No local backend is enabled.
Use `make infra-test` for credential-free mocked validation.

Read the [rollout contract](../../../docs/contracts/cloud-run-rollout.md) and
[release instructions](../../../deploy/README.md) before an owner-controlled apply.
Do not read or publish Terraform plans/state. This source is not live-cloud evidence.
