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

Merge and publish the worker image, deploy its digest, then rerun the authorized
hosted Sandbox probe. Keep #118 open until a new receipt establishes the outcome.
Interactive Link, SQL ingestion and webhook qualification remain under #88.

## PR handoff

Targets main; no migrations or accounting-policy changes. Existing response
profile remains bounded. Review [decoder contract](../contracts/plaid-decoder.md)
and [diagnostic contract](../contracts/plaid-sandbox-qualification.md).
