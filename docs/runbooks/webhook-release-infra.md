# Dedicated webhook infrastructure preparation
<!-- stage-runbook: true -->

## Status and boundary

#271, lease `PLAID-271-PREP-20261022T1725Z` expiry22:00Z, dedicated branch
`issue/271-webhook-release-infra` based on mergeda2c9dee. Engineering preparation
and mocked tests only; no apply, cloud/provider/GitHub operation, credentials,
private Terraform state/plan, merged SQL or existing deploy-workers edits.

## Stage ledger

- Read #271 export, project index/instructions, existing webhook/runtime/release
  contracts, annotated source register and separate worker bootstrap/runtime roots.
- Refreshed official platform sections; versioned Google8.6.0 references used when
  registry HTML unavailable. [Source record](../research/webhook-release-sources.md).
- Coordinator approved separate inert bootstrap/runtime roots, one fixed service,
  two static secret readers, dedicated identity, fixed command, numeric versions,
  separate public approval and request/compute targets before implementation.
- Implemented dedicated roots, copied existing pinned public provider lockfiles,
  isolated credential-free mocked harness, and reviewed release/rollback boundaries.
  Root owns exact release receipts/orchestration and shared index/bibliography.

## Current evidence

- `bash infra/gcp/webhook-runtime/test.sh` PASS: Terraform1.15.7,
  Google8.6.0 pinned lockfiles, formatting, validation and16 mocked plan scenarios
  (bootstrap6/runtime10). Tests use synthetic computed values with
  `override_during=plan`; no real provider operation or apply. Invalid
  project/region/image/secret-version metadata fails as expected.
- `make check test build` PASS with local Go1.26.5/external modfile/cache;
  tracked pinned1.26.10 unchanged. No foundation suite was run.
- Local synthetic outcomes: `/private/tmp/jayburd-271-infra-test.log` and
  `/private/tmp/jayburd-271-infra-core.log`. Initial provider-registry DNS denial
  was retried with approved escalation for the pinned provider download only;
  no TLS/control bypass or cloud identity/credentials were used.
- Staged runbook guard and diff check required before commit. No cloud resource,
  effective inherited IAM or signed provider delivery is claimed by these checks.

## Remaining work

Owner/security approval, separate state provisioning, dedicated SQL bootstrap and
real identity permission proof; trusted worker artifact; guarded exact release and
readback; source consent renewal if explicitly authorized; approved enrollment
configuration and actual W01–W05/W07 delivery/retry/catch-up. No auto authorization.

## PR handoff

New infra roots/tests plus contract/source/runbook only. No migration or financial
policy changes. Existing private API/worker modules and IAM remain untouched.
Mocks verify requested configuration; they cannot establish effective inherited
IAM, deployed route availability or end-to-end provider/SQL/worker safety.
