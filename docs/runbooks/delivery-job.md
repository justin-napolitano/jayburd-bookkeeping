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
- PASS: `make check test build`, all infrastructure mock tests including seven new
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

1. Merge parent/this PR, build the worker image through the existing image workflow,
   and review its successful image receipt/digest. Bootstrap this isolated Terraform
   root with `canary=null` and GitHub federation disabled.
2. In Infisical organization Access Control → Machine Identities, create
   `jayburd-dev-delivery`, remove Universal Auth and choose **GCP ID Token Auth**.
   Set allowed service-account email to
   `jayburd-dev-delivery@jayburd-bookkeeping-dev.iam.gserviceaccount.com`,
   TTL/max TTL 900 seconds. Add it to project `jayburd-bookkeeping` with read access
   only to Development root key `JAYBURD_DELIVERY_CANARY`. No create/edit/delete/list
   permission. Verify the environment slug is `dev`; if plan scope cannot enforce
   exact-key access, stop and review isolation rather than grant whole-root access.
3. Owner creates that canary with a fictional value, separately from Plaid keys,
   and supplies only identity ID, source ID and exact version metadata. Infisical
   project ID is `775ef885-a687-4792-9463-180bbf9eee13`, origin app.infisical.com.
4. Configure the reviewed image and metadata in Terraform, then explicitly enable
   the execution identity. No secret values enter Terraform or GitHub inputs.
5. Dispatch **Run development delivery canary** once from main. Inspect the Cloud Run
   execution and fixed status log. Submission is not success. Independently verify
   the owner-known canary through an authorized owner process; agents receive only
   PASS/PARTIAL/BLOCKED and numeric metadata. Test denied access, rotation, revocation
   and recovery before expanding to Plaid credentials. Unknown intent stays blocked.

## PR handoff

No SQL/API/frontend changes. Initial job cannot read or write Plaid destinations. Parent #113 CI is fully green.
The existing native Infisical connection is not the machine identity needed here.
Actual bootstrap, identity enrollment and hosted canary remain pending.
