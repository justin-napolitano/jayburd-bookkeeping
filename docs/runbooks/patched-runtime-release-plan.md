<!-- stage-runbook: true -->
# Patched runtime release plan —485

## Status and boundary

OFFLINE PLAN ONLY,2026-11-11. Issue485 requires qualifying patched artifacts,
then a separately authorized finite rollout. Issue488 carries this checklist;
it authorizes no workflow dispatch, image publication, cloud lookup, deployment,
provider/consent action, secret access, managed SQL or data mutation. Existing
managed binaries have not been patched by a source/toolchain change.

Consolidation487/bea304ec remains frozen and unmerged while external CI is queued.
Do not bypass it or call queued checks successful. Any future release must select
one exact accepted main source after its dependencies actually merge; this plan
does not preapprove either487 or the new automatic categorization implementation.

This is operational metadata/checklist maintenance, not a new financial message,
accounting policy or ISO conformance claim. Reuse the reviewed
[image delivery](../contracts/image-delivery.md),
[Sandbox release](../contracts/sandbox-release.md),
[connection release](../contracts/connection-release.md),
[migration release](../contracts/migration-release.md),
[migration status](../contracts/migration-status.md) and
[worker release](../contracts/worker-release.md) contracts. Existing bibliography
and their selected source records are reused; no new vendor behavior is asserted.

## Stage ledger

| Stage | Evidence / disposition |
| --- | --- |
| Grounding | Read485/488, AGENTS, automatic-categorization stage, existing build/release/rollback contracts and accepted repository receipts. |
| Source security |483 patches source/build/CI to Go1.26.11 after GO-2026-6859 in1.26.10; accepted synthetic scan reports zero affecting vulnerabilities, not zero findings across every required module. New artifacts and fresh scans are still required. |
| Offline design | Ordered source, dependency, four-image, compatibility, staged runtime, health/transport and rollback gates specified below; all managed steps are future operator work. |
| Execution | NOT STARTED. No build ID, patched candidate digest, new revision, deployment/status receipt or owner sign-off exists for this plan. |

## Current evidence

Repository-only historical baseline:

