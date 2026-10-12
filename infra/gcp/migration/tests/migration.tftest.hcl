mock_provider "google" {}
variables {
  project_id = "jayburd-synthetic-test"
  image      = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/migrate@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
}
run "bounded_migration_job" {
  command = plan
  assert {
    condition     = google_cloud_run_v2_job.migrate.template[0].task_count == 1 && google_cloud_run_v2_job.migrate.template[0].parallelism == 1 && google_cloud_run_v2_job.migrate.template[0].template[0].max_retries == 0 && google_cloud_run_v2_job.migrate.template[0].template[0].timeout == "660s"
    error_message = "A single bounded migration execution must not retry automatically."
  }
  assert {
    condition     = google_cloud_run_v2_job.migrate.template[0].template[0].service_account == "jayburd-dev-migrator@jayburd-synthetic-test.iam.gserviceaccount.com" && google_cloud_run_v2_job.migrate.template[0].template[0].containers[0].args == tolist(["-action=up", "-instance=jayburd-synthetic-test:us-central1:jayburd-dev"])
    error_message = "Only explicit forward migration through the dedicated IAM identity is allowed."
  }
}
run "reject_mutable_image" {
  command = plan
  variables { image = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/migrate:latest" }
  expect_failures = [var.image]
}
