<!-- stage-runbook: true -->
# Provider secret adapters

## Status and boundary

Implemented approved finite-pool writer, durable registry and runtime reader.
Based on ab2e1f4, while predecessor PR109 was open at task assignment. This lane
must remain stacked until that dependency is actually merged.

## Stage ledger

- Read source/runtime contracts, SQL reference allocation and official Infisical/
  Google API sections. No credential stores, environment values or secret outputs
  were accessed.
- Coordinator approved eight precreated slots, immutable GCS create intent and
  propagation pins. Implemented one source POST only on fresh acknowledged intent;
  recovery is read-only at the source and never repeats exchange or rotation.
- Slot source keys slot-00..07 and prefix provider-credentials-v1 frozen with
  worker/runtime and Terraform lanes. No ordinal mapping or direct-read fallback.

## Current evidence

`go test -race ./internal/providersecrets`: PASS, 11 top-level groups, including
source version 7 / logical version 4 / destination version 23; unknown source and
mapping acknowledgements; concurrent same-reference creation; delayed propagation;
pool exhaustion; original-digest conflicts; empty destination requirement; wrong Item,
project, numeric alias, name and CRC; HTTP bounds and redacted preflight.
No hosted credentials or live qualifications were used.

## Remaining work

Run default gates and integrate worker/API composition. Hosted IAM, account
capabilities, propagation and operator-injected credentials remain explicit gates.

## PR handoff

Own providersecrets package plus linked contract/runbook only. No SQL, Terraform,
shared dependencies or cloud mutations. Preserve one-way runtime secret delivery.

Default `make test build`: PASS. The initial `make check` rejected the requested
secret-shaped documentation filename; coordinator approved provider-vault.md instead
of weakening the protected-path guard. Final check and staged guard follow.
Final `make check`: PASS after the approved document rename. Final package race
suite: PASS, 11 groups. Independent worker-lane review found no additional resolver
or GCP-reader blocker after the bounded-token validator fix. Exact canonical numeric
Google project names are configured explicitly; no project alias is inferred.
