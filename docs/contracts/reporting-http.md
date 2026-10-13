# Financial reporting HTTP

## Decision and reviewed scope

Coordinator approved this JSON-only refinement of [financial reporting](financial-reporting.md)
on 2026-10-13 before implementation. SQL and monetary semantics remain owned by
[reporting SQL](reporting-sql.md). No SQL migration, CSV, provider, new accounting
policy or live-use approval. OpenAPI is the typed transport authority.

## Wire contract

Four additive GET routes below `/v1/entities/{entity}/books/{book}/reports/`:
trial-balance, general-ledger, balance-sheet and profit-and-loss. The old
trial-balance route remains unchanged. Verified bearer identity supplies the actor;
no caller actor field. Every request authenticates before parsing/report access.
SQL authorizes current scope before checking snapshot revisions or aggregating.

Query names and required date/account combinations follow the SQL contract.
Unknown/duplicate/empty fields, malformed URL escapes, request bodies, HEAD and
noncanonical integer limits reject with 400 invalid_request. The complete request URI caps at 4,096 bytes, matching the existing server and
bridge. The transformed SQL request additionally caps at 8,192 bytes. SQL validates dates, UUIDs, revisions, allowed
combinations, policies and cursor membership. Limit defaults 100 and caps 500.

Response preserves the SQL envelope and exact string amounts, with
contract_version financial-reporting-v1 and layout: account-v1 for TB/P&L,
posting-v1 for GL, basic-unclassified-v1 for BS. Layout is fixed, never caller
selected. Existing SQL provenance, totals, comparisons and selectors are unchanged.
Responses buffer entirely, including encoding, before headers; maximum 2 MiB,
Cache-Control:no-store. Service returns only after successful transaction commit.

Cursor is unpadded canonical base64url of JSON containing exactly version, report,
layout, revision, filter_hash and last_key. Version is financial-reporting-v1;
report/layout must match route; revision must match explicit snapshot_revision.
Maximum 2,048 encoded bytes. Duplicate/unknown/missing keys, alternate encodings,
CRLF, padding, invalid UTF-8, unpaired UTF-16 surrogate escapes and
trailing JSON reject. Generated cursors retain valid Unicode as UTF-8 directly. Decode only transports revision/hash/key
into SQL, which binds entity/book/normalized filters and validates key membership.
No signature or access grant is inferred. Every page reauthorizes.

Errors reuse safe problem responses: 400 syntax, 401 identity, 403 insufficient read,
404 inaccessible scope/account, 409 report_snapshot_changed, 422 unsupported policy
or result_out_of_range, 503 deadline/retry failure, 500 unexpected fault. No rows,
SQL, cursor or token in errors/logs. Existing 9-second service budget includes at most
two attempts, with 5-second statement/1-second lock timeouts. No partial success.

## Sources and applicability

Existing FR01–05 and financial-reporting.md standards assessment apply unchanged:
internal application reporting JSON is not a selected ISO 20022 network message;
ISO 4217 reference identity stays preserved; normative audit mappings remain deferred.
No blanket ISO/GAAP claim. RH01: [RFC 4648](https://www.rfc-editor.org/rfc/rfc4648),
October 2006, sections 3.2/3.3/3.5/5 reviewed 2026-10-13: padding may be excluded by
an explicit referring profile; reject nonalphabet bytes and noncanonical pad bits.
RH02: [Go encoding/base64](https://pkg.go.dev/encoding/base64), RawURLEncoding,
Strict and DecodeString reviewed 2026-10-13. Strict still ignores CR/LF; exact
re-encoding equality closes that gap. These sources specify transport, not accounting.
Opaque server-held cursors would add durable state; plain query JSON permits awkward
escaping. Bounded base64url with SQL revalidation fits the current revision model.

## Evidence and remaining work

TestReportingHTTPContract covers transport bounds/redaction/auth/cursor mapping;
TestReportingHTTPPostgres covers actual scoped reports/pagination/revision/revocation
through the handler. Existing SQL tests own golden balances and 1,024 posting proof.
See [living evidence](../runbooks/reporting-http.md). Foundation materialization
caps, supported synthetic USD functional policy, absent CSV/full FX and hosted
performance gates remain unchanged. JSON route delivery does not complete #51/#53.
