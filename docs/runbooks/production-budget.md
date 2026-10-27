<!-- stage-runbook: true -->
# Budget and offline release sequence — #337/#316

## Status and boundary

BUDGET337-25; docs/337-budget-25 fromb75d7c7. Documentation only.
Owner target approximately USD25/month for GCP only, excluding Plaid; no cloud/provider/secret/private data,
Terraform state/plan access, resource creation, resize, shutdown or spend.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live337/316 OPEN; lease65 BUDGET337-25; merged315/349 source retained |
| Review | PP11 official selected public pricing; source dev tier only; actual sizing unavailable |
| Decision proposal | Costed micro alternatives, Sandbox cost included, fees/usage unknown; no guaranteed cap |
| Sequence | Bank chosen privately in Plaid; fictional offline316 preparation may proceed; actual337 decision gates provisioning/activation |
| Validation | make check/test/build PASS; synthetic loopback tests rerun outside sandbox; staged gate required |

## Current evidence

Local checks used installed Go1.26.5 with external temporary modfile/cache;
tracked1.26.10 unchanged. Initial sandbox socket denial was environmental.
Public list-price arithmetic is recorded in the mandate; no live sizing or bill
is established. Existing Infisical secret entry remains owner-reported complete.

## Remaining work

Actual nonsecret resource sizing, itemized service/provider estimate, synthetic
capacity/restore proof, accepted dev availability and final owner/security mandate.
Earlier production-pilot-mandate runbook records historical initial questions;
this revision supersedes its upfront bank/budget and source316 waiting statements.
Source-only316 work remains pending; no final release or issue completion implied.

## PR handoff

Refs #337/#316. Documentation only; no SQL, runtime, IAM or deployment changes.
Both decisions and actual release evidence remain pending.
