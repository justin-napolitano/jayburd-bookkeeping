locals {
  labels           = { application = "jayburd", environment = "dev", managed_by = "terraform" }
  accounts         = { for component in ["api", "publisher", "scheduler", "task-invoker"] : component => "jayburd-dev-${component}@${var.project_id}.iam.gserviceaccount.com" }
  runtime_accounts = { publisher = local.accounts.publisher, sync = google_service_account.sync.email }
  origins          = { for component in ["publisher", "sync"] : component => "https://jayburd-dev-acquisition-${component}-${var.project_number}.${var.region}.run.app" }
  containers       = toset(concat(["client-id", "client-secret"], [for i in range(8) : format("slot-%02d", i)]))
  services         = var.runtime == null ? toset([]) : toset(["publisher", "sync"])
}

# Existing foundation owns API/publisher/scheduler/task-invoker identities. Acquisition has a separate SQL role.
data "google_service_account" "caller" {
  for_each   = toset(["scheduler", "task-invoker"])
  project    = var.project_id
  account_id = "jayburd-dev-${each.key}"
}
resource "google_project_service" "worker" {
  for_each           = toset(["cloudtasks.googleapis.com", "cloudscheduler.googleapis.com", "secretmanager.googleapis.com"])
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}
resource "google_service_account" "sync" {
  project      = var.project_id
  account_id   = "jayburd-dev-sync"
  display_name = "Jayburd dev acquisition SQL worker"
}
resource "google_project_iam_member" "sql_client" {
  for_each = local.runtime_accounts
  project  = var.project_id
  role     = "roles/cloudsql.client"
  member   = "serviceAccount:${each.value}"
}
resource "google_project_iam_member" "sql_login" {
  for_each = local.runtime_accounts
  project  = var.project_id
  role     = "roles/cloudsql.instanceUser"
  member   = "serviceAccount:${each.value}"
}
resource "google_sql_user" "worker" {
  for_each = local.runtime_accounts
  project  = var.project_id
  instance = "jayburd-dev"
  name     = trimsuffix(each.value, ".gserviceaccount.com")
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
  lifecycle {
    prevent_destroy = true
  }
}

# Values and versions are absent from Terraform. No secret accessor is granted to publisher/scheduler/deployer.
resource "google_secret_manager_secret" "provider" {
  for_each  = local.containers
  project   = var.project_id
  secret_id = "jayburd-dev-bank-sync-${each.value}"
  labels    = local.labels
  replication {
    auto {}
  }
  lifecycle {
    prevent_destroy = true
  }
  depends_on = [google_project_service.worker]
}
resource "google_secret_manager_secret_iam_member" "sync_reader" {
  for_each  = local.containers
  project   = var.project_id
  secret_id = google_secret_manager_secret.provider[each.value].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.sync.email}"
}
resource "google_secret_manager_secret_iam_member" "api_reader" {
  for_each  = local.containers
  project   = var.project_id
  secret_id = google_secret_manager_secret.provider[each.value].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${local.accounts.api}"
}
resource "google_project_iam_custom_role" "version_metadata" {
  project     = var.project_id
  role_id     = "jayburdProviderVersionMetadata"
  title       = "Jayburd provider propagation metadata"
  permissions = ["secretmanager.versions.list"]
}
resource "google_secret_manager_secret_iam_member" "api_version_metadata" {
  for_each  = toset([for i in range(8) : format("slot-%02d", i)])
  project   = var.project_id
  secret_id = google_secret_manager_secret.provider[each.value].secret_id
  role      = google_project_iam_custom_role.version_metadata.name
  member    = "serviceAccount:${local.accounts.api}"
}
resource "google_project_iam_custom_role" "secret_sync" {
  project     = var.project_id
  role_id     = "jayburdProviderSecretSync"
  title       = "Jayburd precreated provider secret delivery"
  permissions = ["secretmanager.secrets.get", "secretmanager.versions.add", "secretmanager.versions.list"]
}
resource "google_secret_manager_secret_iam_member" "infisical_sync" {
  for_each  = var.infisical_sync_service_account == null ? toset([]) : local.containers
  project   = var.project_id
  secret_id = google_secret_manager_secret.provider[each.value].secret_id
  role      = google_project_iam_custom_role.secret_sync.name
  member    = "serviceAccount:${var.infisical_sync_service_account}"
}

