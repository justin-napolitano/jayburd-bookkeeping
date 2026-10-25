<!-- stage-runbook: true -->
# Private identity transport — #299

## Status and boundary

Lease IDENTITY299-PRIVATE-TRANSPORT through2026-10-25T01:44Z, source-only.
Dedicated feat/299-private-identity-transport, initially3763b579 and rebased onto
merged330/main1bacecda. No SQL, IAM,
deployment, secret retrieval, provider call, personal data or consent renewal.

## Stage ledger

| Stage | Evidence / limit |
| --- | --- |
| Resume | Index/bibliography,299/300/272/65, current main, verifier/projector contracts/source/runbook reviewed;328 source lease released into330 review |
| Decision | Official sync request/account/count/readiness and custom-user date sections reviewed before implementation; narrow owner-private producer chosen over hosted authority expansion |
| Implementation | Separate default-inert owner-private command, fixed one-request Sandbox/account/version profile, fixture-before-credential validation, closed injection fields, bounded in-memory projection/private map |
| Timeout finding | Initial actual inherited pipe hung in syscall.read despite context Close; stopped only that synthetic test, reviewed Go NewFile pollable source, corrected shared opener without increasing deadline |
| Focused gates | Exact request/version/account, changed observed finance with fixed identical map, private descriptor/file/symlink/replay, redirect/HTTP/oversize/foreign/incomplete/duplicate/cancel failures PASS; actual1second blocked input expiry now PASS |
| Core | `make check test build` PASS, including full race suite and new local binary; local cached Go1.26.5, tracked1.26.10 unchanged |
| Comparison regression | `make sandbox-verification-test` PASS against actual restricted disposable PostgreSQL, including expected values/review identity/foreign history/pins/concurrent revoke; no hosted proof |
| Final review | Actual raw blocking pipe deadline and private socket tests, lost terminal acknowledgement/repeat-without-I/O cases and focused race PASS; final rebased `make check test build` PASS after330 merge |
| Canonical CI correction | Head246ff62 Linux1.26.10 rejected two positive fixtures that wrapped already Go-owned os.Pipe descriptors again. Reproduced the same failure with pre-fix Linux binary in non-root/read-only/network-none container. Corrected fixtures use actual inherited-style raw blocking descriptors with one Go owner; preserve all assertions and require successful open plus actual expiry in the blocked-read case. No production opener/deadline weakened or timeout increased; corrected Linux/local results recorded before push |

## Current evidence

Offline projector and new private producer validate exact preauthored tags and
copy only IDs. Tests prove observed finance cannot alter expected fields or map.
Both use the corrected type-checked nonblocking/pollable descriptor opener. Default
is offline; invalid fixture/hash/scope/expiry opens no credential pipe. One HTTP
response enters the projector directly; no raw output/file, cursor or DB exists.
An acknowledgement failure preserves the complete private map, with repeat refused
before credential/provider work; this does not claim power-loss filesystem durability.
Actual authorized owner injection/transport and subsequent comparison remain missing.
Human security-sensitive source review of330/331 is still pending.330 was merged
after CI without a recorded human review;328 and dependent319 were reopened so
the repair activation closure gate remains blocked. No live activation occurred.

## Remaining work

Final corrected-head CI and human transport/security review. Owner must separately
supply an authorized private injection path and fresh finite source scope; helper
does not retrieve/decrypt an application Item or grant such authority.300 actual
equality/cleanup stays open; no existing source consent or approvals are renewed.

## PR handoff

Refs299/272/300/88/90. No migration or browser/API/hosted runtime/IAM change. Local
binary is built by Makefile; it is not silently added to a deployed service/job.
Required core and focused synthetic gates, marked runbook and staged validation
before every commit. Canonical exact-head CI remains required. No actual provider
transport/financial equality, owner credential access or live readiness is claimed.
