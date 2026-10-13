# Provider ingestion and worker contract review

<!-- stage-runbook: true -->

## Status and boundary

Design-only follow-up to merged #69 (`2428096`): #70 specifies Plaid acquisition;
#63 specifies generic outbox/worker delivery. Both proposals require review before
runtime, SQL or infrastructure implementation. No provider call, credential access
or live-account connection is included.

## Stage ledger

| Stage | Evidence / result | Next action |
| --- | --- | --- |
| Dispatch | #69 merged; #70/#63 open with no native blockers or competing PRs at dispatch; bounded leases recorded in #65 | Two isolated agents research contracts |
| Existing authority | Migrations 000001/000002/000005 and command-recovery contract reviewed | Preserve immutable events and accounting receipts |
| Independent review | Read-only reviewer identified account-filtered Plaid cursors and missing import/connect permissions | Drafters assess scoped stream choice and future grants |
| Worker integration | Agent commit `288bc73` integrated; intermediate-step receipt is explicitly distinct from terminal completion | Validate alongside provider contract |
| Provider integration | Agent commit `9f8b0d6` integrated; account-filtered cursors, generation-independent source identity and Item-wide credential lifecycle reviewed | Review proposed profile before implementation |

## Current evidence

Repository inspection establishes these integration constraints:

- `integration.outbox_message` contains immutable `journal.posted` facts. Delivery
  status, attempts and publisher leases require distinct proposed state.
- `integration.operation_receipt` records posting/reversal equality, not arbitrary
  provider jobs. `ledger.interactive_command` only supports explicit setup/post
  recovery with one unacknowledged command per principal; it is not a job queue.
- Provider observations, cursor progress and wakeup generations need their own
  consistency boundary. A provider update cannot silently rewrite a journal or
  masquerade as a new ledger book revision.
- Item-wide acquisition must not let one account binding authorize another LLC's
  data. Current connection/account authorization and revocation need explicit
  behavior during acquisition, publication and replay.
- The current private API bridge cannot serve external Plaid callbacks. A narrow
  verified public ingress needs its own proposed permission/deployment boundary;
  this does not authorize opening the bookkeeping API to anonymous requests.

The existing #7 first consumer remains deliberately small: reload and validate the
committed journal event, then record durable completion without another posting.
Its existing acceptance identifiers remain the implementation handoff. Provider
acquisition uses the same reliability principles but is a distinct operation.

Integration review will check notification arrival during sync, full pagination
restart, crash/ack races, stale fences, original amount/currency preservation,
cross-source overlap and the difference between source correction and accounting
correction. Named contract tests are planned evidence, not passing runtime tests.

## Remaining work

Both agents stopped at commit handoff. Shared source/index links are integrated.
Review the proposed non-ISO adapter mapping and scoped connection profile, then
specify physical SQL/API/permission contracts and implement synthetic fault tests.
Shared CSV adapter identity remains an explicit contract-alignment gate. No current
SQL connect/import/sync grant is implied. Reporting implementation remains separate.

## PR handoff

`make check test build` PASS on the integrated branch: documentation links,
migration checksums/file faults, runbook structure, Go vet/race tests and builds.
These are existing repository checks; the new provider/worker fault matrices are
requirements for subsequent implementation, not executed runtime proofs.

No SQL migration or runtime rollback is needed
for this documentation-only slice. Keep #70/#63 open until their acceptance is
reviewed; do not equate a merge with provider or cloud acceptance.
