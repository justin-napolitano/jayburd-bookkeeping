<!-- stage-runbook: true -->
# Optional hosted Sandbox connections

## Status and boundary

Coordinator approved managed API composition, same verified API pool, explicit Link
events acknowledgement, canonical numeric secret versions and absolute local consent
expiry at most30days. Owned cmd/api/main.go, internal/connectionruntime and these
contract/runbook documents. No frontend, SQL, shared server, IAM or deployment edit.

## Stage ledger

Reviewed actual server.Dependencies/Managed, ledger.Service.DB, connection Service,
providersecrets reader/registry/writer and managed metadata interfaces before code.
Identified Infisical identity UUID audience mismatch; worker author corrected its
metadata adapter using reviewed official source. Numeric GCP project response names
are pinned explicitly; no project ID alias inference. Implemented complete opt-in
validation and cleanup, static credential reads, HTTPProvider and SecretStore/service
composition. Disabled startup is inert. Fixed errors suppress underlying diagnostics.

## Current evidence

Focused connectionruntime race tests and API command build PASS. Tests prove disabled
and invalid settings never touch dependencies; enabled startup reads exactly two
numeric versions with CRC32C and no DB transaction, Infisical identity login, writer
mutation or Plaid request. Failure tests cover canceled startup, wrong version,
transport diagnostics and invalid secret payload. Final default gates pending.
Dependencies providersecrets/workerruntime and clouddb changes were copied locally
for compilation and remain outside this commit; coordinator owns their integration.

## Operator configuration

Keep optional settings absent until the full hosted/secret/Link readiness review.
The [contract](../contracts/hosted-connections.md) lists exact identifier flags.
The local consent expiry is a fixed RFC3339 instant, never a rolling thirty-day
extension. This is an owner policy ceiling, not provider-reported consent lifetime.
Provider manifest applies the earlier actual expiry where present. Static credentials
must already exist at their exact numeric Google versions; no startup alias lookup,
Infisical fallback, rotation, Link creation or credential publication occurs.

Cold restart after that absolute expiry intentionally fails; obtain explicit new
local consent approval instead of a rolling extension. The registry bucket/prefix
and two static containers are fixed to the Terraform namespace in the contract;
foreign or swapped containers fail before I/O.

The API command shares its existing restricted pool and closes the injected direct
HTTP transport with normal server cleanup. Metadata supplies managed tokens. Runtime
writer permissions and the one-way Infisical propagation into the eight precreated
slots require separate deployed qualification. Existing API12s request budget stays;
uncertain work must use the existing durable reconciliation routes.

## Remaining work

Frontend real Link remains unavailable and Link-events acknowledgement is an operator
assertion, not proof. Verify actual Plaid enablement or implement/review Hosted Link,
then perform full Sandbox qualification. Hosted identity, static secret delivery,
Infisical token TTL, writer access and asynchronous propagation are not exercised by
these synthetic tests. DB readiness is not provider or propagation readiness.

## PR handoff

No migration or financial contract changes. Integrate after providersecrets and
workerruntime, including UUID audience support. Add canonical project number and the
explicit identifier flags through the coordinator's reviewed deployment workflow;
do not enable from defaults. Primary source selection is recorded in the contract.
