# Product workspace and collection foundations

<!-- stage-runbook: true -->

## Status and boundary

Build on merged #79 (`5490f6d`). Owner requested product-quality navigation and
parallel progress toward data collection. Current bounded children #80/#81/#82
cover the workspace, source registry and private synthetic staging.

## Stage ledger

- Checked all prior agent threads were completed; reused three stopped threads.
- Created separate worktrees and exclusive scopes. Root reserves integration,
  planning, shared CI and schema-version fixtures. Source registry is sole SQL writer.
- Reviewed UI component/navigation plan, owner-only synthetic source enrollment
  and immutable staging API/crash/cancellation boundaries before implementation.
- Published updates to #8/#9/#50/#52/#55/#57/#58/#65/#70, preserving original
  acceptance and checking for concurrent edits. Added native sub-issue links
  #80→#52 and #81/#82→#8.
- Integrated staging agent `1a7019d` as `280e5cf`; macOS race and isolated Linux
  evidence passed. Source review corrected revocation of dormant reader grants.
- Integrated UI `bc46687` as `3c3cc0e` and source registry `ff1615e` as `4e95718`.
- Inspected desktop/mobile workspace screenshots; balanced fixture rows and
  workspace-versus-report revision wording were corrected and recaptured.
- Independent source review found no blocking design issue; malformed numeric
  JSON now maps to the fixed error profile with a regression fixture.
- Added source-registry CI gate and schema9 stale/newer-version readiness fixtures.

## Current evidence

Combined `make check test build migration-test ledger-test cloud-sql-test
observer-test source-registry-test`: PASS. Migration9 alone is new; migration1–8
checksums remain unchanged. Restricted migration/readiness and existing posting,
reporting and observer regression gates passed. Source registry has 13 actual
PostgreSQL test groups; [named evidence](source-registry.md).

Unchanged integrated UI source passed `make web-check web-test web-build
web-browser`: 151 units and ten browser journeys; [named evidence](book-workspace.md).
The combined `make app-integration` passed browser → production Next bridge → Go
→ PostgreSQL setup/post/lost-response recovery, with exactly one journal. Initial
alternate-port run missed the Playwright proxy port; corrected the local override
and reran successfully. All four harness overrides restored byte-for-byte. No
product fix or authentication bypass was introduced for that environment failure.
Workflow lint (actionlint1.7.7) passed.

[Staging evidence](import-staging.md): 13 groups on macOS with race checks and in
an isolated Linux arm64 container, including real child-process crash recovery.
These are process/filesystem tests, not hardware durability or encryption proof.

Latest synthetic review runs separately at localhost:39383 using this batch's
runtime changes. HTTP200 verified and native browser opened. Existing review
servers were preserved. User approval of the new experience remains separate
from automated checks and agent screenshot inspection.

## Remaining work

Review/merge this batch. Next implement #8 batch/preview publication and explicit
recoverable import into review items, then source-linked posting. Continue #7
private transport and #70 provider acquisition; no uploads or live data yet.

## PR handoff

All bounded implementation and local validation complete. GitHub CI/owner review
remain the handoff gates. Migrations1–8 unchanged; migration9 is forward-only for
product delivery. No cloud activation, real source enrollment, upload endpoint or
provider connection. #8/#50/#52/#55/#57 remain open for broader acceptance.
