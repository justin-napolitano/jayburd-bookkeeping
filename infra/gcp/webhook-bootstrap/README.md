# Dedicated webhook bootstrap

Owner-operated #271 preparation only. Default `enabled=false` creates nothing.
With explicit review, owns dedicated webhook SA/SQL IAM user/client+login grants
and two existing static secret readers. Separate `public_invocation_approved=false`
requires service creation and human/security approval before public invocation.
No private-service IAM, source/provider call, credential value or SQL grant occurs.

Use a separate owner-created backend bucket
`PROJECT-jayburd-dev-webhook-bootstrap-state`, prefix `webhook-bootstrap`.
Do not initialize against existing workers/runtime state. Do not toggle enabled
false after creation as teardown: prevent_destroy preserves identity/SQL evidence.

See [release contract](../../../docs/contracts/webhook-release.md) and run isolated
`bash infra/gcp/webhook-runtime/test.sh` from repository root. That command copies
allowlisted source into temporary directories, uses mocked providers and never
loads owner backend/configuration/variable/credential files. No apply is authorized.
