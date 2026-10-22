# Bounded Plaid page decoding

## Decision and boundary

Issue #102, coordinator approved interface and material-field policy before code,
2026-10-13. Implements only decoding of synthetic `/transactions/sync` responses
for the proposed `jayburd-plaid-transactions/1`, API pin `2020-09-14`. No HTTP client,
SDK, token, provider call, staging, complete-set publication or accounting authority.
The [provider proposal](provider-ingestion.md) remains authoritative for future
acquisition/fencing, whole-set lineage and live qualification.

`DecodePage(raw []byte, Expected) (Page,error)` accepts an approved account ID and
installed ISO currency references (code, UUID, minor unit, inclusive ValidFrom,
exclusive ValidTo; empty end means open). It never derives ownership from payload.
On any error return the zero Page: no partial rows, raw bytes, cursor or account ID.
Errors carry fixed codes only. Input slices are copied on success; callers own
private retention. Returned public structs are ordinary mutable Go values, not
signed attestations or SQL authorization credentials.

## Material mapping

Page JSON fields are `next_cursor,has_more,transactions_update_status,request_id,
sha256,added,modified,removed`. Raw exact page bytes are separately retained and
excluded from JSON. SHA256 identifies evidence but never substitutes for byte or
normalized-field equality. Arrays retain order. The decoder does not invent Item,
environment, stream, fence, page index or request cursor; trusted acquisition must
attach these separately. A complete-set assembler, not this decoder, owns cross-
page identity/lineage and cursor commit. `HasMore=false` is not history readiness.

Each added/modified transaction outputs `account_id,transaction_id,pending,
pending_transaction_id,date,authorized_date,datetime,authorized_datetime,
iso_currency_code,currency_version_id,amount_lexeme,amount,direction,
eligible_for_review,name,merchant_name,original_description`. Nullable fields use
null; omitted optional dates/names normalize to null with exact omission retained
in raw evidence. Pending link is required but nullable. Raw exact transaction
object is separately retained and excluded from JSON. Removed entries expose only
account/transaction IDs, with separately retained raw object.

Positive signed source numbers map to outflow, negative to inflow. Zero, including
negative/exponent zero, retains its original lexeme, unsigned amount `0`, direction
`zero` and readiness false. Pending remains pending, with readiness false. Ordinary
nonzero, nonpending rows are source-ready even with no pending link: readiness
**never** grants journal posting or accounting approval. A pending row cannot
claim a predecessor; self-links reject. Link target existence/removal/state cannot
be established from a single page and stays a complete-set responsibility.

Source date and authorized date are separate full dates; timestamps retain their
original offsets/text with no timezone conversion. Description remains inert source
text. None becomes an approved journal date, category, FX rate or ledger account.
Only exactly one installed currency version valid on source date may resolve;
unofficial/missing ISO and ambiguous or unavailable versions reject the whole page.

## Parser profile and bounds

Issue #258's subsequent [source precision decision](plaid-source-precision.md)
supersedes the currency-minor-unit rejection below: retain exact bounded source
amounts but mark them ineligible and hold their review rows. Magnitude and
nine-decimal storage bounds still reject. Historical diagnostics remain readable.

Issue #258 refines diagnostics without changing the accepted quantity set.
`amount_magnitude_exceeded` means more than 38 significant or 29 integer digits;
`amount_scale_exceeded` means normalized scale above nine;
`currency_precision_exceeded` means a quantity within storage limits that is
not an integral minor unit of the dated ISO 4217 currency. These fixed codes
replace new emissions of the ambiguous `amount_out_of_range`; old receipts
remain readable. No source amount, identifier or payload fragment enters a
diagnostic. SQL retains its exact numeric/lexeme equality and minor-unit checks.
Alternative rounding would lose evidence and is rejected; accepting arbitrary
extra precision would change reviewed storage/accounting contracts. Named proof:
`TestPlaidMoneyDiagnostics`, existing independent-rational fuzz oracle and
`TestDeliveryManualRecovery` for durable classified quarantine/recovery. This is
a diagnostic improvement, not a confirmed repair of the hosted checking cause.

