mock_provider "google" {}
variables { project_id = "jayburd-bookkeeping-dev" }
run "disabled_by_default" {
  command = plan
  assert {
    condition     = length(google_service_account.verifier) == 0 && length(google_sql_user.verifier) == 0 && length(google_storage_bucket.inputs) == 0 && length(google_cloud_run_v2_job.compare) == 0
    error_message = "No platform resource or executable job is created by default."
  }
}
run "identity_without_executable_job" {
  command = plan
  variables { enabled = true }
  assert {
    condition     = length(google_cloud_run_v2_job.compare) == 0 && google_sql_user.verifier[0].name == "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" && google_sql_user.verifier[0].type == "CLOUD_IAM_SERVICE_ACCOUNT" && length(google_project_iam_member.sql) == 2
    error_message = "Separate platform identity must not execute or enroll itself."
  }
  assert {
    condition     = google_storage_bucket.inputs[0].public_access_prevention == "enforced" && google_storage_bucket.inputs[0].uniform_bucket_level_access && !google_storage_bucket.inputs[0].versioning[0].enabled && google_storage_bucket.inputs[0].soft_delete_policy[0].retention_duration_seconds == 0 && google_storage_bucket_iam_member.input_read[0].role == "roles/storage.objectViewer"
    error_message = "Only dedicated private ephemeral synthetic inputs may be read."
  }
}
run "bounded_compare_job" {
  command = plan
  variables {
    enabled = true
    run = {
      image  = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      run_id = "verify-synthetic123"
    }
  }
  assert {
    condition     = google_cloud_run_v2_job.compare[0].template[0].task_count == 1 && google_cloud_run_v2_job.compare[0].template[0].parallelism == 1 && google_cloud_run_v2_job.compare[0].template[0].template[0].max_retries == 0 && google_cloud_run_v2_job.compare[0].template[0].template[0].timeout == "120s"
    error_message = "One bounded comparison with no retries is required."
  }
  assert {
    condition     = google_cloud_run_v2_job.compare[0].template[0].template[0].service_account == "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam.gserviceaccount.com" && google_cloud_run_v2_job.compare[0].template[0].template[0].containers[0].command == tolist(["/sandbox-verify"]) && google_cloud_run_v2_job.compare[0].template[0].template[0].containers[0].args == tolist(["-action=compare", "-manifest=/verification/manifest.json"])
    error_message = "Fixed verifier identity and command cannot become acquisition or posting."
  }
  assert {
    condition     = google_cloud_run_v2_job.compare[0].template[0].template[0].volumes[0].gcs[0].read_only && google_cloud_run_v2_job.compare[0].template[0].template[0].volumes[0].gcs[0].mount_options == tolist(["only-dir=verify-synthetic123", "file-mode=0444", "dir-mode=0555"]) && length(google_cloud_run_v2_job.compare[0].template[0].template[0].containers[0].env) == 0
    error_message = "Inputs must be read-only with no credentials in environment."
  }
}
run "reject_unreviewed_scope" {
  command = plan
  variables { project_id = "jayburd-other-project" }
  expect_failures = [var.project_id]
}
run "reject_disabled_execution" {
  command = plan
  variables {
    run = { image = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", run_id = "verify-synthetic123" }
  }
  expect_failures = [var.run]
}
run "reject_mutable_or_foreign_image" {
  command = plan
  variables {
    enabled = true
    run     = { image = "untrusted.example/worker:latest", run_id = "verify-synthetic123" }
  }
  expect_failures = [var.run]
}
run "reject_directory_escape" {
  command = plan
  variables {
    enabled = true
    run     = { image = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", run_id = "verify-../foreign" }
  }
  expect_failures = [var.run]
}
