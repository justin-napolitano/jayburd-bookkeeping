<!-- stage-runbook: true -->
# Paused Production runtime rollout — #370

## Status and boundary

PR371 merged0ea7eb6 after all five exact-head checks passed and explicit owner
review/deployment approval. Paused Production API/frontend are deployed; owner
enrollment and password/TOTP workspace sign-in are proven. Independent static
readback and personal-bank registration remain pending.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Owner approves369/rollout;369 mergedcf335da after five checks; scanned build40987388375 succeeded |
| Gap | Existing Terraform runtime derives Sandbox; Production API needs explicit expired profile for isolated SQL routing |
| Design | paused-production-runtime contract records source sections, alternatives, boundaries and acceptance |
| Implementation | New /credential-check, inert Production runtime root and API-only revocation lookup role |
| Focused | Exact pins, finite authority, no-I/O denials, fixed output and returned-buffer disposal PASS |
| Infrastructure | Six Production runtime mocks and three foundation mocks PASS; explicit role/member checked with plan-time overrides |
| Core | make check/test/build PASS; local Go1.26.5/external temporary modfile, tracked1.26.10 unchanged |
| Container | Pinned Go1.26.10 offline build and network-disabled/nonroot/read-only command suite PASS |
| Integration | make app-integration and additional --production journey PASS; synthetic identity limitation retained |
| Actual services | Production API00001-wz6 and web00001-7vb Ready; canonical shell HTTP200, private unauthenticated API403 |
| CI correction | Infrastructure CI caught the new worker command missing from webhook ancestry inputs; added its path without removing any guard |
| Owner identity | Exact console-verified issuer/subject enrolled through existing identity-operator function; enabled mapping and exactly one audit row verified; private owner MFA workspace sign-in observed |
| Fresh build | Build41263883683 attempts1/2 timed out at reusable validate20m; no images published; builder disabled after terminal outcome |

## Actual paused deployment

Service-only targeted apply excluded the new credential-check job. All four
historical component receipts from build40987388375 were recomputed; API/web,
internal code, module locks and their Dockerfiles were unchanged between that
source and371. API3b7a503a and weba65d8b03 remain pinned to full digests recorded
below/in production-build operations. Runtime public-metadata input SHA256
209c95a1f30a5ae3a698c1431a9e118b124725ad33bfd16a6467af3f28c05f2a.
Canonical frontend https://jayburd-prod-web-9088794002.us-central1.run.app.
Actual API invoker contains only Production frontend; API identity/image/expired
Production routing arguments independently verified. Single lookup role contains
only firebaseauth.users.get. No migration, instance change or provider call.

Owner separately approved exact Production enrollment. Existing operator SET
capabilities sufficed; no new IAM/role grant. Serializably invoked only
ledger.enroll_principal under identity-operator capability; separate read-only
resolve/audit checks returned true/true. Operator proxy stopped. Owner's new
Production tab reached Financial position with no books or pending command.
No password, code, token, financial record, state/plan or credential value read.
Redacted actual records: issue370 comments6552451191,6552567978,6552602507.

## Current evidence

Production foundation/database/schema35 and static secret version1 deliveries
have actual evidence in shared-production-operations and production-build-operations.
Those operations do not prove a Production app is deployed or a bank is qualified.
The applied root is separate, defaults absent, leaves provider authority
expired and creates no scheduler/worker/webhook. The new command reads only exact
GCP versions in runtime memory with fixed diagnostics; no provider credentials
or payloads belong in source, logs, artifacts or chat. No migration or financial
contract change. Existing verified MFA/current entity authority remain mandatory.

Actual build40987388375 fromcf335dab2f7edf67f67bb9506b64b6c2007ac0cf succeeded for
all four images. Separately authenticated owner/main/workflow/run/attempt metadata
and all component receipts matched. API inventory retains go-jose v4.1.6; no
HIGH/CRITICAL findings at that scan. Worker digest is
sha256:636fcd4f58f84d06880fda19ccc7bf5e6e10d28c6a81f4c8dad606bdfca1dcec.
That worker includes369's CLI fix but not this new credential-check command.
Builder federation readback is disabled. Historical build receipt equality does
not supply full release-bundle proof or live-bank approval. Separate owner371
approval authorized the bounded paused app deployment described above.

The old build target expired2026-10-31T20:00:00Z. This branch proposes a fresh
target through2026-11-03T04:00:00Z, hash
b58f53138cf632257f9c18ef1035622e59b757ab759b8b39dc0780ad096f75b9.
Owner explicitly approved371 and these exact target bytes before the deployment.
No automatic extension follows. Local offline build supplied only go.mod/go.sum-governed
public module cache entries, retaining checksum verification with GOPROXY=off;
TLS was not weakened. No owner environment, state/plan or credential files were
read/copied. Default tests and container proof had no live provider/cloud access.

Before future foundation toggles/apply, refresh the operator's source-only .tf/
lock copies from the exact reviewed commit; do not reuse a stale pre370 foundation
copy that lacks the new identity-lookup resource. Private backend/state remain
in their existing operator locations and are never inspected/copied into source.

## Remaining work

Repair build timeout372, qualify fresh images and deploy/execute exact static
readback. Production reference metadata and first owner-reviewed personal book
remain absent; Sandbox preset is not a live accounting-policy approval. Then
worker SQL/runtime, receiver, remaining mandate/qualification and owner-private
bank pilot remain open in316/337/365/90. Sign-in is proven, bank access stays paused.

## PR handoff

Refs #370/#316/#365/#372.371 exact-head CI and explicit human review passed;
actual paused app/enrollment evidence above. No full pilot qualification or
automatic cloud mutation from default commands.
