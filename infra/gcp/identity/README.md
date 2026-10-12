# Development identity configuration

## Decision

Owner-authorized #6 setup uses a separate owner-operated root. It enables
Identity Platform and configures invited email/password accounts with TOTP.
No users, passwords, browser keys, SQL principals or application deployments are
created by this root. It owns `identitytoolkit.googleapis.com`; the foundation's
previously unapplied declaration is removed to avoid two state owners.

Reviewed 2026-10-12: [I03/I06 sources](../../../docs/research/identity-sources.md).
We choose Terraform over console-only setup for repeatable configuration and
mocked assertions. Provider MFA is ENABLED so a first-factor session can enroll;
[the API](../../../docs/contracts/identity-setup.md) still requires completed TOTP.
One adjacent time interval is our clock-skew policy, not a provider recommendation.
Account deletion stays operator-managed so SQL disabling and provider revocation
can be coordinated. Authorized domains are redirect configuration, not a substitute
for API authorization. ISO financial-message standards do not apply to this
provider configuration; no accounting contracts or SQL migrations change.

## State and execution

Use the existing owner-only foundation bucket with prefix `identity`, never the
runtime/application buckets. Provider computed fields can contain password-hash
signing material; **all state, plan and provider logs remain private**, even if
Terraform does not mark every field sensitive. No outputs or CI apply job exist.
Do not grant the builder, deployer or runtime access to identity state.

Owner/operator installs `backend.tf.example` as ignored `backend.tf`, then runs:

```sh
terraform init -input=false -lockfile=readonly \
  -backend-config='bucket=jayburd-bookkeeping-dev-jayburd-dev-foundation-state' \
  -backend-config='prefix=identity'
terraform plan -input=false \
  -var='project_id=jayburd-bookkeeping-dev' -var='project_number=9088794002'
```

Plans are reviewed only in the owner's private terminal. With explicit owner
setup authorization, an agent may execute this bounded root with stdout/stderr
redirected to ignored mode-600 logs and report exit status only. Never use
`terraform show`, `output`, state downloads or credential-print commands.
Use existing ADC internally; for user ADC set `USER_PROJECT_OVERRIDE=true` and
`GOOGLE_BILLING_PROJECT=jayburd-bookkeeping-dev`. Do not inspect credentials.
After authorized apply, a private `plan -detailed-exitcode` returning zero is
configuration convergence evidence, not a successful user login.

If initialization already exists, stop and have the owner review/import
`projects/jayburd-bookkeeping-dev/config`; do not replace configuration blindly.
If another environment applied the old foundation API declaration, transfer its
state ownership privately before applying this change there. Never destroy the
identity configuration to recover a failed apply.

## Acceptance and follow-up

`make infra-test` uses only synthetic mock providers; `invited_totp_identity`
and the invalid-project tests cover configuration. The living
[identity runbook](../../../docs/runbooks/identity-setup.md) records actual apply
and remaining owner enrollment evidence. #6 remains open until managed sign-in,
revocation, SQL principal enrollment and entity boundaries pass live development
review. Recovery and stronger password/abuse policy need review before wider use.
