<!-- stage-runbook: true -->
# Shared acquisition publisher delivery

## Status and boundary

Issue #156 on `feat/shared-acquisition-publisher`, following merged #229.
Sandbox scheduling only; no automatic journals or live
bank activation.

## Stage ledger

1. **Issue and source review — complete.** #156 is open and claimed; #155 is
   closed. Plaid webhook and cursor behavior were reviewed.
2. **SQL/runtime/release contract — complete.** The shared-v1 keyset cursor,
   authorization, fairness, fixed-v1 rollback and hosted evidence are defined.
3. **Implementation — complete.** Migration 27 adds login-scoped durable
   selection; shared-v1 runs a bounded rotation through the existing SQL relay.
   The worker release policy selects shared-v1 or the binding-v1 rollback mode.
4. **Synthetic validation — complete locally.** Migration fresh/replay/roundtrip,
   restricted SQL dispatch, repository check/test/build, release, and Terraform
   infrastructure suites pass. CI remains to run.
5. **Hosted qualification — pending.** Migration first, deploy shared-v1,
   verify Scheduler/IAM, then prove an enrolled second Sandbox book progresses.

## Current evidence

`make migration-test` passed, including migration 27 fresh/replay/roundtrip.
`scripts/test-sync-dispatch.sh` passed in a disposable PostgreSQL container,
including five-plus-one rotation across distinct service principals, login
scope and disabled exclusion. Worker
release/receipt Node tests, `make check`, `make test`, `make build`,
`make release-test` and `make infra-test` passed. Local Go 1.26.5 uses a temporary modfile
because pinned Go 1.26.9 download returned HTTP 403; CI remains authoritative.
The worker release instructions and contract now describe both explicit
publisher policies, including the shared empty-binding requirement.
PR #230 CI found two Go files missed the separate `gofmt -l cmd internal`
gate; they were formatted and that gate was rerun before merge.
No hosted scheduler acceptance is yet claimed.

## Remaining work

Validate and review the bounded shared publisher. Activate the
Sandbox scheduler only after migration/runtime order and exact cloud IAM review.

## PR handoff

Link #156, validation, migration 27 impact, fixed-binding rollback and the
remaining hosted qualification limits.
