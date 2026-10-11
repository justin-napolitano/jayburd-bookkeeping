locals {
  # Foundation owns these accounts. No service-account creation or remote state lookup here.
  sql_accounts = toset(["api", "migrator"])
  labels       = { application = "jayburd", environment = "dev", managed_by = "terraform" }
}
resource "google_project_service" "sql" {
  project            = var.project_id
  service            = "sqladmin.googleapis.com"
  disable_on_destroy = false
}
resource "google_sql_database_instance" "ledger" {
  project             = var.project_id
  name                = "jayburd-dev"
  database_version    = "POSTGRES_17"
  region              = var.region
  deletion_protection = true
  settings {
    edition                     = "ENTERPRISE"
    tier                        = "db-g1-small"
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
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 7
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
  name     = "jayburd-dev-${each.key}@${var.project_id}.iam"
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
  lifecycle { prevent_destroy = true }
}
resource "google_project_iam_member" "sql_client" {
  for_each = local.sql_accounts
  project  = var.project_id
  role     = "roles/cloudsql.client"
  member   = "serviceAccount:jayburd-dev-${each.key}@${var.project_id}.iam.gserviceaccount.com"
}
resource "google_project_iam_member" "sql_login" {
  for_each = local.sql_accounts
  project  = var.project_id
  role     = "roles/cloudsql.instanceUser"
  member   = "serviceAccount:jayburd-dev-${each.key}@${var.project_id}.iam.gserviceaccount.com"
}
