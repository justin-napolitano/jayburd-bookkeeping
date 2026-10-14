<!-- stage-runbook: true -->
# Canary job delivery

## Status and boundary

PRs #113/#114 merged for #11. Implements a canary-only command, image packaging,
Terraform job/identity and owner-only workflow. See hosted checkpoint below.
[Contract](../contracts/delivery-job.md); [Terraform setup](../../infra/gcp/delivery/README.md).

## Stage ledger

Initial implementation evidence (superseded by hosted checkpoint where noted):

- Verified current GCP job listing: only `jayburd-dev-migrate`; delivery job absent.
- Reused metadata identity and one-shot adapter; separated canary from Plaid values.
- Added private receipt storage, add-only destination IAM and disabled federation.
- Focused command/adapter tests and initial five mocked Terraform checks passed.
- PASS: `make check test build`, all infrastructure mock tests including eight new
  delivery runs, focused command/adapter race tests and actionlint.
- Full local Dockerfile build BLOCKED by dependency-proxy certificate trust at
  `go mod download`; TLS verification remains enabled. Separate offline Linux
  scratch-container proof PASS: six startup rejections, no network. This does not
  replace the full Dockerfile/image-build gate.

## Current evidence

GCP control-plane access works. Owner previously confirmed the native Infisical
App Connection succeeded after IAM propagation. Neither is proof of the new job's
Infisical authentication, secret delivery, SQL connectivity or Plaid Sandbox access.
No end-to-end environment connection is claimed.

## Remaining work

Owner selected a separate Infisical project after the hosted UI gated folder grants
behind a paid plan. This supersedes the original root-key scoped-grant instructions.

| Binding | Owner-supplied metadata |
| --- | --- |
| Canary project slug | `jayburd-delivery-canary-jy-om` |
| Canary project ID | `83f8487f-3c31-42d0-ae16-f04dbf8092f2` |
| Machine identity | `8d510558-6b96-44c0-b25f-ce73ff7e4d74` |
| Environment / path / key | `dev` / `/` / `JAYBURD_DELIVERY_CANARY` |
| Source version | `1` confirmed by owner screenshot |

Owner reports GCP ID Token Auth corrected; exact delivery SA email and TTL/max TTL
900 configured. Identity should have Viewer only on the isolated canary project,
and No Access on the original bookkeeping project. No actual identity login verified.

1. COMPLETE: reviewed image built and isolated Terraform root provisioned.
2. COMPLETE: `canary` uses that image and metadata above, omitting `source_id` with
   `mode="lookup"`; scoped owner invocation enabled.
3. Run the manual workflow once. Lookup requests `viewSecretValue=false`, with
   imports/expansion disabled, and prints only validated `source_id` and
   `source_version` JSON. It performs no GCP secret or registry calls. Read those
   selected log fields; do not dump provider responses, tokens or process environments.
4. Review the identifiers, then configure `mode="execute"` and exact `source_id`.
   Terraform adds the canary-only version-adder grant only in execute mode.
   Invoke separately to deliver the synthetic canary. No automatic lookup-to-write.
5. Independently verify the owner-known canary. Submission/lookup/acknowledgement are
   distinct evidence; rotation, revocation, denied access and recovery remain gates.
   Unknown intent stays blocked. No Plaid credential activation.

## PR handoff

No SQL/API/frontend changes. Initial job cannot read or write Plaid destinations. Parent #113 CI is fully green.
The existing native Infisical connection is not the machine identity needed here.
Bootstrap is complete; hosted lookup outcome and subsequent canary delivery remain separate gates.

## Metadata lookup stage

Added value-disabled exact-version lookup and strict ID/namespace/version validation.
Tests cover success, denied access, wrong key/version/project/folder and malformed ID;
only selected metadata fields can be serialized. CLI rejects mixed modes and no-ID
execution. PASS: updated `make check test build`, `make infra-test` (including all eight
delivery runs), and `git diff --check`. No live lookup or cloud execution yet.

## Hosted bootstrap checkpoint — 2026-10-14

- PASS: image run `38617498600`, main `47e8a415bd2d454e6da8cbdfa3c5c1b6db02d2fb`; worker container and vulnerability gates passed.
- PASS: missing workers state bucket created by a targeted apply through the existing foundation backend, prefix `foundation`. Delivery backend initialized with prefix `delivery`; no competing local state owner.
- GCP rejected 256 MiB for this execution mode. Corrected to its required 512 MiB; `bounded_canary_job` now asserts the limit.
- PASS: delivery Terraform apply completed; lookup job and scoped identities provisioned. No destination version-adder grant in lookup mode.
- PASS: owner workflow `38623665528` submitted execution `jayburd-dev-delivery-canary-kdchm`. Execution later succeeded; see hosted delivery checkpoint for selected metadata evidence.
- PASS: `make check test build`. `make infra-test` passed all roots, including eight delivery runs. No real bank connection; the subsequent canary delivery is recorded below.

## Hosted delivery checkpoint — 2026-10-14

- PASS: metadata lookup execution `jayburd-dev-delivery-canary-kdchm` completed successfully; validated source ID `3ee9fbab-c01c-4c4e-ba40-ba6207e7a120`, version `1`. This establishes workload authentication and canary metadata access.
- Owner authorized one fictional canary delivery. Applied execute mode with this exact ID/version and canary-only version-adder permission. Destination version listing was empty before invocation.
- PASS: delivery workflow `38625292204` submitted execution `jayburd-dev-delivery-canary-dnm59`; one task succeeded and logged `PASS: canary delivery acknowledged; independent verification pending`. Metadata-only destination listing now shows version `1` enabled. No payload retrieved into agent context.
- Independent owner verification of destination contents remains required; this is not Plaid connectivity evidence.

Owner verification: in GCP Secret Manager, open `jayburd-dev-bank-sync-canary`,
select version `1`, and compare its value privately with the fictional source.
Report PASS or mismatch only; do not paste the value. Rotation, revocation and
recovery qualification remain outstanding before Plaid activation.