resource "google_storage_bucket" "private" {
  for_each                    = toset(["acquisition-evidence", "credential-registry"])
  project                     = var.project_id
  name                        = "${var.project_id}-jayburd-dev-${each.value}"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false
  labels                      = local.labels
  soft_delete_policy {
    retention_duration_seconds = 0
  }
  lifecycle {
    prevent_destroy = true
  }
  dynamic "lifecycle_rule" {
    for_each = each.value == "acquisition-evidence" ? [1] : []
    content {
      action {
        type = "Delete"
      }
      condition {
        age            = 7
        matches_prefix = ["raw/"]
      }
    }
  }
}
resource "google_project_iam_custom_role" "evidence_raw" {
  project     = var.project_id
  role_id     = "jayburdAcquisitionRawEvidence"
  title       = "Jayburd bounded raw evidence objects"
  permissions = ["storage.objects.create", "storage.objects.get", "storage.objects.delete"]
}
resource "google_storage_bucket_iam_member" "evidence_raw" {
  bucket = google_storage_bucket.private["acquisition-evidence"].name
  role   = google_project_iam_custom_role.evidence_raw.name
  member = "serviceAccount:${google_service_account.sync.email}"
  condition {
    title      = "raw_evidence_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["acquisition-evidence"].name}/objects/raw/')"
  }
}
resource "google_project_iam_custom_role" "evidence_tombstone" {
  project     = var.project_id
  role_id     = "jayburdAcquisitionTombstone"
  title       = "Jayburd immutable evidence tombstones"
  permissions = ["storage.objects.create", "storage.objects.get"]
}
resource "google_storage_bucket_iam_member" "evidence_tombstone" {
  bucket = google_storage_bucket.private["acquisition-evidence"].name
  role   = google_project_iam_custom_role.evidence_tombstone.name
  member = "serviceAccount:${google_service_account.sync.email}"
  condition {
    title      = "immutable_tombstone_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["acquisition-evidence"].name}/objects/deleted/')"
  }
}
resource "google_project_iam_custom_role" "registry_writer" {
  project     = var.project_id
  role_id     = "jayburdCredentialRegistryWriter"
  title       = "Jayburd immutable credential mapping writes"
  permissions = ["storage.objects.create", "storage.objects.get"]
}
resource "google_project_iam_custom_role" "registry_reader" {
  project     = var.project_id
  role_id     = "jayburdCredentialRegistryReader"
  title       = "Jayburd exact credential mapping reads"
  permissions = ["storage.objects.get"]
}
resource "google_storage_bucket_iam_member" "registry_writer" {
  bucket = google_storage_bucket.private["credential-registry"].name
  role   = google_project_iam_custom_role.registry_writer.name
  member = "serviceAccount:${local.accounts.api}"
  condition {
    title      = "provider_registry_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["credential-registry"].name}/objects/provider-credentials-v1/')"
  }
}
resource "google_storage_bucket_iam_member" "registry_reader" {
  bucket = google_storage_bucket.private["credential-registry"].name
  role   = google_project_iam_custom_role.registry_reader.name
  member = "serviceAccount:${google_service_account.sync.email}"
  condition {
    title      = "provider_registry_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["credential-registry"].name}/objects/provider-credentials-v1/')"
  }
}

resource "google_cloud_tasks_queue" "acquisition" {
  project  = var.project_id
  name     = "jayburd-dev-acquisition"
  location = var.region
  rate_limits {
    max_dispatches_per_second = 1
    max_concurrent_dispatches = 1
  }
  retry_config {
    max_attempts       = 20
    max_retry_duration = "86400s"
    min_backoff        = "5s"
    max_backoff        = "300s"
    max_doublings      = 6
  }
  depends_on = [google_project_service.worker]
}
resource "google_cloud_tasks_queue_iam_member" "publisher" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_tasks_queue.acquisition.name
  role     = "roles/cloudtasks.enqueuer"
  member   = "serviceAccount:${local.accounts.publisher}"
}
resource "google_service_account_iam_member" "task_act_as" {
  service_account_id = data.google_service_account.caller["task-invoker"].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${local.accounts.publisher}"
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
        "-binding-id=${var.runtime.binding_id}", "-queue-location=${var.region}", "-queue-name=${google_cloud_tasks_queue.acquisition.name}",
        "-consumer-url=${local.origins.sync}/internal/acquisition/consume", "-consumer-audience=${local.origins.sync}", "-task-service-account=${local.accounts["task-invoker"]}"
        ] : [
        "-project-number=${var.project_number}",
        "-credential-mapping-bucket=${google_storage_bucket.private["credential-registry"].name}", "-credential-mapping-prefix=provider-credentials-v1",
        "-evidence-bucket=${google_storage_bucket.private["acquisition-evidence"].name}",
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
  depends_on = [
    google_project_iam_member.sql_client,
    google_project_iam_member.sql_login,
    google_sql_user.worker,
    google_secret_manager_secret_iam_member.sync_reader,
    google_storage_bucket_iam_member.evidence_raw,
    google_storage_bucket_iam_member.evidence_tombstone,
    google_storage_bucket_iam_member.registry_reader,
    google_cloud_tasks_queue_iam_member.publisher,
    google_service_account_iam_member.task_act_as
  ]
  lifecycle {
    prevent_destroy = true
  }
}
resource "google_cloud_run_v2_service_iam_binding" "worker" {
  for_each = local.services
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.worker[each.value].name
  role     = "roles/run.invoker"
  members  = ["serviceAccount:${local.accounts[each.value == "publisher" ? "scheduler" : "task-invoker"]}"]
}
resource "google_cloud_scheduler_job" "publisher" {
  count            = var.runtime == null ? 0 : 1
  project          = var.project_id
  name             = "jayburd-dev-acquisition-publisher"
  region           = var.region
  schedule         = "* * * * *"
  time_zone        = "Etc/UTC"
  paused           = !var.scheduler_enabled
  attempt_deadline = "90s"
  retry_config {
    retry_count = 0
  }
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
  depends_on = [google_project_service.worker, google_cloud_run_v2_service_iam_binding.worker]
}
