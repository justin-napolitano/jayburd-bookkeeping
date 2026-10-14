locals {
  # Stable GitHub IDs prevent repository-name reuse from inheriting cloud access.
  repository_id = "1414515318"
  owner_id      = "55423093"
  repository    = "justin-napolitano/jayburd-bookkeeping"
  labels        = { application = "jayburd", environment = "dev", managed_by = "terraform" }
  identities    = toset(["builder", "deployer", "migrator", "api", "frontend", "publisher", "consumer", "scheduler", "task-invoker"])
  foundation_apis = toset([
    "artifactregistry.googleapis.com", "cloudbilling.googleapis.com", "cloudresourcemanager.googleapis.com",
    "billingbudgets.googleapis.com", "iam.googleapis.com", "run.googleapis.com",
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
    workflow_ref        = "${local.repository}/.github/workflows/build-images.yml@refs/heads/main"
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
  for_each                    = toset(["foundation", "application", "runtime", "workers"])
  project                     = var.project_id
  name                        = "${var.project_id}-jayburd-dev-${each.key}-state"
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
  account_id   = "jayburd-dev-${each.key}"
  display_name = "Jayburd dev ${each.key}"
  depends_on   = [google_project_service.foundation]
}

resource "google_artifact_registry_repository" "images" {
  project       = var.project_id
  location      = var.region
  repository_id = "jayburd-dev"
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
resource "google_storage_bucket_iam_member" "application_state" {
  bucket = google_storage_bucket.state["runtime"].name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.component["deployer"].email}"
}

resource "google_iam_workload_identity_pool" "github" {
  project                   = var.project_id
  workload_identity_pool_id = "jayburd-dev-github"
  display_name              = "Jayburd dev GitHub builds"
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

# Alerts are advisory and scoped to this project, not a hard spending limit.
resource "google_billing_budget" "development" {
  billing_account = var.billing_account_id
  display_name    = "Jayburd development monthly target"
  budget_filter {
    projects               = ["projects/${var.project_number}"]
    calendar_period        = "MONTH"
    credit_types_treatment = "EXCLUDE_ALL_CREDITS"
  }
  amount {
    specified_amount {
      currency_code = "USD"
      units         = "50"
    }
  }
  threshold_rules { threshold_percent = 0.5 }
  threshold_rules { threshold_percent = 0.8 }
  threshold_rules { threshold_percent = 1.0 }
  depends_on = [google_project_service.foundation]
}

# A separate pool is essential: repository_id alone must not let build tokens impersonate the deployer.
resource "google_iam_workload_identity_pool" "deployment" {
  project                   = var.project_id
  workload_identity_pool_id = "jayburd-dev-deploy"
  disabled                  = !var.enable_github_deploy_identity
  depends_on                = [google_project_service.foundation]
}
resource "google_iam_workload_identity_pool_provider" "deployment" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.deployment.workload_identity_pool_id
  workload_identity_pool_provider_id = "owner-deploy"
  disabled                           = !var.enable_github_deploy_identity
  attribute_mapping                  = { "google.subject" = "assertion.sub", "attribute.repository_id" = "assertion.repository_id" }
  attribute_condition                = join(" && ", [for claim, value in merge(local.build_claims, { workflow_ref = "${local.repository}/.github/workflows/deploy-dev.yml@refs/heads/main" }) : "assertion.${claim} == '${value}'"])
  oidc { issuer_uri = "https://token.actions.githubusercontent.com" }
}
resource "google_service_account_iam_member" "github_deployer" {
  service_account_id = google_service_account.component["deployer"].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${google_iam_workload_identity_pool.deployment.workload_identity_pool_id}/attribute.repository_id/${local.repository_id}"
}
resource "google_project_iam_member" "run_deployer" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.component["deployer"].email}"
}
resource "google_service_account_iam_member" "runtime_act_as" {
  for_each           = toset(["api", "frontend", "migrator"])
  service_account_id = google_service_account.component[each.key].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.component["deployer"].email}"
}
resource "google_artifact_registry_repository_iam_member" "deployer" {
  project    = var.project_id
  location   = var.region
  repository = google_artifact_registry_repository.images.name
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.component["deployer"].email}"
}
# Revocation verification needs user lookup, not identity provisioning or modification.
resource "google_project_iam_custom_role" "identity_lookup" {
  project     = var.project_id
  role_id     = "jayburdIdentityLookup"
  title       = "Jayburd identity revocation lookup"
  permissions = ["firebaseauth.users.get"]
  depends_on  = [google_project_service.foundation]
}
resource "google_project_iam_member" "identity_lookup" {
  project = var.project_id
  role    = google_project_iam_custom_role.identity_lookup.name
  member  = "serviceAccount:${google_service_account.component["api"].email}"
}
