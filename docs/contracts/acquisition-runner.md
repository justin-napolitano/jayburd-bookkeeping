# Bounded synthetic acquisition runner

## Reviewed decision

Coordinator approved this #102 profile before implementation. Go composes the
existing exact Plaid decoder and SQL complete-set publication. No network or storage
call holds a database transaction; acquisition never posts journals. SQL owns current
consent, service/account authority, generation, fencing and atomic cursor publication.
This slice uses injected synthetic credentials/providers and private local evidence;
it does not activate Plaid, Sandbox, cloud storage, scheduled delivery or live books.
The proprietary `jayburd-plaid-transactions/1`, API `2020-09-14`, ISO currency/date
mapping and STD-01 assessment remain as in [sync SQL](provider-sync-sql.md).

## Callable composition

`New(Beginner, Provider, CredentialResolver, EvidenceStore)` returns a Runner.
`Run(ctx, Scope{EntityID,BookID,StreamID}, claimUUID)` runs one bounded acquisition.
The caller persists/reuses the same canonical claim UUID until its outcome is known;
a new UUID is an explicit fresh attempt, never an automatic unknown-commit retry.
`SQLRepository` supplies compiled, parameterized calls through a restricted sync
pool. No connection string, caller actor, destination URL or secret comes from a
wakeup. Outcome contains only status and counts; errors expose fixed classifications.

`CredentialResolver.Resolve(ctx, CredentialReference{ReferenceID,Version,ItemID,
Environment})` returns memory-only access/client credentials. The resolver validates
stored reference/version/Item/environment binding. It must honor cancellation and
must not log tokens. No resolver discovers files/environment/ADC in this package.
`NewPlaidHTTP(http.RoundTripper)` pins HTTPS sandbox.plaid.com/transactions/sync,
Plaid-Version, account filter, count500 and original-description inclusion.
Credentials are injected on each call; redirects and automatic provider retries are
forbidden. Full response bytes are capped at2MiB and parsed without float conversion.

## SQL and recovery protocol

Each SQL call has4s active budget plus independent1s rollback,3s statement/1s lock
limits and at most3 confirmed40001/40P01 retries. Unknown COMMIT exposes no prepared
result and stops acquisition. Role checks reject SET ROLE and mixed privileged,
API, owner, observer, publisher and identity capabilities. SQL remains authority.

`sync_resume(e,b,stream,claimUUID)` reauthorizes before returning absent, staging,
completed, abandoned or quarantined; it includes set/fence/reason and an optional
committed receipt. Completed replay returns that receipt without provider access.
`sync_context(e,b,stream,set,fence)` reauthorizes a live attempt and returns only
approved account/currency metadata, opaque credential reference/version, Item pins,
cursor/page/counts, lease and exact limits. No token crosses SQL.

Issue #225 adds bounded `currency_versions` for the bound monetary identity.
Resolve each source date against its installed version; retain singleton `currency`
for compatibility. [History contract](plaid-history-currency.md) defines source
evidence, migration ordering and unchanged bookkeeping-date gates.

Order: resume/claim COMMIT → context → existing evidence lookup. For a missing page:
context → credential read → fresh context → provider request → exact decode → fresh
context → durable private evidence. Then stage COMMIT; after the final page, complete
COMMIT. Revocation after an external request may leave in-flight bytes in memory;
no stage/publication can bypass current SQL authorization. Context cancellation never
means a blocked arbitrary dependency was forcibly terminated.

A stage COMMIT with lost reply is recovered by the same claim UUID and SQL page
position; if it did not commit, replay the same retained raw bytes/normalized page,
not a new provider response. Unknown completion resolves through sync_resume.
Unknown claim cannot authorize any credential or provider call until recovered.

A pagination mutation abandons the entire set and restarts from committed C0 once,
using a deterministic second claim UUID. Its reason is durable for process replay.
Other ambiguous provider/transient failures abandon and return retryable status;
new acquisition begins C0 with a new explicitly chosen claim. No partial page retry
or partial cursor publication. Failed cleanup prevents internal restart.

## Bounds and evidence

Run budget60s; provider call10s; credential read5s. SQL permits10pages,500changes/page,
1000changes and2MiB normalized complete set with90s lease. This runner additionally
caps total retained raw bytes at2MiB, a stricter project bound, not a Plaid limit.
One mutation restart at most; no invented durable retry scheduler or50k-row claim.

`LocalEvidence` composes the existing synthetic-only [staging store](import-staging.md).
Generated stable UUIDs bind entity/book/stream/set/page and chunk identity; no caller
path. Up to two1MiB immutable chunks precede an immutable manifest. Read validates
scope, cursor, lengths and digests, then Put replay compares complete bytes. Manifest
publication occurs last. Chunk/manifest fsync/no-replace/0700root/0600file and symlink
rules come from staging. Delete tombstones the manifest before both chunks, preventing
late resurrection; lifecycle metadata remains. Partial writes are private orphans,
not SQL-published evidence. Retention and encrypted hosted storage remain future work.

## Primary sources and alternatives

Reviewed2026-10-13 from PI01/PI02/PI08 in the [source register](../research/plaid-ingestion-sources.md):

- [Plaid Transactions API](https://plaid.com/docs/api/products/transactions/),
  sync pagination, options.account_id, count and response fields. Each filtered
  account has its own cursor. Original source bytes and signed amount lexemes are
  retained; SDK float conversion is deliberately avoided.
- [Sync migration guide](https://plaid.com/docs/transactions/sync-migration/),
  complete update application and pagination mutation restart from original C0.
  Our private staging/SQL publication and bounded restart are project inferences.
- Existing pinned pgx retry/commit evidence in [observer runtime](observer-runtime.md)
  and [import service](import-service.md), and filesystem mechanisms in staging.

Alternative: adopt partially staged provider pages across expired leases. Rejected
for this slice because the merged SQL contract intentionally starts a new set at C0.
Cloud object storage and an SDK may replace injected implementations only after
preserving exact bytes, bounded cancellation and unknown-outcome semantics.

## Provider retry instruction

Retry-After seconds or HTTP-date values are preserved up to24hours. Longer values
produce retry_deferred, never a shortened automatic retry. Invalid or past values
use five seconds. Outcome is a proposed caller instruction only: this slice has no
scheduler and performs no delayed retry. Injected dependency errors are reduced to
a closed public classification; whole credential objects reject JSON and redact
formatted representations. Individual credential fields are deliberately available
only for the provider request builder and must not be logged.
