<!-- stage-runbook: true -->
# Canary job delivery

## Status and boundary

Stacked on PR #113 for #11. Implements a canary-only command, image packaging,
Terraform job/identity and owner-only workflow. No hosted apply or secret retrieval.
[Contract](../contracts/delivery-job.md); [Terraform setup](../../infra/gcp/delivery/README.md).

## Stage ledger

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

1. Merge/build reviewed worker image and bootstrap this isolated Terraform root.
2. Set `canary` to that image and the metadata above, omitting `source_id` and using
   default `mode="lookup"`. Enable owner invocation only after reviewing the job.
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
Actual bootstrap, identity enrollment and hosted canary remain pending.

## Metadata lookup stage

Added value-disabled exact-version lookup and strict ID/namespace/version validation.
Tests cover success, denied access, wrong key/version/project/folder and malformed ID;
only selected metadata fields can be serialized. CLI rejects mixed modes and no-ID
execution. PASS: updated `make check test build`, `make infra-test` (including all eight
delivery runs), and `git diff --check`. No live lookup or cloud execution yet.
