# Self-service bank onboarding

## Decision

Issue #155 supersedes the per-book operator requirement in the Sandbox bootstrap
contract. A verified creator with current read/post access clicks Connect account.
The existing Begin(new) request prepares missing connection infrastructure and
persists its attempt in one serializable transaction before provider work.

## Authority

Runtime supplies pre-provisioned sync/publisher SQL login identifiers. Requests
cannot choose logins, service principals or grants. SQL verifies worker capability,
the exact setup receipt, active creator/read/post access and book/entity state.
It grants only creator import/connect and per-book service read/import, then uses
the existing audited worker/delivery enrollment functions. API receives execution
on bounded wrappers, never owner membership, direct table writes or role grants.

Existing configuration uses ordinary connection authorization. Revoked grants,
disabled principals/bindings, foreign creators, missing worker capability and
conflicts reject without reactivation. Repeated Begin keys and concurrent requests
retain the existing durable-attempt replay contract. Failed attempt creation rolls
back provisioning; provider failures leave a discoverable durable attempt.

## Discovery and consent

List and attempt discovery may return empty results for an eligible unconfigured
creator after the same checks, without writes or provider calls. Clicking Connect
account is consent to prepare this book for bank setup. Plaid consent and exact
source-account/currency/ledger binding review remain separate and explicit.
No journal, initial balance, scheduler activation or cloud resource is created.

## Sources and standards

Reviewed 2026-10-15: PostgreSQL 17 CREATE FUNCTION, “Writing SECURITY DEFINER
Functions Safely” ([S12](../research/database-sources.md#s12--restricted-posting-functions)),
and Transaction Isolation, “Serializable Isolation Level”
([manual](https://www.postgresql.org/docs/17/transaction-iso.html)). Relevant
sections reviewed in full. They establish restricted search paths, explicit
execution grants and whole-transaction retry requirements; creator eligibility
and no-reactivation are project policy, demonstrated separately by tests.

Builds on the reviewed [bootstrap](sandbox-book-bootstrap.md),
[connection SQL](connection-sql.md), and [delivery SQL](acquisition-dispatch-sql.md)
contracts. The alternative operator script leaves a product onboarding gap;
granting runtime owner membership would give unrelated authority. Scoped SQL
wrappers retain the existing security-definer and serializable transaction model.
This internal authorization operation has no applicable ISO 20022 message;
existing provider currency and account contracts retain their ISO mappings.

## Acceptance

Named PostgreSQL tests cover discovery without writes, first use, replay,
concurrency, rollback, cross-scope/disabled/revoked denial, and API privilege
boundaries. Service tests prove Begin provisions before provider work. Browser
and real application tests cover a fresh creator-owned book reaching Link.
Shared multi-book publisher dispatch is tracked in
[#156](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/156) and must
be qualified before claiming automatic acquisition across every registered book.
