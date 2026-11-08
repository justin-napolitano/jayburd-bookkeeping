<!-- stage-runbook: true -->
# In-app onboarding Sandbox rollout — 460/436

## Status and boundary

Source461 exactae9dc70054a74b7b7fc86ec356e76b84039bf53a passed all five
canonical gates (Seed42453606802, Infrastructure42453606808, GitGuardian) and
merged950c26d9e3851a919c8ce75ba7728a4a4a314870 at2026-11-08T16:04:34Z.
Trusted main Build42498017914 attempt1 passed all prerequisites and four image
jobs. Independent complete receipt, source/runtime/scan/SBOM/provenance checks
pass for all four, zero HIGH/CRITICAL each; [build evidence](onboarding-build-evidence.md).
Managed up47, separate clean47 status, API/web rollout and anonymous transport
acceptance are complete. Authenticated owner setup/adoption/review remains pending.
Root PRIVATE-ONBOARDING-460 owns existing Sandbox
API/web/migrator only in jayburd-bookkeeping-dev/us-central1/bookkeeping.
User explicitly requested application tooling, no manual/ad hoc operational SQL.
Normal restricted migrator up/status remains the reviewed deployment mechanism.
Root renewed/reclaimed65 through19:00Z after checking the clock gap/main/claims;
no intervening writer. No UID mapping, owner grant/adoption, provider fetch, consent renewal, worker
deployment/execution/scheduling, new resource/IAM or Production change.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source preparation |461 combines own-workspace setup, complete authorized existing-connection adoption and exact shared-gate recovery.647 units/84 component browsers and12SQL+3actualHTTP groups pass; canonical actual Next/Go/SQL and all old journeys passed. |
| Predecessor reference | Previously accepted API00024-r9d/web00024-cf5 and clean46/default-status migrator. Fresh selected metadata at16:05Z matches the expected exact images/settings, Ready/sole100. |
| Clean predecessor | Existing default status jayburd-dev-migrate-r48x2 completed16:08:01.478547Z, succeeded1/no failures/retries, exact old46 image/status args/restricted identity/task/timeout accepted by migration-receipt.mjs; clean46/full hashes. |
| Recovery metadata | Fresh instance RUNNABLE PostgreSQL17/db-g1-small, backup/PITR enabled/log retention7days. Owner supplied the requested bounded Cloud Shell backup metadata: ID1794117600000, SUCCESSFUL, completed2026-11-08T06:29:22.678Z. This resolves the local SDK cache/browser metadata-read blocker without credential/config copying, auth widening or SQL. Successful backup metadata is not a restore rehearsal. |
| Trusted candidates | Build42498017914 attempt1 exact950c26d9 all jobs SUCCESS. Independent image-receipt recomputation/deep equality and tested/scanned IDs, OCI/BuildKit source and CycloneDX identity pass for all four. Runtime gates pass; HIGH0/CRITICAL0. Worker is build evidence only. |
| Forward execution | Existing migrator image-only update199ed337… preserves selected spec exactly, default status/task1/parallelism1/retry0/660s/restricted identity. Per-execution up jayburd-dev-migrate-6sb26 completed2026-11-08T16:31:11.084243Z, succeeded1/no failures or retries. Existing migration-receipt.mjs accepts exact action/image/identity/task/timeout/completion. Separate default status jayburd-dev-migrate-gz274 completed16:32:57.036718Z, succeeded1/no failures/retries; exact receipt accepted, clean47/full embedded hashes. |
| Image-only staging | API00025-2j2/web00025-5vt exact selected specs match predecessor except image; previous00024 revisions retained sole100 during staging. Exact direct imageDigests accepted. Staged Ready/Retired, ActiveFalse, ResourcesUnknown is import evidence only. |
| API promotion | API00025-2j2 exact accepted image/latestReady/sole100, service Ready/ConfigurationsReady/RoutesReady true. Direct revision Ready/Active/ContainerHealthy/ResourcesAvailable true at16:35Z. Selected spec still matches predecessor except image. |
| Web promotion | Web00025-5vt exact accepted image/latestReady/sole100, service Ready/ConfigurationsReady/RoutesReady true. Direct revision Ready/Active/ContainerHealthy/ResourcesAvailable true at16:36Z. Selected spec matches predecessor except image; existing validateRevisions accepts both exact new revisions/sole100. |
| Transport | At16:37Z anonymous numeric web root/auth-config200; workspace/collection/recovery/onboarding/onboarding pending/v3 taxonomy401; direct API workspace403. Nine expected statuses pass. IPv4 curl, no supplied credentials or redirects, all response bodies discarded. |
## Current evidence

