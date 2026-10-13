# GCP delivery source register

## Decision and review boundary

Owner accepted managed Cloud Run, Cloud SQL and Cloud Tasks for the synthetic
milestone. Google Cloud publishes all sources below; accessed 2026-10-11.
These are mutable provider manuals/pricing pages, not accounting standards.
Selected sections were reviewed, not entire documentation sets. Recheck at
implementation; acceptance belongs to the linked issues in [GCP delivery](../gcp-delivery.md).

| ID / source | Reviewed sections and established facts | Application and limits |
| --- | --- | --- |
| G01 — [Cloud Run pricing](https://cloud.google.com/run/pricing) | Request-based pricing and billable instance time; request work/startup/shutdown are billed, configured minimum instances incur idle charges; Jobs have a one-minute minimum per instance. | Choose zero minimums and bounded HTTP publisher calls. Actual cold-start/scheduler/load costs require measurement. |
| G02 — [Cloud Tasks pricing](https://cloud.google.com/tasks/pricing) | Billable operations and billing notes: first million operations/month/account free, then $0.40/million at the displayed tier; API calls and delivery attempts count in 32 KB chunks. | Transaction count is not operation count. Retries, payload size, shared allowance and network costs affect spend. |
| G03 — [Cloud Tasks issues and limitations](https://docs.cloud.google.com/tasks/docs/common-pitfalls) | Execution order, duplicate execution, backoff and target resource limits. Ordering is not guaranteed; duplicates occur; overloaded targets cause backlog and throttling. | Require durable deduplication, fault tests and bounded queue/pool concurrency in #7. Provider delivery does not prove business completion. |
| G04 — [Connect from Cloud Run](https://docs.cloud.google.com/sql/docs/postgres/connect-run) | Go connector example and connection-pooling guidance. Connector integrates with pgx; public/private connectivity and pool management are explicit choices. | #26 validates encrypted connector access and connection budget. Examples using passwords do not authorize secret retrieval or supersede IAM direction. |
| G05 — [Cloud SQL IAM authentication](https://docs.cloud.google.com/sql/docs/postgres/iam-authentication) | IAM authentication concepts, automatic authentication and performance on shared-core instances. Connector-managed authentication handles short-lived access tokens; database permissions remain separately managed. | Choose automatic IAM authentication; test SQL role grants and managed-service compatibility. Shared-core IAM login can slow or time out under CPU pressure; test bounded connection recovery. No production access has been exercised. |

## Alternatives and uncertainty

Self-hosted Kubernetes/broker adds operational responsibilities; managed request
billing is our preferred fit for intermittent development, not a demonstrated
universal cost win. Pub/Sub fan-out is deferred until independently owned consumers
exist. Neither choice replaces the atomic PostgreSQL ledger.

The under-$50 target excludes bank-provider subscriptions and is not a guaranteed
ceiling. A complete regional estimate remains required by #5, including Cloud SQL
compute/storage/backups, identity, network, scheduler, logs, artifacts and restore
resources. The initial browser extraction of SQL pricing failed; the foundation follow-up
review below retrieved official HTML and records the bounded rate check. Measure
seven-day spend in #12.

## Foundation implementation review — 2026-10-11

- **G06 — HashiCorp, [Terraform provider mocking](https://developer.hashicorp.com/terraform/language/tests/mocking)**: reviewed Mock Providers and Generated Data. Mocks use provider schemas without resource API calls; passing mocks do not prove real IAM or provider behavior. Implemented six named foundation tests with Terraform 1.15.7/provider 8.6.0. Selected sections only.
- **G07 — Google, [WIF for deployment pipelines](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines)**: reviewed GitHub attribute mapping/conditions and default audience. Numeric repository/owner IDs avoid name-reuse trust; mapped claims constrain federation. Our additional owner actor/event/workflow restrictions are a project decision. Actual token denial tests remain cloud evidence.
- **G08 — HashiCorp, [GCS backend](https://developer.hashicorp.com/terraform/language/backend/gcs)**: reviewed bucket prerequisite, locking, versioning and authentication guidance. Bootstrap must create its bucket first; owner then migrates local state privately. Separate application bucket narrows deployer access. No remote state was read or initialized.
- **G09 — Google, [Cloud Scheduler pricing](https://cloud.google.com/scheduler/pricing)**: reviewed Pricing overview/Free usage. Billing is per configured job, not per execution; first three jobs/account are free, then $0.10/job/31 days. Invocation compute is separate; account-wide usage remains unknown.

The checked-in lock file records signed provider package hashes for macOS ARM64
and Linux AMD64. Release identity was checked against the upstream Google provider
v8.6.0 release. Private branch-protection API still returned HTTP 403; the repository
environments API returned an empty list. The [foundation contract](../gcp-foundation.md)
therefore does not grant GitHub apply access or assume an enforced reviewer gate.

- **G10 — Google, [Cloud SQL pricing](https://cloud.google.com/sql/pricing?hl=en)**: reviewed Shared-core CPU/RAM and Iowa storage/backup tables in official HTML after the browser tool rejected its size. db-g1-small $0.035/hour; SSD $0.000232877/GiB-hour; used backups $0.000109589/GiB-hour. Shared-core types are excluded from the Cloud SQL SLA. The 730-hour example in the foundation contract is our calculation, not a full-stack quote or a billing-account-specific offer. No authenticated billing data was retrieved.

## Cloud SQL implementation review — 2026-10-11

- **G11 — Google, [Cloud SQL Go connector v1.25.3](https://github.com/GoogleCloudPlatform/cloud-sql-go-connector/tree/v1.25.3)**: reviewed README pgx/IAM sections and installed options.go token-source, lazy-refresh and public-IP options. Connector provides authenticated TLS; pgx's inner TLS can be disabled for the tunnel. Explicit API/login metadata token sources avoid the library's default ADC discovery. Library requires Go 1.26; the repository pins 1.26.9. Unit checks prove configured behavior, not live TLS/IAM acceptance.
- **G12 — Google, [PostgreSQL users and roles](https://docs.cloud.google.com/sql/docs/postgres/users)**: reviewed superuser restrictions, default users, secure search path and IAM users/database roles. Cloud SQL does not expose PostgreSQL superuser; IAM users do not automatically receive database roles. Our precreated NOLOGIN roles and restricted migrator are a project design. Disposable PostgreSQL proves ordinary role semantics; actual administrator ADMIN-option compatibility remains cloud evidence.
- **G13 — golang-migrate, [pgx v5 driver at v4.20.1](https://github.com/golang-migrate/migrate/blob/v4.20.1/database/pgx/v5/pgx.go)**: reviewed installed Config/WithInstance, advisory locking, version/dirty handling and Close. The library closes its supplied SQL handle and owns a reserved connection. We use a second connection for artifact-registration locking, register hashes before migration and reject altered registered files. Source io/fs embeds reviewed SQL; no arbitrary remote SQL or down/force operation is exposed.

Required future evidence: real project API enablement, connector-only transport,
IAM login and SQL privilege separation, shared-core reconnection latency, backup/PITR
restore and observed costs. Terraform mocks and local non-superuser tests are not
substitutes for these checks. No live credential/state/plan was retrieved.

## G14 — Next.js standalone output

Vercel, [output](https://nextjs.org/docs/app/api-reference/config/next-config-js/output), documentation identifies 16.4.0; accessed 2026-10-11. Reviewed “Automatically Copying Traced Files” and caveats: standalone emits a minimal server and traced dependencies; static assets require separate copying, PORT/HOSTNAME configure listening. Inference: a read-only non-root image can package the existing app without development dependencies. Container proof remains required; this source does not solve runtime browser identity configuration.

## G15 — Build metadata, scanning and publication

Docker, [buildx build metadata](https://docs.docker.com/reference/cli/docker/buildx/build/#metadata-file) and [attestations](https://docs.docker.com/build/ci/github-actions/attestations/), accessed 2026-10-11. Reviewed metadata-file, BUILDX_METADATA_PROVENANCE and exporter limitations. Local load can retain a separate provenance record but not attached attestations. Aqua, [Trivy action](https://github.com/aquasecurity/trivy-action/tree/ed142fd0673e97e23eac54620cfb913e5ce36c25), v0.36.0, reviewed setup/version, image inputs, severity/exit-code and CycloneDX output; scanner pinned v0.75.0. Inference: test/scan a local image before pushing unchanged, retain unsigned metadata/SBOM and explicitly defer attested promotion. Scan findings depend on the fetched vulnerability database; a clean scan is not a security certification.

## G16 — Build identity and private service invocation

Google, [auth action](https://github.com/google-github-actions/auth/tree/7c6bc770dae815cd3e89ee6cdf493a5fab2cc093), v3, reviewed WIF, token lifetime and create_credentials_file/export_environment_variables inputs; and [Cloud Run service authentication](https://docs.cloud.google.com/run/docs/authenticating/service-to-service), reviewed audience and X-Serverless-Authorization sections, accessed 2026-10-11. Auth can deliver a token without generating an ADC file; Cloud Run can validate a service token separately from application Authorization. Inference: authenticate the builder after image checks; a future frontend bridge must supply its own service identity without replacing the user token. No real federation/token exchange was executed. Proposed ingress still needs implementation and denial tests.

## G17 — User ADC quota project during foundation bootstrap

HashiCorp/Google provider [v8.6.0 configuration reference](https://github.com/hashicorp/terraform-provider-google/blob/v8.6.0/website/docs/guides/provider_reference.html.markdown), “Quota Management Configuration”; Google [Set the quota project](https://docs.cloud.google.com/docs/quotas/set-quota-project), request/environment precedence and permissions, accessed 2026-10-12. Reviewed those sections. The provider documents `USER_PROJECT_OVERRIDE=true` with `GOOGLE_BILLING_PROJECT`; the caller needs Service Usage Consumer authority. Applied these non-secret command settings after the budget API rejected user ADC without an explicit quota project; the subsequent foundation apply succeeded. No credential contents were inspected or changed.

## G18 — Runtime configuration and service-token forwarding

Vercel, [Environment Variables](https://nextjs.org/docs/app/guides/environment-variables), “Bundling Environment Variables for the Browser” and “Runtime Environment Variables”; [Route Handlers](https://nextjs.org/docs/app/api-reference/file-conventions/route), HTTP methods and request/response handling; Next.js 16.4.0, accessed 2026-10-12. Reviewed those sections. NEXT_PUBLIC values freeze at build time; request-time server code can expose an explicitly selected public configuration. Google, [Cloud Run service authentication](https://docs.cloud.google.com/run/docs/authenticating/service-to-service), “Use the metadata server” and separate authorization-header sections, reviewed again 2026-10-12. A fixed metadata identity request accepts the service URL as audience; X-Serverless-Authorization allows separate application Authorization. Inference: use a dynamic public allowlist endpoint and bounded frontend bridge, rather than a static rewrite or public API. No new dependency is needed for the documented metadata HTTP request. No token caching in this first slice; load impact and actual IAM/MFA remain cloud acceptance gates. Custom auth/API domains and public API ingress are deferred alternatives, not implied supported configurations.

## Runtime rollout sources — 2026-10-12

- **G19 — Google Cloud, Execute jobs**, [gcloud execution](https://docs.cloud.google.com/run/docs/execute/jobs#execute_jobs). Reviewed execution and roles sections: configured jobs can execute with `--wait`; job configuration is separate from execution. We infer separate migration state plus wait is a clear deployment gate. No live execution tested.
- **G20 — Google Cloud, Rollbacks and traffic migration**, [Terraform traffic](https://docs.cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration). Reviewed Terraform revision/traffic examples; Terraform can own traffic and image rollback. It does not guarantee atomic changes across services or SQL compatibility. [Pinned provider 8.6.0 service source](https://github.com/hashicorp/terraform-provider-google/blob/v8.6.0/google/services/cloudrunv2/resource_cloud_run_v2_service.go), reviewed scaling/audience schema; local schema validation is required.
- **G21 — Google Cloud, Ingress and URLs**, [internal-service access](https://docs.cloud.google.com/run/docs/securing/ingress#access_internal_services), [custom/default URLs](https://docs.cloud.google.com/run/docs/custom-urls). Reviewed ingress routing and default URL sections: Cloud Run calls to internal-only services require suitable VPC routing; deterministic URLs use service/project number/region. We select IAM-restricted network-reachable API plus explicit deterministic token audience, without VPC/NAT. Live reachability/audience acceptance remains unproved.
- **G22 — Google Cloud IAM, Firebase Authentication permissions**, [roles and permissions](https://docs.cloud.google.com/iam/docs/roles-permissions/firebaseauth). Reviewed user-get and viewer/admin lists. Custom API role selects only `firebaseauth.users.get` for revocation lookup instead of broad user administration; actual SDK acceptance is a cloud gate.

- **G23 — Deployment token lifetime and transport**, [pinned auth action access-token inputs](https://github.com/google-github-actions/auth/blob/7c6bc770dae815cd3e89ee6cdf493a5fab2cc093/README.md), [Terraform 1.15.7 GCS backend source](https://github.com/hashicorp/terraform/blob/v1.15.7/internal/backend/remote-state/gcs/backend.go), [gcloud access-token authentication](https://docs.cloud.google.com/sdk/docs/authenticate#use_an_access_token). Reviewed token lifetime/outputs, backend environment defaults and CLI environment transport. A one-hour masked service-account access token covers the bounded job without relying on a short-lived GitHub subject token to refresh credentials after migrations. No credential file or token command is used. Actual exchange remains a live gate.

## G24 — IAM administrator bootstrap research — 2026-10-12

Google, [Add and manage IAM users](https://docs.cloud.google.com/sql/docs/postgres/add-manage-iam-users),
“Assign database roles while adding an IAM account” and role assignment/revocation;
[PostgreSQL users](https://docs.cloud.google.com/sql/docs/postgres/users), default
users and superuser restrictions; HashiCorp/Google,
[sql_user at provider 8.6.0](https://github.com/hashicorp/terraform-provider-google/blob/v8.6.0/website/docs/r/sql_user.html.markdown),
`database_roles` and IAM type arguments. Reviewed selected sections only.

Google documents assigning `cloudsqlsuperuser` to a separate IAM user, including
a Terraform example. IAM users otherwise receive no application privileges by
default. The pinned provider documents `database_roles` as write-only and replacing
existing memberships. Default `postgres` has explicit CREATEROLE/CREATEDB attributes;
membership alone does not establish every authority our bootstrap needs.

Proposal: evaluate a separate owner IAM operator instead of introducing a database
password. Do not apply that role to API or migrator. Alternative: owner-injected
administrator credentials, with additional secret-handling work. Neither approach
is implemented by this provisioning slice. Acceptance requires installed provider
schema checks, administrator ownership/role-authority probes and actual bootstrap
success followed by restricted-login denial checks. #26 owns the decision and
live compatibility evidence; source examples alone do not close that gate.

G10 pricing was rechecked on 2026-10-12 through indexed official regional tables;
the full page exceeded the browser response limit. This is a bounded price check,
not a new full-page review. The [provisioning estimate](../runbooks/cloud-sql-bootstrap.md#provisioning-estimate)
records assumptions separately from actual spend.

## G25 — Owner-operated IAM proxy — 2026-10-12

Google, [Cloud SQL Auth Proxy v2.26.0 README](https://github.com/GoogleCloudPlatform/cloud-sql-proxy/blob/v2.26.0/README.md),
installation, credentials, Unix sockets and automatic IAM authentication;
[proxy connection guide](https://docs.cloud.google.com/sql/docs/postgres/connect-auth-proxy),
Unix socket length and troubleshooting outbound port 3307. Reviewed those sections
and installed proxy help. The proxy consumes ADC internally and supports automatic
IAM login; it does not establish network connectivity or SQL authorization.
Selected source version matches the official connection guide; GitHub's latest
release endpoint returned a v1 tag and was not used as v2 version authority.

Inference: a private 0700 Unix directory confines local access during the operator
check without exposing a TCP listener. Keep this exception outside runtime's
metadata-only connector. A failed TCP 3307 check is evidence of local reachability
failure, not proof of a particular firewall rule. Resume on a permitted network or
review a GCP runner; do not weaken Cloud SQL transport policy. Provider example
commands that print access tokens are not adopted.

## G26 — Cloud Shell command access — 2026-10-12

Google, [Launch Cloud Shell](https://docs.cloud.google.com/shell/docs/launching-cloud-shell),
console launch and local CLI connection sections; installed `gcloud cloud-shell ssh`
help and SDK `command_lib/cloud_shell/util.py`, SSH argument/setup code. Reviewed
these sections, not the entire SDK. Google documents remote command/file access;
the CLI's `--authorize-session` explicitly transfers authorization and was omitted.
The installed custom key-path flag was incorrectly declared boolean; default SDK
SSH handling worked without reading or overwriting keys. Existing Cloud Shell
authorization sufficed for the pinned proxy's IAM login. This is observed owner
administration, not a new runtime authentication path or migration identity.

## G27 — Immutable job execution receipt — 2026-10-12

Google, [Cloud Run v1 Execution](https://docs.cloud.google.com/run/docs/reference/rest/v1/namespaces.executions),
resource and ExecutionStatus; [ExecutionSpec](https://docs.cloud.google.com/run/docs/reference/rest/v1/ExecutionSpec),
task count and template; installed `gcloud run jobs execute` help, `--wait`.
Reviewed selected schema/flag sections. Execution captures immutable configuration;
Completed True and succeeded/failed/cancelled/running/retried counts describe its
result. Waiting without exact configuration readback would not bind evidence to
our approved image and identity. We therefore validate selected execution fields
before emitting a bounded receipt or changing services. Actual provider field
serialization remains a live gate; synthetic fixtures prove rejection logic only.

## G28 — Node runtime image contents — 2026-10-12

Node Docker maintainers, [Best Practices](https://github.com/nodejs/docker-node/blob/main/docs/BestPractices.md),
“CMD” and “Smaller images without npm/yarn” sections. Reviewed these sections,
not the whole repository. They describe direct Node execution and retaining build
tools only in the builder. Our standalone runtime starts `node server.js`;
removing npm/Yarn avoids unnecessary runtime dependencies. Alternative: upgrade
bundled npm, retaining unused code and an unfixed scan finding.
Private Trivy 0.75.0 evidence located all Node findings under npm; no application
package was waived. OpenSSL fixed-version bounds come from the scan's Alpine
package findings. Repaired-image scan and isolated runtime proof passed. Alpine's
package-manager web documentation was unavailable; actual build/scan evidence
verifies the update, not a claim of full advisory review. The base stays
digest-pinned; APK resolution can advance patched packages, so only the final
tested/scanned image digest is eligible for release.

## G29 — Execution field serialization — 2026-10-12

Google, [TaskSpec](https://docs.cloud.google.com/run/docs/reference/rest/v1/TaskSpec),
JSON representation and timeoutSeconds (string/int64), maxRetries (integer);
[resource keys](https://docs.cloud.google.com/sdk/gcloud/reference/topic/resource-keys),
array paths; [projections](https://docs.cloud.google.com/sdk/gcloud/reference/topic/projections),
subset selection. Reviewed these sections and installed SDK execution describe/
wrapper code, not the whole CLI. Actual allowlisted readback proves explicit `[]`
preserves containers and conditions whereas the original nested paths omitted them.
Accept only the exact bounded timeout string or number. Alternative broad numeric
coercion or whole-resource export would weaken validation/data minimization.
Synthetic regression checks cover the selected command paths and representation;
corrected validator also passed against the already completed real execution.
No migration replay was needed to repair evidence; the hosted run remains failed.

## G30 — Read-only migration metadata snapshot — 2026-10-12

PostgreSQL 17, [SET TRANSACTION](https://www.postgresql.org/docs/17/sql-set-transaction.html),
read-only restrictions; [Transaction Isolation](https://www.postgresql.org/docs/17/transaction-iso.html),
repeatable-read snapshot semantics. Reviewed those sections. Read-only mode rejects
persistent DML/DDL/grants; it does not mean PostgreSQL performs no physical writes.
Repeatable read gives both metadata queries a consistent committed snapshot, not
a guarantee that the database cannot change after verification. Apply with the
existing Cloud SQL identity/job sources G27/G29: no added operator grant or new
provider authentication path. A forced-RLS synthetic fixture checks the actual
transaction mode; wrong/dirty versions and artifact set/hash mismatches fail.
