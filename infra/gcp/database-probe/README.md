# API database permission proof

This independent Terraform root creates only a one-shot development Job using
the reviewed API image's `/database-probe` binary and the existing API identity.
It grants no permissions and does not run automatically. See the
[operator procedure and acceptance](../../../docs/runbooks/database-permission-probe.md).

Use a separate GCS backend prefix `database-probe`; never reuse application or
migration state. Supply the approved existing backend bucket privately through
the normal operator backend configuration. State/plans stay outside review output.

`terraform init -backend=false -lockfile=readonly` and `terraform test` use a mock
provider and synthetic digest. They do not create cloud resources.
