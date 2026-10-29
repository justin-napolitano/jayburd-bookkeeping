locals {
  # Stable GitHub IDs prevent repository-name reuse from inheriting cloud access.
  repository_id = "1414515318"
  owner_id      = "55423093"
  repository    = "justin-napolitano/jayburd-bookkeeping"
  labels        = { application = "jayburd", environment = "production", managed_by = "terraform" }
  identities    = toset(["builder", "migrator", "api", "frontend", "publisher", "sync", "webhook", "scheduler", "task-invoker"])
  sql_accounts  = toset(["api", "migrator", "sync", "publisher", "webhook"])
  foundation_apis = toset([
    "artifactregistry.googleapis.com", "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com", "run.googleapis.com",
    "iamcredentials.googleapis.com", "sts.googleapis.com",
    "serviceusage.googleapis.com", "storage.googleapis.com"
  ])
  # Both human initiation and exact workflow/ref are required. No PR or fork token qualifies.
  build_claims = {
    repository_id       = local.repository_id
    repository_owner_id = local.owner_id
    actor_id            = local.owner_id
    ref                 = "refs/heads/main"
    event_name          = "workflow_dispatch"
    workflow_ref        = "${local.repository}/.github/workflows/build-production-images.yml@refs/heads/main"
  }
}

resource "google_project_service" "foundation" {
  for_each           = local.foundation_apis
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

# Separate buckets prevent an application deployer from changing bootstrap state.
resource "google_storage_bucket" "state" {
  for_each                    = toset(["foundation", "migration", "runtime", "worker-runtime", "webhook-runtime"])
  project                     = var.project_id
  name                        = "${var.project_id}-jayburd-prod-${each.key}-state"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false
  labels                      = local.labels
  versioning { enabled = true }
  lifecycle { prevent_destroy = true }
  depends_on = [google_project_service.foundation]
}

resource "google_service_account" "component" {
  for_each     = local.identities
  project      = var.project_id
  account_id   = "jayburd-prod-${each.key}"
  display_name = "Jayburd Production ${each.key}"
  depends_on   = [google_project_service.foundation]
}

resource "google_artifact_registry_repository" "images" {
  project       = var.project_id
  location      = var.region
  repository_id = "jayburd-prod"
  format        = "DOCKER"
  labels        = local.labels
  docker_config { immutable_tags = true }
  cleanup_policies {
    id     = "delete-old-untagged"
    action = "DELETE"
    condition {
      tag_state  = "UNTAGGED"
      older_than = "604800s"
    }
  }
  lifecycle { prevent_destroy = true }
  depends_on = [google_project_service.foundation]
}

# Builder may upload images but cannot read state, deploy, impersonate runtimes or read secrets.
resource "google_artifact_registry_repository_iam_member" "builder" {
  project    = var.project_id
  location   = google_artifact_registry_repository.images.location
  repository = google_artifact_registry_repository.images.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.component["builder"].email}"
}
resource "google_iam_workload_identity_pool" "github" {
  project                   = var.project_id
  workload_identity_pool_id = "jayburd-prod-github"
  display_name              = "Jayburd Production GitHub builds"
  disabled                  = !var.enable_github_build_identity
  depends_on                = [google_project_service.foundation]
}
resource "google_iam_workload_identity_pool_provider" "build" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "owner-build"
  disabled                           = !var.enable_github_build_identity
  attribute_mapping = merge(
    { "google.subject" = "assertion.sub" },
    { for claim, value in local.build_claims : "attribute.${claim}" => "assertion.${claim}" }
  )
  attribute_condition = join(" && ", [for claim, value in local.build_claims : "assertion.${claim} == '${value}'"])
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
    # Empty allowed_audiences enforces Google's canonical provider-resource audience.
  }
}
resource "google_service_account_iam_member" "github_builder" {
  service_account_id = google_service_account.component["builder"].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${google_iam_workload_identity_pool.github.workload_identity_pool_id}/attribute.repository_id/${local.repository_id}"
}


resource "google_project_service" "sql" {
  project            = var.project_id
  service            = "sqladmin.googleapis.com"
  disable_on_destroy = false
}
resource "google_sql_database_instance" "ledger" {
  project             = var.project_id
  name                = "jayburd-prod"
  database_version    = "POSTGRES_17"
  region              = var.region
  deletion_protection = true
  settings {
    edition                     = "ENTERPRISE"
    tier                        = "db-f1-micro"
    availability_type           = "ZONAL"
    disk_type                   = "PD_SSD"
    disk_size                   = 10
    disk_autoresize             = true
    disk_autoresize_limit       = 20
    deletion_protection_enabled = true
    user_labels                 = local.labels
    connector_enforcement       = "REQUIRED"
    ip_configuration {
      ipv4_enabled = true
      ssl_mode     = "ENCRYPTED_ONLY"
      # No authorized networks: clients must use the authenticated connector.
    }
    backup_configuration {
      enabled                        = true
      start_time                     = "06:00"
      location                       = "us"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 8
        retention_unit   = "COUNT"
      }
    }
    database_flags {
      name  = "cloudsql.iam_authentication"
      value = "on"
    }
    database_flags {
      name  = "max_connections"
      value = "50"
    }
    database_flags {
      name  = "log_statement"
      value = "none"
    }
    database_flags {
      name  = "log_min_error_statement"
      value = "panic"
    }
  }
  lifecycle { prevent_destroy = true }
  depends_on = [google_project_service.sql]
}
resource "google_sql_database" "ledger" {
  project  = var.project_id
  name     = "bookkeeping"
  instance = google_sql_database_instance.ledger.name
  lifecycle { prevent_destroy = true }
}
# IAM login authorizes authentication only. Owner-operated SQL bootstrap assigns SQL grants.
resource "google_sql_user" "component" {
  for_each = local.sql_accounts
  project  = var.project_id
  instance = google_sql_database_instance.ledger.name
  name     = trimsuffix(google_service_account.component[each.key].email, ".gserviceaccount.com")
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
  lifecycle { prevent_destroy = true }
}
resource "google_project_iam_member" "sql_client" {
  for_each = local.sql_accounts
  project  = var.project_id
  role     = "roles/cloudsql.client"
  member   = "serviceAccount:${google_service_account.component[each.key].email}"
}
resource "google_project_iam_member" "sql_login" {
  for_each = local.sql_accounts
  project  = var.project_id
  role     = "roles/cloudsql.instanceUser"
  member   = "serviceAccount:${google_service_account.component[each.key].email}"
}

# Explicit operator-only elevation; no runtime membership is changed by this resource.
resource "google_sql_user" "bootstrap_operator" {
  count          = var.bootstrap_operator_email == null ? 0 : 1
  project        = var.project_id
  instance       = google_sql_database_instance.ledger.name
  name           = var.bootstrap_operator_email
  type           = "CLOUD_IAM_USER"
  database_roles = ["cloudsqlsuperuser"]
}
resource "google_project_iam_member" "bootstrap_operator" {
  for_each = var.bootstrap_operator_email == null ? toset([]) : toset(["roles/cloudsql.client", "roles/cloudsql.instanceUser"])
  project  = var.project_id
  role     = each.value
  member   = "user:${var.bootstrap_operator_email}"
}
