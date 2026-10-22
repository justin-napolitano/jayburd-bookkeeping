mock_provider "google" {
  override_during = plan
}
variables {
  project_id     = "jayburd-synthetic-test"
  project_number = "123456789012"
}
run "default_is_inert" {
  command = plan
  assert {
    condition     = length(google_cloud_run_v2_service.webhook) == 0 && output.webhook_url == null
    error_message = "Null runtime must not create a service."
  }
}
run "isolated_route_identity_and_budget" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      client_id_version     = "7"
      client_secret_version = "23"
    }
  }
  assert {
    condition     = length(google_cloud_run_v2_service.webhook) == 1 && google_cloud_run_v2_service.webhook[0].name == "jayburd-dev-webhook" && !google_cloud_run_v2_service.webhook[0].invoker_iam_disabled && google_cloud_run_v2_service.webhook[0].deletion_protection && google_cloud_run_v2_service.webhook[0].template[0].service_account == "jayburd-dev-webhook@jayburd-synthetic-test.iam.gserviceaccount.com"
    error_message = "Only a dedicated identity/host with retained IAM invocation checks is allowed."
  }
  assert {
    condition     = google_cloud_run_v2_service.webhook[0].template[0].containers[0].command == tolist(["/webhook"]) && google_cloud_run_v2_service.webhook[0].template[0].containers[0].image == var.runtime.image && google_cloud_run_v2_service.webhook[0].template[0].containers[0].args == tolist(["-listen=0.0.0.0:8080", "-cloud-sql-instance=jayburd-synthetic-test:us-central1:jayburd-dev", "-project-number=123456789012", "-plaid-client-id-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/7", "-plaid-client-secret-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/23"])
    error_message = "The reviewed worker digest must select only /webhook with fixed paths and pinned versions."
  }
  assert {
    condition     = google_cloud_run_v2_service.webhook[0].template[0].timeout == "10s" && google_cloud_run_v2_service.webhook[0].template[0].max_instance_request_concurrency == 16 && google_cloud_run_v2_service.webhook[0].scaling[0].max_instance_count == 1 && google_cloud_run_v2_service.webhook[0].template[0].scaling[0].max_instance_count == 1 && google_cloud_run_v2_service.webhook[0].template[0].containers[0].startup_probe[0].tcp_socket[0].port == 8080 && google_cloud_run_v2_service.webhook[0].template[0].containers[0].resources[0].limits == tomap({ cpu = "1", memory = "512Mi" })
    error_message = "Request, concurrency, scaling, startup and compute budgets must remain bounded."
  }
  assert {
    condition     = length(google_cloud_run_v2_service.webhook[0].template[0].containers[0].env) == 0 && length(google_cloud_run_v2_service.webhook[0].template[0].volumes) == 0 && output.webhook_url == "https://jayburd-dev-webhook-123456789012.us-central1.run.app/webhooks/plaid"
    error_message = "No secret environment/mounts or arbitrary webhook destination is accepted."
  }
}
run "reject_mutable_image" {
  command = plan
  variables { runtime = { image = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker:latest", client_id_version = "1", client_secret_version = "1" } }
  expect_failures = [var.runtime]
}
run "reject_cross_project_image" {
  command = plan
  variables { runtime = { image = "us-central1-docker.pkg.dev/other-project/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", client_id_version = "1", client_secret_version = "1" } }
  expect_failures = [var.runtime]
}
run "reject_api_image" {
  command = plan
  variables { runtime = { image = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", client_id_version = "1", client_secret_version = "1" } }
  expect_failures = [var.runtime]
}
run "reject_alias_version" {
  command = plan
  variables { runtime = { image = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", client_id_version = "latest", client_secret_version = "1" } }
  expect_failures = [var.runtime]
}
run "reject_overflow_version" {
  command = plan
  variables { runtime = { image = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", client_id_version = "1", client_secret_version = "9223372036854775808" } }
  expect_failures = [var.runtime]
}
run "reject_leading_zero_version" {
  command = plan
  variables { runtime = { image = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", client_id_version = "01", client_secret_version = "1" } }
  expect_failures = [var.runtime]
}
run "reject_project_overflow" {
  command = plan
  variables { project_number = "9223372036854775808" }
  expect_failures = [var.project_number]
}
run "reject_unreviewed_region" {
  command = plan
  variables { region = "europe-west1" }
  expect_failures = [var.region]
}
