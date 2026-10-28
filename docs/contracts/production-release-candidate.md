# Offline Production release candidate — #316 stage B

Coordinator source decision2026-10-27; security review pending before merge.
This bounded stage validates public metadata only. It neither authenticates nor
emits deployment inputs. Success is candidate-valid, never deployment approval.
No runtime, SQL, financial recognition or external message changes. STD-01:
operational release metadata has no applicable ISO20022 message; existing
ISO4217 and ISO21378 applicability/access limits remain unchanged.

## Contract and authority

Accept an exact-byte SHA256 from a separate caller, a closed JSON schema and a
finite seven-day window. schemaVersion1, environment production, deployable false,
operation prepare and scheduling paused are mandatory. Source commit, four image
digests (api/web/worker/migrate), migration35 and checksum, numeric secret versions,
project/number/region and named evidence digests are explicit. Derive expected
Production instance, registry, keys and secret resource names from that same
project; reject development resources and the known development project number.
No secret contents, arbitrary URLs or extra fields are accepted. Reject rollback:
the first pilot has no accepted compatible prior Production deployment.

Evidence fields are SHA256 bindings to separately reviewed mandate/build/schema/
identity/secrets/recovery/qualification receipts, not claims those receipts exist
or passed. This validator cannot prove build ancestry, image provenance, effective
IAM, actual secret delivery, migration status, account consent or hosted tests.
Subsequent stages must fetch independently trusted metadata, compare receipt bytes,
verify their semantics and owner/security approval before any cloud authentication.
Never connect this candidate-only function directly to deployment or activation.

Malformed input returns one fixed error with no submitted content. CLI accepts
only a public candidate path and approved hash; outputs a fixed candidate-only
message. No Terraform variables, files, network calls or process commands emitted.
Encoding is compact JSON.stringify output plus one newline, in submitted member
order. Duplicate keys, alternate whitespace and alternate value encodings fail
roundtrip comparison. Exact bytes, including member order, remain hash-bound.

## Evidence, alternatives and acceptance

Reviewed internal production-runtime.md, production-webhooks.md and existing
release/worker manifest validators at4e4a172. PP08–11 in
[Production sources](../research/production-pilot-sources.md) establish key/reference,
identity and cost mechanisms; naming/closed schema/finite window are project policy.
No new vendor mechanism or normative source claim. Extending the dev manifest was
rejected because it emits dev Terraform inputs and could silently mix profiles.
A standalone candidate validator preserves existing Sandbox behavior while staged
Production infrastructure and trusted evidence verification remain outstanding.

Named tests in scripts/production-release-candidate.test.mjs cover exact bytes,
mixed targets, static version pins, finite windows, paused scheduling, malformed
and unknown input, image pinning and fixed error output. This is release-tooling
coverage, not hosted product qualification. Core gates and existing release tests
must pass; no new migration or live data fixture. Remaining #316 stages C–H and
actual #337 acceptance are required before real deployment.

STRIDE review: tampering/profile substitution(high) is constrained by exact hashes,
closed resource naming and canonical encoding; disclosure(medium) by fixed errors
and public-only fields; authority spoofing(high) remains unresolved until independent
receipt/owner verification and is bounded here by no execution capability. Denial
of service(low for local tooling) has a32KiB parsed-input bound. Runtime key/IAM,
network, provider and bookkeeping boundaries are unchanged. Human review is needed
before integrating this code into a deployment authority path.

## Build receipt comparison — follow-up at05716f1

Before implementation2026-10-28: reuse the reviewed owner/main/run identity
contract from image-delivery.md and scripts/release-manifest.mjs, extending to
four Production images. No new external protocol or vendor mechanism. Input is
candidate bytes/hash, canonical build-receipt bundle bytes, and a separately
trusted GitHub run object. The caller must retrieve that object via authenticated
GitHub metadata; parsing an untrusted file as the trusted run is not verification.
This pure function performs no retrieval or authentication and is not wired to CI
or rollout. Trust separation must be enforced by the future orchestrator.

Pin the proposed Production workflow path build-production-images.yml, canonical
repository ID1414515318, owner actor ID55423093/login, main/workflow_dispatch,
success, exact commit/run/attempt. Existing dev workflow cannot qualify. That
Production workflow is not implemented yet; no real build can be declared proven
by fictional fixtures. Candidate evidence.build binds exact canonical receipt
bundle bytes; all four unique components must match candidate images/source/run.
Each closed receipt records environment production, candidate-only/nondeployable,
passed high/critical scan and component runtime tests; worker requires worker and
webhook tests. Claims in artifacts are only as trustworthy as their publisher;
workflow/scan/ancestry verification remains separate before deployment.

Acceptance: reject failed/fork/foreign actor/event/path/run/attempt/source, changed
bundle hash, missing/duplicate/swapped/tagged images, absent scan or webhook proof,
unknown metadata and secret-bearing extras; retain fixed diagnostics. The result
remains deployable:false. This reduces cross-build confusion; it does not prove
source ancestry, schema compatibility, effective IAM or actual hosted qualification.
