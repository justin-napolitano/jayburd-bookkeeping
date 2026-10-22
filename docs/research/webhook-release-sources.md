# Dedicated webhook release sources — 2026-10-22

Selected official sections were reviewed for #271 before infrastructure changes.
These sources establish platform mechanisms, not approval to deploy or proof of
hosted receipt/publication. No provider payload, credentials, private state or
personalized Dashboard requirement was read. Public provider docs are mutable;
Terraform provider reference below is pinned to repository Google8.6.0.

| ID / authoritative reference | Reviewed sections / fact | Project inference and acceptance |
| --- | --- | --- |
| WR01 [Cloud Run container configuration](https://docs.cloud.google.com/run/docs/configuring/services/containers) | Configure entrypoint and arguments, including Terraform command/args. Overrides select the executable independently of image default. Relevant section read in full. | Same trusted worker digest explicitly selects `/webhook`, never worker default. `isolated_route_identity_and_budget` mocked test; actual revision readback remains required. |
| WR02 [Cloud Run public invocation](https://docs.cloud.google.com/run/docs/authenticating/public) | Assign Invoker to allUsers, Terraform example and organization policy caveat. Relevant section read in full. | Explicit separate approval grants invocation on one fixed webhook service; retain IAM checks, preserve private services. `public_approval_is_one_fixed_service`; human/security review before exposure. No automatic fallback if organization policy rejects it. |
| WR03 [Cloud SQL IAM authentication](https://docs.cloud.google.com/sql/docs/postgres/iam-authentication) | Automatic versus manual authentication; user/service-account administration and required instance flag. Relevant sections read in full. | Existing metadata-only connector and IAM SQL user plus separately reviewed PostgreSQL grants. Terraform user creation does not provide ledger authority. `dedicated_identity_and_two_static_reads` and owner SQL bootstrap proof. |
| WR04 [Secret Manager access control](https://docs.cloud.google.com/secret-manager/docs/access-control) | Secret Accessor permission table and principle of least privilege. Relevant sections read in full. | Exactly two existing secret-level bindings. These cover versions within each secret; exact numeric-version selection is runtime/release validation, not a version-level IAM claim. Bootstrap mocked grant assertions. |
| WR05 [Google provider8.6.0 Cloud Run service reference](https://raw.githubusercontent.com/hashicorp/terraform-provider-google/v8.6.0/website/docs/r/cloud_run_v2_service.html.markdown) | command/args, invoker_iam_disabled, scaling, request concurrency, timeout, resources, startup probe and traffic descriptions; selected relevant fields read. Registry HTML was unavailable; official tagged source used. | Dedicated service root, fixed bounds and TCP probe; mock tests verify requested configuration only. Actual traffic/Ready/image identity and effective IAM remain separate qualification. |
| WR06 [Google provider8.6.0 SQL user reference](https://raw.githubusercontent.com/hashicorp/terraform-provider-google/v8.6.0/website/docs/r/sql_user.html.markdown) | IAM service-account example, username suffix, password/type argument descriptions. Relevant sections read in full. | Derive `.iam` SQL username from dedicated service identity; no password field. Bootstrap mock test verifies type/name/no password. |
| WR07 [Terraform provider mocking](https://developer.hashicorp.com/terraform/language/tests/mocking) | Mock provider data and override timing sections read in full. Computed mock values default to unknown during plan; `override_during=plan` provides synthetic values without real provider operations. Mutable documentation; verified with pinned Terraform1.15.7. | Both suites keep `command=plan` and explicit mock provider. Mock SA email is synthetic; no real apply is needed to assert derived IAM identities. |

Existing P88-14/15 in [Plaid source register](plaid-88-test-sources.md) govern
webhook receipt/verification; [managed runtime](../contracts/webhook-managed-runtime.md)
records the existing proprietary-notification/ISO applicability. No financial
message, currency semantics, SQL schema or accounting policy changes here. Full
ISO text was not reviewed and no conformance is claimed.

Alternative of widening worker/API roots was rejected because it couples private
IAM, caller policy and scheduler authority to a public notification boundary.
Two new roots require separate owner-created backend buckets and explicit release
orchestration, but make ownership reviewable. Mutable platform behavior, cold
start/latency, inherited IAM and eventual consistency remain hosted evidence gaps.
A source citation does not grant cloud mutation or approve residual security risk.
