<!-- stage-runbook: true -->
# Production webhook delivery — #314

## Status and boundary

PRODUCTION314-WEBHOOK; feat/314-production-webhooks from d0b141b.
Source-only implementation; one reserved migration35, no managed apply. No real
key/secret/provider/cloud/public exposure/private data or pilot activation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Fresh314/311/312/313/65; prerequisites closed; isolated branch and sole35 writer |
| Review | Bibliography, SQL/standards/database/Production/owned-repair contracts and current14/32/33/34 functions inspected; source sections refreshed |
| Decision | production-webhooks contract specifies profile/cache/SQL/login/owned-session/key/expiry/rollback before code/DDL |
| Implementation | Explicit key/verifier/SQL/managed receiver profiles; canonical notification opt-in; immutable environment login registry and owned repair in35 |
| SQL review | Collision fixture found remaining global proof lookup;35 now scopes lookup, record, proof and claim by environment. Historical retired Sandbox login backfill proved |
| Synthetic receiver | Signed Production receipt/replay/lost-COMMIT/current-scope and owned-session/credential tests pass; no acquisition or posting from ingress |
| Browser | app-integration and connection-browser-test pass; web-check/test/build/browser pass (342 unit,26 browser tests) |
| Regression | check/test/build, migration-test, ledger-test, webhook-test, webhook-bootstrap-test, connection-state-test, connection-service-test, connection-ops-test, acquisition-test and provider-flow-test pass |
| Role drift | Actual verifier membership, predefined read-all role and direct credential-table SELECT each deny in both Go receiver and SQL entrypoint |

## Current evidence

313 source and schema34 prerequisite complete; actual Production targets and
deployment remain pending337/316. No reuse of expired release or provider approval.

Local Go evidence uses installed1.26.5 with an external temporary modfile/cache,
because the tracked1.26.10 download previously failed and its approved retry
returned403. Tracked Go/dependency pins are unchanged; exact-head CI must prove
the canonical toolchain. Locked npm install reports zero advisories and four
unapproved install-script warnings; no lifecycle script approval was added.

Initial SQL-generator literal-replacement and proof-input fixture failures were
corrected before the passing gates; historical failed logs are not acceptance.
The collision test specifically proves matching Link hashes across environments
cannot affect the owned proof. Distinct signed transaction events may each request
catchup; duplicate delivery creates no second receipt/demand. Source publication
and actual hosted delivery are separate evidence, not asserted from receipt counts.

## Remaining work

Exact-head CI pending PR creation. Human sensitive-source
review before merge. Actual303/320 and315/337/316/90 acceptance stays separate.

## PR handoff

Issue314. Migration35 source only;1–34 immutable. No infra/release/browser wire
changes. Record exact tests/results/limits; no synthetic suite substitutes actual
provider event, key/IAM delivery, finite release or private personal reconciliation.
