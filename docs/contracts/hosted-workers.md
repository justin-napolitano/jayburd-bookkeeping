# Hosted acquisition worker composition

## Reviewed boundary

The coordinator approved separate publisher and sync service identities, Cloud SQL
IAM authentication and a maximum of two pooled connections per process. One worker
mode exposes one fixed private acquisition route. The existing dispatch SQL remains
authority for enrolled logins, current consent, leases, retry budgets and completion.
No provider data is fetched at startup; no migration or enrollment runs in a worker.

`Config` contains identifiers only: mode, instance connection name, explicit TCP
listen address, pinned caller audience/email/subject, and publisher-only binding
and queue identifiers. Publisher binds `/internal/acquisition/publish`; consumer
binds `/internal/acquisition/consume`. Existing Google signature verification and
exact workload policy run before body or SQL access. Cloud Run IAM is an additional
boundary, not a substitute for the application verification. Hosted header/OIDC
behavior must be qualified before activation.

`New(config, Dependencies)` composes injected SQL, verifier, queue or Runner through
the existing authenticated HTTP adapters and bounded worker host. `Managed` builds
actual Cloud SQL, Google-key verification, Cloud Tasks REST and Sandbox acquisition
clients. Consumer construction requires an explicit credential resolver and durable
evidence store. No local temporary evidence fallback, environment token, credential
file or ADC discovery is allowed. Missing adapters prevent construction.

## Managed identity and lifecycle

Cloud SQL login names are fixed `jayburd-dev-publisher@PROJECT.iam` and
`jayburd-dev-sync@PROJECT.iam`; database name remains bookkeeping and region
us-central1. Publisher and sync use the existing IAM connector, lazy refresh and
explicit metadata token sources. Pool maximum is two, minimum zero; no raw TCP
fallback, owner/API role substitution or worker-provided DSN is accepted. The
coordinator owns IAM SQL user creation and restricted SQL role/binding provisioning.

Metadata OAuth and ID-token access use a fixed Google metadata endpoint, required
Google metadata header, five-second request limit, bounded body and no redirects.
Tokens remain in memory and never appear in error text. Explicit injected transports
support synthetic tests. Production HTTP transport disables ambient proxies and
bounds connections per host at two. This is direct metadata credential access, not
ADC discovery. Constructors themselves retrieve no tokens.

The existing worker host sets header/read/write/idle bounds, private route admission
and graceful shutdown. Managed startup has a bounded database connection check;
it does not assert a book mandate is enrolled or a provider qualified. Dependency
cleanup is idempotent and follows request drain. Cancellation remains cooperative.
No hosted activation, account connection or real provider request is part of tests.

## Source review and decisions

Reviewed selected official sections on 2026-10-14, starting with OT01–OT06 and the
GCP source register already indexed in the bibliography:

- **HW01:** Google Cloud [service identity](https://docs.cloud.google.com/run/docs/securing/service-identity),
  service/deployer identities, metadata access-token request and token types.
  Metadata returns the configured service identity; ID tokens and OAuth access
  tokens have different purposes. Google advises against setting a credential-file
  environment variable for Cloud Run service identity. Our fixed endpoint, bounds
  and rejection of ambient overrides are local application policy.
- **HW02:** Cloud SQL [IAM login](https://docs.cloud.google.com/sql/docs/postgres/iam-logins)
  and [IAM users](https://docs.cloud.google.com/sql/docs/postgres/add-manage-iam-users),
  automatic connector authentication and service-account SQL username suffix.
  Authentication does not create SQL privileges; separate owner bootstrap remains
  necessary. The two-connection cap is the project's bounded initial profile.
- **HW03:** Cloud Run [container contract](https://docs.cloud.google.com/run/docs/container-contract),
  listener binding, metadata server and shutdown sections. The container uses an
  explicit public bind argument; standalone configuration defaults to loopback.
  Local host tests prove shutdown mechanics, not Cloud Run delivery behavior.

Pinned local `golang.org/x/oauth2/google` ComputeTokenSource and Cloud SQL connector
remain the existing metadata-only transport implementation. Reusing these and the
reviewed worker host avoids a second SQL connector or invented OIDC verifier.

This configuration is an internal control-plane contract, not a financial ISO 20022
message. Existing standards applicability and source-data currency profiles remain
unchanged. Hosted qualification, secret-version mappings and durable raw evidence
acceptance are explicit additional gates, recorded in the delivery runbook.

## Executable deployment profile

`cmd/worker` accepts only explicit publisher/consumer mode. Both require the Cloud
SQL instance and caller audience/email/numeric subject. Production CLI pins queue
`jayburd-dev-acquisition` in us-central1, publisher caller `jayburd-dev-scheduler`,
consumer caller/task invoker `jayburd-dev-task-invoker`, all in the instance project.
Consumer additionally requires `-project-number` and exact numeric GSM resources
for `jayburd-dev-bank-sync-client-id` and `jayburd-dev-bank-sync-client-secret`.
The project number is separately supplied by reviewed deployment configuration;
no ID/number alias fallback is performed. Registry bucket is PROJECT-jayburd-dev-credential-registry,
prefix provider-credentials-v1; evidence bucket is PROJECT-jayburd-dev-acquisition-evidence.
These constructors compose the concrete providersecrets resolver and cloudevidence
store. No plaintext secret flag or environment read exists.

Metadata IdentityToken also accepts canonical nonzero Infisical machine identity
UUIDs. **HW04:** [Infisical GCP auth](https://infisical.com/docs/documentation/platform/identities/gcp-auth),
ID-token guide/sample metadata request reviewed 2026-10-14: audience is the machine
identity ID, not an HTTPS service URL. Tests reject malformed UUIDs/newlines before
network access. This correction preserves the fixed Google metadata destination.
