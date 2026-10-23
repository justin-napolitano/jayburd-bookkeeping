locals {
  labels = { application = "jayburd", environment = "dev", managed_by = "terraform" }
  login  = "jayburd-dev-verifier@${var.project_id}.iam.gserviceaccount.com"
}

# No key, impersonation, provider secret, public service or business enrollment is created.
resource "google_service_account" "verifier" {
  count        = var.enabled ? 1 : 0
  project      = var.project_id
  account_id   = "jayburd-dev-verifier"
  display_name = "Jayburd finite synthetic comparator"
  lifecycle { prevent_destroy = true }
}
resource "google_project_iam_member" "sql" {
  for_each   = var.enabled ? toset(["roles/cloudsql.client", "roles/cloudsql.instanceUser"]) : toset([])
  project    = var.project_id
  role       = each.key
  member     = "serviceAccount:${local.login}"
  depends_on = [google_service_account.verifier]
}
resource "google_sql_user" "verifier" {
  count      = var.enabled ? 1 : 0
  project    = var.project_id
  instance   = "jayburd-dev"
  name       = trimsuffix(local.login, ".gserviceaccount.com")
  type       = "CLOUD_IAM_SERVICE_ACCOUNT"
  depends_on = [google_project_iam_member.sql]
  lifecycle { prevent_destroy = true }
}

# Synthetic input channel only; no app/provider evidence or token bucket is reused.
resource "google_storage_bucket" "inputs" {
  count                       = var.enabled ? 1 : 0
  project                     = var.project_id
  name                        = "${var.project_id}-jayburd-verifier-inputs"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false
  labels                      = local.labels
  versioning { enabled = false }
  soft_delete_policy { retention_duration_seconds = 0 }
  lifecycle_rule {
    condition { age = 1 }
    action { type = "Delete" }
  }
  lifecycle { prevent_destroy = true }
}
resource "google_storage_bucket_iam_member" "input_read" {
  count      = var.enabled ? 1 : 0
  bucket     = google_storage_bucket.inputs[0].name
  role       = "roles/storage.objectViewer"
  member     = "serviceAccount:${local.login}"
  depends_on = [google_service_account.verifier]
}

# Zero retries and one immutable input directory; no execution occurs during apply.
resource "google_cloud_run_v2_job" "compare" {
  count               = var.run == null ? 0 : 1
  project             = var.project_id
  name                = "jayburd-dev-sandbox-verifier"
  location            = var.region
  deletion_protection = false
  labels              = local.labels
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = local.login
      timeout         = "120s"
      max_retries     = 0
      containers {
        image   = var.run.image
        command = ["/sandbox-verify"]
        args    = ["-action=compare", "-manifest=/verification/manifest.json"]
        resources { limits = { cpu = "1", memory = "512Mi" } }
        volume_mounts {
          name       = "synthetic-inputs"
          mount_path = "/verification"
        }
      }
      volumes {
        name = "synthetic-inputs"
        gcs {
          bucket        = google_storage_bucket.inputs[0].name
          read_only     = true
          mount_options = ["only-dir=${var.run.run_id}", "file-mode=0444", "dir-mode=0555"]
        }
      }
    }
  }
  depends_on = [google_sql_user.verifier, google_storage_bucket_iam_member.input_read]
}