Engineering bounds, not claims about all legal provider payloads:

- 2 MiB exact response, 500 total added+modified+removed changes, nesting depth 32.
- Required arrays are arrays, not null. Accounts contains at most the approved
  account; added/modified require its metadata and type depository. Account mismatch
  in metadata, transactions or tombstones rejects all data.
- IDs are case-sensitive graphic ASCII 1–256 bytes. No URL interpretation or trim.
  Next cursor uses a bounded base64/base64url alphabet, max 256 bytes. Empty cursor
  only for empty no-more initialization, independently of Item update status.
- Required boolean/pending fields cannot be null. Unknown update/payment-channel/
  transaction-type/account-type enums fail closed. Known UNKNOWN update status
  remains explicitly unknown and does not imply completion.
- Strict duplicate decoded keys at every depth, trailing JSON/data, malformed JSON,
  invalid UTF8, lone UTF16 surrogate and decoded NUL are rejected.
- All JSON number tokens, including inert metadata, max 128 bytes and explicit
  exponent in [-100,100]. Amounts remain number lexemes; strings are not accepted.
  Decimal-position arithmetic normalizes zeros without floats, then requires at
  most 38 significant digits, 29 integer digits and 9 fractional digits, plus exact
  installed currency minor precision. Long insignificant zero representations
  within 128 bytes are accepted when normalized quantity fits; never round.
- Name required 1–500 Unicode codepoints; optional merchant/original text max 500.
  Full dates 0001–9999 use strict Gregorian calendar. Timestamps require seconds,
  optional 1–9 fractional digits and explicit offset/Z; leap seconds unsupported.
- Repeated transaction identity within or across added/modified/removed on one
  page rejects. Distinct IDs with identical amount/date/name remain distinct.

Root/account/transaction/removed unknown fields reject. Root allows only the
listed response fields plus accounts. Transaction inert allowlist: account_owner,
check_number, location, payment_meta, personal_finance_category,
personal_finance_category_icon_url, merchant_entity_id, merchant_category_code,
logo_url, website, counterparties, category, category_id; payment_channel and
transaction_type additionally validate known enums. Account inert allowlist:
subtype, balances, mask, name, official_name, verification_status,
verification_name, verification_insights, persistent_account_id, apy,
holder_category. Inert values preserve arbitrary bounded nested JSON (including
unknown nested fields) but contribute no mapped meaning; global duplicate-key,
depth and numeric checks still apply. Account balances are not ledger balances.
`running_balance` accepts null or a JSON number retained only in raw evidence;
its exact lexeme is never converted to float or treated as a ledger balance. This narrower profile may quarantine valid new provider fields.

## Standards and refreshed primary sources

Plaid JSON remains the explicit proprietary-adapter exception described in the
[standards contract](standards.md), not ISO 20022/camt. Installed ISO 4217 currency
versions and RFC 3339 full-date/time syntax retain the existing reference scope.
No new ISO normative text, bank data or provider environment was accessed.

Selected official sections refreshed 2026-10-13:

