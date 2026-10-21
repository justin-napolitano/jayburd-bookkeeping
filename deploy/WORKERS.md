# Paused worker release

Use this after owner bootstrap described in the
[worker release contract](../docs/contracts/worker-release.md). The workflow
deploys two private worker services with scheduling already paused. It does not
bootstrap IAM, migrate SQL, enroll a binding, activate pulls or qualify Plaid.

## Prepare a reviewable release

After this implementation and #111 merge, run **Build candidate images** on main.
Wait for the entire run to succeed. Prepare `deploy/worker-release.json` in a PR:

```json
{
  "schemaVersion": 1,
  "project": "jayburd-synthetic-test",
  "projectNumber": "123456789",
  "operation": "promote",
  "build": {"run": "123", "attempt": "1", "commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
  "image": "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "binding_id": "11111111-1111-4111-8111-111111111111",
  "publisher_policy": "binding-v1",
  "client_id_version": "1",
  "client_secret_version": "2",
  "evidence": {
    "bootstrap": "https://github.com/justin-napolitano/jayburd-bookkeeping/issues/7#issuecomment-1",
    "compatibility": "https://github.com/justin-napolitano/jayburd-bookkeeping/issues/7#issuecomment-1",
    "identity": "https://github.com/justin-napolitano/jayburd-bookkeeping/issues/7#issuecomment-1",
    "secrets": "https://github.com/justin-napolitano/jayburd-bookkeeping/issues/11#issuecomment-1"
  },
  "expiresAt": "2026-10-15T00:00:00Z",
  "previousDeployment": null
}
```

For shared Sandbox acquisition, select `"publisher_policy": "shared-v1"` and
`"binding_id": ""`. The approved SQL publisher login discovers enrolled books
through the [shared acquisition contract](../docs/contracts/shared-acquisition-publisher.md).
The fixed `binding-v1` policy remains the rollback choice.

These are fictional examples, not a dispatchable approval. Replace every pin
with reviewed public metadata. Versions are numeric GSM version identifiers in
the fixed client-id/client-secret containers, never values or aliases. The
binding must already be enrolled by its owner. Evidence comments must document
owner bootstrap, applicable SQL/image compatibility, managed identity and
Infisical propagation. A syntactically valid URL is not evidence verification;
the owner reviews those comments before approving the exact bytes.

For a creator-owned Sandbox book, **Connect account** now enrolls the binding.
The owner can retrieve its active UUID with the read-only
[`show-worker-binding.sh`](../scripts/show-worker-binding.sh) helper using the
reviewed entity and book IDs. See the [operator instructions](../docs/runbooks/worker-binding-lookup.md).

Close #5/#6/#11/#26 only on their actual acceptance evidence. The workflow checks
these closures and merged/ancestral #109/#110/#111. Build commit must be ancestral
to the workflow commit. This tooling does not close those issues to bypass gates.

## Dispatch and review

After the manifest PR merges, compute `shasum -a 256 deploy/worker-release.json`
and dispatch **Deploy paused development workers** on main with that exact hash.
Only the repository owner may dispatch. Approval expires within seven days.
The separate worker federation identity must already be enabled by the owner.

The workflow verifies the canonical worker candidate receipt before GCP auth,
then checks Scheduler is paused before planning, before apply and after rollout.
Only the worker-runtime root and isolated backend are applied. Failure does not
trigger automatic rollback or any SQL operation. Provider diagnostics/state/plans
remain private and are never artifacts. The retained `worker-deployment.json`
contains only selected revision, image and paused-scheduling evidence. It does
not prove IAM, workload OIDC, credential propagation or bank ingestion.

The owner finishes private invoker grants only after the first paused readback.
Pause alone does not cancel queued or in-flight tasks; previously activated
installations require an owner-reviewed quiescence procedure before rollout.
Scheduler activation remains a separate reviewed change and Sandbox gate #88.

## Compatible rollback

Create a new reviewed manifest with `operation: "rollback"` and the prior
successful worker deployment run ID in `previousDeployment`. Image, binding,
project and exact client versions must match that retained receipt. The prior
successful build receipt is independently checked again. New policy or new
versions require forward promotion; rollback never executes down migrations.

## Limited Sandbox qualification

The separately reviewed [Sandbox release contract](../docs/contracts/sandbox-release.md) adds explicit
`sandbox-promote`/`sandbox-rollback` operations for the fixed development project.
Full promote/rollback gates above remain unchanged. Sandbox releases require the
bounded database proof (#130), and connection-enabled/worker releases also require
delivery implementation (#131). Hosted activation is separately tracked in #133.
Scheduling remains paused; no production or full-readiness claim follows.
