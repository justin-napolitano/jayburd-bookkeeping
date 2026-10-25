<!-- stage-runbook: true -->
# Closed provider transports — #312

## Status and boundary

PROFILE312-TRANSPORT through2026-10-25T03:59Z, dedicated
feat/312-closed-plaid-profile from accepted317/mainb64f3150. Source-only component;
no SQL/DDL/runtime/API/CLI/IAM/release/credential origin change or activation.
Default constructors remain Sandbox; new explicit constructors require reviewed
closed profiles.337 deployment/pilot decision and311/313/314 authority remain.

## Stage ledger

| Stage | Evidence / limit |
| --- | --- |
| Resume |65/312/308 and actual317 merge checked; five canonical checks passed and revised308 source baseline closed; no missing predecessor counted complete |
| Source review | Accepted plaid-provider-profile contract, PP01/04/06, standards/decision/database foundations and current SDK/acquisition/runner/tests reviewed |
| Implementation | Closed private-state profile and exact HTTPS/POST/path guards; explicit enrollment/sync constructors, unchanged Sandbox wrappers and trusted SQL fetch environment propagation complete |
| Focused race | New explicit Sandbox/Production SDK wire/product/lifecycle cases, zero/malformed/mixed profile denials, original USD/EUR/JPY/KWD numeric lexemes and retry/pagination/redirect/redaction/body bounds PASS; actual current SQL context still rejects Production |
| Test input correction | Initial retry/redirect cases accidentally reused the pagination-mutation error body; separated synthetic error inputs while retaining the original expected classifications. No product classification rule or deadline changed |
| Access boundary | First ordinary acquisition test was denied a local loopback listener by sandbox; approved retry PASS. No default/provider network or credential discovery introduced |
| Core validation | Focused package race tests and make check test build PASS; local cached Go 1.26.5 with external modfile, tracked toolchain unchanged |
| Integration validation | make acquisition-test connection-service-test provider-flow-test connection-ops-test PASS against disposable synthetic PostgreSQL and injected transports |

## Current evidence

Source profile routing does not prove opaque credential origin, consent or current
SQL authority. Acquisition SQL validContext remains Sandbox-only until311.313
owns AAD/current credential binding and actual managed composition. Production
receiver/repair are separately gated; this source constructor cannot route to
the Sandbox receiver or enable HostedRepair. No actual provider is called by tests.

## Remaining work

Publish source PR and run canonical CI. Source code requires human security
review before merge; source330/331 review still
pending, no generic owner task instruction is recorded as that review.

## PR handoff

Refs312/308/90/311/313/314/315/337. No migration/OpenAPI/UI/runtime switch. Exact
source/runbook lease and staged check every implementation commit. Original and
functional currency semantics unchanged, no ISO conformance or live capability
claim. Registration remains the delivery priority, not complete from source CI.
