locals {
  principal = "serviceAccount:jayburd-dev-webhook-deployer@${var.project_id}.iam.gserviceaccount.com"
  claims = {
    repository_id = "1414515318", repository_owner_id = "55423093", actor_id = "55423093",
    ref           = "refs/heads/main", event_name = "workflow_dispatch",
    workflow_ref  = "justin-napolitano/jayburd-bookkeeping/.github/workflows/deploy-webhook.yml@refs/heads/main",
    workflow_sha  = var.workflow_sha
  }
}
# All resources are opt-in owner bootstrap. This identity cannot create services or change public IAM.
resource "google_service_account" "deployer" {
  count        = var.enabled ? 1 : 0
  project      = var.project_id
  account_id   = "jayburd-dev-webhook-deployer"
  display_name = "Jayburd exact reviewed webhook update only"
  lifecycle { prevent_destroy = true }
}
resource "google_project_iam_custom_role" "update" {
  count       = var.enabled ? 1 : 0
  project     = var.project_id
  role_id     = "jayburdWebhookRelease"
  title       = "Jayburd existing webhook update only"
  stage       = "GA"
  permissions = ["run.services.get", "run.services.getIamPolicy", "run.services.update"]
}
resource "google_cloud_run_v2_service_iam_member" "update" {
  count    = var.enabled ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = "jayburd-dev-webhook"
  role     = google_project_iam_custom_role.update[0].name
  member   = local.principal
}
# Terraform polls regional operations; collector verifies the actual custom role definition.
# Both permissions are read-only project metadata, with no service mutation or IAM administration.
resource "google_project_iam_custom_role" "metadata" {
  count       = var.enabled ? 1 : 0
  project     = var.project_id
  role_id     = "jayburdWebhookReleaseMetadata"
  title       = "Jayburd release operation and role readback"
  stage       = "GA"
  permissions = ["run.operations.get", "iam.roles.get"]
}
resource "google_project_iam_member" "metadata" {
  count   = var.enabled ? 1 : 0
  project = var.project_id
  role    = google_project_iam_custom_role.metadata[0].name
  member  = local.principal
}
resource "google_service_account_iam_member" "act_as" {
  count              = var.enabled ? 1 : 0
  service_account_id = "projects/${var.project_id}/serviceAccounts/jayburd-dev-webhook@${var.project_id}.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountUser"
  member             = local.principal
}
# Existing owner-created runtime state only; no other bucket, bootstrap state or secret access.
resource "google_storage_bucket_iam_member" "state" {
  count  = var.enabled ? 1 : 0
  bucket = "${var.project_id}-jayburd-dev-webhook-runtime-state"
  role   = "roles/storage.objectAdmin"
  member = local.principal
}
# Cloud Run deployment requires image reads; this covers the existing repository, not writes or image-path IAM.
resource "google_artifact_registry_repository_iam_member" "images" {
  count      = var.enabled ? 1 : 0
  project    = var.project_id
  location   = var.region
  repository = "jayburd-dev"
  role       = "roles/artifactregistry.reader"
  member     = local.principal
}
resource "google_iam_workload_identity_pool" "deploy" {
  count                     = var.enabled ? 1 : 0
  project                   = var.project_id
  workload_identity_pool_id = "jayburd-dev-webhook-deploy"
  disabled                  = !var.federation_active
}
resource "google_iam_workload_identity_pool_provider" "deploy" {
  count                              = var.enabled ? 1 : 0
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.deploy[0].workload_identity_pool_id
  workload_identity_pool_provider_id = "owner-webhook-deploy"
  disabled                           = !var.federation_active
  attribute_mapping                  = merge({ "google.subject" = "assertion.sub" }, { for claim, value in local.claims : "attribute.${claim}" => "assertion.${claim}" })
  attribute_condition                = join(" && ", [for claim, value in local.claims : "assertion.${claim} == '${value}'"])
  oidc { issuer_uri = "https://token.actions.githubusercontent.com" }
}
resource "google_service_account_iam_member" "federation" {
  count              = var.enabled ? 1 : 0
  service_account_id = google_service_account.deployer[0].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${google_iam_workload_identity_pool.deploy[0].workload_identity_pool_id}/attribute.repository_id/1414515318"
}
