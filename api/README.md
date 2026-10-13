# Ledger and reporting API

`openapi.json` is the OpenAPI 3.1.1 contract for implemented ledger operations and
bounded JSON reports, pinned journal detail and internal review CSV. Development
accounting policy and deployment evidence are separate. Close and accountant/audit
exports remain deferred. Read [HTTP semantics](../docs/contracts/api.md),
[report transport](../docs/contracts/reporting-http.md) and [SQL reporting](../docs/contracts/reporting-sql.md).

Report review tools follow the [pinned journal contract](../docs/contracts/report-drill.md)
and [fixed CSV profile](../docs/contracts/report-export.md). Both require an explicit
report revision. CSV exports the complete bounded selection; it rejects page
controls and omits arbitrary labels. It is not a mapped ISO audit exchange.

Generate clients with `npm --prefix web run api:generate`. Validate with
`npx --yes @redocly/cli@2.62.1 lint api/openapi.json`. This optional tool may download
packages; default Go checks remain credential-free. Missing-license metadata
warning is expected; no license grant is inferred. Structural validity does not
prove accounting, authorization, concurrent behavior or ISO conformance.

Import confirmation uses the existing saved-command endpoints with operation `import`;
see `ImportCommandBody`/`ImportCommandResult` and the durable import contract. This
adds no public upload, preview publication or arbitrary-row endpoint. The current
workspace supports safe recovery/cancellation/acknowledgement; it does not confirm
an import without a statement-preview client.
