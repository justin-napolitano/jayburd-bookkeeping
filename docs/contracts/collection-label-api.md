# Private collection query and label transport appendix —304/305

## Scope and reviewed protocol

Coordinator-frozen2026-11-06, following accepted owner design238/242 and merged
registry42/447. This appendix describes Go transport only. SQL owns taxonomy,
selection, current source/book grants, preview pins, expiry, replay and durable
command outcomes. The coordinator owns the canonical SQL/API contract, server
composition, generated browser wire types and activation. No provider operation,
credential handling, direct table query, money calculation or automatic journal.

All routes below have prefix `/v1/collections/{workspace}`. Independently verified
actor, canonical lowercase nonzero workspace UUID and command path UUID are
separate SQL parameters. Request JSON remains byte-for-byte unchanged, including
duplicate/unknown keys, so PostgreSQL can reject malformed intent exactly once.

| Method and suffix | Fixed banking function | Body cap | Isolation | Result profile |
| --- | --- | --- | --- | --- |
| GET /taxonomy | get_collection_taxonomy(actor,workspace) | Empty | Repeatable read | private-collection-catalog-v1 |
| POST /review/query | query_collection_review(actor,workspace,raw) |8192bytes| Repeatable read | private-collection-review-v1 |
| POST /labels/preview | preview_collection_labels(actor,workspace,raw) |524288bytes| Serializable | private-collection-preview-v1 |
| POST /label-commands | prepare_collection_label_command(actor,workspace,raw) |4096bytes| Serializable | private-collection-command-v1 |
| GET /label-commands/pending | pending_collection_label_command(actor,workspace) | Empty | Repeatable read | private-collection-command-v1 |
| POST /label-commands/{command}/execute | transition_collection_label_command(actor,workspace,command,'execute',raw) |1024bytes| Serializable | private-collection-command-v1 |
| POST /label-commands/{command}/cancel | transition_collection_label_command(actor,workspace,command,'cancel',raw) |1024bytes| Serializable | private-collection-command-v1 |
| POST /label-commands/{command}/acknowledge | transition_collection_label_command(actor,workspace,command,'acknowledge',raw) |1024bytes| Serializable | private-collection-command-v1 |

The SQL-owned query body is the reviewed UI Query excluding workspace_id; preview
is PreviewRequest excluding workspace_id; prepare is {preview_id,prepare_key};
transition is exactly {}. Go neither reshapes nor independently validates these
objects. Taxonomy preview action carries topics and empty selection; label preview
uses topic/uncategorized and1–100 selected source observation revisions. The larger
preview limit accommodates the reviewed full taxonomy, not unbounded selections.

Catalog includes workspace/authority/taxonomy/ruleset/map revisions, all topics
(at most1000 total/500 active), and at most100 safe source aliases. Snapshot adds
map_revision and decision_revision to the UI's original authority/source/taxonomy/
ruleset pins. Label commands return a wrapper containing profile, workspace_id
and command; null command is valid only for pending discovery. Cancel/pending/
acknowledge can recover stale previews under current grants; execute rechecks
pins and expiry. These are SQL obligations, not claims proved by adapter fixtures.

## Transaction and HTTP boundary

The service uses one4second context across at most three attempts. Every attempt
starts a fresh transaction with local statement/lock timeouts bounded by remaining
time. Reads permit row locks rather than READ ONLY. Preview is serializable because
the inert preview reference is persisted. Every exit rolls back with a separate
bounded1second cleanup context. Results escape only after acknowledged COMMIT.
Only known40001/40P01 aborts retry. A lost/unknown commit reply, connection error,
lock timeout or any other failure returns a fixed failure with no automatic retry.

HTTP authenticates one nonempty Bearer header before path/body parsing. It rejects
query strings (including an empty ?), Idempotency-Key, Content-Encoding, duplicate
media headers, unsupported media, oversized bodies and bodies on GET. POST media
is exactly application/json. HEAD is explicitly405 with Allow:GET. Existing server
Origin and concurrency boundaries must wrap this handler in coordinator composition.

Both adapter and handler require a valid JSON object with the expected profile,
matching workspace and at most1MiB; command envelopes require an object command
except pending's explicit null. This envelope check is not a second taxonomy or
monetary validator. Private JSON is returned unchanged. Responses are no-store
and nosniff. Fixed RFC9457 errors include an independent random UUID request_id:
P2800/22P02/22021→400 invalid_collection; P2801→404 collection_denied;
P2802→409 collection_conflict; P2803→409 collection_snapshot_changed;
P2804→409 collection_not_ready; P2805→422 collection_limit; other errors→503
temporarily_unavailable. HTTP errors cannot relay arbitrary Failure status/code.

## Source review, standards and alternatives

Reuses the accepted private categorization/registry ISO applicability: original
ISO4217 currency metadata and decimal strings pass unchanged. Internal owner
taxonomy/query commands are not an ISO20022 bank exchange, ISO taxonomy or audit
export. No accounting recognition or new ISO conformance claim.

Reviewed2026-11-06, selected sections rather than complete manuals:

- [PostgreSQL17 isolation13.2.2–13.2.3](https://www.postgresql.org/docs/17/transaction-iso.html):
  snapshots, locking conflicts, whole-transaction retry and commit-before-trust.
- [RFC9457 section5](https://www.rfc-editor.org/rfc/rfc9457.html#section-5):
  error details can disclose private data and implementation internals.
- [Go net/http MaxBytesReader and ServeMux](https://pkg.go.dev/net/http):
  bounded reads and GET-pattern HEAD behavior; current online documentation was
  go1.27.3, not a verification of the tracked1.26.10 implementation. Existing
  reviewed collectionread behavior plus pinned canonical CI remain necessary.

Project inference: fixed SQL calls, exact raw bytes, whole-transaction retries and
minimal envelope fences avoid duplicating SQL authority. Alternatives: parse and
reencode selectors in Go (would erase duplicate-key evidence); read tables directly
(widens privilege and duplicates rules); retry every failure (risks hidden outcomes);
use one isolation level (would obscure preview persistence versus read semantics).

Threats: actor spoofing→independent verified identity; malformed intent→SQL closed
validation and finite HTTP caps; stale authority→SQL current-row locks/pins; replay
and lost reply→immutable SQL command/receipt plus explicit recovery; disclosure→
no-store/fixed errors/unchanged authorized projections. Concrete combined source
review and real restricted-role proofs precede activation; local unit tests do not
establish production readiness or SQL correctness.

## Named transport evidence

TestLabelAdapter: all fixed SQL signatures, raw-byte/actor separation, actual
isolation, bounded contexts, acknowledged commit and cleanup. TestLabelTransactionFailures:
closed diagnostics, known-abort retry cap, successful retry, cancellation and lost
commit suppression. TestLabelResultFence: scope/profile/size/command-null guards.
TestLabelHTTPRoutes, TestLabelHTTPFraming and TestLabelHTTPSafeFailures: all routes,
exact caps, authentication-before-parse, fixed failures and unchanged result bytes.
Real SQL integration belongs to the registry lane; combined app and generated API
verification belong to the coordinator. No test authorization adapter is exported.
