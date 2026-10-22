locals {
  labels    = { application = "jayburd", environment = "dev", managed_by = "terraform" }
  secrets   = var.enabled ? toset(["jayburd-dev-bank-sync-client-id", "jayburd-dev-bank-sync-client-secret"]) : toset([])
  sql_roles = var.enabled ? toset(["roles/cloudsql.client", "roles/cloudsql.instanceUser"]) : toset([])
}

# Bootstrap owns only the dedicated notification identity; secret containers already belong to workers.
resource "google_service_account" "webhook" {
  count        = var.enabled ? 1 : 0
  project      = var.project_id
  account_id   = "jayburd-dev-webhook"
  display_name = "Jayburd dev verified Transactions notification ingress"
  lifecycle { prevent_destroy = true }
}
resource "google_project_iam_member" "sql" {
  for_each = local.sql_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.webhook[0].email}"
}
resource "google_sql_user" "webhook" {
  count    = var.enabled ? 1 : 0
  project  = var.project_id
  instance = "jayburd-dev"
  name     = trimsuffix(google_service_account.webhook[0].email, ".gserviceaccount.com")
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
  lifecycle { prevent_destroy = true }
}
# Two secret-level reads only; no bank token slots, version writes, KMS, bucket or queue authority.
resource "google_secret_manager_secret_iam_member" "client" {
  for_each  = local.secrets
  project   = var.project_id
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.webhook[0].email}"
}
# Apply only after the separate runtime exists and public exposure has independent approval.
# This role binding is authoritative only for this one service; private services are absent from this root.
resource "google_cloud_run_v2_service_iam_binding" "public" {
  count    = var.enabled && var.public_invocation_approved ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = "jayburd-dev-webhook"
  role     = "roles/run.invoker"
  members  = ["allUsers"]
}
