# Draft ledger API

`openapi.json` is an OpenAPI 3.1.1 review artifact; `post-journal.example.json`
is a synthetic request, also embedded in the specification for lint validation.
No application server or financial integration exists. Read the
[HTTP semantics](../docs/contracts/api.md) and [SQL contract](../docs/contracts/ledger-sql.md)
together before implementation.

Validate with `npx --yes @redocly/cli@2.62.1 lint api/openapi.json`.
This optional review tool may download packages; it is not part of the default
credential-free Go build and does not call financial providers. A missing-license
metadata warning is expected: no API license grant has been chosen. Do not add a
license merely to silence lint. Structural validity does not prove accounting,
authorization, concurrent transaction behavior or ISO conformance.
