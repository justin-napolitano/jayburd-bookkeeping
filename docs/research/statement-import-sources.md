# Statement-import sources

Reviewed 2026-10-13 for [issue #61](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/61).
The [proposed contract](../contracts/statement-import.md) separates source facts
from project choices. No source below approves a live bank adapter.

| ID / publisher / version / reviewed sections | Established fact | Project inference and limits |
| --- | --- | --- |
| SI01 — Y. Shafranovich / RFC Editor, [RFC 4180](https://www.rfc-editor.org/rfc/rfc4180), October 2005, §§2–3, complete sections | Documents comma separation, quoted fields/escaped quotes, CRLF records and text/csv parameters; it is Informational. | Specify an intentionally narrower internal template, not arbitrary bank CSV. Mandatory header, ASCII-only fixture text and prohibited multiline fields are our profile restrictions. RFC 4180 is not an ISO financial-message definition. |
| SI02 — Klyne/Newman / RFC Editor, [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339.html), July 2002, §5.6 grammar and §5.7 restrictions | Defines full-date components and calendar restrictions within an Internet timestamp profile of ISO 8601. | Use valid YYYY-MM-DD booking dates without inventing a timezone conversion; receipt/audit instants remain UTC timestamps. This is not review of the full ISO 8601 normative text. |
| SI03 — ISO 20022 Registration Authority, [message catalogue](https://www.iso20022.org/catalogue-messages), documentation/version overview | Registered messages have definitions, versions and supporting artifacts. | camt statement exchange is the longer-term integration candidate. No particular bank usage profile or official XSD was verified here; [#17](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/17) remains open. Our synthetic CSV cannot claim camt/ISO 20022 conformance. |
| SI04 — SIX / ISO 4217 Maintenance Agency, [Global Financial Data Standards](https://www.six-group.com/en/products-services/financial-information/market-reference-data/data-standards.html), “We Maintain and Update the ISO Currency Codes” and “Current and Historical Lists” | The maintenance agency supplies current/historical currency lists and amendments. | Bind imported code/date to installed versioned metadata. No new list snapshot was downloaded or licensed/pinned here; reuse reviewed synthetic reference records and require source-release evidence for a real adapter. Minor units do not establish FX valuation or recognition policy. |
| SI05 — OWASP, [File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html), Introduction, Content-Type Validation, File Storage Location, User Permissions, Upload and Download Limits | Uploads require layered type/content checks, authorization, protected storage and bounded resources; client MIME is insufficient proof. | Restrict plaintext synthetic input, disallow archives/network fetch, store private provenance and redact diagnostics. Our numerical limits and retention durations are engineering proposals, not OWASP requirements. No external scanning service receives financial content. |
| SI06 — OpenStax, [Principles of Accounting, Volume 1, §8.6](https://openstax.org/books/principles-financial-accounting/pages/8-6-define-the-purpose-of-a-bank-reconciliation-and-prepare-a-bank-reconciliation-and-its-associated-journal-entries), textbook section reviewed | Bank and company records can differ; reconciliation identifies timing differences and items requiring book adjustments. | A booked statement item is evidence, not authorization to create a journal or recognize an expense. No reconciliation-complete claim without balances/outstanding-item rules. Educational source, not entity-specific GAAP/tax policy. |

## Alternatives and revisit conditions

A bank-specific CSV can be convenient but needs that bank's sign, identity,
correction and date documentation. OFX likewise needs a selected version/profile;
neither was reviewed as a supported adapter. camt carries richer standardized
semantics, but official artifact and usage-profile review remain unresolved.
Propose one internal synthetic CSV to test ingestion boundaries first; require an
explicit standards-deviation decision before implementation, not implied approval
from this source register. Revisit when #8 selects an actual bank/interface.
