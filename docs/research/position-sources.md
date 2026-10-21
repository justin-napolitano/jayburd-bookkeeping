# Position source review and fixed synthetic example

## Review record

Reviewed 2026-10-21 for proposed [Position contract](../contracts/position-read-model.md),
starting at the [bibliography](bibliography.md), [decision evidence](decision-evidence.md),
[database foundations](database-foundations.md), and existing [financial reporting
sources](financial-reporting-sources.md). No new normative accounting approval.

| ID | Reviewed source and depth | Fact established | Project inference / limit |
| --- | --- | --- | --- |
| P01 | OpenStax / Rice University, *Principles of Accounting, Volume 1*, [§2.3 statement relationships](https://openstax.org/books/principles-financial-accounting/pages/2-3-prepare-an-income-statement-statement-of-owners-equity-and-balance-sheet), income statement, owner's equity and balance sheet subsections, reviewed through FR01 | Balance sheet is dated; performance covers a period; earnings/distributions affect equity | Show dated Position and separately labeled change. Introductory example does not decide LLC cash classification, GAAP or spendability. |
| P02 | PostgreSQL Global Development Group, [PostgreSQL 17 §13.2.2 Repeatable Read](https://www.postgresql.org/docs/17/transaction-iso.html#XACT-REPEATABLE-READ), relevant section reviewed through FR02; local reporting contracts and runbooks reviewed in full for API facts | One transaction has stable snapshot; later request does not inherit it. Existing reporting uses current membership and revision validation. | One response should carry current/prior points; cross-request drill must pin/revalidate revision. Documentation alone does not prove our grants or performance. |
| P03 | ISO 20022 Registration Authority [FAQ](https://www.iso20022.org/faq), About/Compliance, and [message catalogue](https://www.iso20022.org/catalogue-messages), catalogue/version overview, reviewed through FR04/R01–R02 | Registered messages have versioned definitions and usage context | Internal Position query has no selected external exchange profile. No individual message or eight-part normative text reviewed; reassess for an actual exchange. |
| P04 | ISO [4217 public overview](https://www.iso.org/iso-4217-currency-codes.html), currency identifier/minor-unit overview, reviewed through FR04/R03 | Maintained codes and minor-unit metadata identify currencies | Preserve existing pinned code/version; no FX method or cash policy follows. No new code-list release assessed. |
| P05 | ISO [21378:2019 standard page](https://www.iso.org/standard/70823.html), abstract/edition metadata, and [21926:2026 official preview](https://www.iso.org/obp/ui?_escaped_fragment_=iso%3Astd%3Aiso%3A21926%3Aed-1%3Av1%3Aen), contents/introduction/§1 through R04–R05 | Audit-data extraction and semantic modelling are relevant future topics | Position is an internal view, not an audit export. Normative field mappings remain unavailable and unresolved. |

Repository facts checked: [financial reporting](../contracts/financial-reporting.md)
defines debit-positive account net and unclassified balance sheet;
[reporting SQL](../contracts/reporting-sql.md) and [HTTP](../contracts/reporting-http.md)
define exact strings, single-book scope, comparison, revision and errors;
[pinned journal](../contracts/report-drill.md) proves a selected GL line. These
are project implementation/contract sources, not accounting standards. Current
[reporting foundation](../runbooks/reporting-foundation.md) and
[HTTP delivery](../runbooks/reporting-http.md) document synthetic tests and
limitations; they do not prove the proposed cash or attention extension.

## Fixed fictional example — independently specified expected values

All IDs below are fixture labels, not database IDs. One synthetic book A uses
functional USD, synthetic accrual/UTC policy and two asset accounts: `A-CASH-USD`
and `A-CASH-EUR`; `A-PREPAID` is noncash. The *fixture's* reviewed cash membership
is `{A-CASH-USD,A-CASH-EUR}` solely for acceptance math; it does not approve a
production policy. Original units are retained per line. All journals balance
in functional USD. `Dr` and `Cr` are sides, never signed input amounts.

| Journal/date | Debit line, original / functional | Credit line, original / functional | Meaning |
| --- | --- | --- | --- |
| J1 / 2026-01-02 | A-CASH-USD USD 1000 / USD 1000 | Equity-capital USD 1000 / USD 1000 | Opening capital actually posted |
| J2 / 2026-01-10 | Expense USD 120 / USD 120 | A-CASH-USD USD 120 / USD 120 | Period expense |
| J3 / 2026-01-11 | A-CASH-USD USD 300 / USD 300 | Loan-liability USD 300 / USD 300 | Borrowing, not income |
| J4 / 2026-01-12 | A-PREPAID USD 40 / USD 40 | A-CASH-USD USD 40 / USD 40 | Asset transfer, not expense |
| J5 / 2026-01-13 | A-CASH-EUR EUR 10 / USD 11 | Revenue USD 11 / USD 11 | Immutable EUR-to-USD spot valuation; rate/reference pinned to J5 |
| J6 / 2026-02-02 | Revenue USD 11 / USD 11 | A-CASH-EUR EUR 10 / USD 11 | Linked reversal of J5; original EUR quantity/rate retained |

No retranslation is performed at either date. Jan 13's EUR 10 has a booked USD
11 functional value; a later market quote cannot change J5. J6 offsets the
original in February, not retroactively in January. Raw original EUR 10 and
USD amounts remain separate; no cross-currency original total exists.

| As of | Cash USD functional | Other assets USD | Total assets USD | Liabilities USD | Direct equity USD | Unclosed earnings USD | Total equity USD |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 2026-01-02 | 1000 | 0 | 1000 | 0 | 1000 | 0 | 1000 |
| 2026-01-31 | 1151 | 40 | 1191 | 300 | 1000 | −109 | 891 |
| 2026-02-02 | 1140 | 40 | 1180 | 300 | 1000 | −120 | 880 |

At Jan 31, cash is A-CASH-USD `1140` plus A-CASH-EUR functional `11`.
Expense net `+120` and revenue net `−11` yield unclosed earnings `−109`.
Assets `1191 = 300 + 891`. Jan 2→Jan 31 **point changes** are cash `+151`,
other assets `+40`, assets `+191`, liabilities `+300`, direct equity `0`,
unclosed earnings `−109`, total equity `−109`; these are not cash-flow categories.
Feb 2 reverses revenue and EUR bank value: assets `1180 = 300 + 880`.

Each account subtotal traces to exactly its listed J lines; the derived
unclosed-earnings line traces through revenue/expense accounts to J2/J5/J6.
For Jan 31, J6 is excluded because its accounting date is February. Selecting
J5's A-CASH-EUR GL line with Jan `[2026-01-01,2026-02-01)` and the report
revision must retrieve full J5, including both currencies and rate provenance.
The current pinned journal operation proves this one selected GL line, not
an unbounded subtotal-to-journal shortcut.

## Negative and boundary fixtures

| Fixture | Fixed expectation |
| --- | --- |
| Authorized empty book with supported policy | Assets/liabilities/equity all exact string `"0"`; `No posted entries`; cash `Not configured` if no membership policy; no journal links. |
| Book A has only source observation for USD 500, unposted draft or pending command | Jan 31 Position stays `1191` assets and `1151` classified cash; separate attention state may mention observation/command, never add USD 500. |
| Book B has J1-like USD 70 capital/cash | Book A remains unchanged; no A+B sum or selector crossing entity/book. |
| Missing credentials, revoked read or foreign book | 401/403/404 respectively; no financial totals, cursor or existence hint. |
| J7 posts after Position revision R was read | Later pinned GL/journal at R returns 409 and no rows; refresh obtains new revision. J7's amount is deliberately unspecified, so no stale result is asserted as current. |
| Source status older than reviewed threshold while book revision is unchanged | Posted Position remains labelled by its own cutoff/revision; separate source warning only. Threshold and feed are not yet approved. |
| Cash membership absent or ambiguous | Show `Not configured`, not total assets, guessed cash or zero; existing BS totals remain available. |

These numbers are hand-fixed expectations independent of a generator or posting
algorithm. Implementers must copy them into a synthetic fixture and test exact
strings plus original-unit detail. If the chosen development posting policy does
not support a fixture operation, split the fixture into an explicitly supported
policy test and a pending policy case; do not weaken the expected accounting math.

## Alternatives, uncertainty and acceptance

Comparing two separate unpinned HTTP reports risks a revision mismatch. The
existing one-response comparison keeps a coherent point change. A retained MVCC
session or materialized snapshot would permit historical navigation but adds
connection/storage and permission lifecycle work. Assigning cash by bank link or
account name avoids policy setup but confuses source accounts and book assets.
The proposed explicit classification has a review cost and needs versioned
ownership. Disconfirming evidence would be a supported cash classification
already in the actual schema or a real policy that defines cash differently;
recheck before implementation.

Acceptance: `TestPositionFixedExample`, `TestPositionCashPolicy`,
`TestPositionRevisionAndAuthorization`, `TestPositionSourceExclusion`,
`TestPositionBounds`, plus owner vocabulary review. These are **planned** tests,
not claims of present runtime coverage. #1 owns accountant-approved real-book
policy; #249 owns browser behavior. Revisit normative ISO mapping if Position
becomes an exchange or audit export.
