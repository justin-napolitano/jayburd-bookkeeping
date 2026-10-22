# Dedicated webhook runtime

Owner-operated #271 preparation only. Default runtime null creates nothing.
Explicit object is `{image,client_id_version,client_secret_version}`: exact trusted
worker image digest and numeric versions. Project ID/number required; us-central1
only. Fixed service `jayburd-dev-webhook`, command `/webhook`, dedicated identity,
TCP startup and bounded request/compute/scaling policy. No IAM/SQL/bootstrap or
provider mutation is performed here.

Use owner-created backend `PROJECT-jayburd-dev-webhook-runtime-state`, prefix
`webhook-runtime`, distinct from bootstrap and existing worker/application state.
Existing deploy-workers is incompatible and must not be repurposed. Root coordinates
new exact release/readback tooling. Do not null a deployed runtime as rollback;
use reviewed immutable-image rollback while preserving evidence.

See [release contract](../../../docs/contracts/webhook-release.md). Run only
`bash infra/gcp/webhook-runtime/test.sh` for isolated mocked checks; it never reads
owner backend/variable/credential files. No real Terraform apply is authorized.
