locals {
  labels           = { application = "jayburd", environment = "dev", managed_by = "terraform" }
  accounts         = { for component in ["publisher", "scheduler", "task-invoker"] : component => "jayburd-dev-${component}@${var.project_id}.iam.gserviceaccount.com" }
  runtime_accounts = { publisher = local.accounts.publisher, sync = "jayburd-dev-sync@${var.project_id}.iam.gserviceaccount.com" }
  origins          = { for component in ["publisher", "sync"] : component => "https://jayburd-dev-acquisition-${component}-${var.project_number}.${var.region}.run.app" }
  services         = toset(["publisher", "sync"])
}

# Existing foundation owns API/publisher/scheduler/task-invoker identities. Acquisition has a separate SQL role.
data "google_service_account" "caller" {
  for_each   = toset(["scheduler", "task-invoker"])
  project    = var.project_id
  account_id = "jayburd-dev-${each.key}"
}
resource "google_cloud_run_v2_service" "worker" {
  for_each             = local.services
  project              = var.project_id
  name                 = "jayburd-dev-acquisition-${each.value}"
  location             = var.region
  labels               = local.labels
  deletion_protection  = true
  ingress              = "INGRESS_TRAFFIC_ALL"
  invoker_iam_disabled = false
  custom_audiences     = [local.origins[each.value]]
  scaling {
    min_instance_count = 0
    max_instance_count = 1
  }
  template {
    service_account                  = local.runtime_accounts[each.value]
    timeout                          = "90s"
    max_instance_request_concurrency = 1
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
    containers {
      image = var.runtime.image
      ports {
        container_port = 8080
      }
      args = concat([
        "-mode=${each.value == "publisher" ? "publisher" : "consumer"}", "-listen=0.0.0.0:8080",
        "-cloud-sql-instance=${var.project_id}:${var.region}:jayburd-dev", "-caller-audience=${local.origins[each.value]}",
        "-caller-email=${local.accounts[each.value == "publisher" ? "scheduler" : "task-invoker"]}",
        "-caller-subject=${data.google_service_account.caller[each.value == "publisher" ? "scheduler" : "task-invoker"].unique_id}"
        ], each.value == "publisher" ? [
        "-publisher-policy=${var.runtime.publisher_policy}", "-binding-id=${var.runtime.binding_id}", "-queue-location=${var.region}", "-queue-name=jayburd-dev-acquisition",
        "-consumer-url=${local.origins.sync}/internal/acquisition/consume", "-consumer-audience=${local.origins.sync}", "-task-service-account=${local.accounts["task-invoker"]}"
        ] : [
        "-project-number=${var.project_number}",
        "-credential-mapping-bucket=${var.project_id}-jayburd-dev-credential-registry", "-credential-mapping-prefix=provider-credentials-v1",
        "-evidence-bucket=${var.project_id}-jayburd-dev-acquisition-evidence",
        "-plaid-client-id-version=projects/${var.project_number}/secrets/jayburd-dev-bank-sync-client-id/versions/${var.runtime.client_id_version}",
        "-plaid-client-secret-version=projects/${var.project_number}/secrets/jayburd-dev-bank-sync-client-secret/versions/${var.runtime.client_secret_version}"
      ])
      resources {
        limits            = { cpu = "1", memory = "512Mi" }
        cpu_idle          = true
        startup_cpu_boost = false
      }
      startup_probe {
        tcp_socket {
          port = 8080
        }
        period_seconds    = 5
        failure_threshold = 12
      }
    }
  }
  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
  lifecycle {
    prevent_destroy = true
  }
}
