<!-- stage-runbook: true -->
# Automatic account binding delivery

## Status and boundary

Issue #209 owns automatic account creation for the exact cash-like depository
allowlist. Issue #210 separates credit/loan liability policy. AB01 records Plaid's
six top-level types and depository subtypes. No provider call, journal, opening
balance or live-account approval is part of this delivery.

## Stage ledger

1. **Issue and decision — complete.** The accepted contract selects the explicit
   subtype allowlist and atomic account creation inside the existing idempotent
   binding command over pre-creation or a two-command workflow.
2. **Database command — complete.** Migration 22 extends the restricted manifest
   and binding functions. Fresh install, replay and latest down/up passed with the
   exact pinned locally cached images. Focused SQL tests prove three-account
   creation, same-key replay, unsupported-subtype rollback and zero journals.
   Legacy manifests without subtype remain valid for existing-account binding;
   only automatic account creation requires an allowlisted subtype.
3. **Service and browser review — complete.** Go/OpenAPI/browser changes preserve
   existing-account requests, carry bounded subtype metadata, propose distinct
   accounts for every eligible account and keep unsupported types visible.
4. **Integrated evidence — complete.** The synthetic browser→API→PostgreSQL
   journey binds checking, savings and money-market accounts to three distinct
   newly created ledger accounts. It observes three source accounts, three active
   streams and zero enrollment journals before continuing through acquisition,
   explicit posting and response-loss recovery.
5. **Hosted acceptance — in progress.** Build run 39355209782 passed repeated
   repository gates, exact-image tests and vulnerability scans for merge commit
   `dfc1772`. Migration-only deployment 39357128631 applied migration 22
   successfully. Matching runtime deployment 39358665783 and its bounded
   anonymous transport verification succeeded. A fresh owner-driven fictional
   Plaid Sandbox enrollment is the remaining hosted acceptance evidence.

## Current evidence

- #209 and #210 record the separated implementation and accounting-policy scopes.
- `docs/research/automatic-account-binding-sources.md` records the selected Plaid
  type/subtype schema, product-matrix review, project interpretation and limits.
- `docs/contracts/automatic-account-binding.md` records the accepted request,
  classification, atomicity, recovery, standards and acceptance boundaries.
- `TestConnectionAutoAccountBinding` and `TestConnectionAutoAccountAtomicity`
  pass against disposable PostgreSQL with migration 22.
- The connection-service suite passes, including legacy credential, lifecycle,
  cancellation and HTTP adapter cases. Frontend unit/type/OpenAPI checks pass with
  284 tests, including three distinct automatic proposals and an unbound credit.
- The connection-state suite passes all legacy manifest, reconnect, rotation,
  expiry, privilege and malformed-container cases after migration 22.
- `make app-integration` passes, and its focused `--connections` journey proves
  the complete three-account enrollment, acquisition, posting and recovery path.
- `make check`, `make test`, `make build` and `make web-browser` pass; the browser
  suite reports 17 passing scenarios.
- Build run 39355209782 published the exact tested image digests. Migration run
  39357128631 completed before runtime run 39358665783, which passed bounded
  anonymous transport verification.

## Remaining work

Repeat the fresh three-account Sandbox enrollment and record only bounded results.
Hosted success must omit provider identifiers and payloads. Live
institutions remain out of scope. Docker Hub's redundant pull returned EOF during
local validation; the same exact pinned digests were already cached and executed.

## PR handoff

The implementation PR must link #209, list validation and migration impact, state
remaining unsupported account types, and require migration-before-runtime release.
Deployment success alone does not close #88 or approve live bookkeeping use.
