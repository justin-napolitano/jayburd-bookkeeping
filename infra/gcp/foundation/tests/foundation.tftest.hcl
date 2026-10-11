# Only a mock provider may run in default CI; no GCP API or credentials are used.
mock_provider "google" {}
variables {
  project_id         = "jayburd-synthetic-test"
  project_number     = "123456789012"
  billing_account_id = "AAAAAA-BBBBBB-CCCCCC"
}
run "secure_foundation_defaults" {
  command = plan
  assert {
    condition     = google_iam_workload_identity_pool.github.disabled && google_iam_workload_identity_pool_provider.build.disabled
    error_message = "Federation must be disabled until an owner reviews promotion."
  }
  assert {
    condition     = alltrue([for b in google_storage_bucket.state : b.uniform_bucket_level_access && b.public_access_prevention == "enforced" && !b.force_destroy && b.versioning[0].enabled])
    error_message = "Both state buckets must be private, versioned and non-destructive."
  }
  assert {
    condition     = google_storage_bucket.state["foundation"].name != google_storage_bucket.state["application"].name && google_storage_bucket_iam_member.application_state.bucket == google_storage_bucket.state["application"].name
    error_message = "Deployer access must be limited to the application state bucket."
  }
  assert {
    condition     = google_artifact_registry_repository.images.docker_config[0].immutable_tags && google_artifact_registry_repository_iam_member.builder.role == "roles/artifactregistry.writer"
    error_message = "Builder needs scoped immutable image publication, not project administration."
  }
  assert {
    condition     = length(google_service_account.component) == 9 && google_billing_budget.development.amount[0].specified_amount[0].units == "50" && google_billing_budget.development.budget_filter[0].projects == toset(["projects/123456789012"])
    error_message = "Separate all component identities and scope the $50 budget to this project."
  }
  assert {
    condition     = google_iam_workload_identity_pool_provider.build.attribute_condition == "assertion.actor_id == '55423093' && assertion.event_name == 'workflow_dispatch' && assertion.ref == 'refs/heads/main' && assertion.repository_id == '1414515318' && assertion.repository_owner_id == '55423093' && assertion.workflow_ref == 'justin-napolitano/jayburd-bookkeeping/.github/workflows/build-images.yml@refs/heads/main'"
    error_message = "Federation must restrict actor, event, ref, immutable repository/owner IDs and exact workflow."
  }
}
run "owner_can_enable_only_build_federation" {
  command = plan
  variables { enable_github_build_identity = true }
  assert {
    condition     = !google_iam_workload_identity_pool.github.disabled && !google_iam_workload_identity_pool_provider.build.disabled && google_service_account_iam_member.github_builder.role == "roles/iam.workloadIdentityUser"
    error_message = "Explicit enablement changes availability, not the identity trust boundary."
  }
}
run "reject_missing_project" {
  command = plan
  variables { project_id = "" }
  expect_failures = [var.project_id]
}
run "reject_invalid_project_number" {
  command = plan
  variables { project_number = "not-a-number" }
  expect_failures = [var.project_number]
}
run "reject_invalid_billing" {
  command = plan
  variables { billing_account_id = "" }
  expect_failures = [var.billing_account_id]
}
run "reject_unreviewed_region" {
  command = plan
  variables { region = "europe-west1" }
  expect_failures = [var.region]
}
