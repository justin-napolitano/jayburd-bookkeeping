# Production static delivery

Source candidate365. Foundation must exist first. This root uses a separate
private backend prefix `production-static-delivery`; no owner backend/variables
are committed. Do not inspect/copy Terraform state or secret values.

Default null delivery creates only the dedicated keyless identity, without job,
registry access or version-adder grant. Configure Infisical GCP ID Token Auth
with its exact email, TTL/max TTL900s, and Viewer only on the otherwise-empty
jayburd-production-credentials project. No Universal Auth credential or broad
mixed-project access. Browser access grants require action-time owner confirmation.

After reviewed merge, exact Production worker build/runtime scan and identity
qualification, supply a pinned image, isolated source project/identity, one fixed
key, exact source version and a future UTC deadline. Lookup returns metadata
only and cannot add destination versions. Runtime enforces canonical expiry
within7days even if a syntactically valid future fixture passes Terraform mocks.
Execute additionally pins the observed source ID and grants only versions.add on
that selected destination. No scheduler, invoke grant, overrides or implicit run.

Keep zero retries. Unknown acknowledgement stays pending; never reset an intent
or rerun under a different source version to bypass recovery. Restore lookup mode
after acknowledged execution and independently qualify runtime/version integrity.
Source-only tests do not establish hosted permissions, freshness or bank readiness.
