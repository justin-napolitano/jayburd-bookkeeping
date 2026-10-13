# Pinned report journal drill

## Decision and scope

Coordinator reviewed and approved 2026-10-13 before migration 7. Refines #53's
[financial reporting contract](financial-reporting.md) without changing accounting
policy. One selected GL line proves journal inclusion in a scoped, revision-pinned
account/date selection. No historical snapshot storage or claim that a selector
was previously issued by the server. Category/subtotal navigation remains deferred.

## SQL and wire contract

`reporting.report_journal(actor uuid,e uuid,b uuid,raw text) returns jsonb` accepts
at most 8,192 UTF-8 bytes, a JSON object with unique keys and exactly these required
fields: `snapshot_revision` (canonical nonnegative bigint string), `account_id` and
`journal_id` (hyphenated UUID strings), `starts_on`, `ends_before` (Gregorian full
dates, half-open range of 1–366 days), `line_number` (JSON integer 1–1000). Nulls,
unknown fields, coercions, invalid Unicode and duplicate keys fail `P2100` /400
`invalid_request`. HTTP: GET `/v1/entities/{entity}/books/{book}/reports/general-ledger/journals/{journal}`;
all other SQL fields are required query parameters. Coordinator owns HTTP/types.

Authorize `read` first using the existing locked principal/entity/membership/book
boundary. Then validate input and compare revision; mismatch fails `P2114` /409
`report_snapshot_changed`. Find the exact scoped account/journal/line with journal
accounting date in the range; absent membership fails `P2101` /404 `not_found`.
Call existing `reporting.get_journal` in this same transaction. Response:
`{contract_version:"report-journal-v1",entity_id,book_id,snapshot_revision,
selector:{account_id,starts_on,ends_before,journal_id,line_number},journal:Journal}`.
`Journal` is the existing immutable journal shape, including original units and
reversal/rate provenance. Its `book_revision` is the posting revision; the outer
`snapshot_revision` is the report observation revision. Maximum response 2 MiB,
otherwise `P2110` /422 `result_out_of_range`. No writes or new table grants.

`Service.ReportJournal` uses repeatable-read, total deadline 9 seconds, statement
5 seconds, lock 1 second, at most two whole-transaction attempts, bounded rollback
cleanup. Return data only after commit acknowledgement; uncertain commit emits
503 with no body. Migration 7 is forward-only, SECURITY DEFINER with fixed safe
search path, PUBLIC execution revoked, wrapper-only grant to bookkeeping_api.

## Sources and alternatives

Reviewed 2026-10-13: PostgreSQL 17 [§13.2.2 Repeatable Read](https://www.postgresql.org/docs/17/transaction-iso.html#XACT-REPEATABLE-READ)
and [CREATE FUNCTION, Writing SECURITY DEFINER Functions Safely](https://www.postgresql.org/docs/17/sql-createfunction.html#SQL-CREATEFUNCTION-SECURITY).
Relevant sections read in full; first-party implementation documentation, not
accounting standards. Stable transaction snapshots require complete retries after
serialization failures. Definer functions require trusted search paths and narrow
execution privileges. Our inference is to reuse the existing locked authority
function and reader in one transaction. A standalone journal read plus client-side
revision check permits a race and cannot prove selection. Persisted report tokens
would prove issuance but add storage/lifecycle obligations unnecessary for this
bounded inclusion proof. Revisit for archived reports or signed exports.

Standards applicability inherits FR04/FR05: this internal query is not an ISO 20022
bank exchange and receives no invented ISO message designation; ISO 4217 currency
code/version and existing date/UUID contracts are preserved unchanged. No new
recognition, FX, close or live accounting-policy approval.

## Acceptance

`TestReportJournalPinned` proves real GL membership, exact original journal,
unchanged durable state and stale revision rejection. `TestReportJournalBoundary`
proves malformed/duplicate input, unrelated line/range, wrong scope, revoked read
and restricted grants. `TestReportJournalReversal` preserves correction links and
original amounts. `TestReportJournalAdapter` proves isolation, retries and no data
after lost commit acknowledgement. Runbook distinguishes planned from observed.
