mock_provider "google" {}
variables {
  project_id       = "jayburd-synthetic-test"
  project_number   = "123456789012"
  firebase_api_key = "synthetic-public"
  images = {
    api = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    web = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  }
}
run "runtime_boundaries" {
  command = plan
  assert {
    condition     = !google_cloud_run_v2_service.api.invoker_iam_disabled && google_cloud_run_v2_service_iam_binding.api.members == toset(["serviceAccount:jayburd-dev-frontend@jayburd-synthetic-test.iam.gserviceaccount.com"]) && google_cloud_run_v2_service_iam_binding.web.members == toset(["allUsers"])
    error_message = "Public sign-in shell must not grant anonymous API invocation."
  }
  assert {
    condition     = alltrue([for service in [google_cloud_run_v2_service.api, google_cloud_run_v2_service.web] : service.scaling[0].min_instance_count == 0 && service.scaling[0].max_instance_count == 2 && service.template[0].scaling[0].max_instance_count == 2 && service.template[0].containers[0].resources[0].cpu_idle && service.deletion_protection])
    error_message = "Bound service/revision capacity with request billing and deletion protection."
  }
  assert {
    condition     = contains(google_cloud_run_v2_service.api.template[0].containers[0].args, "-mode=managed") && contains(google_cloud_run_v2_service.api.template[0].containers[0].args, "-database-mode=cloudsql") && google_cloud_run_v2_service.api.template[0].containers[0].startup_probe[0].http_get[0].path == "/readyz"
    error_message = "API must start through real managed identity, IAM connector and database readiness."
  }
  assert {
    condition     = local.web_environment.BOOKKEEPING_API_ORIGIN == "https://jayburd-dev-api-123456789012.us-central1.run.app" && local.web_environment.BOOKKEEPING_API_MODE == "cloud-run" && length(local.web_environment) == 6 && google_cloud_run_v2_service.api.custom_audiences == tolist([local.api_origin])
    error_message = "Use explicit deterministic origins, custom audience and only public web configuration."
  }
}
run "reject_mutable_or_foreign_images" {
  command = plan
  variables { images = { api = "evil.invalid/api:latest", web = "evil.invalid/web:latest" } }
  expect_failures = [var.images]
}
run "reject_missing_identity" {
  command = plan
  variables { firebase_api_key = "" }
  expect_failures = [var.firebase_api_key]
}
