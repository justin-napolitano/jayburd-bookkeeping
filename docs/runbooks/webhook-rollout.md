# Existing webhook updater delivery
<!-- stage-runbook: true -->

## Status and boundary

#271 wave2, lease `PLAID-88-WAVE2-20261022T1930Z`, expiry23:59Z. Dedicated branch
`issue/271-webhook-rollout`, base24f263ef2cde697cf0f8cf51c1176d5c1521e679.
Implementation/synthetic tests only. No cloud, GitHub, provider, credential, state
or plan access; no enabled manifest, release dates or source consent invented.

## Stage ledger

1. Reviewed existing offline manifest/readback, PR270/273 and separate inert roots.
   Identified missing executable updater, dedicated federation and selected v2
   retrieval; owner-first private creation chosen over widening routine authority.
2. Refreshed [official sources](../research/webhook-rollout-sources.md); coordinator
   approved exact workflow SHA pin, narrow update/metadata/ActAs/state/repository
   grants, actual custom-role definition proof and independent federation deactivation.
3. Added dedicated default-inert delivery root, owner/main workflow, exact tracked
   manifest/provenance/ancestry/current-main gates, fixed existing-service updater
   and in-memory selected collector. No state/plan inspection or IAM bootstrap in
   the updater. Root owns shared target/CI/index integration.
4. Added fake executable sequencing/failure/redaction tests, role-definition and
   collector negatives and mocked federation/IAM/deactivation scenarios.

## Current evidence

- `make check test build` PASS with local Go1.26.5/external modfile/cache;
  tracked pinned toolchain unchanged. No foundation suite was run.
- `make release-test` PASS; explicit `node --test scripts/webhook-release.test.mjs
  scripts/webhook-rollout.test.mjs` PASS11 groups, including actual shell execution
  with fake binaries, default/context/token denial, preflight/main/expiry/provider
  failure sequencing, diagnostic redaction and in-memory collector limits.
- `bash infra/gcp/webhook-delivery/test.sh` PASS7 mocked plan scenarios with
  Terraform1.15.7/Google8.6.0, including inactive defaults, exact claim pin, minimal
  grants, deactivation and invalid activation/configuration. Initial sandbox DNS
  denial was retried with approved access for the pinned provider download only.
- `bash -n` for updater and isolated harness PASS. Staged runbook guard/diff check
  required before commit. Local synthetic logs are
  `/private/tmp/jayburd-271-rollout-{core,release,tests}.log` and
  `/private/tmp/jayburd-271-delivery-infra.log`.

No actual WIF, inherited IAM or updater permission compatibility claim is made
from mocks/source. The source review identifies expected PATCH/operation/GET
permissions; a real permission mismatch must stop, never widen to RunAdmin.

## Remaining work

Owner/security review, first private creation and state handoff, effective IAM,
public approval, SQL mandate/current consent, trusted worker artifact, new exact
main/manifest SHA pin and explicit federation activation. Actual signed delivery,
retry/catch-up and exact/no-journal acceptance remain open. Earlier consent expired
17:00Z; this implementation does not renew it or authorize an account operation.

## PR handoff

Dedicated workflow/root/scripts/contracts/source/runbook only; existing worker
release and private IAM untouched. Migration impact none. Root integrates shared
infra target and test entrypoints. Retain only validated public receipt; no raw
metadata/provider diagnostics/state/plan/token upload. Failures may follow partial
apply and require owner-controlled mandate/public-binding revocation and review.
