locals {
  origin = "https://jayburd-dev-webhook-${var.project_number}.${var.region}.run.app"
  labels = { application = "jayburd", environment = "dev", managed_by = "terraform" }
}

# This state owns one fixed-route service, never identity/bootstrap, private services or scheduler activation.
resource "google_cloud_run_v2_service" "webhook" {
  count                = var.runtime == null ? 0 : 1
  project              = var.project_id
  name                 = "jayburd-dev-webhook"
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
    service_account                  = "jayburd-dev-webhook@${var.project_id}.iam.gserviceaccount.com"
    timeout                          = "10s"
    max_instance_request_concurrency = 16
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
    containers {
      image   = var.runtime.image
      command = ["/webhook"]
      args = [
        "-listen=0.0.0.0:8080",
        "-cloud-sql-instance=${var.project_id}:${var.region}:jayburd-dev",
        "-project-number=${var.project_number}",
        "-plaid-client-id-version=projects/${var.project_number}/secrets/jayburd-dev-bank-sync-client-id/versions/${var.runtime.client_id_version}",
        "-plaid-client-secret-version=projects/${var.project_number}/secrets/jayburd-dev-bank-sync-client-secret/versions/${var.runtime.client_secret_version}"
      ]
      ports { container_port = 8080 }
      resources {
        limits            = { cpu = "1", memory = "512Mi" }
        cpu_idle          = true
        startup_cpu_boost = false
      }
      startup_probe {
        tcp_socket { port = 8080 }
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
}