Source [PR461](https://github.com/justin-napolitano/jayburd-bookkeeping/pull/461),
implementation [contract](../contracts/collection-onboarding.md) and
[stages](collection-onboarding.md); GitHub460/436/65 own execution state.
The existing private flag is already true; no new application arguments or
environments are needed. New managed API mounts self-service only in Sandbox,
with unchanged actual verified authenticator and origin boundary. Anonymous
denial and startup are transport evidence; owner must perform explicit in-app
setup/adoption and review privately after deployment.

## Remaining work

Managed release acceptance is complete. Owner performs the in-app sequence below
and reviews private history.453/436 stay open for that acceptance. No authenticated
financial response was read and deployment created no workspace or adoption.
Retain exact pins for next reviewed Terraform inputs; no state/plan read/apply.
Default migrator remains status. No down/force/reset/unknown retry occurred.

## Accepted pins and preserved settings

All use us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/:

- api@sha256:971a9381b4e82c69a0398623a4fce85f679dab6e7a274f1fa9619b98fee2b74e (jayburd-dev-api-00025-2j2).
- web@sha256:def2c3f5f23a4a509f9af4e2ed5b662e2747316b2c2c17db24ae920c7de9ccfa (jayburd-dev-web-00025-5vt).
- migrate@sha256:199ed337de73e622c7398516a90ef196620b100701cb3f07924e6cd86df5f31a (embedded47, default status).
- worker@sha256:dae08d1d38145a3c326c8bf09b4a73bc855e4ea4e7d2dd9671607fef74be7e0f (build evidence only; no rollout/execution).

Selected pre/post specs compare exactly except image. API identityjayburd-dev-api,
concurrency16/timeout15s/CPU1/512Mi/port8080/startupreadyz; web identityjayburd-dev-frontend,
concurrency32/timeout20s/CPU1/512Mi/port8080/startupauth-config. Both startup probes
retain failure12/period5/timeout3. All20 API arguments, numeric web origin,
database/Infisical/credential-version references, collection-read-enabled=true,
connections-enabled, false Link qualification and consent expiry2026-11-02T04Z
are unchanged. Web command/args stay absent. No environment values retrieved.
Migrator identity/task/parallelism/retry/timeout/defaultstatus stay unchanged.

## Owner application steps

Open [Sandbox](https://jayburd-dev-web-9088794002.us-central1.run.app), sign in
normally and choose All transactions. If offered, choose Set up private
transactions → Save setup review → Create private workspace → Acknowledge
setup result. An already-ready workspace skips this step.

Under Use an existing connection, choose Review connection for the intended
book. Review every registered account and the sync-pause notice, then Save setup
review → Adopt reviewed connection history → Acknowledge setup result. The
private list refreshes with stored history. All registered accounts in that
eligible connection are adopted; existing automatic syncing pauses. No provider
fetch, new bank connection, transaction copy or journal is created.

Use Check saved private setup and exact retry controls for uncertain outcomes;
never start unrelated work or clear claims through SQL. Only currently authorized
eligible connections appear. New account registration still requires a book;
fresh syncing/consent and connect-first240 remain separate.

## Predecessor pins and rollback boundary

Registry prefix us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/:

- api@sha256:88a946acc77363c4eb8e9c9d43133e4a556bff9c5a6cd7816de8cf06246a7404 (jayburd-dev-api-00024-r9d).
- web@sha256:53e0757f4990614919cec13e0fb633abb8d9182d805da5852e26528dbbfa68af (jayburd-dev-web-00024-cf5).
- migrate@sha256:cde714aed27e71fd92e3b2e547b09c23d21de6090a65f90e10d58613137a220d (embedded46).

Any separately reviewed image rollback retains schema47 and a47-aware default-status
migrator; the old migrator pin is historical. Old UI/API cannot recover a pending
onboarding command and safely fail busy, so an image rollback is not proof of
complete workflow recovery. Prefer reviewed forward repair retaining the current
onboarding recovery client. Never clear claims with manual SQL or infer absence
from old clients. Retained onboarding history prohibits schema47 down.

## PR handoff

Source, candidate and managed release acceptance complete. Owner enters no UID
or SQL; the mounted application presents eligibility, review, confirmation,
durable replay/recovery and acknowledgement. Deployment creates no private
workspace or adoption by itself. Fresh bank sync/connect-first240, metadata242,
PFC413 and live bookkeeping policy remain separate.

Independent final documentation review recomputed both complete migration
receipts, checked exact pins/times and selected API/web/migrator image-only
spec deltas, active startup health, sole100 and actual owner UI labels. No
findings; nine anonymous probes are root-reported evidence, not repeated by the
reviewer. make check PASS with installedGo1.26.5/external exact-dependency
modfile; no tracked toolchain/dependency change. This PR changes documentation
only over exact950 source whose complete trusted release checks passed.
