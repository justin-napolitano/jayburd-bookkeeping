<!-- stage-runbook: true -->
# Production static delivery — #365

## Status and boundary

PROD365-DELIVERY; build/316-production-delivery from28c36b7. Owner signed into
Infisical and authorized continued wiring. Source/UI metadata preparation only;
no actual secret values, authentication keys or provider payloads read/copied.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Browser metadata | Existing PLAID_CLIENT_ID/PLAID_PRODUCTION_SECRET names visible masked in main project's Development; Production empty |
| Plan capability | Creating a custom role triggers subscription gate; no role/access grant created |
| Isolation | Created empty jayburd-production-credentials project, Production selected; owner-only existing-value copy requested |
| Source | Explicit Production constructor/CLI, two fixed static keys, prod source, separate registry prefix and finite deadline |
| Synthetic | Focused adapter/CLI controls and four Terraform mock cases PASS |
| Core gates | make check/test/build and focused adapter/CLI tests PASS |
| Worker image | Pinned Go1.26.10 offline build and runtime suite PASS, including Production config success/expiry denial; hosted scan still pending |

## Current evidence

Repository scoped-delivery.md rejects native sync because its destination reads
exceed approved permissions; no native sync was enabled. Same-project UI copy
supports environment/path selection, not another project. Owner must copy only
the two existing entries inside Infisical to the isolated project's prod root.
Custom roles remain unavailable; no upgrade, mixed-project Viewer access,
credential reveal, clipboard operation or secret write was performed by agent.
Project isolation bounds the read-only machine identity; no additional secrets
or environments should be populated without another scope review.

The new job uses a dedicated identity rather than the foundation secret-sync or
API accounts. Default null creates no job/registry/destination grant. Lookup has
no destination write; execute grants versions.add only on the selected static
destination. No scheduler, SQL, invocation or impersonation grants are introduced.
Source metadata errors, deadline/profile rejection and partial outcomes remain
fixed/redacted. Source/receipt transports remain synthetic; no default provider
call or credential discovery. Local Go1.26.5/external temporary modfile; tracked
Go1.26.10 unchanged. No migration or accounting/ISO message change.

Normal Docker dependency download failed local certificate validation. TLS was
not weakened. An ephemeral build context supplied only go.mod/go.sum-listed public
module archives/metadata; GOPROXY=off and network=none retained Go checksum
validation. Source/runtime image and pinned compiler are unchanged; hosted
canonical build and exact published vulnerability scan remain separate gates.

## Remaining work

Owner copies the existing values within Infisical only; request pending. Human
review of new source, scanned exact Production worker build and reviewed job
inputs. Configure exact GCP ID-token identity with TTL/max TTL900 and Viewer only
on isolated project, with action-time owner confirmation before access grants.
Actual metadata-only lookup, one-shot delivery, unknown-outcome recovery,
independent runtime verification, numeric pins/freshness and revocation proof
remain open. Never retrieve values to complete setup or reset a stranded intent.

## PR handoff

Refs #365/#316/#313/#11. No live application or bank activation. Additional
worker grants/hosted identity/recovery gates remain separate from static delivery.
