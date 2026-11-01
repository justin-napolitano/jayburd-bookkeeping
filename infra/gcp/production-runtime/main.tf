locals {
  api_origin    = "https://jayburd-prod-api-${var.project_number}.${var.region}.run.app"
  public_origin = "https://jayburd-prod-web-${var.project_number}.${var.region}.run.app"
  labels        = { application = "jayburd", environment = "production", managed_by = "terraform" }
  # Null pilot preserves expired SQL-only recovery; readback authority alone never enables banking.
  connection_args = var.runtime == null ? [] : concat([
    "-connections-enabled", "-provider-environment=production", "-provider-project-number=${var.project_number}",
    "-infisical-origin=https://app.infisical.com", "-infisical-environment=prod",
    "-infisical-project=cfd371a3-1229-48d8-a9e2-9c2ea37540cc", "-infisical-identity=8d510558-6b96-44c0-b25f-ce73ff7e4d74",
    "-provider-registry-bucket=${var.project_id}-jayburd-prod-credential-registry", "-provider-registry-prefix=provider-credentials-v1",
    "-plaid-client-id-version=projects/${var.project_number}/secrets/jayburd-prod-bank-sync-client-id/versions/${var.runtime.client_id_version}",
    "-plaid-client-secret-version=projects/${var.project_number}/secrets/jayburd-prod-bank-sync-client-secret/versions/${var.runtime.client_secret_version}",
    "-bank-credential-key=projects/${var.project_id}/locations/${var.region}/keyRings/jayburd-prod/cryptoKeys/bank-credentials",
    "-link-handoff-key=projects/${var.project_id}/locations/${var.region}/keyRings/jayburd-prod/cryptoKeys/hosted-link",
    "-plaid-consent-expires-at=${var.pilot == null ? "2020-01-01T00:00:00Z" : var.pilot.approval_expires_at}"
  ], var.pilot == null ? [] : ["-plaid-webhook-url=${var.pilot.webhook_url}"])
  web_environment = var.runtime == null ? {} : {
    BOOKKEEPING_API_MODE         = "cloud-run"
    BOOKKEEPING_API_ORIGIN       = local.api_origin
    BOOKKEEPING_PUBLIC_ORIGIN    = local.public_origin
    JAYBURD_FIREBASE_API_KEY     = var.runtime.firebase_api_key
    JAYBURD_FIREBASE_PROJECT_ID  = var.project_id
    JAYBURD_FIREBASE_AUTH_DOMAIN = "${var.project_id}.firebaseapp.com"
  }
}

// This root owns paused services only; foundation owns identities, SQL, secret containers and project IAM.
resource "google_cloud_run_v2_service" "api" {
  count                = var.runtime == null ? 0 : 1
  project              = var.project_id
  name                 = "jayburd-prod-api"
  location             = var.region
  labels               = local.labels
  deletion_protection  = true
  ingress              = "INGRESS_TRAFFIC_ALL"
  invoker_iam_disabled = false
  custom_audiences     = [local.api_origin]
  scaling {
    min_instance_count = 0
    max_instance_count = 1
  }
  template {
    service_account                  = "jayburd-prod-api@${var.project_id}.iam.gserviceaccount.com"
    timeout                          = "15s"
    max_instance_request_concurrency = 16
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
    containers {
      image = var.runtime.images.api
      args  = concat(["-mode=managed", "-listen=0.0.0.0:8080", "-project=${var.project_id}", "-origin=${local.public_origin}", "-database-mode=cloudsql", "-cloud-sql-instance=${var.project_id}:${var.region}:jayburd-dev"], local.connection_args)
      ports { container_port = 8080 }
      resources {
        limits            = { cpu = "1", memory = "512Mi" }
        cpu_idle          = true
        startup_cpu_boost = false
      }
      startup_probe {
        http_get { path = "/readyz" }
        period_seconds    = 5
        timeout_seconds   = 3
        failure_threshold = 12
      }
      liveness_probe {
        http_get { path = "/healthz" }
        period_seconds    = 30
        timeout_seconds   = 3
        failure_threshold = 3
      }
    }
  }
  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
  lifecycle { prevent_destroy = true }
}
resource "google_cloud_run_v2_service_iam_binding" "api" {
  count    = var.runtime == null ? 0 : 1
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.api[0].name
  role     = "roles/run.invoker"
  members  = ["serviceAccount:jayburd-prod-frontend@${var.project_id}.iam.gserviceaccount.com"]
}

// Public sign-in shell retains verified MFA and current SQL authority for every financial request.
resource "google_cloud_run_v2_service" "web" {
  count                = var.runtime == null ? 0 : 1
  project              = var.project_id
  name                 = "jayburd-prod-web"
  location             = var.region
  labels               = local.labels
  deletion_protection  = true
  ingress              = "INGRESS_TRAFFIC_ALL"
  invoker_iam_disabled = false
  scaling {
    min_instance_count = 0
    max_instance_count = 1
  }
  template {
    service_account                  = "jayburd-prod-frontend@${var.project_id}.iam.gserviceaccount.com"
    timeout                          = "20s"
    max_instance_request_concurrency = 32
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
    containers {
      image = var.runtime.images.web
      ports { container_port = 8080 }
      dynamic "env" {
        for_each = local.web_environment
        content {
          name  = env.key
          value = env.value
        }
      }
      resources {
        limits            = { cpu = "1", memory = "512Mi" }
        cpu_idle          = true
        startup_cpu_boost = false
      }
      startup_probe {
        http_get { path = "/api/auth-config" }
        period_seconds    = 5
        timeout_seconds   = 3
        failure_threshold = 12
      }
    }
  }
  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
  lifecycle { prevent_destroy = true }
  depends_on = [google_cloud_run_v2_service_iam_binding.api]
}
resource "google_cloud_run_v2_service_iam_binding" "web" {
  count    = var.runtime == null ? 0 : 1
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.web[0].name
  role     = "roles/run.invoker"
  members  = ["allUsers"]
}

// Readback creates no endpoint/scheduler/invocation grants; its command performs no SQL or provider calls.
resource "google_cloud_run_v2_job" "readback" {
  count               = var.runtime == null ? 0 : 1
  project             = var.project_id
  location            = var.region
  name                = "jayburd-prod-credential-check"
  deletion_protection = true
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = "jayburd-prod-api@${var.project_id}.iam.gserviceaccount.com"
      timeout         = "60s"
      max_retries     = 0
      containers {
        image   = var.runtime.images.worker
        command = ["/credential-check"]
        args    = ["-execute", "-project-id=${var.project_id}", "-project-number=${var.project_number}", "-client-id-version=${var.runtime.client_id_version}", "-client-secret-version=${var.runtime.client_secret_version}", "-approval-expires-at=${var.runtime.readback_expires_at}"]
        resources { limits = { cpu = "1", memory = "512Mi" } }
      }
    }
  }
  lifecycle { prevent_destroy = true }
}
