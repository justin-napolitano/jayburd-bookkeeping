# Provider webhook delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #103, lease WEBHOOK-GO-103. Implements the reviewed
[webhook contract](../contracts/provider-webhooks.md) with synthetic inputs only.
No application API mount, provider call, real credentials or cloud deployment.

## Stage ledger

- Read project instructions, source register and provider/runtime contracts.
- Refreshed official Plaid verification and Transactions notification sections.
- Coordinator approved64KiB/future60second profile, maintained JWT verifier,
  bounded key cache, separate ingress and narrow SQL receipt/catch-up adapters.
- Sole SQL author froze migration14 interfaces and role ownership.
- Implemented explicit dependencies, fixed Sandbox key endpoint, exact signed-byte
  verification, body schema, short transactions and safe acknowledgement behavior.

## Current evidence

- `make check test build` passed: vet, race tests, migration file checks and builds.
- `go test -race ./internal/webhook -count=1` passed.
- `bash scripts/test-webhook.sh` passed15 groups and16 subtests, including three
  actual PostgreSQL groups against final migration14 fromc00325c. Tested up-file
  SHA256:e5e19f843358b17b425db265a2b4b914064aefc8053c03371d68be08e7beee33.
- Real restricted sessions commit a receipt/wakeup before injected response loss;
  HTTP returns503, then identical signed delivery resolves204 with exactly one
  receipt and intent. Valid high/low-S equivalents also share that same receipt.
- Revoked replay, unknown Item, API/sync/mixed-role denial create no new intent.
  Catch-up lost-commit replay preserves the same tick, bounded cursor and one intent;
  revoked streams are skipped. Source observations, journals and financial outbox
  remain unchanged. Key/credential callbacks assert no open SQL transaction.
- Signed synthetic tests cover exact original bytes, strict duplicates/claims/body,
  stale/future age, malformed metadata, cached expiry, bounded rotation/misses,
 16-request admission, single-lookup cancellation and fixed key transport bounds.
- Independent peer review identified ECDSA equivalent-signature replay identity.
  Coordinator-approved canonical-S fingerprinting fixed it; independent re-review
  found no remaining blocker. Original signatures still use maintained JWT verify.

Tests generate P256 keys in memory and inject HTTP; no real credentials or provider
network is contacted. Temporary SQL/harness copies were used only for proof and
removed before commit. The coordinator owns their final integration.

## Remaining work

Coordinator integrates migration/harness/shared dependency promotion and runs the
combined status/UI/transport gates. Hosted endpoint,
credentials, scheduler cadence, provider qualification and operational load remain
separate approval gates; no claim of deployed or timely transaction acquisition.

## PR handoff

Only internal/webhook and its contract/runbook are owned by this lane. No SQL or
shared server/deployment/API modifications. Local proof is complete; hosted qualification is not claimed.
