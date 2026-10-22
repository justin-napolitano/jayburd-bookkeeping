# Plaid qualification source register — 2026-10-22

## Scope and review depth

Official public documentation, selected sections below read on 2026-10-22.
These are mutable web pages, not a versioned certification checklist. Current
adapter API version is `2020-09-14`; that does not version these guides.
The personalized Dashboard Launch Center was not inspected. Its review is a
separate outstanding acceptance row. No credentials or provider payloads were read.

| ID / authoritative source | Sections reviewed / established facts | Matrix use |
| --- | --- | --- |
| P88-01 [Launch checklist](https://plaid.com/docs/launch-checklist/) | Opening replacement notice; Production, Link, storage/logging, Item management and Transactions recommendations. Launch Center supersedes this unmaintained checklist. | G01–G04, D01–D02 |
| P88-02 [Sandbox overview](https://plaid.com/docs/sandbox/) | Bypassing Link; simulating events; live testing; differences. Automated build gates should bypass provider UI; Sandbox cannot establish institution-specific live behavior. | L01, applicability, G05 |
| P88-03 [OAuth guide](https://plaid.com/docs/link/oauth/) | Support, prerequisites, Testing OAuth, consent revocation. Test every supported client, app installed/absent and update mode; simulated consent is not institution-specific evidence. | O01–O06 |
| P88-04 [Hosted Link](https://plaid.com/docs/link/hosted-link/) | Overview/Who should use it; creation; obtaining public token. SDK is recommended for owned frontends. Session results have a six-hour retrieval window; completion webhook is recommended. | L02–L04, integration choice |
| P88-05 [Update mode](https://plaid.com/docs/link/update-mode/) | When to use; login repair; new permissions/accounts; expiration; Testing update mode. Existing Items need a repair/consent flow. | I01–I05 |
| P88-06 [Duplicate Items](https://plaid.com/docs/link/duplicate-items/) | How duplicates arise; callback or Hosted Link metadata; pre-exchange comparison. Institution alone is not a sufficient universal duplicate rule. | L05 |
| P88-07 [Invalid Link token](https://plaid.com/docs/link/handle-invalid-link-token/) | Invalid/expired token handling and fresh initialization; SDK iframe cleanup. | L04 |
| P88-08 [Sandbox test credentials](https://plaid.com/docs/sandbox/test-credentials/) | Transactions, MFA, reCAPTCHA and error lists; special users need non-OAuth institutions. Product-specific unrelated test users excluded explicitly. | E01–E05 |
| P88-09 [Transactions overview](https://plaid.com/docs/transactions/) | Integration overview; Sandbox testing; Refresh/Recurring add-ons. Custom additions and persona users support distinct tests; an application sync request is not a provider Refresh call. | T01–T03, T09–T10 |
| P88-10 [Transactions API](https://plaid.com/docs/api/products/transactions/) | `/transactions/sync`: initialization, `account_id`, cursor, update status, history request fields. Separate filtered streams require separate cursors; an empty result need not mean history complete. | T01–T03, T07–T08 |
| P88-11 [Transaction states](https://plaid.com/docs/transactions/transactions-data/) | Pending/posted and testing sections. Removal and replacement can span pages; posted source data can later change. Dynamic student-loan refresh does not advance its repayment schedule. | T04–T06 |
| P88-12 [Transactions errors](https://plaid.com/docs/errors/transactions/) | Pagination mutation and Item limit sections. Mutation recovery restarts the whole update from its original cursor. | T07, R01 |
| P88-13 [Transactions webhooks](https://plaid.com/docs/transactions/webhooks/) | Configuration, activation, update notifications and Refresh. First sync activates sync notifications; readiness flags distinguish initial/history availability. | W01–W02 |
| P88-14 [Webhooks API](https://plaid.com/docs/api/webhooks/) | Retries, best practices and Sandbox testing. Duplicate/out-of-order delivery and missed-webhook recovery must be handled; receiver must respond within ten seconds. | W03–W05 |
| P88-15 [Webhook verification](https://plaid.com/docs/api/webhooks/webhook-verification/) | ES256/kid, key metadata, signature, issued-at age and exact-body SHA256. Plaid describes verification as optional; Jayburd requires it under its existing contract. | W06–W07 |
| P88-16 [Transactions setup](https://plaid.com/docs/transactions/add-to-app/) | Item creation, access-token exchange and fetch/update workflow. Initial history request is bounded by provider availability. | T01, G03 |
| P88-17 [Link troubleshooting](https://plaid.com/docs/link/troubleshooting/) | Link errors, missing institutions/accounts, OAuth configuration and consent/use-case configuration. | E03–E05, G01 |

## Interpretation and limits

The [matrix](../plans/plaid-88-test-matrix.md) is our acceptance mapping for
Transactions-only US web bookkeeping, including loan/card transaction feeds.
Binding a loan does not implement Plaid's separate Liabilities product. Public
examples are possible scenarios, not evidence that Jayburd passed them.

We choose named rows with separate synthetic and hosted results over a single
"Sandbox passed" checkbox: the latter hides missing lifecycle and security
evidence. New matrix identifiers are planned acceptance IDs, not invented Go
test names. Existing tests are cited only where their source was inspected.
Missing evidence remains open even where implementation exists.

The claimed scope is the reviewed sections, not every Plaid product/platform,
unseen Dashboard requirement, full API schema or paid ISO text. Recheck sources
before execution and add newly applicable scenarios. Launch Center changes must
be mapped, not silently ignored. No financial message, SQL, API or accounting
policy changes here: existing proprietary-Plaid/ISO applicability in
[qualification](../contracts/plaid-sandbox-qualification.md) remains authoritative.

Unavailable guessed paths `/transactions/testing/` and
`/sandbox/test-institutions/` were not used as evidence. Official navigation led
to Transactions overview and `/sandbox/institutions/`; the latter was inspected
only for navigation, not used to qualify any institution.
