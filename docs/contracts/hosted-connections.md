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
absolute local consent expiry. Project Sandbox expiry policy: future and at most30days
from startup using an explicitly supplied absolute instant, with no renewal default. The writer has exactly
eight Terraform-provisioned slots, named slot-00 through slot-07; acquisition uses
the separately composed read-only resolver.

Construction validates identifiers before I/O, builds providersecrets reader,
registry and writer, then reads only the two exact static credential versions with
a bounded startup deadline. Values stay inside memory/Secret wrappers. The existing
Plaid HTTP provider and connection service reuse SQL current permission, one-shot
claims, manifest verification, immutable credential readback and recovery semantics.
No startup Plaid calls, Infisical writes, Link session or account binding occur.
The command uses direct managed metadata transport without proxy/ADC discovery.

## Link and operational limits

The [Hosted Link enrollment profile](hosted-link-enrollment.md) creates a
Plaid-hosted session and resolves server completion before exchange. It does not
require ordinary-SDK Link-events enablement. Retained LinkEventsQualified metadata
cannot establish that Plaid enabled it. Reconnect remains unsupported; actual
interactive session/secret/SQL/webhook qualification remains open.

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
UUID audience support is owned by worker runtime; connection composition must not
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
`-plaid-client-secret-version`, `-plaid-consent-expires-at`. Existing `-project`,
`-mode managed`, and `-database-mode cloudsql` remain mandatory for enabled hosting.
No secrets are accepted by these flags. Existing URL-database behavior is unchanged
when optional connections are disabled; this composition introduces no new discovery.

Hosting metadata is fixed to the Terraform namespace: project IDs have6–30characters;
registry bucket is `{project-id}-jayburd-dev-credential-registry`, prefix
`provider-credentials-v1`; static containers are `jayburd-dev-bank-sync-client-id`
and `jayburd-dev-bank-sync-client-secret`. Swapped, foreign or arbitrary containers
fail before secret access even if other IAM grants accidentally exist. Cold restart
after the absolute local consent expiry intentionally fails until an explicit newly
approved expiry is configured; there is no startup renewal.
