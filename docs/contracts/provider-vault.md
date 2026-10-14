# Provider credential storage and propagation

## Reviewed decision

Keep Infisical as source and GCP Secret Manager as the sole acquisition runtime
reader. The coordinator approved a separately scoped connection writer and an
**eight-slot, never-reused pool**, preserving Terraform ownership of destination
containers. Source keys slot-00 through slot-07 in /bank-sync map to precreated
jayburd-dev-bank-sync-slot-00 through slot-07 containers. Only environment dev
and provider sandbox are accepted. Static client-id/client-secret use separate
precreated containers and explicitly configured numeric version resources.

SQL credential_version is logical, not a source or Google ordinal. The private GCS
registry bucket is `${project}-jayburd-dev-credential-registry`, prefix
`provider-credentials-v1`. Conditional creation (`ifGenerationMatch=0`) records
immutable slot intent before source POST; intent binds logical reference/version,
Item/environment, configured namespace and envelope SHA-256. Registry objects
contain no token values. The writer has object create/read, never update/delete;
read-only acquisition gets only object read. Slot capacity is explicit and bounded;
this is not an unlimited enrollment or scale claim.

A committed fresh intent permits one source create only after the precreated Google
container is confirmed empty. Existing or uncertain intent never authorizes another
POST, including when source GET returns 404. Slots may remain unusable after an
interrupted attempt; recovery never recycles them or guesses that an external
operation failed. A separate reviewed pool expansion/recovery contract is required.

The writer can recover a source acknowledgement through exact-key GET with imports
and reference expansion disabled. It verifies namespace and original envelope
digest before considering propagation. This source lookup is control-plane evidence,
not an acquisition direct-read fallback. Up to 16 numeric Google versions are
listed and independently accessed; all must match the expected envelope and CRC32C.
Only an observed numeric resource is pinned, selecting the lowest matching numeric
version deterministically. Source version, Google version and SQL logical version
are independent. Any nonmatching/unavailable version or larger history fails closed.

The immutable mapping records reference, source ID/version, exact Google resource
and envelope digest. Its conditional publication acknowledges an existing identical
mapping after a lost response, but never replaces a conflict. SQL remains
credential_pending until Store.Read verifies the Google payload. Acquisition reads
only the mapping and pinned Google/static versions, with no latest alias or source
fallback. Missing propagation is PARTIAL, not permission to repeat exchange/rotation.

## Concrete interfaces and bounds

`NewGCSRegistry(GCSConfig{Bucket,Prefix,Tokens,Transport})`,
`NewGCPReader(GCPConfig{ProjectID,Tokens,Transport})`, and
`NewResolver(reader,registry,ResolverConfig{ClientIDVersion,ClientSecretVersion})`
provide acquisition's CredentialResolver. `NewStore(StoreConfig,registry,reader)`
provides connections.SecretStore and includes fixed Infisical origin/project/identity,
IdentityTokenSource, transport and the eight Slot definitions. Constructors perform
no I/O or environment/ADC discovery. Managed identity dependencies supply OAuth and
audience-bound ID tokens explicitly.

Only the three selected Infisical Cloud origins and Google's fixed HTTPS hosts are
permitted. Calls have three-second limits, one application attempt, no redirects and
128 KiB response limits; Store operations have eight-second aggregate budgets.
JSON rejects duplicate/case-alias keys, excessive nesting and trailing values.
Google payloads require exact response name and CRC32C. Secrets exist in memory only
outside approved stores; Go strings are not claimed zeroizable. Errors and Preflight
contain fixed codes only, never vendor diagnostics. PASS means one exact runtime
reference was read, not provider functionality, IAM qualification or bookkeeping
approval. Preflight performs no source mutation or provider request.

API service identity may perform this writer capability only through SQL-authorized
connection operations. Its Infisical identity is exact-service-account scoped,
create/read-only for the selected dev folder/keys, TTL at most 900 seconds. It has
no Google secret write permission. The separate delivery identity may add
versions only to precreated destinations; deletion is disabled, key schema is fixed,
and delivery IAM must be qualified without project-wide secret administrator permission.
The [owned delivery adapter](scoped-delivery.md) supersedes the native-sync assumption;
its hosted composition and canary qualification remain pending.
Direct Infisical acquisition reads and dynamic destination creation are not selected.

## Primary source review

Reviewed selected public sections on 2026-10-14; no account configuration was read.

- [Infisical create secret API](https://infisical.com/docs/api-reference/endpoints/secrets/create):
  v4 create returns a secret ID and version. Project, environment and folder are
  explicit request fields. This is not evidence that our account grants create-only
  permissions or that an unknown request did not succeed.
- [Infisical retrieve API](https://infisical.com/docs/api-reference/endpoints/secrets/read):
  exact source version can be requested; imports and reference expansion are
  independently switchable. Source versions do not identify Google versions.
- [Infisical GCP authentication](https://infisical.com/docs/documentation/platform/identities/gcp-auth):
  ID-token login supports Cloud Run with exact allowed service-account emails. The
  documented metadata token audience is the Infisical identity ID. GCE project/zone
  restrictions are not a substitute for a Cloud Run service-account restriction.
  Remove default Universal Auth and bound token TTL under the reviewed deployment.
- [Infisical GCP sync](https://infisical.com/docs/integrations/secret-syncs/gcp-secret-manager):
  scoped key schema, optional automatic sync and deletion disabling are documented.
  Selected documentation does not promise a source-version/destination-version
  bijection or atomic read-after-write propagation. Sync-job success alone is
  insufficient credential publication proof.
- [Google access secret version](https://docs.cloud.google.com/secret-manager/docs/reference/rest/v1/projects.secrets.versions/access)
  and [integrity](https://docs.cloud.google.com/secret-manager/docs/data-integrity):
  read an explicitly addressed resource; check response identity and CRC32C before
  parsing payload. Transport success alone does not prove reference/Item ownership.

These vendor facts support an explicit private version map rather than an ordinal
assumption. Direct Infisical acquisition reads would change the existing availability
and IAM model and are not selected. Dynamic container creation also needs a reviewed
exception to Terraform-owned containers; this document does not grant that exception.

## Applicability and limits

The storage envelope is an internal security protocol, not an ISO 20022 financial
message. It carries no balances, transaction money or accounting dates. Existing
ISO4217 and accounting requirements remain on provider transaction publication.
No deployed IAM, Infisical plan capabilities, synchronization freshness or Sandbox
credentials have been verified. Preflight reports PASS/PARTIAL/BLOCKED with fixed
reason codes only; values, authorization headers and vendor error bodies are never
report output.

## Proposed acceptance evidence

Implemented synthetic tests cover exact numeric resource addressing, mismatched
response name, CRC32C corruption, cross-project mapping, logical/physical version
inequality, wrong Item/environment, missing mapping, duplicate JSON keys, oversized
or redirect responses, unknown write acknowledgement, immutable mapping conflict,
static provider credentials by resource name, and redacted errors/preflight output.
The runbook records executed evidence; these tests do not establish hosted access.

Google configuration accepts an exact project ID or numeric project number. Because
live responses may use a canonical project representation, deployment must select
and qualify that representation (Terraform can supply the project number). The
reader does not infer alias equivalence or relax exact response-name checking.
