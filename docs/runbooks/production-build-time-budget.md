<!-- stage-runbook: true -->
# Full release validation time budget — #372

## Status and boundary

PROD372-CI; build/372-release-validation from0ea7eb6. Increase only the reusable
backend validation timeout20->40minutes. Every test, image proof, dependency scan,
owner-only publication condition and finite target remains unchanged. No product,
SQL/migration, financial message, IAM or provider authority change.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Actual Production owner sign-in proven; issue370 redacted operation records |
| Diagnosis | Build41263883683 attempts1/2 cancelled at validate20m; frontend/integration passed, image publication never started |
| Decision |40m cold-cache budget preserves all checks; alternatives skipping checks or repeatedly retrying20m rejected |
| Implementation | Only validate timeout changed; actual rollout status corrected |
| Validation | make check/test/build and release-test PASS; pinned actionlint1.7.7 PASS; exact-head CI pending |

## Current evidence

Source review: existing pinned seed workflow validate job and authenticated
GitHub run/job metadata, including completed SQL checks followed by timeout during
container proof. These establish measured operational duration, not a failed
assertion or an application defect. No new vendor mechanism or financial standard
mapping; no external source added. Application/identity/secrets/public-exposure
controls are unchanged. Operational timeout gives CI more time, not more cloud
authority. Production builder disabled after each terminal attempt.
Local core checks used host Go1.26.5 with the existing external temporary modfile;
tracked Go1.26.10 remains unchanged. Initial sandbox loopback restrictions stopped
listener tests; the authorized synthetic rerun passed. Pinned actionlint download
needed network authorization; TLS/checksum verification remained enabled.

## Remaining work

All exact-head CI; merge; fresh owner/main build with current exact target; all
four immutable runtime/scan/provenance receipts; disable builder after completion.
Actual new credential-check readback remains370/365. Owner bank pilot remains90.

## PR handoff

Refs #372/#316. No migration or deployment authority renewal. Retain all existing
checks and bounded cloud capacity; do not infer bank readiness from a build.
