mock_provider "google" {
  mock_resource "google_service_account" {
    defaults = {
      email = "jayburd-dev-sync@jayburd-synthetic-test.iam.gserviceaccount.com"
    }
  }
  mock_data "google_service_account" {
    defaults = {
      name      = "projects/jayburd-synthetic-test/serviceAccounts/jayburd-dev-task-invoker@jayburd-synthetic-test.iam.gserviceaccount.com"
      unique_id = "123456789012345678901"
    }
  }
}
variables {
  project_id     = "jayburd-synthetic-test"
  project_number = "123456789012"
}
run "infrastructure_only_defaults" {
  command = plan
  assert {
    condition     = length(google_cloud_run_v2_service_iam_binding.worker) == 0 && google_cloud_scheduler_job.publisher[0].paused
    error_message = "Bootstrap must keep scheduling paused and invocation unbound until owner confirmation."
  }
  assert {
    condition     = length(google_secret_manager_secret.provider) == 10 && length(google_secret_manager_secret_iam_member.infisical_sync) == 0
    error_message = "Only ten empty provider containers; no implicit Infisical sync grant."
  }
  assert {
    condition     = alltrue([for b in google_storage_bucket.private : b.uniform_bucket_level_access && b.public_access_prevention == "enforced" && !b.force_destroy && b.soft_delete_policy[0].retention_duration_seconds == 0])
    error_message = "Evidence and registry must be private and retain no hidden soft-deleted copies."
  }
}
run "least_privilege_object_boundaries" {
  command = plan
  assert {
    condition     = toset(google_project_iam_custom_role.evidence_raw.permissions) == toset(["storage.objects.create", "storage.objects.get", "storage.objects.delete"]) && endswith(google_storage_bucket_iam_member.evidence_raw.condition[0].expression, "/objects/raw/')")
    error_message = "Raw object delete authority must be prefix scoped."
  }
  assert {
    condition     = toset(google_project_iam_custom_role.evidence_tombstone.permissions) == toset(["storage.objects.create", "storage.objects.get"]) && endswith(google_storage_bucket_iam_member.evidence_tombstone.condition[0].expression, "/objects/deleted/')")
    error_message = "Tombstones require create/get only and cannot be deleted by the worker."
  }
  assert {
    condition     = toset(google_project_iam_custom_role.registry_writer.permissions) == toset(["storage.objects.create", "storage.objects.get"]) && toset(google_project_iam_custom_role.registry_reader.permissions) == toset(["storage.objects.get"]) && endswith(google_storage_bucket_iam_member.registry_writer.condition[0].expression, "/objects/provider-credentials-v1/')") && endswith(google_storage_bucket_iam_member.registry_reader.condition[0].expression, "/objects/provider-credentials-v1/')")
    error_message = "Registry facts must be immutable and acquisition must remain read-only."
  }
  assert {
    condition     = google_storage_bucket_iam_member.registry_writer.member == "serviceAccount:jayburd-dev-api@jayburd-synthetic-test.iam.gserviceaccount.com" && google_cloud_tasks_queue_iam_member.publisher.role == "roles/cloudtasks.enqueuer"
    error_message = "Only the API writes registry facts; publisher only enqueues in its queue."
  }
}
run "api_metadata_is_slot_only" {
  command = plan
  assert {
    condition     = length(google_secret_manager_secret_iam_member.api_version_metadata) == 8 && toset(google_project_iam_custom_role.version_metadata.permissions) == toset(["secretmanager.versions.list"])
    error_message = "Only propagation slots need version listing; static client reads need no metadata discovery."
  }
}
run "scoped_infisical_destination_writer" {
  command = plan
  variables {
    infisical_sync_service_account = "synthetic-sync@jayburd-synthetic-test.iam.gserviceaccount.com"
  }
  assert {
    condition     = length(google_secret_manager_secret_iam_member.infisical_sync) == 10 && toset(google_project_iam_custom_role.secret_sync.permissions) == toset(["secretmanager.secrets.get", "secretmanager.versions.add", "secretmanager.versions.list"])
    error_message = "Infisical may add only versions in precreated containers, without project administrator access."
  }
}
run "reject_foreign_sync_identity" {
  command = plan
  variables { infisical_sync_service_account = "synthetic-sync@foreign-project.iam.gserviceaccount.com" }
  expect_failures = [var.infisical_sync_service_account]
}
run "reject_runtime_sync_writer" {
  command = plan
  variables { infisical_sync_service_account = "jayburd-dev-api@jayburd-synthetic-test.iam.gserviceaccount.com" }
  expect_failures = [var.infisical_sync_service_account]
}
run "reject_noncanonical_project_number" {
  command = plan
  variables { project_number = "012345678901" }
  expect_failures = [var.project_number]
}

run "worker_release_has_no_activation_or_admin_authority" {
  command = plan
  assert {
    condition     = google_project_iam_custom_role.worker_release.permissions == toset(["run.services.create", "run.services.get", "run.services.update", "run.operations.get", "cloudscheduler.jobs.get"]) && toset(keys(google_service_account_iam_member.worker_release_act_as)) == toset(["publisher", "sync"])
    error_message = "Release identity cannot mutate IAM, secrets, SQL, scheduler or delete services."
  }
  assert {
    condition     = google_project_iam_custom_role.worker_caller_read.permissions == toset(["iam.serviceAccounts.get"]) && toset(keys(google_service_account_iam_member.worker_caller_read)) == toset(["scheduler", "task-invoker"])
    error_message = "Immutable caller ID lookup must not grant token creation or impersonation."
  }
}
run "owner_installs_exact_invokers_after_services_exist" {
  command = plan
  variables { worker_services_exist = true }
  assert {
    condition     = google_cloud_run_v2_service_iam_binding.worker["publisher"].members == toset(["serviceAccount:jayburd-dev-scheduler@jayburd-synthetic-test.iam.gserviceaccount.com"]) && google_cloud_run_v2_service_iam_binding.worker["sync"].members == toset(["serviceAccount:jayburd-dev-task-invoker@jayburd-synthetic-test.iam.gserviceaccount.com"]) && google_cloud_scheduler_job.publisher[0].paused
    error_message = "Owner binding must retain exact callers and paused scheduler."
  }
}
run "reject_release_identity_as_secret_writer" {
  command = plan
  variables { infisical_sync_service_account = "jayburd-dev-worker-deployer@jayburd-synthetic-test.iam.gserviceaccount.com" }
  expect_failures = [var.infisical_sync_service_account]
}
