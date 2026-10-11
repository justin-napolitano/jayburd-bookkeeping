# API composition and workspace discovery

## Decision

Implementation review for #3/#6. `cmd/api` defaults to a loopback health-only
listener: `/healthz` is 200, `/readyz` is 503 and bookkeeping routes are absent.
No SDK, environment secret or database is consulted in this mode. Managed mode
is explicit; migration and identity enrollment remain separate operator actions.
Sources, alternatives and limits are in [server sources](../research/server-sources.md).

## Managed configuration

Start with `-mode=managed -listen=127.0.0.1:8080 -origin=https://books.example.invalid
-project=OWNER_SELECTED_PROJECT`. The owner injects `JAYBURD_DATABASE_URL` through
the approved secret workflow; never put a DSN on a command line, in logs or Git.
The DSN must be a PostgreSQL URL with explicit user, host and database. Only
`sslmode` may be a query parameter: default `verify-full` uses system roots;
`disable` is permitted only for a loopback database proxy. Certificate, passfile,
service and inherited PG configuration are not supported. File defaults are
explicitly overridden; no user credential file is loaded.

The Admin SDK receives an explicit project and metadata-service token source.
No ADC credential discovery is used. Emulator, metadata-host override, Firebase
config and application credential-file environment presence are rejected before
client construction. Runtime verification also rejects emulator presence.
This requires a separately provisioned GCP workload identity; no cloud resource
or IAM permission is created. Provider verification/revocation and TOTP are the
[identity contract](identity-setup.md), not health-probe assertions.

Legacy URL mode is bounded to eight connections. The [Cloud SQL mode](cloud-sql.md)
uses a five-connection IAM pool and explicit nonsecret instance flags; it rejects
ambient DSN presence. Readiness checks database
reachability, runtime-role membership, unsafe inherited roles, table authority,
DDL privileges, enrollment exclusion and migration4's discovery entrypoint.
It does not call Identity Platform or prove live-bookkeeping approval. The
container runs as numeric non-root with a static executable and system CA bundle.
TLS termination, network access and abuse rate limits remain deployment gates.

## HTTP lifecycle

The bearer-only frontend uses a same-origin `/v1/` rewrite. Configure one exact
HTTPS origin; explicit `http://localhost` or `http://127.0.0.1` origins support local
development. Origin changes grant no CORS or cookie authority. No query token,
public signup, actor header or close endpoint exists.

Limits: 32 concurrent handlers, 4KiB request URI, 32KiB headers, 1MiB request body,
5-second header timeout, 15-second read/write timeout, 30-second idle timeout and
12-second handler context. Existing auth/transaction budgets remain shorter.
SIGINT/SIGTERM removes readiness, drains at most 15 seconds, then closes the pool.
API admission, shutdown and URI-limit failures use safe RFC 9457 JSON with a
server-generated request ID. No request or provider access logging.

## Workspace contract

`GET /v1/workspace` takes no query/body and authenticates before SQL. It returns
`entities`, `currencies` and `development_policy: dev-accrual-usd-v1`. Each entity
has `entity_id`, `name`, `kind`, current `permissions` and active `books`. Each book
has `book_id`, `name`, string `book_revision`, `policies`, `periods`, `accounts`.
Policy records contain IDs, ISO 4217 functional currency and effective dates;
periods contain ID/dates/state; accounts contain ID/code/name/category/normal side,
monetary flag and activity dates. Currency versions contain ID/alphabetic code,
minor unit and validity dates. Nullable end dates remain null; collections are
always arrays. Account/reference history is configuration, not journal disclosure.

Actor-first membership indexing bounds discovery to granted scopes.
Principal/entity/membership/book share locks protect a repeatable-read snapshot.
Only active entities with active read permission appear; post alone is insufficient.
Enrolled users with no memberships receive currencies and empty entities to begin
setup. Disabled/unknown principals receive the existing safe 404; bad authentication
receives 401. Responses exceeding 100 entities, 200 total books, 200 accounts/book,
100 policies/book, 240 periods/book, 1000 currency versions or 1MiB fail with 413;
no partial list is returned. Pagination is a future contract before larger scale.

These are internal control/configuration operations, not ISO 20022 exchanges;
existing ISO 4217 version references remain authoritative. No conformance claim.

## Acceptance and limits

`TestServerBoundary`, `TestServerAdmission` and `TestManagedConfiguration` test inert defaults, saturation, safe
configuration, origins, readiness and bounds. `TestWorkspaceAuthorization`,
`TestWorkspaceRevocation` and `TestWorkspaceBounds` exercise actual runtime roles,
read/post separation, both revocation lock orders and oversize failure. Existing
posting/setup suites still apply. `TestRuntimeReadiness` uses isolated roles to
reject table, enrollment, owner/operator and schema authority.
`bash scripts/test-api-container.sh` builds the strict source allowlist and probes
the non-root/read-only health-only image over an internal Docker network; no host
ports or environment secrets are passed. `--local-build` is an explicit fallback
using the host Go compiler and identical pinned CA/scratch runtime; it does not
validate the canonical source-builder stage. Keep that stage required in CI. The [branch runbook](../runbooks/first-app.md)
owns observed execution evidence. Live metadata, IAM, provider and TLS behavior
remain unverified until a separately approved synthetic deployment.
