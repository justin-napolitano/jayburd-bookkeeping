locals {
  labels           = { application = "jayburd", environment = "production", managed_by = "terraform" }
  accounts         = { for c in ["publisher", "sync", "scheduler", "task-invoker", "webhook"] : c => "jayburd-prod-${c}@${var.project_id}.iam.gserviceaccount.com" }
  origins          = { for c in ["publisher", "sync"] : c => "https://jayburd-prod-acquisition-${c}-${var.project_number}.${var.region}.run.app" }
  webhook_origin   = "https://jayburd-prod-webhook-${var.project_number}.${var.region}.run.app"
  instance         = "${var.project_id}:${var.region}:jayburd-dev"
  bank_key         = "projects/${var.project_id}/locations/${var.region}/keyRings/jayburd-prod/cryptoKeys/bank-credentials"
  link_key         = "projects/${var.project_id}/locations/${var.region}/keyRings/jayburd-prod/cryptoKeys/hosted-link"
  runtime_services = var.runtime == null ? toset([]) : toset(["publisher", "sync"])
}
// Existing foundation owns identities and database/resource capabilities; only exact caller IDs are read.
data "google_service_account" "caller" {
  for_each   = var.runtime == null ? toset([]) : toset(["scheduler", "task-invoker"])
  project    = var.project_id
  account_id = "jayburd-prod-${each.key}"
}
resource "google_cloud_run_v2_service" "worker" {
  for_each             = local.runtime_services
  project              = var.project_id
  name                 = "jayburd-prod-acquisition-${each.value}"
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
    service_account                  = local.accounts[each.value]
    timeout                          = "90s"
    max_instance_request_concurrency = 1
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
    containers {
      image = var.runtime.image
      ports { container_port = 8080 }
      args = concat([
        "-mode=${each.value == "publisher" ? "publisher" : "consumer"}", "-listen=0.0.0.0:8080",
        "-provider-environment=production", "-provider-approval-expires-at=${var.runtime.approval_expires_at}",
        "-cloud-sql-instance=${local.instance}", "-caller-audience=${local.origins[each.value]}",
        "-caller-email=${local.accounts[each.value == "publisher" ? "scheduler" : "task-invoker"]}",
        "-caller-subject=${data.google_service_account.caller[each.value == "publisher" ? "scheduler" : "task-invoker"].unique_id}"
        ], each.value == "publisher" ? [
        "-publisher-policy=binding-v1", "-binding-id=${var.runtime.binding_id}", "-queue-location=${var.region}", "-queue-name=jayburd-prod-acquisition",
        "-consumer-url=${local.origins.sync}/internal/acquisition/consume", "-consumer-audience=${local.origins.sync}", "-task-service-account=${local.accounts["task-invoker"]}"
        ] : [
        "-project-number=${var.project_number}", "-bank-credential-key=${local.bank_key}", "-link-handoff-key=${local.link_key}",
        "-credential-mapping-bucket=${var.project_id}-jayburd-prod-credential-registry", "-credential-mapping-prefix=provider-credentials-v1",
        "-evidence-bucket=${var.project_id}-jayburd-prod-acquisition-evidence",
        "-plaid-client-id-version=projects/${var.project_number}/secrets/jayburd-prod-bank-sync-client-id/versions/${var.runtime.client_id_version}",
        "-plaid-client-secret-version=projects/${var.project_number}/secrets/jayburd-prod-bank-sync-client-secret/versions/${var.runtime.client_secret_version}"
      ])
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
resource "google_cloud_run_v2_service_iam_binding" "worker" {
  for_each = local.runtime_services
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.worker[each.value].name
  role     = "roles/run.invoker"
  members  = ["serviceAccount:${local.accounts[each.value == "publisher" ? "scheduler" : "task-invoker"]}"]
}
// Delivery retries are bounded; current SQL receipts/fences remain authoritative for every attempt.
resource "google_cloud_tasks_queue" "acquisition" {
  count    = var.runtime == null ? 0 : 1
  project  = var.project_id
  name     = "jayburd-prod-acquisition"
  location = var.region
  rate_limits {
    max_dispatches_per_second = 1
    max_concurrent_dispatches = 1
  }
  retry_config {
    max_attempts       = 5
    max_retry_duration = "600s"
    min_backoff        = "5s"
    max_backoff        = "60s"
    max_doublings      = 3
  }
}
resource "google_cloud_tasks_queue_iam_member" "publisher" {
  count    = var.runtime == null ? 0 : 1
  project  = var.project_id
  location = var.region
  name     = google_cloud_tasks_queue.acquisition[0].name
  role     = "roles/cloudtasks.enqueuer"
  member   = "serviceAccount:${local.accounts.publisher}"
}
resource "google_service_account_iam_member" "task_act_as" {
  count              = var.runtime == null ? 0 : 1
  service_account_id = data.google_service_account.caller["task-invoker"].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${local.accounts.publisher}"
}
// No input can resume scheduling. Real first dispatch is a separate reviewed operator action.
resource "google_cloud_scheduler_job" "publisher" {
  count            = var.runtime == null ? 0 : 1
  project          = var.project_id
  name             = "jayburd-prod-acquisition-publisher"
  region           = var.region
  schedule         = "* * * * *"
  time_zone        = "Etc/UTC"
  paused           = true
  attempt_deadline = "90s"
  retry_config { retry_count = 0 }
  http_target {
    uri         = "${local.origins.publisher}/internal/acquisition/publish"
    http_method = "POST"
    body        = base64encode("{}")
    headers     = { "Content-Type" = "application/json" }
    oidc_token {
      service_account_email = local.accounts.scheduler
      audience              = local.origins.publisher
    }
  }
  depends_on = [google_cloud_run_v2_service_iam_binding.worker, google_cloud_tasks_queue_iam_member.publisher, google_service_account_iam_member.task_act_as]
}
// Signature verification is the public boundary; no bank-token/KMS/queue capability belongs to this identity.
resource "google_cloud_run_v2_service" "webhook" {
  count                = var.runtime == null ? 0 : 1
  project              = var.project_id
  name                 = "jayburd-prod-webhook"
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
    service_account                  = local.accounts.webhook
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
        "-listen=0.0.0.0:8080", "-provider-environment=production", "-provider-approval-expires-at=${var.runtime.approval_expires_at}",
        "-cloud-sql-instance=${local.instance}", "-project-number=${var.project_number}",
        "-plaid-client-id-version=projects/${var.project_number}/secrets/jayburd-prod-bank-sync-client-id/versions/${var.runtime.client_id_version}",
        "-plaid-client-secret-version=projects/${var.project_number}/secrets/jayburd-prod-bank-sync-client-secret/versions/${var.runtime.client_secret_version}"
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
resource "google_cloud_run_v2_service_iam_binding" "webhook" {
  count    = var.runtime == null ? 0 : (var.runtime.public_webhook_approved ? 1 : 0)
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.webhook[0].name
  role     = "roles/run.invoker"
  members  = ["allUsers"]
}
