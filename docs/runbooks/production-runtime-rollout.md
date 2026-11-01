<!-- stage-runbook: true -->
# Paused Production runtime rollout — #370

## Status and boundary

PROD370-RUNTIME; build/370-production-runtime fromcf335da. Source candidate for
paused API/frontend and independent static readback, not an applied deployment.

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
| Actual services | Only existing Sandbox services present; canonical Sandbox shell HTTP200, no Production application deployed |

## Current evidence

Production foundation/database/schema35 and static secret version1 deliveries
have actual evidence in shared-production-operations and production-build-operations.
Those operations do not prove a Production app is deployed or a bank is qualified.
Current candidate root is separate, defaults absent, leaves provider authority
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
not supply a current application deployment approval or full release-bundle proof.

The old build target expired2026-10-31T20:00:00Z. This branch proposes a fresh
target through2026-11-03T04:00:00Z, hash
b58f53138cf632257f9c18ef1035622e59b757ab759b8b39dc0780ad096f75b9.
No automatic extension, dispatch or apply follows. Human review covers new exact
bytes and source first. Local offline build supplied only go.mod/go.sum-governed
public module cache entries, retaining checksum verification with GOPROXY=off;
TLS was not weakened. No owner environment, state/plan or credential files were
read/copied. Default tests and container proof had no live provider/cloud access.

Before future foundation toggles/apply, refresh the operator's source-only .tf/
lock copies from the exact reviewed commit; do not reuse a stale pre370 foundation
copy that lacks the new identity-lookup resource. Private backend/state remain
in their existing operator locations and are never inspected/copied into source.

## Remaining work

Human security review, exact scanned image, reviewed public
metadata input/hash and isolated backend apply. Independently verify readiness,
database/identity routing, private API denial, public auth config and static readback.
Then worker SQL/runtime, receiver, mandate/qualification and owner-private pilot
remain open in316/337/365/90. No bank login requested by this source-only slice.

## PR handoff

Refs #370/#316/#365. Source/local synthetic evidence above; exact-head CI and human
review pending. No deployability or live
bookkeeping approval. No automatic grant or cloud mutation from default commands.
