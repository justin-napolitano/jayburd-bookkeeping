# Synthetic import service orchestration

## Boundary

Issue #93 under #8. Coordinator reviewed the Service/SQLRepository separation
before implementation. This local adapter connects the existing private staging
and strict decoder to the durable SQL import boundary. It creates no journals,
executes no import command automatically, and introduces no HTTP endpoint, cloud
object store or provider acquisition. Explicit import confirmation uses the existing
interactive command recovery gate; the service does not provide a bypass.

## API and SQL boundary

`New(Beginner, Storage)` composes caller-owned dependencies. `Register(ctx, Scope,
key, Registration)` registers profile/source/namespace/binding/date identity;
`Batch(ctx, Scope, batchID)` and `Review(ctx, Scope, reviewID)` read public scoped
wrappers. `Upload(ctx, Scope, batchID, staging.Source)` transfers source ownership,
stages and previews; `Preview(ctx, Scope, batchID)` reconciles existing retained
bytes. Upload/Preview use a 20-second overall context budget. No method accepts a
filesystem path, URL, caller digest or decoded row list.

The private repository calls exactly `banking.register_import_batch`,
`banking.import_context`, `banking.publish_import_preview`,
`banking.get_import_batch` and `banking.get_review_item`. All arguments use
parameter binding. Context contains SQL-generated object identity and approved
currency/date expectations; the service validates canonical UUIDs, bigint-string
revisions and decoder expectations before I/O. Registration returns private
service metadata, so it must not be exposed unchanged through a future public
HTTP route. Public Batch uses the separate SQL wrapper without object details.

Publication contains profile, server SHA256 and byte count, plus either complete
rows preserving amount/canonical_amount/currency_version_id or decoder diagnostics,
total and truncated. SQL independently checks the full shape and semantics.
An unknown publication result returns no preview. `Preview` explicitly retries
from retained exact bytes; successful SQL replay owns the stable preview identity.
Scope.Actor must come from verified identity in future composition, never a body.

SQL P2400/P2401/P2402/P2403/P2404 map to fixed invalid_request/not_found/
import_conflict/stale_preview/import_limit. Infrastructure/source errors retain no
cause text. Actual storage object_conflict maps to statement_identity_conflict;
missing/deleted evidence maps to evidence_unavailable. Durable decoder rejection
returns a successful rejected batch with bounded diagnostics, not an exception
containing rejected values.

## Ownership and transactions

The authenticated composition supplies a verified principal and entity/book scope.
These values are not credentials or evidence of permission: SQL checks current
membership and source grants on every call. SQL commits authorization and returns
server-owned batch/object/decoder metadata before any storage access. Input source
ownership transfers at Upload entry; denied/invalid requests still close it with
an independent one-second cleanup context. Once passed to staging.Put, staging
owns exactly one close. Sources must honor context in Read and Close; an arbitrary
blocked io.Reader is not made cancellable by an adapter or goroutine.

No SQL transaction spans storage, reading, or decoding. Private staging provides
immutable exact-byte equality on replay, including full-byte comparison rather
than hash-only identity. Only SQL-returned scoped UUIDs select an object; user
paths, URLs and user-supplied digests are never accepted. Bounded staged bytes are
decoded using SQL-returned expectations, then SQL independently reauthorizes and
publishes the complete preview or bounded diagnostics atomically. Decoding failure
has no usable partial rows. Preview alone creates no source observations/review
items or journal; explicit import is separate.

Each SQL call has a four-second active budget, three-second statement timeout,
one-second lock timeout and four-second idle-transaction timeout. The independent
one-second cleanup budget preserves the five-second transaction bound. Only 40001/40P01 confirmed
rollback outcomes retry the complete transaction, at most three attempts with
bounded jitter. Cleanup always uses an independent one-second context. Response
JSON is capped at 2 MiB. Unknown COMMIT suppresses result and automatic replay;
return `commit_unknown`, then reconcile using the same batch/statement identity.
SQLSTATE 08007/40003 are explicitly ambiguous. Storage publication can similarly
be uncertain: no compensating delete is attempted because published evidence
may already be referenced by a committed or uncertain preview.

## Sources and applicability

This continues the reviewed mechanisms and alternatives in
[durable import](durable-import.md) (DI01–DI02),
[private staging](import-staging.md) (ST01–ST03),
[strict decoding](statement-decoder.md) (SI01–SI06), and
[observer runtime](observer-runtime.md) (OR01: PostgreSQL17 error-code table).
The review covers the cited selected sections and existing implementations, not
additional normative standards text. Internal synthetic CSV remains the recorded
standards exception; ISO4217 reference currency IDs retain their meaning. This
service is not an ISO20022 financial message or a conformance implementation.

## Validation status

Implementation and fault evidence are recorded in the
[living runbook](../runbooks/import-service.md). Hosted encryption, retention
cleanup, real ownership/provider qualification and HTTP upload composition remain
separate acceptance gates.
