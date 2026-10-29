// A new identity isolates delivery from API, workers and the historical native sync connection.
resource "google_service_account" "delivery" {
  project      = var.project_id
  account_id   = "jayburd-prod-delivery"
  display_name = "Jayburd Production exact-key static delivery"
  lifecycle { prevent_destroy = true }
}
resource "google_storage_bucket_iam_member" "registry" {
  count  = var.delivery == null ? 0 : 1
  bucket = "${var.project_id}-jayburd-prod-credential-registry"
  role   = "projects/${var.project_id}/roles/jayburdProdCredentialRegistryWriter"
  member = "serviceAccount:${google_service_account.delivery.email}"
  condition {
    title      = "production-static-delivery-only"
    expression = "resource.name.startsWith('projects/_/buckets/${var.project_id}-jayburd-prod-credential-registry/objects/production-static-delivery-v1/')"
  }
}
// Only the selected static destination may receive a version, and only in execute mode.
resource "google_secret_manager_secret_iam_member" "add_version" {
  count     = var.delivery == null ? 0 : (var.delivery.mode == "execute" ? 1 : 0)
  project   = var.project_id
  secret_id = "jayburd-prod-bank-sync-${var.delivery.key == "PLAID_CLIENT_ID" ? "client-id" : "client-secret"}"
  role      = "roles/secretmanager.secretVersionAdder"
  member    = "serviceAccount:${google_service_account.delivery.email}"
}
// No HTTP listener, Scheduler, invocation grant, impersonation grant or implicit execution.
resource "google_cloud_run_v2_job" "delivery" {
  count               = var.delivery == null ? 0 : 1
  project             = var.project_id
  location            = var.region
  name                = "jayburd-prod-static-delivery"
  deletion_protection = true
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = google_service_account.delivery.email
      timeout         = "60s"
      max_retries     = 0
      containers {
        image   = var.delivery.image
        command = ["/deliver"]
        args    = concat(["-${var.delivery.mode}", "-environment=production", "-approval-expires-at=${var.delivery.expires_at}", "-key=${var.delivery.key}", "-project-id=${var.project_id}", "-project-number=${var.project_number}", "-infisical-project=${var.delivery.infisical_project}", "-infisical-identity=${var.delivery.infisical_identity}", "-source-version=${var.delivery.source_version}"], var.delivery.mode == "execute" ? ["-source-id=${var.delivery.source_id}"] : [])
        resources { limits = { cpu = "1", memory = "512Mi" } }
      }
    }
  }
  depends_on = [google_storage_bucket_iam_member.registry, google_secret_manager_secret_iam_member.add_version]
  lifecycle { prevent_destroy = true }
}
