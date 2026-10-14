<!-- stage-runbook: true -->
# Sandbox decoder compatibility

## Status and boundary

Issue #118. Existing keys authenticated in hosted execution
`jayburd-dev-sandbox-probe-k6cd9`; initial decoding blocked and Item cleanup
succeeded. The old generic reason cannot establish its exact cause. This branch
fixes a source-confirmed transaction_code compatibility gap and adds redacted
classifications. No provider payloads or credentials were inspected.

## Stage ledger

- Reviewed PI01's documented nullable classification and the decoder contract.
- Added bounded inert classification, retaining exact money/account checks.
- Added closed error classification; arbitrary errors remain generic.

## Current evidence

PASS: `make check test build`, plus `git diff --check`. The first test attempt
hit sandbox loopback binding restrictions; the permitted rerun passed.
Named tests: TestPlaidTransactionCode, TestDecoderReasonRedaction,
TestProbeFailuresStayPartialOrBlocked. Fixtures are synthetic and network-free.

## Remaining work

The #120 worker was built, deployed and tested; initial sync still blocks.
Keep #118 open. Merge this diagnostic refinement, publish its worker image and
rerun the authorized probe to distinguish cursor encoding from empty progress.
Interactive Link, SQL ingestion and webhook qualification remain under #88.

## PR handoff

Targets main; no migrations or accounting-policy changes. Existing response
profile remains bounded. Review [decoder contract](../contracts/plaid-decoder.md)
and [diagnostic contract](../contracts/plaid-sandbox-qualification.md).

## Hosted retest — 2026-10-14

- Merge commit: 796a73ab0a5abfb26fadde11330daeb20a623f8d.
- [Build 38635261732](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38635261732): all validation and four image builds PASS.
- Worker digest: sha256:f3b7181b43d1675ebd66e76ec586a51581dbc6928f1b922905dfddf9431f5932.
- Terraform apply PASS; only probe image changed. Numeric credential pins remain
  1/1, dedicated read-only identity, retries zero; delivery remains lookup-only.
- [Invocation 38637132583](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38637132583) submitted execution jayburd-dev-sandbox-probe-bk9vn.
- Actual execution failed: authentication PASS, initial sync BLOCKED
  decoder_invalid_cursor, zero decoded pages/changes, cleanup PASS.
- Zero decoded pages is not evidence of an empty account. No SQL imports occurred.
- Refined the two cursor predicates into separate redacted error codes without
  changing acceptance. Synthetic tests cover both; `make check test build` and diff checks PASS.
