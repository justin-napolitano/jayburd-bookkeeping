# Explicit connection configuration in releases

## Decision

Runtime releases use schemaVersion2 and require `connections`: null explicitly
disables enrollment; a complete metadata object enables the existing Sandbox-only
composition. Omission and legacy runtime schemaVersion1 are rejected before cloud
authentication. Migration/verification releases retain schemaVersion1 and cannot
carry connection settings. This closes the silent-default reset from PR110.

The object uses the existing Terraform names: infisical_origin, infisical_project,
infisical_identity, link_events_qualified, client_id_version, client_secret_version,
consent_expires_at. `link_events_qualified` remains a required boolean, but may
be false for the [Hosted Link profile](hosted-link-enrollment.md). It describes
ordinary-SDK account enablement, not overall Sandbox qualification. No credentials, destination overrides or extra fields. Validate
exact approved origins/nonzero UUIDs, canonical same-project numeric resources and
positive int64 project/version IDs. Absolute consent expiry must remain future, at most30
days away, and later than release authorization expiry. Owner review binds the
exact file hash; existing closure/build/migration gates remain unchanged.

The CLI passes this object unchanged to runtime.tfvars.json. The transport receipt
records a canonical digest of the approved configuration, not evidence of provider
readiness or observed runtime settings. Rollback requires the prior successful
receipt's same configuration digest, including expiry; changed or expired connection
policy needs a separately reviewed forward promotion. Legacy receipts without that
digest cannot establish configuration-preserving rollback.

## Evidence and alternatives

Reviewed selected HashiCorp input-variable assignment/default/precedence sections
and Google Cloud Run revision management on2026-10-14:
[Terraform variables](https://developer.hashicorp.com/terraform/language/values/variables),
[Cloud Run revisions](https://docs.cloud.google.com/run/docs/managing/revisions).
Variables omitted from the generated file fall back to configuration defaults;
Cloud Run revisions capture configuration changes. Requiring an explicit state is
our release-policy inference, not a vendor accounting or security guarantee.

An optional field would retain the omission hazard. Reading live configuration to
silently preserve it would weaken exact-hash review and introduce another authority.
We choose an explicit versioned manifest. Infrastructure/internal configuration is
not an ISO20022 financial message; existing transaction/currency mappings stand.

## Acceptance and limits

Named release tests cover enabled/null propagation, omission/legacy rejection,
partial/cross-project/alias/overflow metadata, expiry, unchanged migration isolation,
configuration-bound rollback and deterministic receipt digest. Default tests use
fictional metadata and no provider/cloud calls. This does not create a release,
provision Infisical, implement Link/reconnect, deploy workers or qualify Sandbox.
#27/#11 retain delivery/configuration acceptance; #88 retains provider qualification.

## First Sandbox writer configuration

The first hosted qualification selects project
`ec11c550-d143-4902-be67-e33266753539` and writer identity
`56d2a541-a48d-4bcc-ae0c-11f48c945ba2`, origin `https://app.infisical.com`.
The [isolated role exception](provider-vault.md#approved-isolated-sandbox-role-exception)
is owner-approved; the manifest evidence must link the configuration confirmation.
Use the existing Hosted Link profile (`link_events_qualified=false`); do not infer
ordinary-SDK qualification from saved identity settings. A complete reviewed
manifest still requires the public Firebase browser identifier, exact build/image
receipts and numeric static GSM pins. No placeholder identifier may be deployed.


## Encrypted bank-token runtime transition (#162–#164)

The new API/consumer image pair selects encrypted PostgreSQL storage, with fixed
KMS resource `projects/{projectNumber}/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials`.
This name is frozen in source and matches workers Terraform; it is not chosen by
an HTTP request, ambient environment or mutable alias. Existing schemaVersion2
Infisical/mapping identifiers remain validated compatibility metadata, but the new
dynamic-token runtime never reads them. Static client versions remain authoritative.

Apply workers KMS resources/grants, forward migration 18, and matching API/consumer
images before owner enrollment. Keep scheduling paused. After encrypted credentials
are activated, legacy-image rollback is incompatible; use a compatible encrypted
image or reviewed forward repair. Deployment acceptance must enforce this build
compatibility, not rely on the unchanged configuration digest. Hosted KMS/IAM and
owner acceptance remain #164; synthetic tests do not close that gate.
