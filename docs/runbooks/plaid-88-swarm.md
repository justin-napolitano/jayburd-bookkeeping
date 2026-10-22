<!-- stage-runbook: true -->
# Plaid #88 swarm integration

## Status and boundary

Owner authorized three parallel lanes plus coordinator on2026-10-22. Lease
`PLAID-88-SWARM-20261022T1540Z`, expiry20:00Z, implementation only. Existing
17:00Z source consent is not extended. Baseline52ef702 includes merged #269;
its validate/frontend/application-integration/foundation/secret checks passed.
No real account, provider call, new IAM grant or cloud exposure in this stage.

Dedicated coordinator branch `issue/88-swarm-integration` in
`/private/tmp/jayburd-plaid-fixes`. Agents own isolated worktrees/reservations
recorded in #65. Root owns shared connector/SQL/API/deployment/index integration.

## Stage ledger

1. **Dispatch complete:** WEBHOOK103 host, CONNECTION104 recovery and VERIFY85
   independent lifecycle/fault proof. Existing issue bodies/dependencies exported
   from fresh GitHub reads; no canceled predecessor treated as complete.
2. **Design review:** injected exact-route webhook host approved; existing-wire
   strict Hosted exit/recovery fix approved. Do not relax reconnect/session gates.
   [Managed webhook contract](../contracts/webhook-managed-runtime.md) records
   dedicated connector identity and future release/security conditions.
3. **Coordinator connector complete:** allow only the dedicated webhook
   connector component with existing narrow pool settings; no user/grant creation.
   `go test -count=1 ./internal/clouddb` and `make check test build` pass using
   the documented external local Go1.26.5 modfile. Initial core run was blocked
   only by a synthetic listener sandbox denial; approved rerun passed. No
   foundation or hosted test was run. Diff and staged runbook guard before commit.
4. **Lane integration complete:** host66886ac, lifecyclea6cddf6,
   signed-socket SQL flow2835b2d, connection recovery41fd55e and managed
   runtime43b81d6 integrated after source review and staged guards. Independent
   review found no blocking initial host/recovery defect; the identified missing
   real-SQL socket and competing/denied/scope-changed UI tests were added and pass.
   Each lane's runbook names exact tests and synthetic limitations.
5. **Packaging/integrated validation in progress:** Makefile builds the webhook
   command; the existing worker image includes its explicit `/webhook` executable
   and offline exact-image failure proofs. No fifth image/release pipeline and
   no public deployment or grant are introduced. Combined core check/test/build
   and acquisition/provider-flow/webhook SQL gates pass on integrated source.
   Full local worker Dockerfile build failed its dependency download certificate
   chain, including approved outside-sandbox retry. TLS validation was retained;
   exact-image startup proof is not claimed locally. CI's existing worker image
   build/proof remains required. Log `/private/tmp/jayburd-88-worker-image-proof.log`.

Independent managed/CLI review found no blocking defect. Root review confirms
fixed paths and identity, no eager key/provider calls and reverse resource cleanup.
Connection lane passed324 unit tests,24 Chromium cases, generated API/types,
production web build and both full browser/API/disposable SQL journeys. Its
temporary dependency symlink was not committed. Sync lane passed acquisition,
sync-state/provider-flow/ledger (including restore); host passed socket/race and
SQL receipt tests. None is a real provider or live-account result.

## Current evidence

Plaid Dashboard is a separate sign-in; Launch Center review is pending owner
authentication in the opened Dashboard tab. Independent local work continues.
No secret/key page or credential was read. The existing bookkeeping tab's
authenticated state was not mistaken for Dashboard access.

## Remaining work

Integrate passing bounded changes, review managed construction, prepare the
concrete release scope and execute authorized hosted cases only after prerequisites.
Full #88 qualification remains open, including the live signed chain and exact
hosted comparator; #90 is separately gated. New public exposure and sensitive
runtime changes require human security review under the applied security skill.

## PR handoff

Refs #88/#103/#104/#85/#87. No product migration yet, no changes to merged SQL.
Do not infer deployed behavior from synthetic tests or this implementation plan.
