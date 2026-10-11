# ADR 0004: Domain schemas and SQL ownership

## Decision and status

Accepted by owner for namespace/naming direction, 2026-10-11. Physical grants and
posting implementation below are proposed for JB-002 review. Keep application
objects out of `public`; schemas express responsibility, not individual LLCs.

| Schema | Responsibility | Example |
| --- | --- | --- |
| `ledger` | Entities, authorization membership, books, accounts, periods and journals | `ledger.journal_entry` |
| `banking` | Source accounts, observations and reconciliation | `banking.statement` |
| `reference` | Currency definitions and immutable rate observations | `reference.currency` |
| `audit` | Actor/action evidence | `audit.action` |
| `integration` | Idempotency receipts and outbox intent | `integration.operation_receipt` |
| `reporting` | Authorized derived reports | `reporting.trial_balance` |
| `deployment` | Bootstrap/migration metadata, inaccessible to application roles (#26) | `deployment.migration_artifact` |

Create a schema with its first implemented reviewed slice. Use lowercase
`snake_case`, singular table names, `<subject>_id` identifiers, `_date` for business
dates and `_at` for instants. Use schema-qualified references in SQL. Constraints
use `<table>_<purpose>_<pk|fk|uq|ck>`; indexes use `<table>_<purpose>_idx`.
Avoid quoted mixed-case identifiers, reserved `pg_` prefixes and redundant `tbl_`.
These names are project conventions, not ISO-prescribed field names.

## Ownership and evidence

Use shared domain tables with entity/book-scoped references. A schema per LLC
would multiply migrations and complicate cross-book reporting; separate databases
remain an option if future isolation or recovery requirements justify them.
Schemas alone do not enforce tenant security. See [SQL contract](../contracts/ledger-sql.md)
for the proposed role matrix and controlled write/read boundary.

PostgreSQL 17 [§5.10](https://www.postgresql.org/docs/17/ddl-schemas.html), reviewed
§§5.10.1–6, establishes namespace qualification, separate object/schema grants and
search-path trust. Inference: migration ownership plus restricted runtime roles
fits our single ledger authority. Reject runtime schema ownership and writable
untrusted search paths. Set grants explicitly rather than depending on database
creation/upgrade defaults. Review extension schema requirements separately.

Done when `TestSchemaPrivileges` proves no application objects in `public`, no
runtime DDL, no unchecked ledger DML and no function/search-path privilege escape.
This test is specified, not implemented. Owner/schema reviewer resolves grants
before the first product migration; revisit only for a concrete isolation need.

## Cloud SQL migration namespace — #26

The [Cloud SQL contract](../contracts/cloud-sql.md) adds deployment before product
migrations run. Owner bootstrap owns its schema; the restricted migrator owns its
artifact receipts and golang-migrate version table. This avoids public objects and
leaves merged product migrations unchanged. The tool-owned `schema_migrations`
name is an explicit exception to singular naming. `TestRestrictedMigration` and
`TestRuntimeReadiness` cover runtime denial; this is operational metadata, not an
ISO financial message or an additional ledger.