| Item | Accepted evidence; not a fresh observation |
| --- | --- |
| Project | Existing Sandbox `jayburd-bookkeeping-dev`, number9088794002, regionus-central1; no Production/new resources. |
| Web |00027-6jn, `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:e4521574edecc6a5901c7f590341112484370e6cf93d63a3f13e30c79f3ba16d`. |
| API |00026-5nt, `us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:9747304373c1c9923518b0675cb85a2fcdde1ed1f397244df59208683fc790e0`; accepted artifact-only enforcement, unchanged during web release. |
| Release source/build |9b2a6568ffb05ed4e66c77857caa0ad07cc5509b, trusted Build42561705439 attempt1. Four images accepted; only web deployed in that release. New API/migrate/worker candidates from that build were not deployed. |
| Selected settings | Web identity jayburd-dev-frontend, concurrency32/timeout20s/CPU1/512Mi/port8080, startup `/api/auth-config`, failure12/period5/timeout3, no command/args. API identity jayburd-dev-api, concurrency16/timeout15s/CPU1/512Mi/port8080, `/readyz`, all20 approved arguments retained. |
| Authority/schema | Last release records schema47, collection reads/connections enabled, false Link qualification and expired consent2026-11-02T04Z. No consent renewal or acquisition authority follows from patching code. No current worker revision/pin is established by the web-only receipt. |
| Sources | [Build evidence](customer-ui-build-evidence.md), [rollout/settings/rollback](customer-ui-web-release.md), [artifact API baseline](artifact-sandbox-rollout.md), [toolchain research](../research/go-toolchain-2026-11.md), [final source acceptance483](https://github.com/justin-napolitano/jayburd-bookkeeping/pull/483). |

`deploy/dev-release.json` and `deploy/worker-release.json` contain expired older
release metadata; do not replay or refresh them implicitly. Artifact retention
also expires: an inaccessible historical evidence object is not verified by its
filename. Preserve an approved complete receipt before its retention deadline.

### Gate1 — exact source and finite scope

- Record full source commit, tree, merged ancestry/dependencies and clean tracked
  diff. Recheck that selected main contains the Go patch and unchanged locked
  module/npm dependencies except separately reviewed changes. Keep source/app
  version distinct from the deployed revision. No arbitrary-ref build input.
- Require every exact-head prerequisite: core/check/test/build; immutable SQL
  checks and relevant migration/readiness/least-privilege proofs; frontend
  generated types/units/build/browser; actual source0.10/replay/report/CSV/drill,
  setup/adoption/recovery/Production-shaped/demo; restore and1000 linked-post
  proof; reachable-vulnerability gate on the actual patched toolchain.
- Treat skipped, cancelled, externally queued, partial or superseded checks as
  unresolved. Local EPERM/compile-only evidence does not replace actual SQL tests.
- Decide which component(s) will change and why. A patch-only API/worker/migrator
  replacement is not equivalent to web-only presentation release. Worker rollout
  requires its separate existing worker-release profile and pause/demand review;
  building its candidate does not authorize deploying or invoking it.
- If selected source includes488/new schema or background classification, stop
  reusing a schema47 patch-only assumption. Require that feature's reviewed
  forward-migration, schema/runtime/saved-command compatibility and inactive/
  active execution gates separately. Never add DDL, a grant or automatic work
  merely to make the patch rollout start. No migration number is allocated here.
- For488, record API background CPU/instance lifecycle or an explicitly reviewed
  invocation schedule, cancellation/drain before pool closure, and active/paused
  recovery evidence. The default-off API classifier and consumer PFC flag are
  separate opt-ins; a successful web build supplies neither hosted processing
  continuity nor authority to activate them.

### Gate2 — trusted complete artifact set

- Future authorized owner dispatch must be the existing main-only
  `.github/workflows/build-images.yml`: repository1414515318,
  owner/actor55423093 and expected triggering actor/workflow path. Record exact
  run/attempt, workflow source, public project identity and all prerequisite jobs.
- Require all four api/migrate/web/worker candidates from one source/run/attempt,
  even when the finite rollout selects fewer components. Each is built once for
  linux/amd64, tested by its exact local image ID, scanned before cloud auth,
  inventoried and published without rebuilding. No tag-only promotion.
- Independently recompute every complete candidate.json with the existing
  `scripts/image-receipt.mjs` semantics and actual build/repository metadata.
  Require tested/scanned identity equality, OCI revision, BuildKit source/revision/
  component entrypoint, architecture and one bound published repository digest.
  A BuildKit buildDigest need not equal repository digest; compare the receipt's
  defined identities, not a guessed string substitution.
- Retain nonempty matching CycloneDX inventory and provenance metadata for each
  tested image; verify source/image/tool bindings. These unsigned metadata are
  not a signed attestation or SLSA certification.
- Require exact-candidate container proofs, pinned scanner installation/integrity,
  fresh HIGH/CRITICAL image gate including unfixed findings, and module scan.
  Record actual findings/affected versus uncalled dependency distinction; no
  scanner exclusion, ignore-unfixed change, suppressed gate or older scan reuse.
- Missing/expired artifact, mismatched digest, rejected-scan artifact, failed
  matrix member, wrong identity or new finding stops promotion. Do not deploy a
  partial set because the desired component happened to publish successfully.

### Gate3 — compatible configuration and reviewed promotion input

- Future operator obtains fresh narrowly selected service/revision metadata under
  a new finite claim. Compare it to the repository receipt; if drift exists,
  review it before proceeding. This offline plan did not inspect cloud metadata.
  Do not retrieve environment values, secrets, state, plans, financial bodies or
  provider payloads. Evidence contains approved identifiers/digests/outcomes only.
- Prove selected source/schema/API/bridge/auth/argument compatibility, including
  retained prepared/committed commands, recovery IDs and original history.
  Check changed flags/defaults: omission must not reset approved settings or
  quietly enable classification/acquisition/provider work. Reuse the existing
  release validator and exact same-build receipts; no generic shell/SQL input.
- Preserve service identity, ports, resources, scaling/concurrency, timeouts,
  probes, ingress/private invocation/origin/auth, complete argument list and
  numeric secret-version references. Keep consent expiry, qualification/caps,
  scheduler pause and registered scope unchanged unless a separate approved
  change explicitly says otherwise. Do not print argument/secret values.
- Manifest operation/project/component/build/digests/compatibility evidence/
  expiry and rollback input must be exact, hashed and reviewed. Review complete
  input, not a friendly alias. Expired historical manifests are not candidates.
- Use the approved schema status-only profile if the finite release requires
  it: exact candidate migrator image, clean expected schema and complete embedded
  hash set, independently validated execution receipt. Status does not mean up.
  No force/down/reset/automatic grant or manual SQL repair.

### Gate4 — staged runtime and bounded transport

- Keep accepted old traffic until the staged exact-image revision is genuinely
  Ready. Inactive/Retired or deployment-command completion is not startup health.
  Reconcile async operation, generation, observed generation and selected spec.
- After an explicitly authorized promotion, require expected exact image,
  latestReady, one intended sole100 traffic target with no unexpected tag/route,
  service Ready/ConfigurationsReady/RoutesReady and direct revision
  Ready/Active/ContainerHealthy/ResourcesAvailable. Retain old revision identity
  for diagnosis; do not delete it as part of promotion.
- Deep-compare selected before/stage/active settings; only reviewed differences
  may remain. Separately verify any component declared unchanged, including its
  image/revisions/traffic/spec. A changed API must still enforce artifact-only
  public posting and current auth; private claims/financial operations are not
  smoke-test inputs.
- Repeat bounded anonymous IPv4 transport with bodies discarded, no redirects,
  empty Authorization/Cookie/serverless-auth: web root/auth-config200; proxied
  workspace/collection/recovery/onboarding/onboarding pending/v3 taxonomy401;
  direct private API workspace403. Retain numeric statuses/time/target only.
  This is transport/denial evidence, not successful sign-in/MFA/private review.
- Stop on any mismatch/unhealthy revision/unexpected status or unapproved drift.
  Do not weaken a readiness flag, grant public IAM, renew consent, call a provider,
  acknowledge saved work, create a journal or inspect private data as a workaround.

### Gate5 — rollback decision, not automatic fallback

- Prefer hold before traffic or a compatible forward repair. Select a rollback
  only through a fresh reviewed finite action, comparing prior image/configuration,
  current schema and retained new-format data/commands. Never down/force schema,
  clear claims with SQL, restore data or replace credentials to fix a binary issue.
- The historically accepted API00026 may preserve artifact policy but was built
  before the Go1.26.11 patch. Returning to it reintroduces the known old-runtime
  security risk; document severity/exposure/mitigation and obtain explicit bounded
  operator disposition. Its old acceptance/scan is not current security clearance.
- A pre-artifact API is additionally prohibited: it permits manual posting. Do
  not choose it even if it starts or matches an old tag. A web rollback can
  reintroduce old presentation and must retain a compatible enforcing API; it
  does not patch vulnerable backend/migrator/worker binaries.
- A new-schema automatic categorization release may make old code incompatible
  with saved decisions/job state. Stop until concrete compatibility is proven;
  keeping SQL48+ while guessing an older app is safe is not a rollback contract.
- Unknown promotion/rollback outcome requires selected readback of the same
  intended operation/revision. Never infer absence/success from a timeout or
  blindly dispatch a replacement. Re-run health/settings/traffic/transport checks
  after any reviewed rollback and preserve both receipts and failure evidence.

## Remaining work

All five gates above are future execution/approval work. Patched candidate build,
receipts, runtime pins, fresh scans, finite component scope/compatibility and
managed health/transport readback remain unset for485. Owner review, provider/MFA
qualification, data reconciliation and live bookkeeping acceptance remain separate.
No current source/private categorization proof is a deployment authorization.

## PR handoff

Offline checklist ready for review, not executable release approval. Root owns
staging/commit and any future GitHub/managed action. This lane edited only this
allocated marked runbook plus its separate acceptance inventory; no cloud or
provider action occurred. Record final actual source/run/digests and outcomes in
new explicit execution evidence rather than changing historical receipts to imply
they describe the patched binaries.
