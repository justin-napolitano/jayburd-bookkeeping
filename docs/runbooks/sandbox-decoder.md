<!-- stage-runbook: true -->
# Sandbox decoder compatibility

## Status and boundary

Issue #118 decoder blocker resolved by hosted execution
`jayburd-dev-sandbox-probe-g6wfc`: authentication PASS, initial sync PASS (one page,
six changes), repeat delta PASS (one page, zero changes), cleanup PASS. Overall
receipt remains PARTIAL: historical completeness, interactive Link, application
SQL import, pending transitions and real signed webhook delivery remain unqualified
under #88. No provider payloads or credential values were inspected.

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

Continue #88 with the reviewed Link/session association path and application
acquisition-to-review persistence, then dynamic pending transitions and signed
webhook delivery. Historical completeness was not established by this smoke run.
Six decoded changes are not six posted journals or verified accounting balances.

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

## Hosted cursor diagnosis and readiness correction — 2026-10-14

- [Build 38637872109](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38637872109): PASS for merged commit 309f19d961c9cb86fdc61a2083d1bcc1f5c87be2.
- Worker digest: sha256:15b10c0a7ed68c47464fec440684014874f3c44b588bb3f65350eed8b82dd88f.
- Terraform twice failed during metadata refresh with a network socket error.
  Applied the same desired image using gcloud jobs update; verified image, identity,
  pins 1/1 and retries zero. Desired Terraform input matches; state reconciliation
  remains pending restored metadata access. No IAM or secret value changes.
- [Invocation 38639677554](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38639677554) submitted jayburd-dev-sandbox-probe-k4z8f.
- Actual receipt: authentication PASS, initial sync BLOCKED
  decoder_invalid_cursor_progress, zero decoded pages/changes, cleanup PASS.
- Narrow source-backed correction: empty no-change/no-more account snapshots reach
  existing bounded readiness polling regardless of Item status. Cursor required for
  changes or further pages. Full hosted sync is not yet proven.
- PASS: make check test build and git diff --check; named regression tests are in the
  [decoder contract](../contracts/plaid-decoder.md).

Next: merge readiness correction, publish/deploy its image and repeat the authorized
probe. Keep #118 open; this execution is diagnostic evidence, not successful sync.

## Readiness retest and public-schema review — 2026-10-14

- [Build 38641168793](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38641168793): all validation/image jobs PASS, commit 09215afc45cacfb9d8bdeda58d627b476bd0aef3.
- Worker digest: sha256:dc5582f1f39d7393108d35151e6a3f9444ed01b5d4089426c8a0e9e7ab4b73a2.
- Terraform apply PASS; prior manual deployment reconciled. Verified image,
  dedicated identity, pins 1/1 and retries zero. Delivery remains lookup-only.
- [Invocation 38642580248](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38642580248) submitted jayburd-dev-sandbox-probe-r295g.
- Receipt: authentication PASS; initial sync BLOCKED decoder_unknown_field,
  zero decoded pages/changes; cleanup PASS. No successful sync or SQL import.
- Reviewed the complete direct public schema field inventory (PI11). Added two
  inert enrichment fields and exact numeric/null running balance retention, per
  owner direction to retain all reviewed fields for future use. No ledger mapping. The actual unknown field is not exposed by the receipt.
- PASS: make check test build and git diff --check. TestPlaidReviewedEnrichment
  and TestPlaidFullPublicSchemaFixture cover exact retention and all declared
  direct fields: 8 response, 13 account, 33 transaction, 2 removal fields.

Next: merge schema compatibility change, publish/deploy and rerun the authorized
probe. Keep #118 open. All raw provider data and credentials remain uninspected.

## Successful hosted smoke — 2026-10-14

- [Build 38648733412](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38648733412): all validation/image jobs PASS, commit 4e0c192595f2f50ced65d4fe66c9b270e0fab5a2.
- Worker digest: sha256:caf1001dc84b0850f5bf08b0e79e15d2f563e03860cd63207ad7a53fe2540b07.
- Terraform apply PASS. Verified deployed digest, dedicated identity, credential
  pins 1/1 and retries zero. Delivery remains lookup-only; no write grant restored.
- [Invocation 38650337250](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38650337250) submitted jayburd-dev-sandbox-probe-g6wfc.
- Cloud Run succeeded at 2026-10-14T22:52:26Z. Receipt: authentication PASS;
  initial sync exact_decode_complete, one page/six changes; repeat sync
  empty_delta_observed, one page/zero changes; cleanup removal_acknowledged PASS.
- Overall PARTIAL: historical_readiness reports history_not_complete; four
  application qualification checks remain blocked/application_proof_required.
- This resolves the decoder rejection observed in #118. It does not establish
  the exact unknown field present in earlier private responses or live-bank readiness.

PASS: make check test build, git diff --check and staged runbook gate.
