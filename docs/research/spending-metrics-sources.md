# Spending metric source register —417/428

Selected public sections reviewed2026-11-03. No bank record, credential, actual
provider object or full paywalled standard read. This supports a proposal, not an
implemented financial API, jurisdictional policy or conformance statement.

| ID/source | Review depth / version | Fact and project inference |
| --- | --- | --- |
| SM01 [OpenStax Financial Accounting §8.6](https://openstax.org/books/principles-financial-accounting/pages/8-6-define-the-purpose-of-a-bank-reconciliation-and-prepare-a-bank-reconciliation-and-its-associated-journal-entries) | Fundamentals and reconciliation explanation, selected paragraphs, accessed2026-11-03 | Bank and ledger balances can differ. Infer separate source/posted measures; this educational source does not approve our recognition policy |
| SM02 [Plaid Transactions API](https://plaid.com/docs/api/products/transactions/) | Transactions/sync field definitions amount/date/name/merchant_name/pending/pending_transaction_id; selected fields, API adapter pins2020-09-14 | Direction, booked-day, nullable enrichment and pending lineage differ. Infer exact original-source projection, no purpose inferred from sign/description; no Item capability/history content inspected |
| SM03 [ISO4217 overview](https://www.iso.org/iso-4217-currency-codes.html) | Public2015 overview: codes, minor units, historical references/maintenance,2026-11-03; full standard unread | Currency metadata requires version context. Reuse installed dated references, no new code download/FX/unit equivalence claim |
| SM04 [Existing standards assessment](../contracts/standards.md) | ISO20022 FAQ/catalogue overview and ISO21378:2019 abstract/metadata review reused, not new normative-text access | Internal analytics gets an applicability record, not an invented message identifier; audit export conformance remains unresolved |
| SM05 Repository normalized/current-fact review | internal/ingestion/plaid/types.go Transaction; migration12 publish_sync_row/review detail and last_completed_at; migrations39/40 read boundaries; relevant source sections read | Normalized merchant_name is stored in provider_fields while display description uses name; no dedicated normalized PFC. Pending may lack review rows. Infer a separate approved minimal projection, current-revision inventory and exact merchant drill; do not assume40 already implements metrics |
| SM06 [ISO8601-1:2019](https://www.iso.org/standard/70907.html), [Amd1:2022](https://www.iso.org/standard/81801.html), [RFC3339](https://www.rfc-editor.org/rfc/rfc3339.html#section-5.6) | ISO edition/public metadata only, full normative text unread; RFC July2002 §5.6 grammar and notes read2026-11-03 | Distinguish a full calendar day from an offset-bearing timestamp. Reuse YYYY-MM-DD and existing RFC3339 UTC output; no expanded/week/ordinal date adoption or claim that the RFC certifies the 2019 ISO edition |

SM07, reviewed2026-11-03 for429: PostgreSQL17
[§9.21](https://www.postgresql.org/docs/17/functions-aggregate.html), count/sum,
empty-set and ordered aggregation entries;
[§9.3](https://www.postgresql.org/docs/17/functions-math.html), trim_scale/trunc entries;
[§13.2.2](https://www.postgresql.org/docs/17/transaction-iso.html#XACT-REPEATABLE-READ),
full subsection. Facts: numeric aggregates retain numeric arithmetic, empty sums
need explicit treatment, trailing scale differs from value, and RepeatableRead
requires careful locking/retry for consistency. Project inference: exact complete
unit-group sums, no precision rounding into eligibility, fresh authorization locks
and pinned drill snapshots. These mechanisms do not prove our future joins,
privilege boundaries, snapshot invalidation or latency; actual runtime tests remain
required. No full-dataset/payload access or dependency added by this review.
Also selected PostgreSQL17
[§4.1.2.2](https://www.postgresql.org/docs/17/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS-ESCAPE)
escape table/notes and
[§23.2.2.1](https://www.postgresql.org/docs/17/collation.html#COLLATION-MANAGING-STANDARD)
C/default entries: octal denotes a byte while an unlisted escape preserves its
letter; C ordering is byte-based for an encoding. Infer explicit ASCII blank
handling and C group/equality behavior, with a one-letter-label regression case.

Contract choices, alternatives, threats and acceptance are in
[spending metrics](../contracts/spending-metrics.md). Original worked examples
and hand-fixed expected quantities live in the [oracle](../../tests/corpus/spending/README.md).
No external examples/code copied, model/runtime/engine dependency or private data
backfill proposed. References establish mechanisms/field meanings, not accuracy
of a future classifier or approved GAAP/tax accounting treatment.
