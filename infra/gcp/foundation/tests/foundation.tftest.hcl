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
    error_message = "All state buckets must be private, versioned and non-destructive."
  }
  assert {
    condition     = google_storage_bucket.state["foundation"].name != google_storage_bucket.state["application"].name && google_storage_bucket.state["workers"].name != google_storage_bucket_iam_member.application_state.bucket && length(google_storage_bucket.state) == 5 && google_storage_bucket_iam_member.application_state.bucket == google_storage_bucket.state["runtime"].name
    error_message = "Deployer access must be limited to the runtime state bucket."
  }
  assert {
    condition     = google_artifact_registry_repository.images.docker_config[0].immutable_tags && google_artifact_registry_repository_iam_member.builder.role == "roles/artifactregistry.writer"
    error_message = "Builder needs scoped immutable image publication, not project administration."
  }
  assert {
    condition     = length(google_service_account.component) == 10 && google_billing_budget.development.amount[0].specified_amount[0].units == "50" && google_billing_budget.development.budget_filter[0].projects == toset(["projects/123456789012"])
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

run "deployment_trust_and_authority" {
  command = plan
  assert {
    condition     = google_iam_workload_identity_pool.deployment.disabled && google_iam_workload_identity_pool_provider.deployment.disabled && google_iam_workload_identity_pool.deployment.workload_identity_pool_id != google_iam_workload_identity_pool.github.workload_identity_pool_id
    error_message = "Deploy trust must be independently disabled and isolated from builder claims."
  }
  assert {
    condition     = google_iam_workload_identity_pool_provider.deployment.attribute_condition == "assertion.actor_id == '55423093' && assertion.event_name == 'workflow_dispatch' && assertion.ref == 'refs/heads/main' && assertion.repository_id == '1414515318' && assertion.repository_owner_id == '55423093' && assertion.workflow_ref == 'justin-napolitano/jayburd-bookkeeping/.github/workflows/deploy-dev.yml@refs/heads/main'"
    error_message = "Deploy federation must require the exact owner, repo, event, main and deployment workflow."
  }
  assert {
    condition     = toset(keys(google_service_account_iam_member.runtime_act_as)) == toset(["api", "frontend", "migrator"]) && google_project_iam_member.run_deployer.role == "roles/run.admin" && google_artifact_registry_repository_iam_member.deployer.role == "roles/artifactregistry.reader"
    error_message = "ActAs is confined to three runtime identities; deployer reads but cannot publish images."
  }
  assert {
    condition     = google_project_iam_custom_role.identity_lookup.permissions == toset(["firebaseauth.users.get"])
    error_message = "API can check identity revocation, not manage users."
  }
}
run "deploy_enable_does_not_enable_builder" {
  command = plan
  variables { enable_github_deploy_identity = true }
  assert {
    condition     = !google_iam_workload_identity_pool.deployment.disabled && !google_iam_workload_identity_pool_provider.deployment.disabled && google_iam_workload_identity_pool.github.disabled
    error_message = "Owner controls the two federation pools independently."
  }
}

run "worker_release_isolated_by_default" {
  command = plan
  assert {
    condition     = google_iam_workload_identity_pool.worker_deployment.disabled && google_iam_workload_identity_pool_provider.worker_deployment.disabled && google_iam_workload_identity_pool.worker_deployment.workload_identity_pool_id != google_iam_workload_identity_pool.deployment.workload_identity_pool_id
    error_message = "Worker tokens must not inherit application deployer trust."
  }
  assert {
    condition     = google_storage_bucket_iam_member.worker_runtime_state.bucket == google_storage_bucket.state["worker-runtime"].name && google_storage_bucket_iam_member.worker_runtime_state.bucket != google_storage_bucket.state["workers"].name
    error_message = "CI must never receive bootstrap state access."
  }
  assert {
    condition     = google_iam_workload_identity_pool_provider.worker_deployment.attribute_condition == "assertion.actor_id == '55423093' && assertion.event_name == 'workflow_dispatch' && assertion.ref == 'refs/heads/main' && assertion.repository_id == '1414515318' && assertion.repository_owner_id == '55423093' && assertion.workflow_ref == 'justin-napolitano/jayburd-bookkeeping/.github/workflows/deploy-workers.yml@refs/heads/main'"
    error_message = "Worker federation requires the exact owner-dispatched main workflow."
  }
}
run "reject_noncanonical_project_number" {
  command = plan
  variables { project_number = "012345678901" }
  expect_failures = [var.project_number]
}
