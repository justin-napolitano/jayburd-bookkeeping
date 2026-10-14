locals {
  labels = { application = "jayburd", environment = "dev", managed_by = "terraform" }
}
# This identity is distinct from the Infisical native app-connection and all API/worker identities.
resource "google_service_account" "delivery" {
  project      = var.project_id
  account_id   = "jayburd-dev-delivery"
  display_name = "Jayburd development canary delivery"
}
# Canary values are owner-injected into Infisical; Terraform owns only an empty destination.
resource "google_secret_manager_secret" "canary" {
  project   = var.project_id
  secret_id = "jayburd-dev-bank-sync-canary"
  labels    = local.labels
  replication {
    auto {}
  }
  lifecycle { prevent_destroy = true }
}
resource "google_secret_manager_secret_iam_member" "add_canary" {
  count     = var.canary == null ? 0 : (var.canary.mode == "execute" ? 1 : 0)
  project   = var.project_id
  secret_id = google_secret_manager_secret.canary.secret_id
  role      = "roles/secretmanager.secretVersionAdder"
  member    = "serviceAccount:${google_service_account.delivery.email}"
}
resource "google_storage_bucket" "registry" {
  project                     = var.project_id
  name                        = "${var.project_id}-jayburd-dev-delivery-registry"
  location                    = var.region
  labels                      = local.labels
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false
  soft_delete_policy { retention_duration_seconds = 0 }
  lifecycle { prevent_destroy = true }
}
resource "google_project_iam_custom_role" "registry" {
  project     = var.project_id
  role_id     = "jayburdDeliveryRegistry"
  title       = "Jayburd immutable delivery receipt writer"
  permissions = ["storage.objects.create", "storage.objects.get"]
}
resource "google_storage_bucket_iam_member" "registry" {
  bucket = google_storage_bucket.registry.name
  role   = google_project_iam_custom_role.registry.name
  member = "serviceAccount:${google_service_account.delivery.email}"
  condition {
    title      = "delivery-prefix-only"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.registry.name}/objects/delivery-v1/')"
  }
}
# Creating this definition does not execute it. No Scheduler or runtime token-creator grant is installed.
resource "google_cloud_run_v2_job" "canary" {
  count               = var.canary == null ? 0 : 1
  project             = var.project_id
  name                = "jayburd-dev-delivery-canary"
  location            = var.region
  labels              = local.labels
  deletion_protection = true
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = google_service_account.delivery.email
      max_retries     = 0
      timeout         = "60s"
      containers {
        image   = var.canary.image
        command = ["/deliver"]
        args    = concat(["-${var.canary.mode}", "-project-id=${var.project_id}", "-project-number=${var.project_number}", "-infisical-project=${var.canary.infisical_project}", "-infisical-identity=${var.canary.infisical_identity}", "-source-version=${var.canary.source_version}"], var.canary.mode == "execute" ? ["-source-id=${var.canary.source_id}"] : [])
        resources { limits = { cpu = "1", memory = "512Mi" } }
      }
    }
  }
  depends_on = [google_secret_manager_secret_iam_member.add_canary, google_storage_bucket_iam_member.registry]
  lifecycle { prevent_destroy = true }
}
# The workflow caller may invoke the canary job but cannot impersonate the runtime or override its arguments.
resource "google_service_account" "runner" {
  project      = var.project_id
  account_id   = "jayburd-dev-delivery-runner"
  display_name = "Jayburd owner canary invocation"
}
resource "google_project_iam_custom_role" "invoke" {
  project     = var.project_id
  role_id     = "jayburdCanaryInvoker"
  title       = "Jayburd canary job invocation only"
  permissions = ["run.jobs.get", "run.jobs.run"]
}
resource "google_cloud_run_v2_job_iam_member" "invoke" {
  count    = var.enable_github_canary ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_job.canary[0].name
  role     = google_project_iam_custom_role.invoke.name
  member   = "serviceAccount:${google_service_account.runner.email}"
}
resource "google_iam_workload_identity_pool" "canary" {
  project                   = var.project_id
  workload_identity_pool_id = "jayburd-dev-delivery"
  disabled                  = !var.enable_github_canary
}
resource "google_iam_workload_identity_pool_provider" "canary" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.canary.workload_identity_pool_id
  workload_identity_pool_provider_id = "owner-canary"
  disabled                           = !var.enable_github_canary
  attribute_mapping                  = { "google.subject" = "assertion.sub", "attribute.repository_id" = "assertion.repository_id" }
  attribute_condition = join(" && ", [for claim, value in {
    repository_id = "1414515318", repository_owner_id = "55423093", actor_id = "55423093",
    ref           = "refs/heads/main", event_name = "workflow_dispatch",
    workflow_ref  = "justin-napolitano/jayburd-bookkeeping/.github/workflows/run-delivery-canary.yml@refs/heads/main"
  } : "assertion.${claim} == '${value}'"])
  oidc { issuer_uri = "https://token.actions.githubusercontent.com" }
}
resource "google_service_account_iam_member" "runner_federation" {
  service_account_id = google_service_account.runner.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${google_iam_workload_identity_pool.canary.workload_identity_pool_id}/attribute.repository_id/1414515318"
}
