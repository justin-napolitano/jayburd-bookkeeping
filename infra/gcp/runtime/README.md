# Development runtime infrastructure

This root owns the API/frontend services, IAM bindings and public identity configuration.
Foundation owns identities and API enablement; application owns Cloud SQL.
The deployment workflow selects immutable images from a reviewed release, using
`PROJECT-jayburd-dev-runtime-state` with prefix `runtime`. No local backend is enabled.
Use `make infra-test` for credential-free mocked validation.

Read the [rollout contract](../../../docs/contracts/cloud-run-rollout.md) and
[release instructions](../../../deploy/README.md) before an owner-controlled apply.
Do not read or publish Terraform plans/state. This source is not live-cloud evidence.

Optional `connections` metadata is null by default: no enrollment flags are added.
An owner-reviewed object may specify `infisical_origin`, `infisical_project`,
`infisical_identity`, `link_events_qualified = true`, `client_id_version`,
`client_secret_version`, and an absolute `consent_expires_at`. The static versions
must be numeric immutable resources in this project number's exact
`jayburd-dev-bank-sync-client-id` and `jayburd-dev-bank-sync-client-secret`
containers. The environment, registry prefix and project-derived bucket are fixed.
The expiry must be future and at most 30 days from the plan; API startup rechecks
this ceiling and deliberately fails cold starts after expiry. Do not automatically
roll the timestamp forward. Link qualification is an explicit prerequisite, not a
claim established by this Terraform setting.

This metadata adds no IAM, provider calls or secret values. The worker foundation's
secret containers, registry, identity permissions and Infisical propagation must be
qualified separately. The current release manifest/workflow does **not** expose
this object and cannot enable connections; changing that delivery path requires
separate owner review. No apply was performed for this implementation.
