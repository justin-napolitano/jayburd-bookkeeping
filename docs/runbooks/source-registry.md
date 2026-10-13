# Source registry delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #81, child #8/#55; lease SRC81-20261013T184500Z. Synthetic registry only,
no uploads, posting, HTTP, provider, cloud or secrets. Exact contract is
[source-registry](../contracts/source-registry.md).

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Review | Existing source/standards/database contracts and fresh issue81 read; exact provisioning proposal sent to coordinator before DDL. |
| Implementation | Coordinator approved owner-only enrollment plus scoped runtime reads before migration9. Four registry/receipt tables; immutable identity, explicit read grants, CAS lifecycle and bounded namespaces. No old migration changed. |
| Review | Independent agent found numeric JSON overflow needed safe mapping; fixed and covered. Root requested dormant-principal revocation; both disabled/member-revoked fixtures pass. |
| Verification | `make check test build` and `bash scripts/test-source-registry.sh` PASS, 2026-10-13. |

## Current evidence

Actual PostgreSQL17 proof covers enrollment/replay and concurrent retries, malformed
JSON (including huge exponents), USD/JPY/KWD metadata, current permissions/scopes,
revocation races, dormant grants, lifecycle/namespace limits, immutable evidence,
role denial and late audit-write rollback. Registry operations leave journal/outbox
counts and book revision unchanged. Logs: `/private/tmp/jayburd-source-registry-proof.log`
and `/private/tmp/jayburd-source-registry-default.log`. These are synthetic local
SQL proofs, not a deployed operator or real ownership proof.

## Remaining work

Coordinator runs integrated migration, ledger and restricted Cloud SQL gates
with version9 expectations; lane default and focused SQL proofs passed. Coordinator owns
shared version expectations/Makefile/CI. Import batches, provider namespace profiles,
live account ownership, retention and user-facing operator enrollment remain gated.

## PR handoff

Lane owns only migration9/checksums, internal/source, focused harness and these
two documents. Staged runbook/whitespace guard required before commit. Coordinator
integrates and publishes; no lane push/deployment.
