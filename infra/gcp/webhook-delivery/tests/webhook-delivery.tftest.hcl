mock_provider "google" {
  override_during = plan
  mock_resource "google_project_iam_custom_role" {
    defaults = { name = "projects/jayburd-synthetic-test/roles/synthetic" }
  }
}
variables {
  project_id     = "jayburd-synthetic-test"
  project_number = "123456789012"
}
run "default_no_federation_or_grants" {
  command = plan
  assert {
    condition     = length(google_service_account.deployer) == 0 && length(google_iam_workload_identity_pool.deploy) == 0 && length(google_cloud_run_v2_service_iam_member.update) == 0 && length(google_service_account_iam_member.act_as) == 0 && length(google_storage_bucket_iam_member.state) == 0 && length(google_artifact_registry_repository_iam_member.images) == 0 && length(google_project_iam_member.metadata) == 0
    error_message = "Default delivery must create no identities, federation or runtime capability."
  }
}
run "exact_workflow_and_minimal_existing_service_update" {
  command = plan
  variables {
    enabled           = true
    workflow_sha      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    federation_active = true
  }
  assert {
    condition     = google_service_account.deployer[0].account_id == "jayburd-dev-webhook-deployer" && toset(google_project_iam_custom_role.update[0].permissions) == toset(["run.services.get", "run.services.getIamPolicy", "run.services.update"]) && google_cloud_run_v2_service_iam_member.update[0].name == "jayburd-dev-webhook" && google_cloud_run_v2_service_iam_member.update[0].role == google_project_iam_custom_role.update[0].name
    error_message = "Updater is fixed to one existing host with no create/delete/IAM administration."
  }
  assert {
    condition     = toset(google_project_iam_custom_role.metadata[0].permissions) == toset(["run.operations.get", "iam.roles.get"]) && google_service_account_iam_member.act_as[0].service_account_id == "projects/jayburd-synthetic-test/serviceAccounts/jayburd-dev-webhook@jayburd-synthetic-test.iam.gserviceaccount.com" && google_service_account_iam_member.act_as[0].role == "roles/iam.serviceAccountUser"
    error_message = "Only read-only operation/role metadata and the webhook runtime identity may be used."
  }
  assert {
    condition     = google_storage_bucket_iam_member.state[0].bucket == "jayburd-synthetic-test-jayburd-dev-webhook-runtime-state" && google_storage_bucket_iam_member.state[0].role == "roles/storage.objectAdmin" && google_artifact_registry_repository_iam_member.images[0].repository == "jayburd-dev" && google_artifact_registry_repository_iam_member.images[0].role == "roles/artifactregistry.reader"
    error_message = "State is isolated to runtime bucket; image repository read never grants image/secret writes."
  }
  assert {
    condition     = alltrue([for claim, value in { repository_id = "1414515318", repository_owner_id = "55423093", actor_id = "55423093", ref = "refs/heads/main", event_name = "workflow_dispatch", workflow_ref = "justin-napolitano/jayburd-bookkeeping/.github/workflows/deploy-webhook.yml@refs/heads/main", workflow_sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } : strcontains(google_iam_workload_identity_pool_provider.deploy[0].attribute_condition, "assertion.${claim} == '${value}'")]) && google_iam_workload_identity_pool_provider.deploy[0].oidc[0].issuer_uri == "https://token.actions.githubusercontent.com"
    error_message = "Federation must bind immutable owner/repository, dispatch/main, exact workflow and reviewed SHA."
  }
}
run "deactivate_preserves_grants_but_disables_federation" {
  command = plan
  variables {
    enabled           = true
    workflow_sha      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    federation_active = false
  }
  assert {
    condition     = google_iam_workload_identity_pool.deploy[0].disabled && google_iam_workload_identity_pool_provider.deploy[0].disabled && length(google_service_account.deployer) == 1 && length(google_cloud_run_v2_service_iam_member.update) == 1
    error_message = "Owner can disable future federation while preserving protected identity/grants; existing tokens still expire separately."
  }
}
run "reject_active_without_resources" {
  command = plan
  variables { federation_active = true }
  expect_failures = [var.federation_active]
}
run "reject_enabled_without_exact_sha" {
  command = plan
  variables { enabled = true }
  expect_failures = [var.workflow_sha]
}
run "reject_mutable_sha" {
  command = plan
  variables {
    enabled      = true
    workflow_sha = "main"
  }
  expect_failures = [var.workflow_sha]
}
run "reject_unreviewed_region" {
  command = plan
  variables { region = "europe-west1" }
  expect_failures = [var.region]
}
