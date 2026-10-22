# Existing webhook updater bootstrap

Owner-only #271 preparation; `enabled=false` creates no resources, and separate
`federation_active=false` denies federation while retaining prepared resources.
Exact reviewed main workflow SHA required. No first service creation, public
invoker, SQL, secret or private-worker permissions are granted.

Owner must first create the private webhook service in its dedicated runtime
backend and separately approve public invocation. This root binds only existing
webhook update/read policy, read-only operation/role metadata, ActAs on its runtime
identity, one runtime state bucket and Artifact Registry repository reader.
Actual IAM/WIF compatibility remains an operator qualification gate.

Use separate owner-created `PROJECT-jayburd-dev-webhook-delivery-state` bucket,
prefix `webhook-delivery`. Do not use runtime/bootstrap/application state or turn
off protected resources to revoke tokens. Deactivate federation separately;
already-issued short-lived tokens expire independently.

Run `bash infra/gcp/webhook-delivery/test.sh` for an isolated mocked plan; it never
loads owner backend/variable/credential files or calls cloud APIs. See the
[updater contract](../../../docs/contracts/webhook-rollout.md). No apply is authorized.
