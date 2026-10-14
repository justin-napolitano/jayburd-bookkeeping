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
    condition     = length(google_cloud_run_v2_service.worker) == 0 && length(google_cloud_scheduler_job.publisher) == 0 && !var.scheduler_enabled
    error_message = "Default configuration must not deploy runtime or enable scheduling."
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
run "runtime_is_private_bounded_and_paused" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "7"
      client_secret_version = "23"
    }
  }
  assert {
    condition     = length(google_cloud_run_v2_service.worker) == 2 && google_cloud_scheduler_job.publisher[0].paused && alltrue([for s in google_cloud_run_v2_service.worker : !s.invoker_iam_disabled && s.deletion_protection && s.template[0].max_instance_request_concurrency == 1 && s.template[0].timeout == "90s" && s.template[0].scaling[0].max_instance_count == 1])
    error_message = "Runtime must retain IAM invocation, bounded capacity, deadline and paused scheduling."
  }
  assert {
    condition     = contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-project-number=123456789012") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-credential-mapping-prefix=provider-credentials-v1") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-plaid-client-id-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/7") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-plaid-client-secret-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/23")
    error_message = "Consumer uses canonical numeric resources and the frozen immutable mapping prefix."
  }
  assert {
    condition     = google_cloud_run_v2_service_iam_binding.worker["publisher"].members == toset(["serviceAccount:jayburd-dev-scheduler@jayburd-synthetic-test.iam.gserviceaccount.com"]) && google_cloud_run_v2_service_iam_binding.worker["sync"].members == toset(["serviceAccount:jayburd-dev-task-invoker@jayburd-synthetic-test.iam.gserviceaccount.com"])
    error_message = "No anonymous or cross-handler invocation authority."
  }
}
run "reject_mutable_image" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker:latest"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "1"
      client_secret_version = "1"
    }
  }
  expect_failures = [var.runtime]
}
run "reject_alias_version" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "latest"
      client_secret_version = "1"
    }
  }
  expect_failures = [var.runtime]
}
run "reject_scheduler_without_runtime" {
  command = plan
  variables { scheduler_enabled = true }
  expect_failures = [var.scheduler_enabled]
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
run "reject_unbounded_version" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "9223372036854775808"
      client_secret_version = "1"
    }
  }
  expect_failures = [var.runtime]
}
