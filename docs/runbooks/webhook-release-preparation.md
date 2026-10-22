# Webhook release preparation — #271
<!-- stage-runbook: true -->

## Status and boundary

Lease `PLAID-271-PREP-20261022T1725Z`, implementation only, expires22:00Z.
Base: merged PR270 `a2c9dee793f81ed483055b0ee916876b59265d27`.
Prepare isolated infrastructure, bootstrap, enrollment and exact release evidence.
No provider/cloud operations, credential access, consent renewal or deployment.
New public exposure requires owner review of concrete release artifacts.

## Stage ledger

- Rechecked live #65/#271, recorded owner approval and merged unchanged green270.
- Claimed successor; reserved three independent lanes and reviewed interfaces.
- Reviewed existing release, connection, runtime, webhook and research contracts.
- Preparing strict offline manifest/readback and optional fixed enrollment metadata.
- Integrated enrollment8265166 and infrastructure4a2eab6 after source review.
- Integrated explicit SQL bootstrapcc1705c; independent reviews confirmed socket
  comma denial, current-state replay protection and emergency owner disable path.
- Independent review found and resolved manual-scaling budget, omitted ProtoJSON
  defaults, disabled canonical URL/IAP, and h2c readback gaps. Regression fixtures
  pass; reviewer re-read confirms resolution. No actual readback claimed.
- Wired default CI to new isolated infrastructure and disposable SQL bootstrap
  checks; optional scoped infra targets permit avoiding unrelated foundation runs.
- Root review found powerful predefined SQL roles lack superuser flags. Extended
  the existing per-request receiver check to reject inherited `pg_` memberships,
  matching bootstrap policy. Added actual-session grant-drift denials without
  server file access or program execution. Migration/financial semantics unchanged.

## Current evidence

PASS: complete `make release-test`, including five new fixture groups and all
existing application/worker release regression tests. Root targeted
`scripts/test-infra.sh runtime webhook-bootstrap webhook-runtime` PASS:
existing runtime plus two webhook roots (16 new mocked scenarios). Initial scoped
run stopped on formatting, corrected; sandbox DNS retried with approved escalation.
Provider downloads only; no real provider resources.

PASS: `make app-integration connection-browser-test`, both actual Chromium →
production Next bridge → Go → disposable PostgreSQL journeys, using unchanged
locked dependencies via local untracked node_modules symlink. This includes
enrollment/acquisition/review and explicit synthetic posting/recovery. Test-only
identity limitation remains; no actual Plaid, MFA, cloud or live-use evidence.
Initial loopback denial was retried with approved escalation.

PASS: integrated `make webhook-bootstrap-test` and `make webhook-test` against
disposable PostgreSQL29. Actual restricted-session denial covers later grants of
API, program, file and all-data roles with zero receipts. Bootstrap covers replay,
revocation, mixed/delegated/direct privileges and sanitized wrapper execution.
Docker sandbox denials retried with approved escalation; no actual provider/SQL
host was contacted. Bootstrap lane also passed migration/cloud-sql/ledger gates.

PASS: final `make check test build` (project/link/runbook/migration checks, Go vet,
full race suite, all executable builds). New runbook headings were corrected;
loopback denial retried without weakening tests. Local Go1.26.5 uses the external
modfile/cache recorded in the preceding swarm; tracked Go1.26.10 is unchanged and
remains CI's required toolchain. Root logs are `/private/tmp/jayburd-271-*.log`.

PASS: `make web-check web-test web-build web-browser`:324 unit tests,24 Chromium
browser tests, generated API drift, type check and production build. Browser
loopback retry passed; no skipped scenario is counted. Temporary dependency
symlink removed after completion. No hosted qualification claimed. No manual
foundation or volume proof was rerun.

## Remaining work

Submit the preparation PR. Human review, a fresh trusted
image, concrete public exposure approval and current source consent precede actual
Sandbox qualification. #271/#88/#272 remain open; no cloud execution authorized.

## PR handoff

Dedicated branch `issue/271-webhook-release`, root worktree
`/private/tmp/jayburd-271-release`. Original checkout/lock untouched. New roots and
bootstrap are explicit operator tools; no active release file or WIF expansion.
