<!-- stage-runbook: true -->
# Binding review recovery delivery

## Status and boundary

Issue #215 owns transparent binding readiness and provider-backed cancellation
from `binding_review`. It does not add credit/loan policy, create journals, or
approve live institutions.

## Stage ledger

1. **Issue and source review — complete.** #215 records the bounded outcome.
   Plaid Items and Item-error documentation support provider removal and exact
   already-removed recovery.
2. **Contract — complete.** The accepted design keeps binding as the primary
   path and serializes cancellation against it before provider I/O.
3. **Database and service — complete.** Migration 23 and the Go coordinator
   implement prepare/remove/finalize with replay, failure, and bind-versus-cancel
   race evidence.
4. **Browser recovery — complete.** The UI names every binding blocker and
   exposes explicit cancel confirmation for current or pending cancellation.
5. **Integrated evidence — complete.** The disposable browser/API/PostgreSQL
   proof removes one unbound Item, starts a distinct replacement enrollment,
   binds three supported accounts, and completes acquisition and posting.
6. **Hosted acceptance — pending release.** Apply migration 23 before the API
   revision, then repeat the owner-driven Sandbox flow against development.

## Current evidence

Migration fresh/replay/down/up, connection-service SQL, bind/cancel race, Go
service, web unit, production build, component-browser, and full disposable
connection journey checks pass with migration 23.

## Remaining work

Apply migration 23 before the API revision, then complete the owner-driven
Plaid Sandbox acceptance check in development. Credit and loan binding remains
outside this issue.

## PR handoff

The PR must link #215, list migration and rollback impact, cite exact validation,
and state the remaining provider-retention and hosted-acceptance limits.
