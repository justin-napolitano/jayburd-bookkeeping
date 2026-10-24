# Optional hosted connection composition

## Approved bounded composition

The API keeps connection enrollment disabled unless explicit complete metadata
configuration is supplied. `plaid-link-events-qualified` remains ordinary-SDK
account metadata; false is supported by the Sandbox Hosted Link profile.
Enabled composition requires managed mode and the existing Cloud SQL IAM API pool;
no second pool, alternate DSN, provider credential argument or environment lookup
is introduced. Disabled composition reads no metadata token or secret and mounts
no optional connection routes. Partial or contradictory settings fail startup.

Configuration identifies GCP project ID plus its explicitly configured canonical numeric project number, approved Infisical origin/project UUID/dev
namespace/identity UUID, GCS metadata registry bucket/prefix, two distinct exact
numeric Secret Manager versions in projects/{number} resources for static Plaid client credentials, and a fixed
absolute local consent expiry. Live Sandbox approval is at most30days from startup
using an explicitly supplied instant, with no renewal default. Well-formed expired
optional enrollment starts SQL-only bookkeeping/attempt inspection without any
dependency access; malformed or unsafe configuration still rejects. This is the
accepted281 availability boundary, not a consent extension.

Current encrypted composition validates identifiers before I/O, constructs the
fixed canonical-project-ID KMS wrapper and PostgreSQL bank store, then reads only
the two exact static credential versions with a bounded startup deadline. Registry/
Infisical identifiers remain validated compatibility metadata; the API no longer
uses finite per-account secret slots, registry discovery or a dynamic secret writer.
Acquisition uses its separately scoped encrypted read-only resolver. Values stay
inside memory/Secret wrappers. The existing
Plaid HTTP provider and connection service reuse SQL current permission, one-shot
claims, manifest verification, immutable credential readback and recovery semantics.
No startup Plaid calls, Infisical writes, Link session or account binding occur.
The command uses direct managed metadata transport without proxy/ADC discovery.

## Link and operational limits

The [Hosted Link enrollment profile](hosted-link-enrollment.md) creates a
Plaid-hosted session and resolves server completion before exchange. It does not
require ordinary-SDK Link-events enablement. Retained LinkEventsQualified metadata
cannot establish that Plaid enabled it. Owned Hosted repair requires the separate
default-false `HostedRepair` capability and exact fixed Sandbox receiver, schema33
and compatible binary/bootstrap. Existing webhook-only configuration stays repair-
disabled. Operator flag `-plaid-hosted-repair=true` is private deployment metadata,
never an HTTP/browser-selected permission. Current SQL still owns actor, Item,
source, consent, proof and claim authority. See the [owned repair protocol](hosted-reconnect.md)
and [exact release profile](connection-release.md). Actual interactive session/
secret/SQL/webhook qualification remains open; a configured flag is not acceptance.

Existing12s API budget is unchanged. Unknown exchange/write/propagation outcomes
follow durable service recovery; a slow propagation cannot authorize a new exchange
or blind secret mutation. Existing health/readiness is DB/auth readiness, not a
claim that Plaid, Infisical delivery or all qualification steps passed.

## Sources and standards

This composition adds no financial message and retains existing proprietary Plaid
2020-09-14 and ISO mappings in connection/provider contracts. Reviewed official
[Infisical GCP Auth](https://infisical.com/docs/documentation/platform/identities/gcp-auth)
GCP ID Token Auth, sample identity request and login sections: audience is the
Infisical identity UUID, not an invented HTTPS URI. Configure the writer identity's
short token TTL to satisfy the providersecrets adapter's bounded policy. Metadata
UUID audience support is owned by worker runtime; the retained legacy writer
source review does not imply current encrypted API writer use. Composition must not
silently substitute a different audience. Source review does not qualify deployed
IAM or secret propagation. The coordinator links the annotated source in shared
bibliography; dependency secret and worker contracts own their wire details.

## Callable boundary and command metadata

`connectionruntime.New(ctx, Config, connections.Beginner, http.RoundTripper, Tokens)`
returns the existing connection Service or a fixed error. It returns nil/nil when
fully disabled. `Config.Validate` and `ManagedScope` reject partial metadata,
non-managed/API-URL mode, unmatched hosting project, mutable version aliases and
cross-number credential resources before startup secret access. Construction has a
15s total static-read budget; source adapters retain their stricter request bounds.
Project ID-to-number association is explicitly operator/Terraform configured, not
inferred or proven by string validation. Exact canonical numeric response names and
CRC32C are checked by GCPReader; no ID/number alias equality is assumed.

Command flags (identifiers and policy only): `-connections-enabled`,
`-plaid-link-events-qualified`, `-provider-project-number`, `-infisical-origin`,
`-infisical-project`, `-infisical-environment`, `-infisical-identity`,
`-provider-registry-bucket`, `-provider-registry-prefix`, `-plaid-client-id-version`,
`-plaid-client-secret-version`, `-plaid-consent-expires-at`, optional fixed
`-plaid-webhook-url` and separate default-false `-plaid-hosted-repair`. Existing `-project`,
`-mode managed`, and `-database-mode cloudsql` remain mandatory for enabled hosting.
No secrets are accepted by these flags. Existing URL-database behavior is unchanged
when optional connections are disabled; this composition introduces no new discovery.

Hosting metadata is fixed to the Terraform namespace: project IDs have6–30characters;
registry bucket is `{project-id}-jayburd-dev-credential-registry`, prefix
`provider-credentials-v1`; static containers are `jayburd-dev-bank-sync-client-id`
and `jayburd-dev-bank-sync-client-secret`. Swapped, foreign or arbitrary containers
fail before secret access even if other IAM grants accidentally exist. Cold restart
after well-formed optional approval expiry preserves SQL-only bookkeeping/attempt
inspection and performs no dependency I/O. Provider entries stay unavailable before
SQL/secret/KMS/provider work; warm outgoing transport also rechecks the same exact
deadline. Any new provider approval is separately explicit; there is no grace or
startup renewal. Disabled composition remains entirely inert.
