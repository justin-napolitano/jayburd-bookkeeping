# Development image delivery

## Decision and scope

Issue #27 first implementation slice, 2026-10-11: owner-triggered, main-only
publication of tested API, migrator, worker and standalone Next.js images. No deployment
is coupled to publication. Sources G14–G16 in the [register](../research/gcp-sources.md)
support the packaging, metadata and authentication choices; live federation is unverified.

The build workflow reuses Seed validation before building linux/amd64 candidates.
Each image is built once, tested by its local content ID, scanned with pinned Trivy
for HIGH/CRITICAL vulnerabilities including unfixed findings, inventoried as
CycloneDX, then pushed without rebuilding. Buildx metadata records build provenance.
The receipt links commit, run/attempt, tested local content ID (configuration or manifest/index, depending on Docker store) and registry
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
A failed matrix can leave a partial set of images; require the complete four-component build to succeed. The existing application
release validator still consumes its three API/migrator/web receipts; the worker
receipt is a separate candidate and does not authorize worker deployment. Failed publication
never grants rollout authority. Artifacts expire after 30 days; retain an approved
release receipt separately before this expiry. Old tagged candidates require a
reviewed retention policy; current cleanup only deletes old untagged images.

## Frontend boundary and next slice

Failed HIGH/CRITICAL scans retain only `receipts/vulnerabilities.json` in a
separate `rejected-scan-*` artifact for seven days. This vulnerability-only report
is diagnostic evidence, never a candidate receipt or release authorization. Later
authentication/publication steps stay skipped after scan failure. Missing reports
(for example scanner download failure) warn without changing the failed outcome.

The non-root/read-only standalone image includes traced runtime files and static
assets. The [runtime identity/private API bridge](private-api-bridge.md) now reads
operator-supplied public identifiers at request time and separates the frontend
service token from the user's bearer. An unconfigured image remains unavailable.
No identity identifiers or backend destination are baked into its build.
Runtime excludes npm/Yarn and requires patched OpenSSL packages; the builder
retains npm for locked installation. APK patch resolution may advance, so release
authority binds the final tested/scanned digest. See [source G28](../research/gcp-sources.md#g28--node-runtime-image-contents--2026-10-12).

The next deployment slice must wire runtime settings and the API Invoker grant,
then prove real IAM/MFA behavior. Candidate receipts remain non-deployable until
migration/rollout and cloud acceptance are reviewed; a passing bridge test does
not approve publication or establish real authentication.

No financial message or SQL contract changes; existing ISO assessments remain
applicable. Acceptance: receipt integrity tests and isolated runtime proofs run
in CI. Actual registry publication, IAM denial, real MFA, Infisical acceptance,
serialized migration/rollout and compatible rollback remain #27 release gates.

## Acquisition worker candidate

`Dockerfile.worker` has an explicit allowlisted build context and a non-root scratch
runtime. Its isolated container proof checks fail-closed startup without any cloud
network. Real managed startup still needs reviewed metadata, Cloud SQL role grants,
private callers and exact provider-store versions. The owner-operated workers root
owns initial infrastructure; the application deployment identity receives no new
bootstrap/secret authority. Automated worker promotion is not implemented here.
