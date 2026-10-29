# Production static credential delivery — #365

Candidate extension of the existing owned scoped-delivery adapter. The account's
custom roles are gated by its plan; mixed-project Viewer access would expose
unrelated secrets. Use the empty dedicated Infisical project
jayburd-production-credentials, Production/prod root, containing only
PLAID_CLIENT_ID and PLAID_PRODUCTION_SECRET. Owner copies existing values inside
Infisical; agents never reveal, copy or upload them. No native sync, upgrade,
shared-project Viewer grant or source reference expansion is approved.

Explicit Production constructor/command profile maps those two keys only to
jayburd-prod-bank-sync-client-id/client-secret. Sandbox/canary/slots retain their
existing behavior and cannot select Production destinations. Metadata lookup
requests viewSecretValue=false, exact positive versions and disabled imports/
references. Execution additionally binds exact source IDs. Existing immutable
intent/receipt fence precedes the sole addVersion request and retains ambiguity
as pending; no automatic retries/cleanup or destination payload reads.

Production command requires a canonical future UTC deadline no more than7days
away, checked before metadata authentication and used as a context deadline.
Registry uses the dedicated Production credential bucket and a separate
production-static-delivery-v1 prefix with create/get permission only. The
keyless GCP service identity is separate from API/runtime identities; Infisical
Viewer must be confined to this otherwise-empty project. Identity/access grants
require action-time browser confirmation. Job defaults to absent/lookup, one
task, no retries, no scheduler. No live transfer follows source-only success.

Source basis: scoped-delivery.md and delivery-job.md, selected Infisical exact
version/value-disabled API and GCP ID-token auth references. Actual UI inspection
2026-10-29 confirms plan gate and same-project copy dialog; no secret values read.
Isolation replaces unavailable key-level roles; document any future extra source
keys/environments before adding them. ISO20022 is inapplicable to internal secret
transport; accounting/financial schemas remain unchanged. Elevation/disclosure
risks: isolated source, fixed mappings, exact IDs/versions, keyless auth and no
payload diagnostics. Availability risk: stranded delivery intent, reviewed recovery.

Selected UI source review2026-10-29: Infisical
[CopySecretsSheet at f943528](https://github.com/Infisical/infisical/blob/f943528f8f51d469fa6ef4ccb62d84d14b291cb8/frontend/src/pages/secret-manager/OverviewPage/components/CopySecretsSheet/CopySecretsSheet.tsx),
projectId, source/destination environment and duplicateSecret call sites;
[SecretEditTableRow](https://github.com/Infisical/infisical/blob/f943528f8f51d469fa6ef4ccb62d84d14b291cb8/frontend/src/pages/secret-manager/OverviewPage/components/SecretTableRow/SecretEditTableRow.tsx),
Copy Secret dropdown versus clipboard-value control. Selected functions only,
not a full frontend/deployed-version review. UI inspection independently confirms
the copy dialog offers environment/path destinations and custom-role creation
opens the subscription gate. Our inference: isolated-project owner injection is
needed with the available controls; no broader mixed-project Viewer grant.

Acceptance: synthetic Production mapping/lookup/replay/unknown-ack/deadline and
cross-profile denials; unchanged legacy controls; core/image/mock gates; actual
keyless access isolation and numeric receipt/freshness qualification. Payload
verification is performed by approved runtime or owner without agent disclosure.
