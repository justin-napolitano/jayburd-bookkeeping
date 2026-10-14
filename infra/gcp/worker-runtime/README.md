# Worker runtime deployment

This root owns only the two bounded acquisition Cloud Run services. It requires
owner bootstrap, separate worker-deployer federation and a dedicated backend:
`${project}-jayburd-dev-worker-runtime-state`, prefix `worker-runtime`.

Pass explicit project ID/number, region and a required runtime object with digest
image, enrolled binding UUID, exact numeric client-ID/secret version identifiers.
No secret values, bootstrap IAM or scheduler activation are accepted here.

See [worker release contract](../../../docs/contracts/worker-release.md) for the
mandatory first-deployment ordering and owner removed/import migration. Do not
point this root at workers-state or the application's runtime-state bucket.

Run `scripts/test-infra.sh` for isolated mocked validation; do not initialize a
real backend or use live Terraform state in agent/default CI work.
