# Development image delivery

## Decision and scope

Issue #27 first implementation slice, 2026-10-11: owner-triggered, main-only
publication of tested API, migrator and standalone Next.js images. No deployment
is coupled to publication. Sources G14–G16 in the [register](../research/gcp-sources.md)
support the packaging, metadata and authentication choices; live federation is unverified.

The build workflow reuses Seed validation before building linux/amd64 candidates.
Each image is built once, tested by its local content ID, scanned with pinned Trivy
for HIGH/CRITICAL vulnerabilities including unfixed findings, inventoried as
CycloneDX, then pushed without rebuilding. Buildx metadata records build provenance.
The receipt links commit, run/attempt, local configuration digest and registry
manifest digest. Tests reject missing build evidence and mismatched identities.

Metadata is unsigned evidence, **not a signed attestation or SLSA conformance**.
Docker's load exporter does not retain image attestations. Keeping unsigned
build metadata is an explicit first-slice limitation; registry-first attested
publication would require a separately controlled quarantine/promotion path.
Do not use these receipts as an automated deployment authorization.

## Identity and ownership

The workflow accepts no arbitrary ref or shell input. Both its guard and existing
foundation WIF require owner actor ID 55423093, repository ID 1414515318, main and
workflow_dispatch; provider additionally pins owner ID and workflow path.
`GCP_PROJECT_ID` and `GCP_PROJECT_NUMBER` are public repository variables. Foundation
WIF stays disabled until owner review and explicit apply; this change does not
enable it. Registry writer is the builder's only cloud resource role.

Cloud authentication occurs after build/test/scan. The action creates no credential
file; a short-lived masked access token passes directly to Docker login. Docker
configuration is runner-temporary and excluded from all image contexts/artifacts.
No Terraform state, secret-manager access or runtime impersonation is involved.
Credential-free PR CI only builds/tests locally. Default commands never publish.

Tags include commit, run and attempt, compatible with immutable registry tags.
A failed matrix can leave a partial set of images; require all three same-run
successful artifacts before considering any future release. Failed publication
never grants rollout authority. Artifacts expire after 30 days; retain an approved
release receipt separately before this expiry. Old tagged candidates require a
reviewed retention policy; current cleanup only deletes old untagged images.

## Frontend boundary and next slice

The non-root/read-only standalone image includes traced runtime files and static
assets. It intentionally has no build-time Firebase identifiers and shows unavailable
sign-in. Next public variables and rewrite destinations are baked into builds;
setting Cloud Run environment variables later does not repair that image.

Before deployment, implement and test runtime public identity configuration and
an explicit backend-for-frontend path: preserve the end-user Authorization token
and add a server-obtained service token in X-Serverless-Authorization for the
private Cloud Run API. A plain rewrite cannot do this. This is a proposed design
for the next slice, not an implemented authentication guarantee. Alternatives are
a public API with reviewed end-user ingress or environment-specific frontend builds;
those introduce different exposure or build-promotion tradeoffs.

No financial message or SQL contract changes; existing ISO assessments remain
applicable. Acceptance: receipt integrity tests and isolated runtime proofs run
in CI. Actual registry publication, IAM denial, real MFA, Infisical acceptance,
serialized migration/rollout and compatible rollback remain #27 release gates.
