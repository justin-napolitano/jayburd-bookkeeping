# Dedicated Sandbox webhook release evidence

## Decision

Engineering contract under #271, reviewed by the coordinator2026-10-22.
New release metadata is separate from application and paused-worker manifests.
Only `sandbox-promote` in the fixed development project is accepted. A compatible
rollback is a newly reviewed forward release selecting a previously qualified
immutable image; there is no implicit rollback, SQL down migration or reused
expired approval. Existing private services and Scheduler are outside this root.

The offline validator approves exact public file bytes, validates the retained
trusted worker build receipt, and generates inputs for the isolated runtime.
It never authenticates, deploys, fetches evidence, reads state/plans or grants IAM.
No enabled manifest or new deployment federation is committed in this slice.
An owner must separately review bootstrap/public-invoker changes, database ingress
mandate, image provenance, finite qualification scope and cleanup before applying.
This preserves an explicit approval boundary while infrastructure is prepared.

## Metadata and acceptance

Schema1 has exactly project/projectNumber, operation, build(run/attempt/commit),
worker image digest, numeric client version pins, evidence(security/bootstrap/
identity/secrets/scope/compatibility), expiresAt and qualificationEndsAt.
Authorization expires within24hours; qualification ends after authorization and
within48hours. These timestamps constrain tooling approval, not service lifetime.
Owner scope evidence must bind actual source/Item/book/interval, finite current
SQL consent, ingress enable/disable operation keys and explicit cleanup ownership.
No tokens, financial rows or personal account identifiers belong in release files.

Readback must show the exact reconciled revision/image, dedicated service account,
`/webhook` command, complete exact arguments, sole port8080,10s timeout,
concurrency16,1CPU/512Mi, max1/min0 at both service and revision, TCP startup,
IAM checking enabled, internet ingress and untagged100% latest-ready traffic.
Automatic scaling, serving canonical URL, HTTP/1 and no IAP are required. Omitted
implicit false/zero ProtoJSON defaults are accepted; null/wrong types are rejected.
Separate IAM readback must contain only `allUsers` Invoker (no conditions/other
bindings). This proves selected control-plane settings, not secret propagation,
SQL authority, actual delivery, data equality or absence of journals.

## Sources, standards and alternatives

Applies the [managed webhook profile](webhook-managed-runtime.md),
[release infrastructure review](../research/webhook-release-sources.md), and
existing exact-byte candidate/receipt policy. Internal infrastructure metadata
is not an ISO20022 exchange; no financial message, amount, currency or accounting
rule changes. Existing ISO4217 mapping remains authoritative. No conformance claim.
Adding webhook fields to paused-worker deployment was rejected because its
services/callers/Scheduler have a different authority. Automatically expanding
deployment federation was deferred until this separate public root is reviewed.

Named offline evidence: `webhook-release.test.mjs`; mocked infrastructure and
actual disposable SQL bootstrap evidence are recorded in their lane runbooks.
Actual signed Sandbox chain, retry timing and independent exact/no-journal proof
remain open under #88/#271/#272. A new approval is required before execution.

Operator preparation and failure/cleanup order: [release review](../../deploy/WEBHOOK.md).
