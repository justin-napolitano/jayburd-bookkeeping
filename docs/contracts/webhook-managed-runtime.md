# Dedicated Sandbox webhook runtime

## Decision and boundary

Engineering composition for #103/#87 under owner-authorized #88 swarm, reviewed
by the coordinator 2026-10-22. Extends the existing
[verified ingress](provider-webhooks.md) and [hosted identity](hosted-workers.md)
profile to a separate executable host. No new financial service, database,
business authority or webhook type. Human security review and exact release
authorization are required before new internet exposure or IAM grants; neither
is supplied by this document. Repository Infisical/GCP instructions govern over
the security skill's generic Azure examples.

The host runs only the existing exact `/webhooks/plaid` handler. It never mounts
bookkeeping, setup, posting, debug or credential routes. It accepts only the
reviewed Sandbox Transactions notification. Item events, Hosted completion
webhooks and Production need their own explicit profiles; this host does not
pretend to implement them. No migration or enrollment runs at startup.

## Composition and capabilities

- Cloud SQL component `webhook` derives only
  `jayburd-dev-webhook@PROJECT.iam`, bookkeeping database, us-central1, existing
  metadata-only IAM connector; pool maximum2/minimum0, no raw TCP/ADC/DSN or
  API/sync/owner identity substitution. This is constructor support, not a grant.
- Operator provisions the dedicated Cloud SQL IAM user and only existing
  `bookkeeping_webhook` inherited capability, without ADMIN/SET or direct table
  privileges. The existing SQLReceiver verifies that restricted authority on
  each call before invoking `integration.accept_verified_webhook`.
- Credentials come from existing runtime-managed exact numeric secret versions,
  fixed `jayburd-dev-bank-sync-client-id` and `jayburd-dev-bank-sync-client-secret`
  resources in the explicit numeric project. No arbitrary secret path, aliases,
  environment plaintext or agent retrieval. Infisical remains the secret owner.
- Reuse existing bounded metadata token source, CRC32C reader, fixed Sandbox
  key endpoint, verifier and SQLReceiver. Key cache misses resolve credentials
  only in memory. Request-controlled key identifiers never select secret paths.
- Configuration validates before database/secret/provider access. Startup checks
  database transport only; per-call SQLReceiver inspection remains authority.
  No bank acquisition or queue call.
  Durable wakeups are consumed by the already separate publisher/worker.
- Cleanup drains requests, closes the SQL pool/connector and idle HTTP connections
  once. Cancellation or ambiguous SQL commit never acknowledges success early.

Future deployment must pin a trusted image, exact SQL/secret-version/identity
metadata and unchanged private API IAM; separately review public invocation of
this one host and enrollment webhook URL. Default local/CI commands stay offline
and use only synthetic keys and disposable databases.

The existing worker image packages `/webhook` alongside its other explicit
executables. Default entrypoint stays `/worker`; a separately reviewed webhook
service must explicitly select `/webhook`. Reuse the trusted worker-image build,
scan/SBOM and immutable candidate receipt without inventing a fifth unreviewed
image promotion channel. Offline exact-image startup proof covers invalid webhook
configuration and proves no network/default enrollment behavior.

## Standards, sources and alternatives

This is an internal operational capability and proprietary Plaid notification,
not an ISO20022 financial message. No amount/currency mapping or accounting rule
changes; existing ISO4217/source decoder profile remains authoritative. No ISO
conformance claim. Source applicability: selected current official sections in
[P88-18–P88-20](../research/plaid-88-test-sources.md); existing P88-14/15 cover
signature, body hash, age and response constraints. Full ISO text not reviewed.

Alternative: mount on the application API or borrow sync credentials. Rejected:
it widens routing/credential authority and cannot pass SQLReceiver role checks.
Alternative: remain injected-only. Safe for synthetic host proof, but cannot
receive real delivery; accepted only as an intermediate stage, not W01 clearance.

## Threat review and acceptance

| Threat / initial risk | Mitigation and required evidence |
| --- | --- |
| Spoofing/tampering/replay, high | Existing ES256/raw-body/age/key validation and durable replay receipt; negative signatures never reach SQL; W06/W07. |
| Privilege escalation, high | Dedicated derived login, SQLReceiver role inspection, no direct table authority/private routes; constructor and SQL role tests. |
| Credential/data disclosure, high | Fixed resources, no logging/reflection/persistence; synthetic redaction tests and reviewed runtime identity policy. |
| Resource exhaustion, high | Header/body/socket/context/admission bounds, small pool/key-cache budgets and Cloud Run instance cap; real-socket host tests and hosted retry/load evidence. |
| Lost/duplicate work, high | Receipt+wakeup atomicity before acknowledgement, uncertain commit retry, existing generation fences and catch-up; SQL receipt/dispatch and W02–W05. |

Residual hosted risk remains until dedicated IAM/SQL provisioning, routing,
trusted image, actual signed chain and bounded overload/recovery tests are
reviewed. Loopback or Terraform mocks do not qualify cloud identity/delivery.
Root constructor acceptance: `TestConnectorConfiguration` verifies derived
webhook identity, pool bounds and no raw fallback; invalid role aliases fail.
Host/managed acceptance must additionally prove inert constructors, fixed secret
configuration, route isolation and fixed/redacted errors without real credentials.
