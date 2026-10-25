# Production pilot source review

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
