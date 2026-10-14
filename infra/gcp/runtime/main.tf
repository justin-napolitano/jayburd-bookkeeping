locals {
  # No metadata means no activation flags, secret reads or enrollment composition.
  connection_args = var.connections == null ? [] : [
    "-connections-enabled", "-plaid-link-events-qualified",
    "-provider-project-number=${var.project_number}",
    "-infisical-origin=${var.connections.infisical_origin}",
    "-infisical-project=${var.connections.infisical_project}",
    "-infisical-environment=dev",
    "-infisical-identity=${var.connections.infisical_identity}",
    "-provider-registry-bucket=${var.project_id}-jayburd-dev-credential-registry",
    "-provider-registry-prefix=provider-credentials-v1",
    "-plaid-client-id-version=${var.connections.client_id_version}",
    "-plaid-client-secret-version=${var.connections.client_secret_version}",
    "-plaid-consent-expires-at=${var.connections.consent_expires_at}"
  ]
  public_origin = "https://jayburd-dev-web-${var.project_number}.${var.region}.run.app"
  api_origin    = "https://jayburd-dev-api-${var.project_number}.${var.region}.run.app"
  labels        = { application = "jayburd", environment = "dev", managed_by = "terraform" }
  web_environment = {
    BOOKKEEPING_API_MODE         = "cloud-run"
    BOOKKEEPING_API_ORIGIN       = local.api_origin
    BOOKKEEPING_PUBLIC_ORIGIN    = local.public_origin
    JAYBURD_FIREBASE_API_KEY     = var.firebase_api_key
    JAYBURD_FIREBASE_PROJECT_ID  = var.project_id
    JAYBURD_FIREBASE_AUTH_DOMAIN = "${var.project_id}.firebaseapp.com"
  }
}
# Foundation owns API enablement, identities and project IAM. This state owns only runtime services.
resource "google_cloud_run_v2_service" "api" {
  project             = var.project_id
  name                = "jayburd-dev-api"
  location            = var.region
  labels              = local.labels
  deletion_protection = true
  # Network reachable without VPC/NAT; IAM requires the frontend service identity on every request.
  ingress              = "INGRESS_TRAFFIC_ALL"
  invoker_iam_disabled = false
  custom_audiences     = [local.api_origin]
  scaling {
    min_instance_count = 0
    max_instance_count = 2
  }
  template {
    service_account                  = "jayburd-dev-api@${var.project_id}.iam.gserviceaccount.com"
    timeout                          = "15s"
    max_instance_request_concurrency = 16
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }
    containers {
      image = var.images.api
      args  = concat(["-mode=managed", "-listen=0.0.0.0:8080", "-project=${var.project_id}", "-origin=${local.public_origin}", "-database-mode=cloudsql", "-cloud-sql-instance=${var.project_id}:${var.region}:jayburd-dev"], local.connection_args)
      ports { container_port = 8080 }
      resources {
        limits            = { cpu = "1", memory = "512Mi" }
        cpu_idle          = true
        startup_cpu_boost = false
      }
      startup_probe {
        http_get { path = "/readyz" }
        initial_delay_seconds = 0
        period_seconds        = 5
        timeout_seconds       = 3
        failure_threshold     = 12
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
  lifecycle {
    prevent_destroy = true
    precondition {
      condition     = var.connections == null ? true : try(timecmp(var.connections.consent_expires_at, plantimestamp()) > 0 && timecmp(var.connections.consent_expires_at, timeadd(plantimestamp(), "720h")) <= 0, false)
      error_message = "Connection consent requires an explicit future absolute expiry within 30 days; never roll it forward automatically."
    }
  }
}
# Authoritative invoker membership removes drifted public grants for this role.
resource "google_cloud_run_v2_service_iam_binding" "api" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  members  = ["serviceAccount:jayburd-dev-frontend@${var.project_id}.iam.gserviceaccount.com"]
}
resource "google_cloud_run_v2_service" "web" {
  project              = var.project_id
  name                 = "jayburd-dev-web"
  location             = var.region
  labels               = local.labels
  deletion_protection  = true
  ingress              = "INGRESS_TRAFFIC_ALL"
  invoker_iam_disabled = false
  scaling {
    min_instance_count = 0
    max_instance_count = 2
  }
  template {
    service_account                  = "jayburd-dev-frontend@${var.project_id}.iam.gserviceaccount.com"
    timeout                          = "20s"
    max_instance_request_concurrency = 32
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }
    containers {
      image = var.images.web
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
# Public sign-in shell only. Go still requires verified MFA and current entity authority for finance.
resource "google_cloud_run_v2_service_iam_binding" "web" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.web.name
  role     = "roles/run.invoker"
  members  = ["allUsers"]
}
