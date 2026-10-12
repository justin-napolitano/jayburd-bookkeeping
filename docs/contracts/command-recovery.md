# Recoverable bookkeeping commands

## Decision and scope

Implementation decision for #9, owner project maintainer, 2026-10-11. Store one
immutable reviewed setup/post command before execution. One unacknowledged command
per principal prevents a lost result from silently becoming a new interactive
request. This deliberately serializes one user's interactive workflow across tabs;
it does not limit entity/account counts. No queue, worker or automatic execution.
Sources and alternatives: [recovery research](../research/recovery-sources.md).

## State and authority

A command is `prepared`, `committed` or `cancelled`. Registration persists exact
validated JSON and its principal, operation and target; it never posts a journal.
Execution invokes existing SQL setup/post functions and marks the command committed
in the same serializable transaction. The existing receipt, audit and outbox
invariants remain authoritative. Failed execution rolls back, leaving prepared.
Only explicit user confirmation executes; discovery never executes or acknowledges.

All transitions lock the active principal first, then follow existing scope/book
lock order and command locking consistently. Execution/cancellation serialize;
cancellation cannot report success after posting committed. Acknowledgement is
explicit and terminal-only; it releases the one-unacknowledged-command gate but
never deletes immutable request/result history. Runtime has restricted functions,
no table access. Same registration key with changed content conflicts.

Discovery/results reauthorize the current principal and current read/post scope.
Another principal cannot discover, execute, cancel or acknowledge a command.
Revoked scopes fail closed without returning request/result financial data. An
unacknowledged inaccessible command blocks new work pending operator restoration;
this availability limitation is explicit, not a reason to bypass authorization.
Command storage has the existing 1MiB request bound; it is not a general draft API.
The gate applies to this interactive workflow. Existing direct setup/post API
clients retain their original idempotency obligations; this is not semantic
duplicate detection across independently submitted business transactions.

## HTTP contract

- `POST /v1/commands`, required `Idempotency-Key`: register
  `{operation: setup|post, entity_id: uuid|null, book_id: uuid|null, body: object}`.
  Setup requires null targets; post requires both. Return 201 `Command`.
- `GET /v1/commands/pending`: return `{command: Command|null}`; no query/body.
- `POST /v1/commands/{command_id}/execute`, `/cancel`, `/acknowledge`: body `{}`,
  return 200 `Command`. Command identity supplies retry identity; no extra key
  authorizes a changed operation. Execution returns the stored result on replay.

`Command` fields: `command_id`, `operation`, nullable `entity_id`/`book_id`,
`state`, immutable `body`, nullable `result`, `created_at` (RFC3339),
`acknowledged` (boolean). `result` is the existing SetupResult or Journal object.
All responses are no-store. Existing RFC9457/auth/body rules apply. New409 codes
may distinguish a pending command or invalid state; no sensitive raw error detail.
Registration input rejects duplicate/unknown JSON fields, including nested input.
Internal recovery is not an ISO20022 exchange; unchanged financial payloads retain
ISO4217 version references. No new accounting or standards-conformance claim.

## Browser and acceptance

Load pending state before enabling new work. Review registers a durable command;
confirmation executes it. Session/tab loss clears local state but leaves server
recovery. Ambiguous responses trigger discovery/retry of the same command; no new
key/body or acknowledgement is synthesized. A committed result remains visible
until the user acknowledges it. Cancelling a prepared command requires server
confirmation, followed by acknowledgement before new work.

Named proofs: `TestCommandLifecycle`, `TestCommandAuthorization`,
`TestCommandConcurrency`, `TestCommandLostAcknowledgement`; synthetic browser
setup/post/report, dropped real response followed by remount/recovery, and denied
scope. Full Go/PostgreSQL regression and existing browser unit/component gates
remain required. Live Firebase and deployed GCP verification remain separate.
