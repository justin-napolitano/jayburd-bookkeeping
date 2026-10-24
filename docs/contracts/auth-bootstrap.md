# Public sign-in configuration availability

## Decision and applicability

#289, 2026-10-24. This presentation/composition slice keeps the existing
[browser authentication](browser-auth.md) authority and public configuration
allowlist intact. Bootstrap availability is separate from verified session state.
It is not a financial exchange or an ISO20022 message; financial/currency payloads
and standards mappings are unchanged. No API/SQL/schema, persistence or new identity
mechanism. Reviewed existing browser-auth and identity-setup contracts and their
canonical browser-auth-sources reference selections; no new external API claim.

## States and recovery

Initial mount, source change and explicit retry render loading, without login
inputs or the missing-installation warning. Runtime configuration fetch retains
no-store/omit credentials/no redirect and a five-second timeout. Network,
timeout, non-success HTTP and incomplete JSON transport failures show generic
temporary unavailability with one explicit retry control. No automatic retries,
raw diagnostics, secrets or session resurrection.
The pre-hydration server shell also renders loading; it cannot classify the
browser's runtime configuration before that request. The exact-image smoke
requires this shell without password inputs, while separately requiring absent
runtime configuration to return503/no-store and finance/demo routes to deny.

Successful HTTP/supplied configuration must pass the existing exact public-field
validation. Invalid configuration, unavailable SDK adapter or conflicting SDK
installation remains operator-only invalid configuration with no fallback source.
Only a valid initialized controller enters ready. Existing verified email/TOTP,
memory-only persistence, fresh token/revocation and API permission rules stand.

An exact configuration-source identity and attempt revision fence snapshots.
Source/retry changes immediately mask the previous controller before effect
cleanup; abort/unmount fences late fetch/JSON responses even if transport ignores
abort. Cleanup signs out any initialized old controller, invalidating pending
SDK results through existing epochs. Retry is offered only for unavailable
bootstrap, never to reset an active verified session. No browser persistence.

## Alternatives and evidence

Keeping unconfigured for every failure conflates installation with loading and
offers no recovery. Automatic retries add repeated requests and hide the user
action; choose bounded fetch plus explicit retry. Replacing controller auth phases
would mix availability with verified identity and expand the trust boundary;
choose separate bootstrap state and preserve controller/Firebase code.

Acceptance: delayed valid config never displays invalid installation; HTTP503,
network and timeout retry restores inputs; invalid/supplied-conflicting settings
stay denied; unmounted/stale requests cannot initialize SDK or resurrect a session;
source changes sign out and mask the prior controller. Component and browser
fixtures remain credential-free. Required core/web/full-stack gates and a marked
runbook; actual managed-browser sign-in evidence remains separate. Sensitive
authentication composition needs review before release, without relaxing identity.
