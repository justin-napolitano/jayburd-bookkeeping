# Production pilot source review

## PP12 — shared-instance database isolation,2026-10-29

[PostgreSQL17 database roles](https://www.postgresql.org/docs/17/database-roles.html),
role scope paragraph reviewed: roles span the cluster's databases.
[Privileges5.8](https://www.postgresql.org/docs/17/ddl-priv.html), CONNECT and PUBLIC
default sections reviewed: CONNECT is checked at startup; databases initially grant
PUBLIC connection rights. Our explicit database allowlist, separate IAM logins and
atomic bootstrap checks are engineering controls, not physical/admin isolation.
No entire PG manual, live SQL catalog or private records reviewed. Read-only GCP
metadata confirms existing PostgreSQL17/us-central1/ZONAL/db-g1-small/10GiB and
only bookkeeping/postgres databases. Shared instance selection is owner direction;
actual CONNECT grants, capacity and managed migration remain unproved.

## PP11 — public pilot pricing,2026-10-27

[Cloud SQL pricing](https://cloud.google.com/sql/pricing): selected Iowa/us-central1
shared-core, SSD and backup rows reviewed from public HTML (browser extraction
exceeded size limit). Rates: micro0.0105/hour, small0.035/hour;
SSD0.000232877/GiB-hour, backup-used0.000109589/GiB-hour. Shared CPU lacks SQL SLA.
[Cloud Run pricing](https://cloud.google.com/run/pricing): selected request-based
free tier/billing-account aggregation and active/idle sections reviewed. Allowances
are shared across projects; scale-to-zero and low usage do not guarantee zero cost.
[Plaid billing](https://plaid.com/docs/account/billing/): pricing/plans/Trial sections
reviewed; public docs do not quote this owner's actual rates.
[Plaid plan eligibility](https://support.plaid.com/hc/en-us/articles/16110502116887-What-are-Plaid-s-prices-and-pricing-plans-and-how-do-they-differ):
selected Trial eligibility restrictions reviewed; existing Production applications
can exclude Trial. No Dashboard/actual bill or private account inspected.

Arithmetic uses730hours,10GiB SSD and10GiB total backup-used allowance; capacity,
usage, region and prices need verification before provisioning. Approximately25USD
for GCP only (Plaid excluded) is owner direction; candidate sizing/alerts/Sandbox scheduling are engineering
proposals, not vendor guarantees or accepted purchases. Current source specifies
db-g1-small; gcloud sizing readback unavailable (command not installed). No actual
running size, free-tier balance or provider plan is established. PP10 budget
limitations still apply. See mandate budget table and production-budget runbook.

## PP10 — pilot isolation/recovery proposal,2026-10-26

[Cloud Run service identity](https://docs.cloud.google.com/run/docs/securing/service-identity),
identities/types/least-permission/metadata sections240–280 reviewed: deployer and
service identities differ; a user-managed minimal-permission identity is recommended.
[Cloud SQL backup options](https://docs.cloud.google.com/sql/docs/postgres/backup-recovery/backup-options),
standard backup/storage/retention and transaction-log sections244–297 reviewed:
retention/location need explicit configuration; on-demand backups can persist;
PITR log and backup windows must be coordinated. [Billing budgets](https://docs.cloud.google.com/billing/docs/how-to/budgets),
overview/caution89–110,343–347 reviewed: alerts-only budgets notify rather than
prevent spend; service-specific cap support needs separate confirmation.
Selected public sections only, not entire guides, actual cloud configuration,
price quote, legal retention rule or normative ISO. Facts support mechanisms;
the resource/retention/duration/cost choices in
[the337 proposal](../contracts/production-pilot-mandate.md) are project inference
requiring owner/security acceptance. No resources or billing account inspected.

## PP09 — signed webhook profile, 2026-10-26

[Plaid verification](https://plaid.com/docs/api/webhooks/webhook-verification/),
header/ES256/JWK retrieval+metadata/age/raw-body validation sections reviewed;
[Link SESSION_FINISHED](https://plaid.com/docs/api/link/#session_finished),
full modern field table incl environment and Hosted defaults reviewed. Different
profiles retain separate hosts/cache; raw-body verification precedes field routing.
Vendor fields do not prove Item ownership or consent; our stored-attempt association
and current SQL authority are engineering policy. [PG17 CREATE FUNCTION](https://www.postgresql.org/docs/17/sql-createfunction.html),
Writing SECURITY DEFINER Functions Safely266–292 reread for search_path and atomic
PUBLIC revoke/selective grants. Selected public sections only; no whole manuals,
Dashboard/keys/payloads or normative ISO reviewed. Applied decision:
[Production webhooks](../contracts/production-webhooks.md); actual signed delivery
and independent security review remain separate from source/injected tests.

## PP08 — credential/runtime source review, 2026-10-26

[Google KMS AAD](https://docs.cloud.google.com/kms/docs/additional-authenticated-data),
overview/When to use AAD/confused-deputy explanation reviewed: equal AAD is required
to decrypt; it authenticates context, not permissions. [Secret Manager version access](https://docs.cloud.google.com/secret-manager/docs/access-secret-version),
intro/required-role/version-ID sections reviewed: access returns a value and accepts
versions or aliases. We choose exact numeric pins and owner delivery evidence,
not mutable latest. [Plaid API overview](https://plaid.com/docs/api/), protocol,
host/storage sections refreshed: closed environments and protected credentials;
no account/Dashboard/secret access. These are selected public guide sections, not
entire API manuals, normative ISO, IAM deployment proof or cryptographic certification.
Applied engineering decision: [Production runtime](../contracts/production-runtime.md).

#308/#90; reviewed selected public official sections on2026-10-24. These are
engineering sources, not accounting-policy or Production-access approval. No
Dashboard keys, private profile/questionnaire answers or financial records read.
Contract: [proposed pilot profile](../contracts/production-pilot.md).
Refresh2026-10-25: PP01 full protocol/header/host/storage paragraphs, PP04 API
sync request/cursor/count/account filter/readiness sections and PP06 below read
before the closed source transport decision. API2020-09-14/SDK49.0.0 unchanged.

| ID | Authoritative source / selected sections | Fact established | Interpretation / limit |
| --- | --- | --- | --- |
| PP01 | [Plaid API overview](https://plaid.com/docs/api/), API protocols/headers and API host | Distinct Sandbox/Production hosts; Items cannot transfer between them; API uses HTTPS/JSON and client credentials | Create a fresh Production Item; never relabel Sandbox data. Root-bundle TLS validation remains intact. Exact app access is not established by this page |
| PP02 | [Launch checklist](https://plaid.com/docs/launch-checklist/), replacement notice, Production setup, webhook configuration and Item management | Personalized Launch Center replaces this unmaintained checklist; server/client settings and Production credentials must correspond; Item removal is an explicit lifecycle action | Discovery/reference only, not a current exhaustive gate list. #274 owns actual app-specific requirements; no new request follows from the already observed approval header |
| PP03 | [OAuth guide](https://plaid.com/docs/link/oauth/), introduction, prerequisites, platform reinitialization table, Hosted Link and testing | Financial-institution flows require OAuth support; access/registration depends on actual app/plan; Hosted Link has no Link reinitialization requirement; test supported client platforms and update mode | Keep Hosted Link scope; do not add a custom SDK redirect merely from a generic example. No browser token persistence. Desktop-only pilot/platform exclusions need explicit review, not a viewport substitution |
| PP04 | [Transactions introduction](https://plaid.com/docs/transactions/), history and integration overview | History request differs from supplied coverage; first sync activates notifications and may be empty; pagination/mutation restarts and saved cursors are required | Preserve current atomic acquisition/readiness protocol and independent equality; no90-day completeness claim from readiness alone. Investment history is a different product |
| PP05 | [Hosted Link](https://plaid.com/docs/link/hosted-link/), overview and Obtaining a public token | Retain user-associated link token; completion uses SESSION_FINISHED or link/token/get; modern results are recommended and lookup is available for a bounded period | Existing owned attempt/completion association still needs #187/#275 evidence; hosted frontend/redirect does not grant backend authority. Reconnect's exact update-result shape needs further selected API review |
| PP06 | [Transactions states](https://plaid.com/docs/transactions/transactions-data/), Pending and posted transactions including added/removed linkage and posted modifications, reviewed2026-10-25 | Pending and settled details may differ; settled transactions can change; sync added/modified/removed updates preserve current provider state | Keep source revisions and explicit accounting corrections. Vendor feed mechanics do not select a book's recognition/category/FX rule or justify blanket ledger prohibition; owner approved the bookkeeping direction2026-10-25. Exact policy/current authority and independent replay/correction evidence remain project obligations |

Review depth: PP01 selected complete protocol/host sections; PP02 selected notice
and operational sections, explicitly historical; PP03 selected prerequisite/table/
Hosted/testing passages; PP04 selected history/integration steps; PP05 overview
and completion token passages. Videos, whole API reference, current personalized Dashboard and paid ISO
normative text were not reviewed here. API-Version stays2020-09-14 unless a
separate compatibility decision changes it. Existing pinned SDK remains unchanged.

## Code evidence at main74dd4b1

| Boundary | Reviewed source | Confirmed limit |
| --- | --- | --- |
| Enrollment | internal/connections/http_provider.go: NewHTTPProvider and boundedPlaidTransport | SDK host and request allowlist require Sandbox |
| Acquisition | internal/acquisition/plaid_http.go: Fetch; sql.go: validContext | Sync URL and accepted execution context require Sandbox |
| Credential identity | internal/bankcredential/envelope.go: Binding.aad | Environment is authenticated in AAD and validated Sandbox-only; ciphertext cannot be relabeled safely |
| Source/Item SQL | migrations12/13 and current replacements22/25/26/28 | Source namespace and Item constraints, enrollment/binding functions and worker contexts encode Sandbox and synthetic provenance |
| Credential SQL | migration18 integration.bank_credential and addressed functions | Environment check and read/write binding are Sandbox-only |
| Webhook | internal/webhook/key_http.go, verifier.go, sql.go; migration14 | Key-fetch host, event/environment validation and SQL receipt scope require Sandbox |
| Runtime/release | internal/connectionruntime/runtime.go; deploy/README.md; infra/gcp/runtime, workers and webhook roots | Current configuration/resources are development-scoped; #281 only repairs optional expiry, not Production composition |
| Diagnostics | internal/sandboxqualification/hosted.go; finite verifier contracts | Sandbox endpoints/fixtures and zero-journal proof are intentionally bounded; they must not gain Production authority |

These are reviewed examples, not a claim that grep exhausts every SQL replacement,
worker/evidence/grant path. Each implementation child must inventory the latest
owned functions and callers before changing them, with negative cross-environment
tests. Merged SQL remains immutable; no migration number reserved.

## PP07 — Production SQL authority source refresh2026-10-25

[PostgreSQL17 CREATE FUNCTION](https://www.postgresql.org/docs/17/sql-createfunction.html),
full Writing SECURITY DEFINER Functions Safely section266–292 reviewed: safe
search_path, PUBLIC EXECUTE revocation and selective grant in one transaction.
[Constraints5.5.1](https://www.postgresql.org/docs/17/ddl-constraints.html), selected
NULL and cross-table CHECK limitations paragraphs92–101 reviewed: NOT NULL must be
explicit and cross-table runtime authority is not an immutable CHECK expression.
Plaid PP01 full host/storage sections refreshed: Items do not move environments,
tokens and consumer data require protected handling. Mechanism facts support the
book-scoped proposal in [Production source SQL](../contracts/production-source-sql.md);
the specific capability/trigger/role design is our inference, not vendor policy.
No whole PG chapter, Dashboard, ISO normative or accounting policy text reviewed.

## Source baseline refresh at main1bacecda

Rechecked2026-10-25: connection HTTPConfig/SDK/boundedPlaidTransport still use
Sandbox host; Hosted repair is a separate false-by-default capability and its
receiver is the fixed Sandbox URL. Acquisition.NewPlaidHTTP and Fetch still fix
Sandbox, runner supplies AccountID/cursor/credentials but not environment metadata;
SQL validContext still requires sandbox. #312 may add explicit profile construction
and trusted fetch metadata without changing SQL or runtime. Binding/AAD/routing
and current roles remain separately gated311/313/314. Closed source profiles do
not authorize Production ingress, secrets, resource creation or real-account use.