| Source | Reviewed sections and factual evidence | Project interpretation |
| --- | --- | --- |
| [Plaid Transactions API](https://plaid.com/docs/api/products/transactions/) | Sync account filter/count/pagination; response accounts, amount/currency, date/timestamps, pending/link, transaction ID, removed, cursor, update status. Amount direction is provider-defined; pending links may be absent; cursors are account-specific; all pages matter. | Exact lexemes, separate dates/state, explicit same-account check; no pending-to-journal shortcut. |
| [Plaid API versioning](https://plaid.com/docs/api/versioning/) | Nonbreaking changes, explicit version header, 2020-09-14. Additive fields/enums and identifier changes can occur without a version change. | Fixed reviewed allowlists may intentionally quarantine additions; pinning alone is insufficient. |

This refresh supplements PI01/PI03/PI08 in the
[source register](../research/plaid-ingestion-sources.md). Reviewed selected sections,
not every field or provider guarantee. Alternative float SDK decoding loses original
number tokens; unrestricted unknown-field acceptance conceals material changes;
rejecting all inert nested additions is stricter but unnecessarily broad for fields
with no financial interpretation. Tests prove this local parser profile only.

## Transaction classification compatibility — 2026-10-14

Issue #118: accept optional `transaction_code` as null or a string of at most
256 Unicode codepoints (our bound). Retain it only in private raw evidence;
never derive direction, ledger category or posting authority from it. Unknown
classification strings remain inert, like other unmapped metadata. Wrong JSON
types reject. Unknown field names and all money/account checks still reject.
The proprietary adapter/ISO applicability above is unchanged.

PI01 documents this nullable classification even where no institution code is
available. Synthetic reproduction proved our previous allowlist rejected its
null form. This is a confirmed compatibility defect, not proof of the cause of
the earlier hosted rejection. Rejecting the documented field prevents valid
pages; accepting every unknown field would remove the material-schema boundary.
`TestPlaidTransactionCode` verifies the narrow alternative, exact cents, raw
retention, omitted normalized mapping, invalid types and account isolation.

## Cursor diagnostic refinement — 2026-10-14

Hosted retest after #120 returned decoder_invalid_cursor. PI01's request cursor
section still documents an upper bound of 256 base64 characters; its response
next_cursor section permits empty text when transactions are not yet available.
Update status describes the Item. We cannot infer from the old error whether
encoding or our empty-progress rule rejected this response. Keep both rules
unchanged and classify them as invalid_cursor_encoding or invalid_cursor_progress.
TestProbeFailuresStayPartialOrBlocked exercises both through the probe. No raw
cursor, length or provider status is added to the receipt.

## Account readiness correction — 2026-10-14

PI01 describes transactions_update_status at Item scope, next_cursor as empty
when transactions are unavailable, and account_id as a separate filtered stream.
Therefore Item completion cannot by itself require a filtered account cursor.
Accept empty cursor only with no changes and has_more=false for any known Item
status. The diagnostic still reports provider_not_ready and never passes sync
without a usable cursor. Unknown status, invalid cursor encoding, changes or
has_more with an empty cursor still reject. This narrow interpretation avoids
both declaring history complete and loosening cursor encoding constraints.
TestPlaidEmptyAccountReadiness covers all four known statuses and rejects all
three change groups/has_more without a cursor; TestHostedScopeAndCleanup covers
one initial wait followed by sync, repeat delta and single Item cleanup.
The hosted progress code identifies the branch but does not expose which
predicate fired; this documented contract correction still requires hosted proof.

## Public schema compatibility review — 2026-10-14

PI11 reviewed every direct field in TransactionsSyncResponse, AccountBase,
Transaction (including TransactionBase through allOf) and RemovedTransaction at
public schema commit 325e2e192bcb422df708029bafe9d950c94df2fd. Root, account and
removed allowlists cover their declared fields. Three remaining transaction fields
needed explicit disposition: business_finance_category and client_customization
are optional null/object enrichment retained only as private raw evidence;
running_balance accepts null or a JSON number. Owner explicitly approved retaining
all reviewed fields on 2026-10-14. Global numeric lexical/depth/byte bounds remain;
balance strings, objects, arrays and booleans reject. Exact source number bytes
remain private evidence, with no currency conversion, rounding or ledger mapping. This does not infer ledger
categories, identities or balances from enrichment. Unknown new field names still
reject, as do scalar/array enrichment values. Existing nested JSON bounds apply.

TestPlaidReviewedEnrichment covers both enrichment forms, exact cents, account
isolation, no normalized metadata leakage, exact balance lexemes beyond float
precision, malformed balance types and unknown fields.
TestPlaidFullPublicSchemaFixture uses fictional values for all declared direct
fields: 8 response, 13 account, 33 transaction, 2 removal fields.
Public schema compatibility does not prove which field a private hosted page
contained; a successful hosted sync remains required. Proprietary API and ISO
applicability are unchanged; no financial message or accounting policy is added.
