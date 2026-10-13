# Observer runtime delivery

<!-- stage-runbook: true -->

## Status and boundary

Build the bounded local Go publisher/consumer adapter for #7/#57 on merged #78.
No SQL migration, hosted transport, cloud mutation or financial effect.

## Stage ledger

- Assigned one implementation agent a bounded adapter/test scope; coordinator
  reviewed the API before implementation and requested independent review.
- Documented project status and exact frontend navigation after owner review.
- Integrated agent commit `78c09b5` as `fa470e8`; independent review verified
  ambiguous COMMIT SQLSTATE classification and regression tests after correction.
- Recorded owner navigation feedback and proposed next UI slice in issue #52.
- Started merged-main synthetic demo on separate loopback ports 39281–39283,
  preserving the existing demo. Port overrides remain in the review worktree.

## Current evidence

Local review frontend returned HTTP 200 at localhost:39283 and was opened in the
system browser. Owner supplied a screenshot of the running synthetic page.
Read-only Cloud Run listing across regions returned an empty service array for
jayburd-bookkeeping-dev. No deployed frontend claim or cloud mutation.
Combined `make check test build` passed after integration. The unchanged adapter
source passed `make observer-test`: 19 existing SQL tests, seven unit groups and
six PostgreSQL adapter groups. [Named evidence](observer-runtime.md) covers all
nine primitives, role separation, deadline cancellation, retry/fencing, queue
calls after commit, duplicate/reordered delivery and lost commit replies.
Independent reviewer reran the corrected unknown-commit regression tests.
Merged baseline PR #78 now has all hosted CI checks passing.

No API/UI/SQL changes are included; existing browser and ledger proof is linked
rather than rerun as evidence for unchanged components.

## Remaining work

Add separately reviewed service composition, queue/OIDC transport and hosted
acceptance. Implement the navigable book overview tracked in #52; durable import
storage/review remains a separate slice.

## PR handoff

Local integration and validation passed. No DDL, hosted service activation or
provider connection. Keep #7/#57 open for remaining hosted acceptance. The local
demo is a separate review worktree; its temporary port overrides are not in this PR.
