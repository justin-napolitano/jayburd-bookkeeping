# One-click synthetic acquisition

## Boundary and decision

Issue #97 supplies a **test-only** loader in integration-tagged server test files.
The ordinary application binary has no demo endpoints or synthetic identity.
The coordinator approved three fictional LLCs, one bank account each, 3,000 source
transactions, and bounded explicit request-driven progress. No bank connection,
real upload, automatic journal, cloud workload or background runner is introduced.

The fixture administrator separately provisions source read/import/review grants
through their owner-only functions. Book setup, batch registration, staging,
decoding, preview and explicit import use the existing restricted runtime pool.
The runtime loader never retains the administrator connection. The root composition
owns a private staging Store and keeps its directory for the disposable database's
lifetime. Rebuilding the handler against that database reconstructs identities from
setup/source/registration receipts; deleting the object directory while retaining
its database is unsupported and must not be called successful recovery.

## Hook and HTTP protocol

`bootstrapDemoAcquisition(ctx, admin, pool, store)` returns an `http.Handler`.
Bootstrap creates three idempotent fictional books/sources and registers empty
batches. It does not decode/import transactions or create journals. Composition
mounts it only in the disposable demo integration binary, under the existing
BrowserBoundary for the local demo origin. The handler independently requires the
exact fictional demo bearer identity before returning status or touching SQL.

- `GET /v1/demo/acquisition`: return durable progress; no query parameters.
- `POST /v1/demo/acquisition/step`, body exactly `{}`: perform at most one stage,
  then return freshly read progress. No scope, file, row, amount or URL is accepted.
- Both return `{profile:"synthetic-acquisition-v1",total_count:"3000",
  imported_count:string,complete:boolean,blocked:boolean,
  pending_command_id:string|null,batches:[{name,entity_id,book_id,
  source_account_id,batch_id,state,imported_count:string}]}`.
- Errors are `{error:{code}}`: unauthorized (401), invalid_request (400),
  demo_busy/demo_blocked (409), or demo_retry_required (503). Internal details
  never appear. Responses are no-store. A 503 is not proof of rollback; read status
  and resume the same durable identities.

One explicit Load/Resume action authorizes the synthetic import sequence. The UI
makes successive bounded POSTs only while the user keeps that action active.
Pause stops new requests; no hidden goroutine continues processing. Each request
has a 20-second overall context. A local TryLock rejects overlapping step calls;
SQL receipts, source/statement uniqueness and the existing recovery gate provide
durability and retry safety, not this process lock.

A stage is one of: upload+strict preview, register reviewed import command, execute
that command, or acknowledge its committed result. Existing pending commands must
match the exact demo batch, scope and pinned latest preview before execution or
acknowledgement. Unrelated, cancelled or changed approval blocks progress; it is
never silently discarded. A lost response resumes via Batch and Recovery.pending.
Only imported batches count toward imported_count; completion waits for the last
demo command acknowledgement. This is synthetic import confirmation, not approval
to create financial journals. Journal posting remains a separate review action.

## Dataset and independent oracle

Each LLC has 1,000 deterministic CRLF CSV rows, within the strict decoder's maximum:
500 inflows of USD 100.01 and 500 outflows of USD 25.02, alternated over January 2026.
Names explicitly say Fictional. Descriptions identify customer payments and
operating expenses; there are no real names, bank identifiers or credentials.
Entry IDs are stable within the registered fictional namespace. Integer cents
produce exact decimal strings; no float, rounding or balance plugs are used.

Independent expected values are 1,000 observations/review items/provenance links
per LLC, 3,000 overall, USD 50,005.00 inflows and USD 12,510.00 outflows per LLC,
and USD 37,495.00 net each / USD 112,485.00 overall. Import creates zero journals,
zero ledger outbox messages and changes no book revision. Imported observations
are real database records; they are not prefilled chart values or generator-only
validation. Account setup is outside the before/after import revision assertion.

## Sources and standards

This is a project-authored deterministic fixture, not a bank export or a licensed
external dataset. The [statement decoder](statement-decoder.md),
[private staging](import-staging.md), [import orchestration](import-service.md),
[SQL import](import-sql.md) and [command recovery](command-recovery.md) remain the
reviewed authorities. Their SI/ST/DI/OR source sections establish existing syntax,
exact-value, transaction and ownership mechanisms; no new external source or
normative text was consulted for this fixture. The synthetic CSV exception applies;
ISO4217 USD metadata is retained, and no ISO20022 exchange/conformance claim is made.
USD-only demo acquisition is not a product-scope reduction: broader multicurrency
requirements and existing EUR/JPY/KWD proofs remain unchanged.
