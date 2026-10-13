# Synthetic acquisition demo delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #97; bounded integration-only server fixture. Three fictional LLCs and 3,000
source transactions flow through the actual importer and explicit recovery gate.
Root owns composition, UI/shared scripts; this lane owns the test-only backend
and these contract/runbook files. No production route, cloud or provider mutation.

## Stage ledger

- Reviewed AGENTS, import/staging/decoder/recovery contracts and approved protocol.
- Added idempotent synthetic ownership bootstrap separated from runtime importer.
- Added request-driven stage/status handler with pending-command protection.
- Added independent cents/count checks, HTTP admission and actual SQL proof for
  all 3,000 rows, unrelated commands, lost commit replies and handler reconstruction.

## Current evidence

2026-10-13:

- `TestDemoAcquisitionGenerator` and `TestDemoAcquisitionHTTPBoundary`: PASS.
- `TestDemoAcquisitionPostgres`: PASS on actual disposable PostgreSQL 17.11 and
  private filesystem. All 3,000 observations, review items and provenance links
  were created through runtime importer/command execution; no journal, outbox
  message or book revision changed. Per-LLC counts and independent exact cent
  totals matched. Runtime has no table DML or administrative connection.
- Unrelated pending setup was left untouched; a real import COMMIT with an injected
  lost application reply resumed through pending-command discovery. Rebuilding
  the handler from idempotent setup/source/batch receipts completed once. This
  is not a TCP fault, provider delivery or whole-machine crash proof.
- First actual run caught an incorrect assumption about the existing pending
  wrapper; parsing now honors `{command:...}` before any step decisions.
- `make check test build`: PASS before final handoff.

Focused proof used migration 11 up SHA256
`b57974247dc00aea1408c9a152896858c0c7310f1f7603a70fb50abae8fe9ecd`.
The temporary migration copy/checksum was removed/restored before commit.
Logs: /private/tmp/jayburd-demo-backend-proof.log and
/private/tmp/jayburd-demo-backend-default.log. Root repeats relevant proof after
integrating the SQL lane's final version.

## Browser-discovered permission correction

The first integrated category-draft browser request exposed missing entity prepare
membership: setup grants only read/post. The fixture now explicitly provisions
prepare through its owner connection before runtime registration, without weakening
SQL authority or restoring an existing revoked grant. The extended actual
PostgreSQL proof passes: restricted-runtime category lookup and draft creation
succeed after all 3,000 imports, exact draft replay creates no second draft, and
journal/outbox counts, book revisions and source net remain unchanged. Root reruns
the browser workflow after integration.

## Remaining work

Root composes BrowserBoundary and private staging in the demo binary and integrates
UI Load/Resume loops plus browser proof. No background work runs after Pause.
The raw object root must survive any handler restart that retains its database.
No imported row is automatically categorized, drafted or posted by this loader.

## PR handoff

No push/deployment from this lane. Commit includes owned test-only Go files and
living runbook; root integrates SQL/composition and records combined browser proof.
