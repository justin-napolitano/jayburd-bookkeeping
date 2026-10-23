# Webhook updater sources — 2026-10-22

Selected official sections reviewed before the #271 updater design. Source review
establishes mechanisms, not cloud authorization or hosted qualification. No
credential, provider payload, personal financial data, Terraform state or plan was
read. This extends the [release source register](webhook-release-sources.md).

| ID / source | Reviewed section and fact | Interpretation / evidence |
| --- | --- | --- |
| WU01 [Google federation for deployment pipelines](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines) | GitHub claims, attribute mapping/conditions and service-account impersonation sections. Numeric repository/owner IDs identify the intended source. | Separate owner-created pool/provider binds repository, owner, actor, dispatch/main and workflow path. Mocked delivery tests inspect every claim; no federation was activated. |
| WU02 [GitHub OIDC reference](https://docs.github.com/en/actions/reference/security/oidc) | Token claim table and trust-condition sections, including workflow_ref and workflow_sha. | Exact reviewed main SHA pin is a separate owner configuration. Updating main does not silently update federation. It pins workflow source, not the arbitrary dispatch input; exact manifest hash is independently validated. |
| WU03 [Cloud Run IAM roles](https://docs.cloud.google.com/run/docs/reference/iam/roles) | Resource-level role applicability, get/update/getIamPolicy and operation permission lists, and IaC image-read requirement. Relevant sections read. | Service-scoped custom role is only get/update/getIamPolicy. Project read-only operation/role metadata, one runtime ActAs, one runtime-state bucket and read-only artifact repository are separate grants. No RunAdmin fallback. Repository reader covers repository images, not one image path. |
| WU04 [Google8.6.0 service implementation](https://raw.githubusercontent.com/hashicorp/terraform-provider-google/v8.6.0/google/services/cloudrunv2/resource_cloud_run_v2_service.go) | resourceCloudRunV2ServiceUpdate request, operation wait and final read; Read request and fixed ForceNew identity fields. Selected routines read. | Provider uses service PATCH, operation polling and final GET. No create/delete permission is granted to the updater. Actual IAM compatibility still needs operator proof; mocks cannot establish it. |
| WU05 [Cloud Run PATCH](https://docs.cloud.google.com/run/docs/reference/rest/v2/projects.locations.services/patch) | HTTP method, allowMissing and operation response sections. | Missing service creation requires create permission. Existing-service preflight plus no-create IAM fails closed; an owner must first create/adopt the fixed runtime in its dedicated backend. |
| WU06 [Service GET](https://docs.cloud.google.com/run/docs/reference/rest/v2/projects.locations.services/get), [IAM GET](https://docs.cloud.google.com/run/docs/reference/rest/v2/projects.locations.services/getIamPolicy), [operation GET](https://docs.cloud.google.com/run/docs/reference/rest/v2/projects.locations.operations/get) | Fixed v2 GET paths, response types, IAM policy absence of inherited bindings and operation polling descriptions. | Collector uses fixed destinations and bounded server-side field selection; raw responses stay in memory. Only validated receipt is written. Direct service policy does not prove absence of inherited project/organization grants. |
| WU07 [IAM roles GET](https://docs.cloud.google.com/iam/docs/reference/rest/v1/projects.roles/get) | Custom-role resource response: name, includedPermissions, stage, deleted. | Optional updater binding needs actual matching GA/nondeleted role with exact three permissions. A matching role name alone is insufficient. |

Relevant public sections were read, not whole vendor manuals. Google/GitHub pages
are mutable; the Terraform source is tagged8.6.0. Existing repository source
records cover Terraform1.15.7 mocked plans and exact trusted build evidence. No ISO
text or personalized Launch Center was reviewed. This operational metadata adds
no financial message, amount, currency, accounting policy or ISO designation.
