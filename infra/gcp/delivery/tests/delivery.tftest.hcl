mock_provider "google" {
  mock_resource "google_service_account" {
    defaults = { email = "jayburd-dev-delivery@jayburd-synthetic-test.iam.gserviceaccount.com" }
  }
}
variables {
  project_id     = "jayburd-synthetic-test"
  project_number = "123456789012"
}
run "default_has_no_job" {
  command = plan
  assert {
    condition     = length(google_cloud_run_v2_job.canary) == 0 && google_secret_manager_secret_iam_member.add_canary.role == "roles/secretmanager.secretVersionAdder"
    error_message = "No executable job by default; destination permission must be add-only."
  }
  assert {
    condition     = google_service_account.delivery.account_id == "jayburd-dev-delivery" && google_secret_manager_secret.canary.secret_id == "jayburd-dev-bank-sync-canary"
    error_message = "Canary identity/destination must remain distinct from Plaid."
  }
}
run "private_immutable_receipts" {
  command = plan
  assert {
    condition     = google_storage_bucket.registry.uniform_bucket_level_access && google_storage_bucket.registry.public_access_prevention == "enforced" && !google_storage_bucket.registry.force_destroy && google_project_iam_custom_role.registry.permissions == toset(["storage.objects.create", "storage.objects.get"]) && endswith(google_storage_bucket_iam_member.registry.condition[0].expression, "/objects/delivery-v1/')")
    error_message = "Receipt authority must be private create/get-only within its own prefix."
  }
}
run "bounded_canary_job" {
  command = plan
  variables {
    enable_github_canary = true
    canary = {
      image              = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      infisical_project  = "00000000-0000-4000-8000-000000000001"
      infisical_identity = "00000000-0000-4000-8000-000000000002"
      source_id          = "00000000-0000-4000-8000-000000000003"
      source_version     = "7"
    }
  }
  assert {
    condition     = google_cloud_run_v2_job.canary[0].template[0].task_count == 1 && google_cloud_run_v2_job.canary[0].template[0].parallelism == 1 && google_cloud_run_v2_job.canary[0].template[0].template[0].max_retries == 0 && google_cloud_run_v2_job.canary[0].template[0].template[0].timeout == "60s" && google_cloud_run_v2_job.canary[0].template[0].template[0].containers[0].command == tolist(["/deliver"])
    error_message = "Canary must be one bounded task, with no platform retry."
  }
}
run "reject_unpinned_image_and_alias" {
  command = plan
  variables {
    canary = { image = "worker:latest", infisical_project = "invalid", infisical_identity = "invalid", source_id = "invalid", source_version = "latest" }
  }
  expect_failures = [var.canary]
}
run "reject_other_region" {
  command = plan
  variables { region = "europe-west1" }
  expect_failures = [var.region]
}
run "github_execution_disabled_and_narrow" {
  command = plan
  assert {
    condition     = google_iam_workload_identity_pool.canary.disabled && google_iam_workload_identity_pool_provider.canary.disabled && length(google_cloud_run_v2_job_iam_member.invoke) == 0 && google_project_iam_custom_role.invoke.permissions == toset(["run.jobs.get", "run.jobs.run"])
    error_message = "No default execution or override/admin authority."
  }
  assert {
    condition     = strcontains(google_iam_workload_identity_pool_provider.canary.attribute_condition, "assertion.actor_id == '55423093'") && strcontains(google_iam_workload_identity_pool_provider.canary.attribute_condition, "assertion.repository_id == '1414515318'") && strcontains(google_iam_workload_identity_pool_provider.canary.attribute_condition, "assertion.workflow_ref == 'justin-napolitano/jayburd-bookkeeping/.github/workflows/run-delivery-canary.yml@refs/heads/main'") && strcontains(google_iam_workload_identity_pool_provider.canary.attribute_condition, "assertion.event_name == 'workflow_dispatch'")
    error_message = "Federation must bind exact owner, repository, main workflow and manual dispatch."
  }
}
run "reject_execution_without_job" {
  command = plan
  variables { enable_github_canary = true }
  expect_failures = [var.enable_github_canary]
}
