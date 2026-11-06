# Private source read transport —239/304

## Decision and scope

Implementation slice following owner approval recorded in GitHub238/242 on
2026-11-06. GitHub owns approval and scheduling. This contract covers transport
for the three private-registry read functions; it does not implement taxonomy,
assignment, acquisition, commands, spending totals or posting. Source API and
305's future analytic review port are distinct versioned contracts.

Managed runtime mounts these routes only with `-collection-read-enabled=true`;
default is false. Startup/readiness require all three read functions and grants,
deny operator provisioning, and retain existing runtime safety checks. No cloud
flag, bridge allowlist, browser navigation or deployment changes in this slice.

| Route | Restricted SQL | Result profile |
| --- | --- | --- |
| GET /v1/collection | banking.get_collection_workspace(uuid) | private-source-workspace-v1 |
| POST /v1/collections/{workspace}/sources/query | banking.list_collection_sources(uuid,uuid,text) | private-source-page-v1 |
| POST /v1/collections/{workspace}/observations/query | banking.query_collection_observations(uuid,uuid,text) | private-source-query-v1 |

The registry contract owns exact closed selectors, current principal grants,
immutable observation revisions, source snapshot and continuation validation.
Actor is independently resolved by the existing verified-token authenticator;
no body field or UUID grants authority. Queries keep their original bytes so SQL
can reject duplicate/unknown keys. No raw provider identifiers or legacy book
IDs are introduced into the source-only projection.

## Framing, transaction and error boundary

Existing managed browser Origin boundary wraps every route. Exactly one Bearer
header is required before private parsing. Query strings, encoded requests and
Idempotency-Key are rejected. POST requires exactly `application/json`, maximum
8192 bytes. Workspace path is canonical lowercase nonzero UUID. Workspace GET
rejects bodies; HEAD cannot inherit GET behavior. Responses are noncached JSON,
maximum1MiB, with the expected versioned object profile. Decimal strings and
dated currency metadata pass unchanged; Go computes no money or totals.

Each read uses one bounded repeatable-read transaction. It is not READ ONLY:
SQL's current-authority locks require row locks. Four-second context and local
statement/lock timeouts bound the operation; rollback uses a separate cleanup
context. Only known40001/40P01 aborts retry, at most three attempts. Response
escapes only after acknowledged commit; connection/unknown-commit errors return
503 without another attempt. SQL must reject stale snapshot even on page one.

Closed errors: P2800→400 invalid_collection; P2801→404 collection_denied;
P2802→409 collection_conflict; P2803→409 collection_snapshot_changed;
P2804→409 collection_not_ready; P2805→422 collection_limit. Known malformed
UUID/encoding SQLSTATEs map400. Other diagnostics become503
temporarily_unavailable. No private database/auth text is reflected. Problems
use RFC9457 application/problem+json and independent random UUID request_id.

## Standards, evidence and alternatives

ISO applicability follows standards.md and private registry: original source
currency is versioned ISO4217 metadata; this is an internal authorized projection,
not an ISO20022 bank exchange or audit export. No new ISO message designation,
recognition rule or conformance claim. Existing source evidence remains intact.

Reviewed2026-11-06: official Go net/http MaxBytesReader/ServeMux sections
([Go API](https://pkg.go.dev/net/http)); PostgreSQL17 transaction isolation,
repeatable-read and abort retry sections
([PG17](https://www.postgresql.org/docs/17/transaction-iso.html)); RFC9457
sections3/5 ([RFC](https://www.rfc-editor.org/rfc/rfc9457.html)). Relevant sections
reviewed, not complete manuals. Facts: bounded reader reports excess input;
GET patterns include HEAD; repeatable read can require transaction retry;
problem responses must avoid sensitive diagnostics. Project inference: explicit
HEAD denial, closed errors and fixed profile budget fit the private boundary.

Alternative: parse selectors again in Go or query tables directly. Rejected
because those duplicate SQL authority/selector semantics and widen runtime
privilege. Read-committed would weaken a page's stable selection; READ ONLY
would prevent current-authority locks. Locks/snapshot races require disposable
database evidence, beyond fake-transaction unit results.

Threats and controls: actor spoofing→verified identity; tampered selectors→SQL
closed validation; information leakage→private aliases/fixed diagnostics;
resource exhaustion→caps/timeouts; privilege escalation→read-only entrypoint
surface and denied provision function; replay/stale page→current SQL grant and
snapshot checks. Root source review and real restricted-role integration remain
required before activation. Issue304 owns full analytic query/preview/application.

## Acceptance evidence

`TestCollectionTransport`, `TestCollectionTransportDenies`,
`TestCollectionSafeFailures`: identity/body separation, framing, no private
diagnostics, fixed failure status and malformed/oversized/wrong-profile denial.
`TestCollectionReadAdapter`/`TestCollectionReadFailure`: exact SQL and isolation,
commit-before-result, abort retries, unknown commit, cleanup, invalid envelopes.
`TestCollectionComposition`: managed flag/auth/Origin/default-off composition.
These are synthetic transport proofs, not private SQL permission/race proofs.
