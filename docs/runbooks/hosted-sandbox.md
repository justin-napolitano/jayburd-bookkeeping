<!-- stage-runbook: true -->
# Hosted acquisition and Sandbox delivery

## Status and boundary

User authorized worker/IAM, Infisical integration and Plaid Sandbox testing.
Implementation is integrated; actual hosted/provider qualification is unperformed.
Work began at PR109 head ab2e1f4; GitHub now confirms merge2115e3c. The
delivery targets main. Real accounts remain excluded.
The last read-only cloud check found no regional services and Secret Manager API
not enabled. Infisical project metadata and owner-injected credentials are unknown.

## Stage ledger

Three agents delivered worker composition, provider vault and Sandbox diagnostic;
the API lane added explicit opt-in connection composition. Root added durable GCS
evidence, owner-operated Terraform/IAM, worker SQL bootstrap, image/CI integration
and sources. Review fixes include canonical numeric project names, immutable mapping,
non-reusable slots, tombstone IAM, dedicated bootstrap state, and rejecting another
grantor's SQL delegation. Migrations1–15 are unchanged.

## Current evidence

- `make check test build`: PASS, including all-package race tests.
- Web checks/build: PASS;244 unit tests and13 browser journeys passed.
- `make app-integration connection-browser-test`: both PASS using temporary
  ports39481–39483; test files restored exactly afterward.
- Six clean-copy Terraform modules:49 mocked tests PASS; empty environment,
  backend disabled and pinned Google8.6.0 mirror. Registry DNS prevented the
  ordinary online init; equivalent source/lock tests used the cached provider.
- `test-worker-bootstrap.sh`: PASS with non-superuser operator, replay, capability,
  SET/DDL denials and foreign-grantor delegation rejection.
- Workflow lint PASS. Image/release receipt tests:19 PASS. Offline Sandbox diagnostic PASS; no provider calls.
- Four isolated worker startup denials PASS using a cross-compiled scratch image.
  Original Dockerfile build remains blocked locally by dependency-proxy certificate
  trust; no TLS bypass. CI must prove the full image before publication.

Test logs use `/private/tmp/jayburd-hosted-*`; Terraform proof logs use
`/private/tmp/jayburd-combined-tf-proof.log` and `jayburd-workers-tf-proof.log`.

## Remaining work

Owner configuration/canary, hosted IAM/SQL/binding evidence and actual Sandbox
Link/dynamic transactions/webhooks/cursor-to-review acceptance remain open.
Eight never-reused credential slots and one publisher book binding are bounded
Sandbox capacity, not linear scaling. Existing application promotion omits optional
connections and would disable them; automated worker promotion and real Link UI/
reconnect require follow-up. See [platform contract](../contracts/hosted-platform.md).

## PR handoff

#7/#11/#88 remain open; #65 owns leases. Review against main after merged PR109.
No cloud apply, secret retrieval, actual Sandbox execution or live-use approval.
