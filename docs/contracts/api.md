# Ledger HTTP API contract

## Status and authority

OpenAPI 3.1.1 implementation-review contract: [machine-readable definition](../../api/openapi.json).
An in-process HTTP adapter exists with an injected identity interface; no
deployed verifier, listener or public server is provided. A managed identity adapter
and atomic `POST /v1/setup` are specified by the
[identity/setup contract](identity-setup.md). The specification describes the first posting/read/reversal/
close/report interface; transport validation is not accounting approval. Every
operation also follows [SQL invariants](ledger-sql.md) and [money semantics](multicurrency.md).
The [posting boundary](posting-boundary.md) specifies the first executable slice.
Close is explicitly deferred pending reviewed close-evidence storage/workflows;
its documented route must not be mounted by the first server.

## Operations and permissions

Ledger routes start `/v1/entities/{entity_id}/books/{book_id}`; the separate
`POST /v1/setup` route creates a new scope. Authentication is
bearer-token transport; issuer/audience binding, MFA and revocation follow the identity/setup contract;
the actual GCP project remains a deployment decision. No actor ID is accepted from request JSON.

| Method and suffix | Permission | Result |
| --- | --- | --- |
| POST `/journals` | post | 201 committed manual identity/spot journal |
| GET `/journals/{journal_id}` | read | 200 immutable journal with valuation provenance |
| POST `/journals/{journal_id}/reversals` | post | 201 exact linked reversal; no client-generated replacement lines |
| POST `/periods/{period_id}/close` | close | 200 closed period; fails without valid reviewed evidence |
| GET `/trial-balance?through_date=YYYY-MM-DD` | read | 200 cumulative debit/credit balances through date, policy currency and snapshot revision |

These are local bookkeeping operations, not registered ISO 20022 messages.
Applicability review: currency codes use ISO 4217 metadata; banking/payment
exchange definitions are not assigned to these internal operations. ISO audit-data
mappings remain required for future exports; the trial-balance HTTP response is
not asserted to be an ISO audit export. Any later financial interchange adapter
must select its exact definition/profile under the [standards contract](standards.md).

## Request and replay rules

Amounts are decimal strings, revisions are unsigned decimal strings within the
SQL bigint range, and IDs are UUID strings. Business dates have no timezone;
operational timestamps use RFC 3339 UTC. Reject unknown/duplicate JSON fields.
The selected request limit is 1 MiB and 1,000 journal lines; these are engineering
bounds for the first implementation. Unknown account/currency/rate references,
invalid dates and overflow fail before any durable journal writes.

Ledger mutation requests require `Idempotency-Key` and `expected_revision`.
Setup requires the key but creates its initial revision in SQL. The SQL
contract defines scope/canonicalization/actor binding. The expected revision is
optimistic concurrency, not authorization. Successful retries return original
status and body with `Idempotent-Replay: true`, even if the period later closes,
provided the actor is still authorized. A stale revision on new work conflicts;
a known successful receipt is resolved first. After timeout or 503, resend the
same key and body. If a receipt may exist, do not change the expected revision
under the same key. Changes after a confirmed conflict use a new key.

## Errors and response safety

Use RFC 9457 `application/problem+json` with stable type URI, title, status,
project `code`, and request ID. Detail contains a safe actionable explanation,
never SQL, tokens, provider payloads or another entity's identifiers. Public
problem-document URLs require a reviewed deployment base. Until deployment,
the review convention is `https://bookkeeping.example.invalid/problems/{code}`;
this is a non-routable example, not a deployed documentation URL. Select the real
base before publishing `/v1` and keep each problem type stable thereafter.

| Status | Code examples / handling |
| --- | --- |
| 400 | invalid_request: malformed JSON, duplicate fields, missing headers |
| 401 | unauthenticated: missing/invalid bearer token; WWW-Authenticate: Bearer |
| 403 | permission_denied: known accessible scope, insufficient operation permission |
| 404 | not_found: absent resource or inaccessible entity/book; indistinguishable |
| 409 | revision_conflict, idempotency_conflict, period_closed, already_reversed, close_evidence_stale |
| 413 | request_too_large: body limit exceeded |
| 422 | invalid_journal, invalid_amount, unsupported_policy, result_out_of_range, close_evidence_required |
| 500 | internal_error: unexpected failure, generic detail and request ID only |
| 503 | temporarily_unavailable: exhausted retry/deadline, Retry-After; replay same key |

All financial responses use `Cache-Control: no-store`. Journal responses preserve
original currency amounts, functional currency, policy and selected rate IDs.
No partial response claims posting succeeded. No POST is acknowledged with 202
for a journal whose transaction has not committed.

## Acceptance and evolution

OpenAPI documents shape, not cross-line sums, database reference validity or
permission enforcement. `TestAPIContract` must cover valid requests, extra fields,
malformed money, absent credentials, denied scope, redacted problems, replay after
close/revocation and revision conflicts. SQL test names remain in the SQL contract.
Keep changes backward-compatible within `/v1`; new required request fields or
semantic reinterpretations require an explicit version/migration decision.

Further entity/book/account management, draft approvals, journal lists/pagination, statement
import, reconciliation, settlement, revaluation and exports are separate next
slices. Their durable obligations are recorded in the SQL contract; they have no
invented placeholder HTTP endpoints here. First implementable slice is manual
identity/spot posting and reads under ADR 0005 and the posting boundary, then exact reversal.
An internal synthetic identity adapter may exercise SQL tests, but cannot stand
in for a production bearer-token verifier.
