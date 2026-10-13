# Synthetic statement decoder

## Decision and boundary

Issue #73 implements only the pure Go decoding portion of
[the statement proposal](statement-import.md). Owner direction permits this
synthetic internal CSV starting point; broader #8 approval gates remain open.
This is not an ISO 20022 message or a bank adapter. Source rationale, syntax and
standards limits remain [SI01–SI06](../research/statement-import-sources.md).

## Exact API

`ingestion.Decode(io.Reader, Expectations) (Statement, error)` consumes at most
1 MiB + one sentinel byte. Expectations supplies an exact source account reference,
statement starts-on/ends-before dates and one currency snapshot: code, immutable
version ID, minor unit (0–9), valid-from and optional valid-to (exclusive).
Source account references match `[A-Za-z0-9_-]{1,64}`. Currency version ID
is a lowercase hyphenated UUID and cannot be all zero. Both interval boundaries
must be valid, with starts-on strictly before ends-before; currency validity
has the same ordering rule. All dates are four-digit Gregorian full dates,
years 0001–9999. Invalid caller
expectations fail before reading. The caller owns reader deadlines/cancellation;
a byte limit cannot interrupt a blocking reader. The decoder validates equality, not authority;
callers remain responsible for approved metadata and binding permissions.

A successful Statement contains profile version, exact byte SHA-256 and byte count,
plus rows in original order. Each row retains its one-based physical record ordinal
(header is 1), seven decoded fields, original amount lexeme, canonical amount and
currency version ID. Canonical amount removes leading integer zeros and trailing
fractional zeros without rounding. No totals, classifications or functional
currency conversions are computed. No raw file is retained by the result.

Failure returns a zero Statement and `*DecodeError` with bounded Diagnostics
(code, record ordinal, field name), total error count and truncated flag. No raw
values, input reader errors or paths enter error strings. Record 0 means file or
expectation error. Structural failures stop immediately with one structural diagnostic (earlier
semantic diagnostics are discarded); semantic errors collect
at most 100 diagnostics while counting errors across the bounded file. An I/O
failure rejects all rows. No usable partial success is returned.

## Profile clarifications

Exact unquoted header bytes are required. Files contain 1–1000 data records;
header-only input is rejected. CRLF separates physical records; final CRLF is
optional. The 2048-byte physical record cap excludes CRLF and applies to header
and rows. Quoted commas and doubled quotes are accepted; embedded record breaks,
blank rows, BOM, controls and non-ASCII bytes are rejected. No trimming occurs.

Field bounds and account/date/sign/currency rules follow the statement proposal.
Amount limits count lexical digits (29 integer, 9 fractional), including zeros;
insignificant fractional zeros may exceed minor unit within the nine-digit cap.
Duplicate IDs reject even identical rows; different IDs preserve identical
purchases. One supplied currency version must cover each booking date; there is
no default, metadata lookup or cross-file duplicate handling.

## Implementation evidence and alternatives

Coordinator reviewed this API before implementation on 2026-10-13. The Go
[encoding/csv reference](https://pkg.go.dev/encoding/csv), Overview and Reader
fields, reviewed 2026-10-13, documents blank-line skipping, CRLF normalization
and multiline quoted fields. Local implementation uses the repository-pinned
Go 1.26.9 standard library. Our inference: prevalidate bytes and physical records,
then use its strict quoted-field parser per line. Using the reader alone would
accept inputs this profile prohibits; writing a second quoting parser would add
unnecessary ambiguity and maintenance. No external parser dependency is added.

## Acceptance

Named tests cover profile quoting, valid boundaries, 0/2/3-decimal exact money,
invalid caller expectations, invalid fields/dates/account, duplicate identity,
size/record/error bounds and I/O failure redaction. They use independent synthetic
expected values. Current evidence belongs to the
[living runbook](../runbooks/statement-decoder.md). No persistence, draft, API,
provider, SQL or posting behavior is implemented by this package.
