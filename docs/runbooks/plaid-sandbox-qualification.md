<!-- stage-runbook: true -->
# Plaid Sandbox qualification diagnostic

## Status and boundary

Issue #88; coordinator-approved bounded read-only harness. No provider, secret,
cloud or real application database has been accessed by this lane. Overall Sandbox
qualification remains unperformed. The diagnostic deliberately cannot emit an
overall pass, even when exact-decoding and repeat-delta checks pass.

## Stage ledger

Reviewed AGENTS, issue #88, decoder/acquisition/connections and webhook boundaries,
current official Sandbox/Link/Transactions docs. Reported real Link prerequisite:
ordinary Link session event enablement or reviewed Hosted Link implementation is
needed before current VerifySession can qualify association. Coordinator approved
an offline-default pipe-fed sync diagnostic and explicitly deferred webhook firing.
Implemented strict bounded input, production HTTP/decoder reuse, aggregate page/
change/byte budgets, cancellation, fixed redacted receipt and executable offline
proof. Inherited pipes become nonblocking before Go wrapping so five-second input
read deadlines actually apply. Invalid flags are never echoed as diagnostics.

## Current evidence

`make check test build` PASS, including all-package race tests.
`bash scripts/test-sandbox-qualification.sh` PASS (focused race tests, executable
build and default blocked/offline receipt with exit2). Logs:
`/private/tmp/jayburd-sandbox-default.log` and
`/private/tmp/jayburd-sandbox-offline.log`. Synthetic tests exercise exact cents above float precision,
pagination and repeat cursor, scope mismatch, parser rejection, missing/null currency
precision, credential redaction, opt-in denial, ordinary-file descriptor rejection,
actual pipe read deadline, cancellation and aggregate budgets. Tests perform no
provider calls; an injected fake RoundTripper receives fictional credentials only.

## Operator sequence

1. Run `bash scripts/test-sandbox-qualification.sh` for offline proof, or build
   `go build -o /your/private/output/sandbox-qualify ./cmd/sandbox-qualify`.
   Default executable exits 2 with a blocked/offline JSON receipt. This is expected.
2. Complete coordinator readiness review: Sandbox account only, real Link proof
   path, owner-approved secret injection, deployed app/SQL evidence and hosted IAM.
3. Only after explicit provider execution authorization, the operator's private
   secret injector supplies the [contract input](../contracts/plaid-sandbox-qualification.md)
   on inherited pipe FD >=3. Invoke the binary with
   `-mode sandbox-sync -allow-sandbox -input-fd 3`. Never put values in arguments,
   process environment, repository files, shell history, screenshots or agent output.
   The injector must finish/close its write end within five seconds. Regular files,
   stdin/stdout/stderr and TTY input are rejected. No agent reads that pipe.
4. Save only the redacted receipt if desired. Exit 3 means partial diagnostic
   evidence, never completed qualification; exit 2 means blocked. Investigate fixed
   blocked checks using owner-controlled provider/app tooling, without copying raw
   errors or payloads to this repository. Provider-not-ready is not an empty book.

This mode does not modify source SQL, generate transactions, refresh an Item,
exchange/rotate tokens, fire webhooks or post journals. A read-only provider probe
cannot prove application no-journal effects. Repeat delta can legitimately change;
no automatic retry is attempted. Cursors are not retained for later runs.

## Remaining work

Full #88 real Link/exchange, dynamic pending-to-posted, signed webhook delivery,
provider-to-review exact consistency, persisted cursor/idempotency and no-journals
proof remain explicit blocked receipt checks. Infisical/runtime evidence design and
hosted deployment qualification belong to the coordinator's other lanes. Sandbox
simulation is not production institution history/balance/consent correctness.

## PR handoff

Owned new qualification package, executable, offline script, contract and this
runbook only. No SQL migration, shared API, CI, dependency or production adapter
change. Primary source annotations SQ01–SQ05 are in the contract; coordinator owns
shared bibliography/index links. Do not close #88 from this partial harness.
