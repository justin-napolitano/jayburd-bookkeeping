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
  assert {
    condition     = length(local.connection_args) == 0 && length(google_cloud_run_v2_service.api.template[0].containers[0].args) == 6
    error_message = "Default deployment must not enable connections or add any connection metadata flags."
  }
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

run "enabled_connection_metadata" {
  command = plan
  variables {
    connections = {
      infisical_origin      = "https://us.infisical.com"
      infisical_project     = "11111111-1111-4111-8111-111111111111"
      infisical_identity    = "22222222-2222-4222-8222-222222222222"
      link_events_qualified = true
      client_id_version     = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1"
      client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2"
      consent_expires_at    = timeadd(timestamp(), "24h")
    }
  }
  assert {
    condition     = length(local.connection_args) == 12 && alltrue([for arg in local.connection_args : contains(google_cloud_run_v2_service.api.template[0].containers[0].args, arg)]) && contains(local.connection_args, "-provider-registry-bucket=jayburd-synthetic-test-jayburd-dev-credential-registry") && contains(local.connection_args, "-infisical-environment=dev") && contains(local.connection_args, "-provider-registry-prefix=provider-credentials-v1") && contains(local.connection_args, "-plaid-link-events-qualified")
    error_message = "Enabled metadata must produce the complete fixed Sandbox API profile."
  }
}

run "reject_unqualified_link" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { link_events_qualified = false }) }
  expect_failures = [var.connections]
}

run "reject_foreign_static_version" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { client_id_version = "projects/999999999999/secrets/jayburd-dev-bank-sync-client-id/versions/1" }) }
  expect_failures = [var.connections]
}

run "reject_alias_version" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/latest" }) }
  expect_failures = [var.connections]
}

run "reject_zero_identity" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { infisical_identity = "00000000-0000-0000-0000-000000000000" }) }
  expect_failures = [var.connections]
}

run "reject_expired_consent" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { consent_expires_at = "2000-01-01T00:00:00Z" }) }
  expect_failures = [google_cloud_run_v2_service.api]
}

run "reject_unbounded_consent" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { consent_expires_at = "2099-01-01T00:00:00Z" }) }
  expect_failures = [google_cloud_run_v2_service.api]
}

run "reject_project_zero" {
  command = plan
  variables { project_number = "0" }
  expect_failures = [var.project_number]
}

run "reject_project_leading_zero" {
  command = plan
  variables { project_number = "0123456789012" }
  expect_failures = [var.project_number]
}

run "reject_project_overflow" {
  command = plan
  variables { project_number = "9223372036854775808" }
  expect_failures = [var.project_number]
}

run "reject_client_id_version_overflow" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/9223372036854775808" }) }
  expect_failures = [var.connections]
}

run "reject_client_secret_version_overflow" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/9223372036854775808" }) }
  expect_failures = [var.connections]
}

run "accept_exact_int64_version_maximum" {
  command = plan
  variables { connections = merge({ infisical_origin = "https://us.infisical.com", infisical_project = "11111111-1111-4111-8111-111111111111", infisical_identity = "22222222-2222-4222-8222-222222222222", link_events_qualified = true, client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", consent_expires_at = timeadd(timestamp(), "24h") }, { client_id_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/9223372036854775807", client_secret_version = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/9223372036854775807" }) }
}
