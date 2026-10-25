# Owner-private Sandbox identity transport — #299

## Decision and scope

Engineering profile reviewed by the coordinator2026-10-24 before implementation;
human security/owner acceptance is required before actual secret/provider use.
Resolve the missing private producer for the existing tagged identity projector,
not acquisition or financial comparison. One freshly controlled synthetic checking
account, complete posted-only fixture authored before provider data, at most500
rows. The existing projector and financial comparison remain authoritative.

A separate local `sandbox-identity-readback` command defaults offline. Explicit
`-action=readback` requires the absolute independently authored fixture.json,
its exact SHA256, exact synthetic account ID, an absolute expiry no more than60
minutes ahead, and a separate owner-private pipe/socket descriptor3..1024.
The owner supplies exactly client_id, secret and access_token JSON through that
descriptor; token must have the Sandbox prefix. No credentials in flags, env,
files, agent messages or browser storage. No secret/ADC/SQL/vault discovery.
The owner checks Item/account/source consent before injection; this local deadline
does not attest SQL ownership, renew consent or grant access to any existing Item.

Before opening the private descriptor, validate fixture bytes/hash/posted-only
identity profile, account and finite deadline. The total operation has15seconds,
capped by the supplied expiry, including pipe read and response. Credential input
is bounded16KiB, closes on all paths and never appears in diagnostics. One HTTPS
POST to fixed https://sandbox.plaid.com/transactions/sync, API2020-09-14, no proxy,
redirect, retry, refresh, paid operation or cursor. Request count500 with exact
options.account_id and omitted cursor. Expiry is rechecked before request; TLS
verification remains standard. No arbitrary endpoint or Production profile.

Response headers are bounded16KiB with at most one connection. Read at most the
existing2MiB combined fixture/response body bound. In-memory response
goes directly to the existing projector; it must be complete historical readiness,
has_more=false, no modified/removed, exact account, exact preauthored unique tags
and unique IDs. Do not parse financial values to select identities or alter
expectations. Neither raw response nor credentials are written or printed.
Only fixed sibling identity-map.json is written create-only0600 after full
validation. Invalid input/transport leaves no map; an existing map/symlink is never overwritten.
Success emits only readiness Boolean and bounded count. If final stdout delivery
fails after the complete map is committed, preserve that map and treat the
acknowledgement as unknown; do not automatically repeat the provider request.
Inspect the owner's private result before any deliberate next operation.
It is not an equality,
acquisition, current SQL identity or zero-journal receipt.

## Evidence, alternatives and limits

[Reviewed source record](../research/sandbox-identity-transport-sources.md) separates
vendor semantics from this narrow policy. Existing `sandbox-identity-map` remains
an offline alternative when an owner already has an authorized private response
pipe. A hosted bridge was rejected for this slice because it adds deployed
provider/credential/SQL authority before the first controlled proof. General curl,
saved payloads or SQL-derived expectations lack this closed output and independent
oracle boundary. No owner is asked to send a credential or financial record to
the agent. The source helper's availability does not establish how an owner may
retrieve a particular stored Item credential; absent private authorized injection,
leave the actual run blocked rather than export/decrypt through an admin query.

Named acceptance: default/invalid/expired command does zero dependency work;
fixture/hash/tag validation before credential open; exact fixed request/version/
account/omitted cursor with one call; financial-field mutations produce identical
map and unchanged authored bytes; foreign/missing/duplicate/incomplete input fails;
oversize/HTTP/error/redirect/cancel/expiry/pipe failure produces fixed diagnostic
and no map; actual synthetic pipe/file rejection/create-only0600 and cleanup.
Core/race/build plus existing projector/comparator gates remain required.

## Standards and security

Proprietary Plaid API2020-09-14 identity association, not an ISO20022 financial
exchange or ISO21378 audit export. The existing ISO4217 currency/version mapping
and original money lexemes remain unchanged in the independent comparison; no
normalization, FX, accounting policy or conformance claim is introduced.

Threats: supplied wrong Item is constrained by account-filter and exact authored
tags but this does not attest application consent; owner finite scope is required.
Secret/response disclosure is constrained by private bounded input, fixed errors,
no filesystem/raw output and TLS/fixed origin. Stalled input/output is bounded;
owner controls ancestor directory, process and private descriptor. A compromised
owner process or host is outside this helper's guarantees. Expiry stops this
invocation only, not Plaid credentials or existing services. Cleanup closes pipe
and response and removes only a partial newly created map; owner separately
revokes the actual #300 mandate and deletes generation-pinned hosted inputs.

Actual human-reviewed transport/credential injection and #300 comparison/cleanup
remain unproved. Old Sandbox approvals remain expired. No personal-account use.

The shared private descriptor opener validates pipe/socket type, sets nonblocking
mode before os.NewFile and requires read-deadline support. Closing an inherited
blocking descriptor was experimentally insufficient to stop syscall.read on the
local platform. Both producer and existing offline projector use the corrected
opener; invalid descriptors never become ordinary file/standard-stream reads.

After actual human transport/scope acceptance, an owner-side launcher may provide
the private pipe on descriptor3 to this invocation template (placeholders are not
approved values):

```text
bin/sandbox-identity-readback -action=readback
  -fixture=/owner/private/run/fixture.json -fixture-sha256=<reviewed independent hash>
  -account-id=<exact synthetic account> -expires-at=<fresh approved absolute deadline>
  -credential-fd=3
```

The launcher must not echo credentials, use shell tracing, save a credential JSON
file or send values through agent tools. The three envelope fields are exactly
client_id, secret and access_token; only the owner injects their values. This
template supplies no token retrieval/decryption permission. Review private map
readiness/count only, hash the map owner-side for the subsequent finite #300
mandate, and keep all financial comparison/cleanup requirements unchanged.
