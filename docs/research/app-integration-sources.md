# Application integration sources

Reviewed 2026-10-11. These are engineering sources, not accounting or ISO
conformance evidence. Component-specific source reviews remain with their owners.

## A01 — openapi-typescript 7.x

[Official introduction](https://openapi-ts.dev/introduction), Setup and Basic
usage sections reviewed. The generator supports OpenAPI 3.1 and local schemas,
producing TypeScript declarations without a running API. Inference: generate UI
types from the repository contract and check for drift in CI. Types do not
validate runtime payloads or prove SQL invariants. The competing hand-maintained
interface approach is simpler initially but adds a second contract authority.
Acceptance: generated-file check, frontend typecheck and API fixture tests.
The installed 7.13.0 package's peer metadata requires TypeScript 5.x; pin 5.9.3
instead of forcing an incompatible major. The lockfile and typecheck validate
this compatibility choice, not merely a latest-version preference.

## A02 — Node.js release policy

[Official releases](https://nodejs.org/en/about/previous-releases), release policy
and supported version table reviewed. Node 24 is LTS; production should use an
LTS release. Inference: pin an available patched Node 24 version for web CI/builds
rather than the current non-LTS major. Actual package pins and dependency-audit
evidence belong in the lockfile and application runbook.

## A03 — Playwright CI setup

[Official CI guide](https://playwright.dev/docs/ci-intro), installation and test
workflow sections reviewed. Browser binaries/dependencies are explicitly installed
before tests. Inference: a distinct frontend CI job uses the lockfile, runs unit
and type checks, then synthetic Chromium journeys. No production provider or data
is required; mocked browser journeys prove UI behavior, not a live identity flow.

## A04 — GitHub setup-node

[Official setup-node repository](https://github.com/actions/setup-node), v6
reference resolved through GitHub to immutable commit
`249970729cb0ef3589644e2896645e5dc5ba9c38`. The action selects the CI Node toolchain;
pin it consistently with the existing immutable checkout/setup-go action policy.
No npm registry credential or cloud identity is needed for public dependencies.
