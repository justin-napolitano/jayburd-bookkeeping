mock_provider "google" {}
variables {
  project_id = "jayburd-bookkeeping-dev"
  image      = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
run "bounded_api_permission_probe" {
  command = plan
  assert {
    condition     = google_cloud_run_v2_job.probe.template[0].task_count == 1 && google_cloud_run_v2_job.probe.template[0].parallelism == 1 && google_cloud_run_v2_job.probe.template[0].template[0].max_retries == 0 && google_cloud_run_v2_job.probe.template[0].template[0].timeout == "120s"
    error_message = "Probe must be single-task, bounded and without retries."
  }
  assert {
    condition     = google_cloud_run_v2_job.probe.template[0].template[0].service_account == "jayburd-dev-api@jayburd-bookkeeping-dev.iam.gserviceaccount.com" && google_cloud_run_v2_job.probe.template[0].template[0].containers[0].command == tolist(["/database-probe"]) && google_cloud_run_v2_job.probe.template[0].template[0].containers[0].args == tolist(["-action=probe", "-instance=jayburd-bookkeeping-dev:us-central1:jayburd-dev"])
    error_message = "Probe must use exact API identity and reviewed command."
  }
}
run "reject_mutable_image" {
  command = plan
  variables { image = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api:latest" }
  expect_failures = [var.image]
}
run "reject_other_project" {
  command = plan
  variables {
    project_id = "jayburd-other-project"
    image      = "us-central1-docker.pkg.dev/jayburd-other-project/jayburd-dev/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  }
  expect_failures = [var.project_id]
}
