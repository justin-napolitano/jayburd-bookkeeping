// Values/versions remain Infisical-owned; Terraform manages empty destinations only.
resource "google_project_service" "resources" {
  for_each           = toset(["secretmanager.googleapis.com", "cloudkms.googleapis.com", "cloudtasks.googleapis.com", "cloudscheduler.googleapis.com"])
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}
resource "google_secret_manager_secret" "provider" {
  for_each  = toset(concat(["client-id", "client-secret"], [for i in range(8) : format("slot-%02d", i)]))
  project   = var.project_id
  secret_id = "jayburd-prod-bank-sync-${each.value}"
  labels    = local.labels
  replication {
    auto {}
  }
  lifecycle { prevent_destroy = true }
  depends_on = [google_project_service.resources]
}
resource "google_secret_manager_secret_iam_member" "reader" {
  for_each  = { for pair in setproduct(["api", "sync"], keys(google_secret_manager_secret.provider)) : "${pair[0]}-${pair[1]}" => pair }
  project   = var.project_id
  secret_id = google_secret_manager_secret.provider[each.value[1]].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.component[each.value[0]].email}"
}
resource "google_secret_manager_secret_iam_member" "webhook_reader" {
  for_each  = toset(["client-id", "client-secret"])
  project   = var.project_id
  secret_id = google_secret_manager_secret.provider[each.value].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.component["webhook"].email}"
}
// Delivery can add/list versions but cannot access secret payloads or administer containers.
resource "google_project_iam_custom_role" "secret_sync" {
  project     = var.project_id
  role_id     = "jayburdProdProviderSecretSync"
  title       = "Jayburd Production precreated secret delivery"
  permissions = ["secretmanager.secrets.get", "secretmanager.versions.add", "secretmanager.versions.list"]
}
resource "google_secret_manager_secret_iam_member" "secret_sync" {
  for_each  = google_secret_manager_secret.provider
  project   = var.project_id
  secret_id = each.value.secret_id
  role      = google_project_iam_custom_role.secret_sync.name
  member    = "serviceAccount:${google_service_account.component["secret-sync"].email}"
}
resource "google_project_iam_custom_role" "version_metadata" {
  project     = var.project_id
  role_id     = "jayburdProdProviderVersionMetadata"
  title       = "Jayburd Production slot version metadata"
  permissions = ["secretmanager.versions.list"]
}
resource "google_secret_manager_secret_iam_member" "version_metadata" {
  for_each  = toset([for i in range(8) : format("slot-%02d", i)])
  project   = var.project_id
  secret_id = google_secret_manager_secret.provider[each.value].secret_id
  role      = google_project_iam_custom_role.version_metadata.name
  member    = "serviceAccount:${google_service_account.component["api"].email}"
}
// Separate buckets contain no Sandbox data and expose no public IAM.
resource "google_storage_bucket" "private" {
  for_each                    = toset(["acquisition-evidence", "credential-registry"])
  project                     = var.project_id
  name                        = "${var.project_id}-jayburd-prod-${each.key}"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false
  labels                      = local.labels
  versioning { enabled = true }
  lifecycle { prevent_destroy = true }
}
resource "google_kms_key_ring" "provider" {
  project    = var.project_id
  name       = "jayburd-prod"
  location   = var.region
  depends_on = [google_project_service.resources]
}
resource "google_kms_crypto_key" "provider" {
  for_each        = toset(["bank-credentials", "hosted-link"])
  name            = each.value
  key_ring        = google_kms_key_ring.provider.id
  purpose         = "ENCRYPT_DECRYPT"
  rotation_period = "7776000s"
  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = "SOFTWARE"
  }
  lifecycle { prevent_destroy = true }
}
resource "google_kms_crypto_key_iam_member" "api" {
  for_each      = google_kms_crypto_key.provider
  crypto_key_id = each.value.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${google_service_account.component["api"].email}"
}
resource "google_kms_crypto_key_iam_member" "sync" {
  crypto_key_id = google_kms_crypto_key.provider["bank-credentials"].id
  role          = "roles/cloudkms.cryptoKeyDecrypter"
  member        = "serviceAccount:${google_service_account.component["sync"].email}"
}

resource "google_project_iam_custom_role" "evidence_raw" {
  project     = var.project_id
  role_id     = "jayburdProdAcquisitionRawEvidence"
  title       = "Jayburd bounded raw evidence objects"
  permissions = ["storage.objects.create", "storage.objects.get", "storage.objects.delete"]
}
resource "google_storage_bucket_iam_member" "evidence_raw" {
  bucket = google_storage_bucket.private["acquisition-evidence"].name
  role   = google_project_iam_custom_role.evidence_raw.name
  member = "serviceAccount:${google_service_account.component["sync"].email}"
  condition {
    title      = "raw_evidence_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["acquisition-evidence"].name}/objects/raw/')"
  }
}
resource "google_project_iam_custom_role" "evidence_tombstone" {
  project     = var.project_id
  role_id     = "jayburdProdAcquisitionTombstone"
  title       = "Jayburd immutable evidence tombstones"
  permissions = ["storage.objects.create", "storage.objects.get"]
}
resource "google_storage_bucket_iam_member" "evidence_tombstone" {
  bucket = google_storage_bucket.private["acquisition-evidence"].name
  role   = google_project_iam_custom_role.evidence_tombstone.name
  member = "serviceAccount:${google_service_account.component["sync"].email}"
  condition {
    title      = "immutable_tombstone_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["acquisition-evidence"].name}/objects/deleted/')"
  }
}
resource "google_project_iam_custom_role" "registry_writer" {
  project     = var.project_id
  role_id     = "jayburdProdCredentialRegistryWriter"
  title       = "Jayburd immutable credential mapping writes"
  permissions = ["storage.objects.create", "storage.objects.get"]
}
resource "google_project_iam_custom_role" "registry_reader" {
  project     = var.project_id
  role_id     = "jayburdProdCredentialRegistryReader"
  title       = "Jayburd exact credential mapping reads"
  permissions = ["storage.objects.get"]
}
resource "google_storage_bucket_iam_member" "registry_writer" {
  bucket = google_storage_bucket.private["credential-registry"].name
  role   = google_project_iam_custom_role.registry_writer.name
  member = "serviceAccount:${google_service_account.component["api"].email}"
  condition {
    title      = "provider_registry_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["credential-registry"].name}/objects/provider-credentials-v1/')"
  }
}
resource "google_storage_bucket_iam_member" "registry_reader" {
  bucket = google_storage_bucket.private["credential-registry"].name
  role   = google_project_iam_custom_role.registry_reader.name
  member = "serviceAccount:${google_service_account.component["sync"].email}"
  condition {
    title      = "provider_registry_objects"
    expression = "resource.name.startsWith('projects/_/buckets/${google_storage_bucket.private["credential-registry"].name}/objects/provider-credentials-v1/')"
  }
}

