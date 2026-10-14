<!-- stage-runbook: true -->
# Hosted worker composition delivery

## Status and boundary

Runnable local composition is implemented. No deployment, credentials, live provider
call or hosted-readiness claim. Root owns Terraform, bootstrap and delivery workflows;
parallel lanes own the versioned secret resolver and durable cloud evidence store.

## Stage ledger

- Read current agent instructions, source index and worker/Cloud SQL contracts.
- Reviewed current Google service-identity, IAM login and container contract sections.
- Coordinator approved fixed publisher/sync logins, pool cap2 and mandatory durable
  evidence plus versioned credential resolver for the consumer.
- Implemented injected runtime composition, managed clients, bounded metadata tokens
  and proxy-free production HTTP transport. Default constructors perform no provider I/O.
- Integrated concrete versioned GSM/registry resolver and durable GCS evidence constructors
  in the consumer executable; strict production CLI resource pins match coordinator policy.
- Fixed an interrupted import edit and verified both worker modes again. Added a single-use
  runtime guard so a second Serve cannot close active dependencies.
- Added strict worker Docker context allowlist and bounded offline container proof script.

## Current evidence

Focused `go test -race ./internal/workerruntime ./internal/clouddb` passes with
synthetic metadata responses and temporary loopback listeners. Tests cover rejected
identity before SQL, cleanup ownership, fixed metadata endpoints/expiry/body limits,
invalid configuration, missing consumer adapters, cancellation, canonical Infisical
UUID audiences, CLI resource/version pins and rejection of a second Serve. Loopback tests required normal
sandbox escalation. No external service was contacted.

`make check test build` and explicit Linux/arm64 worker compilation passed with
final providersecrets f0ea1ad and provisional coordinator cloudevidence source.
Default log: `/private/tmp/jayburd-hosted-workers-default.log`.

The original Dockerfile.worker build reached dependency download but failed on
Docker's local TLS trust for proxy.golang.org (unknown certificate authority).
No TLS verification was disabled. A separately built scratch image containing the
cross-compiled worker passed four --network=none, read-only, dropped-capability,
nonroot startup rejection checks via scripts/test-worker-container.sh. This proves
binary startup behavior only; it does not prove the original multi-stage image build.

## Remaining work

Root must qualify actual Cloud SQL login,
IAM/OIDC and durable secret/evidence wiring before enabling hosted acquisition.

## PR handoff

Owned cmd/worker, internal/workerruntime, internal/clouddb, Dockerfile.worker,
its source allowlist, scripts/test-worker-container.sh and this contract/runbook only.
No SQL or infrastructure mutation. Parallel packages were copied only for local
compilation and are excluded from this commit. Root owns combined integration and
the original image build in CI.
