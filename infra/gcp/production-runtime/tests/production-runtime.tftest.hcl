mock_provider "google" {}
variables {
  project_id     = "jayburd-bookkeeping-dev"
  project_number = "9088794002"
}
run "inert_default" {
  command = plan
  assert {
    condition     = length(google_cloud_run_v2_service.api) == 0 && length(google_cloud_run_v2_service.web) == 0 && length(google_cloud_run_v2_job.readback) == 0 && length(google_cloud_run_v2_service_iam_binding.api) == 0 && length(google_cloud_run_v2_service_iam_binding.web) == 0
    error_message = "No service, job or runtime IAM without explicit configuration."
  }
}
run "paused_production_profile" {
  command = plan
  variables {
    runtime = {
      images = {
        api    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        web    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        worker = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
      firebase_api_key      = "synthetic-public-browser-id"
      client_id_version     = "7"
      client_secret_version = "9"
      readback_expires_at   = "2030-01-01T00:00:00Z"
    }
  }
  assert {
    condition     = contains(local.connection_args, "-provider-environment=production") && contains(local.connection_args, "-infisical-environment=prod") && contains(local.connection_args, "-plaid-consent-expires-at=2020-01-01T00:00:00Z") && !contains(local.connection_args, "-plaid-consent-expires-at=2030-01-01T00:00:00Z") && contains(google_cloud_run_v2_service.api[0].template[0].containers[0].args, "-cloud-sql-instance=jayburd-bookkeeping-dev:us-central1:jayburd-dev")
    error_message = "Explicit Production SQL routing, permanently expired bank authority."
  }
  assert {
    condition     = google_cloud_run_v2_service.api[0].name == "jayburd-prod-api" && google_cloud_run_v2_service.web[0].name == "jayburd-prod-web" && google_cloud_run_v2_service.api[0].template[0].service_account == "jayburd-prod-api@jayburd-bookkeeping-dev.iam.gserviceaccount.com" && google_cloud_run_v2_service.web[0].template[0].service_account == "jayburd-prod-frontend@jayburd-bookkeeping-dev.iam.gserviceaccount.com" && google_cloud_run_v2_service_iam_binding.api[0].members == toset(["serviceAccount:jayburd-prod-frontend@jayburd-bookkeeping-dev.iam.gserviceaccount.com"]) && google_cloud_run_v2_service_iam_binding.web[0].members == toset(["allUsers"]) && !google_cloud_run_v2_service.api[0].invoker_iam_disabled
    error_message = "Private API belongs only to Production frontend; public sign-in shell retains IAM enforcement."
  }
  assert {
    condition     = alltrue([for s in [google_cloud_run_v2_service.api[0], google_cloud_run_v2_service.web[0]] : s.scaling[0].min_instance_count == 0 && s.scaling[0].max_instance_count == 1 && s.template[0].scaling[0].max_instance_count == 1 && s.template[0].containers[0].resources[0].cpu_idle && !s.template[0].containers[0].resources[0].startup_cpu_boost])
    error_message = "Bounded min0/max1 with idle CPU billing and no boost."
  }
  assert {
    condition     = local.web_environment.BOOKKEEPING_API_ORIGIN == "https://jayburd-prod-api-9088794002.us-central1.run.app" && local.web_environment.BOOKKEEPING_PUBLIC_ORIGIN == "https://jayburd-prod-web-9088794002.us-central1.run.app" && google_cloud_run_v2_service.api[0].custom_audiences == tolist([local.api_origin])
    error_message = "Both browser origin and private bridge audience use exact Production destinations."
  }
  assert {
    condition     = google_cloud_run_v2_job.readback[0].template[0].task_count == 1 && google_cloud_run_v2_job.readback[0].template[0].parallelism == 1 && google_cloud_run_v2_job.readback[0].template[0].template[0].max_retries == 0 && google_cloud_run_v2_job.readback[0].template[0].template[0].timeout == "60s" && google_cloud_run_v2_job.readback[0].template[0].template[0].containers[0].command == tolist(["/credential-check"]) && google_cloud_run_v2_job.readback[0].template[0].template[0].service_account == "jayburd-prod-api@jayburd-bookkeeping-dev.iam.gserviceaccount.com"
    error_message = "One bounded readback under the actual API identity, no task retries."
  }
}
run "reject_sandbox_image" {
  command = plan
  variables {
    runtime = {
      images = {
        api    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        web    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        worker = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
      firebase_api_key      = "synthetic-public-browser-id"
      client_id_version     = "7"
      client_secret_version = "9"
      readback_expires_at   = "2030-01-01T00:00:00Z"
    }
  }
  expect_failures = [var.runtime]
}
run "reject_mutable_version" {
  command = plan
  variables {
    runtime = {
      images = {
        api    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        web    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        worker = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
      firebase_api_key      = "synthetic-public-browser-id"
      client_id_version     = "latest"
      client_secret_version = "9"
      readback_expires_at   = "2030-01-01T00:00:00Z"
    }
  }
  expect_failures = [var.runtime]
}
run "reject_other_project" {
  command = plan
  variables { project_id = "synthetic-other-project" }
  expect_failures = [var.project_id]
}
run "reject_mixed_number" {
  command = plan
  variables { project_number = "123456789012" }
  expect_failures = [var.project_number]
}

run "explicit_finite_pilot" {
  command = plan
  variables {
    runtime = {
      images = {
        api    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        web    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        worker = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
      firebase_api_key      = "synthetic-public-browser-id"
      client_id_version     = "7"
      client_secret_version = "9"
      readback_expires_at   = "2030-01-01T00:00:00Z"
    }
    pilot = { approval_expires_at = timeadd(timestamp(), "24h"), webhook_url = "https://jayburd-prod-webhook-9088794002.us-central1.run.app/webhooks/plaid" }
  }
  assert {
    condition     = contains(local.connection_args, "-plaid-consent-expires-at=${var.pilot.approval_expires_at}") && contains(local.connection_args, "-plaid-webhook-url=https://jayburd-prod-webhook-9088794002.us-central1.run.app/webhooks/plaid") && !contains(local.connection_args, "-plaid-consent-expires-at=2030-01-01T00:00:00Z") && !contains(local.connection_args, "-plaid-hosted-repair=true")
    error_message = "Only separate finite pilot authority activates the matching receiver; no repair opt-in/readback-date reuse."
  }
}
run "reject_expired_pilot" {
  command = plan
  variables {
    runtime = {
      images = {
        api    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        web    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        worker = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
      firebase_api_key      = "synthetic-public-browser-id"
      client_id_version     = "7"
      client_secret_version = "9"
      readback_expires_at   = "2030-01-01T00:00:00Z"
    }
    pilot = { approval_expires_at = "2020-01-01T00:00:00Z", webhook_url = "https://jayburd-prod-webhook-9088794002.us-central1.run.app/webhooks/plaid" }
  }
  expect_failures = [var.pilot]
}
run "reject_long_pilot" {
  command = plan
  variables {
    runtime = {
      images = {
        api    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        web    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        worker = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
      firebase_api_key      = "synthetic-public-browser-id"
      client_id_version     = "7"
      client_secret_version = "9"
      readback_expires_at   = "2030-01-01T00:00:00Z"
    }
    pilot = { approval_expires_at = timeadd(timestamp(), "192h"), webhook_url = "https://jayburd-prod-webhook-9088794002.us-central1.run.app/webhooks/plaid" }
  }
  expect_failures = [var.pilot]
}
run "reject_foreign_receiver" {
  command = plan
  variables {
    runtime = {
      images = {
        api    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        web    = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        worker = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
      firebase_api_key      = "synthetic-public-browser-id"
      client_id_version     = "7"
      client_secret_version = "9"
      readback_expires_at   = "2030-01-01T00:00:00Z"
    }
    pilot = { approval_expires_at = timeadd(timestamp(), "24h"), webhook_url = "https://foreign.invalid/webhooks/plaid" }
  }
  expect_failures = [var.pilot]
}
run "reject_pilot_without_runtime" {
  command = plan
  variables {
    runtime = null
    pilot   = { approval_expires_at = timeadd(timestamp(), "24h"), webhook_url = "https://jayburd-prod-webhook-9088794002.us-central1.run.app/webhooks/plaid" }
  }
  expect_failures = [var.pilot]
}
