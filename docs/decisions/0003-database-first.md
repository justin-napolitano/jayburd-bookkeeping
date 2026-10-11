# ADR 0003: SQL-first contracts and golang-migrate

## Decision

Accepted migration tool: `golang-migrate/migrate` v4.20.1. Use reviewed PostgreSQL
SQL files, with paired six-digit sequential `up.sql` and `down.sql` filenames.
The disposable tooling proof uses PostgreSQL 17.11; cloud-version selection stays
in JB-005. Image indexes are pinned by digest for amd64/arm64 reproducibility.

No ORM is needed. Recommend `pgx/v5` for application connections and `sqlc` to
generate typed Go methods from reviewed SQL when queries exist. Those application
dependencies are not installed yet. The migrate CLI's PostgreSQL driver is an
implementation detail, independent of the future application driver.

## Why

An ORM maps tables to objects and can provide relationships, change tracking and
query construction. It does not establish balanced journals, entity isolation,
posting/close locking or idempotency. Explicit SQL makes these boundaries easier
to review. sqlc removes repetitive parameter/scanning code without owning schema
changes. Generated row types are persistence types, not HTTP or domain contracts.

## Contract order

Review a small accounting workflow and invariants, then its logical relationships,
physical columns/constraints, transaction protocol and role matrix. Implement the
corresponding migration and PostgreSQL contract tests before application features.
Iterate per slice; do not predesign every future provider or report.

Use [the database contract](../contracts/database.md) as the design home. SQL is
the executable schema authority. Mermaid is a review aid; do not introduce DBML
or ORM models as competing schema authorities. OpenAPI later owns the HTTP boundary,
not internal table definitions.

## Limits

Migration manifests validate the submitted source bundle; they do not prove a
running database has that bundle. Before cloud deployment, implement a durable
release/checksum receipt and verify prior applied contents. `schema_migrations`
alone stores version/dirty state. No automatic `force`, drop, or production down
operation. Ordinary migration files use explicit transactions; nontransactional
DDL needs a separately reviewed migration and failure-recovery procedure.

## References

- [golang-migrate format](https://github.com/golang-migrate/migrate/blob/v4.20.1/MIGRATIONS.md)
- [Failure and locking](https://github.com/golang-migrate/migrate/blob/v4.20.1/FAQ.md)
- [sqlc](https://docs.sqlc.dev/)
