# Allowlisted secret delivery

## Decision and scope

Selected for implementation by the coordinator under owner direction, 2026-10-14;
hosted activation remains gated by #11. Replace the assumed native Infisical sync
with an owned delivery adapter. Infisical remains source; GCP remains the runtime
reader. No new database, API route or continuously running service is introduced.
This change implements a callable adapter, not a deployed job.

Source mapping is fixed: root `PLAID_CLIENT_ID` and `PLAID_SECRET` map to
`jayburd-dev-bank-sync-client-id` and `jayburd-dev-bank-sync-client-secret`.
Eight `/bank-sync/slot-00` through `slot-07` keys map to corresponding destinations.
A separate root `JAYBURD_DELIVERY_CANARY` maps only to the precreated canary
destination. The [controlled job](delivery-job.md) initially accepts only this key.
Original root values stay in place; references and whole-folder imports are disabled.
Every invocation selects an exact positive source version and expected source ID.
The configured project/environment and returned identity/version must match.

An immutable GCS intent keyed by destination and source version precedes the sole
Google addVersion attempt. Concurrent calls, conflicting source identity and unknown
intent acknowledgement cannot authorize another write. A lost Google or receipt
acknowledgement returns PARTIAL/unknown: a later call returns a matching immutable
receipt if present, otherwise stops for reviewed recovery. No automatic rotation,
retry or cleanup is implied. Dynamic slots remain never-reused; static rotation
requires another explicit source version. Delivery receipts are transport evidence,
not runtime verification, freshness evidence or permission to use revoked versions.

The adapter has exact-key source read permission, destination versions.add, and
create/read access to its own private registry prefix. It has no destination value
read, project secret listing, source write, container create or delete capability.
Separate runtime verification retains CRC32C/payload matching and numeric pinning.
The new worker identity, registry prefix IAM, owner-injected synthetic canary and
controlled job composition must be reviewed before any hosted execution.

## Sources and alternatives

Reviewed relevant implementation and REST sections on 2026-10-14:

- [Infisical GCP implementation](https://github.com/Infisical/infisical/blob/bd18bfdc374008c79581812529af7672fce71cc2/backend/src/services/secret-sync/gcp/gcp-sync-fns.ts):
  `getGcpSecrets` lists project secrets and reads latest payloads before the sync
  loop filters keys. This establishes upstream behavior, not the hosted version.
- [Google addVersion](https://cloud.google.com/secret-manager/docs/reference/rest/v1/projects.secrets/addVersion):
  accepts an existing parent and requires versions.add; no idempotency key is
  documented. Our inference is to reserve before attempting and stop on ambiguity.
- [Infisical retrieve](https://infisical.com/docs/api-reference/endpoints/secrets/read):
  exact version, folder, project, environment and disabled expansion/imports allow
  explicit reads. Account permissions still require actual qualification.

Alternative: isolate native sync in a separate GCP project and authorize read-back.
That adds infrastructure and changes runtime project configuration while retaining
broader access inside that project. Direct runtime Infisical reads would change
startup availability. Neither is selected. The owned adapter adds maintenance and
can strand an intent after a crash; recovery is deliberately manual until reviewed.

## Acceptance and applicability

`TestDeliveryCanary` proves fictional root-key delivery, exact source version,
numeric destination receipt and replay without a second write. Failure tests cover
concurrency, lost acknowledgements, mismatched source/receipt, invalid targets,
denied requests and redacted diagnostics. All transports are injected fakes.
Hosted isolation, revocation, rotation and recovery remain required #11 evidence.
This is an internal credential protocol; ISO 20022 does not apply. No accounting
or financial-message contract changes. See the living delivery runbook.
