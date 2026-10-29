mock_provider "google" {}
variables {
  project_id     = "jayburd-bookkeeping-dev"
  project_number = "9088794002"
}
run "shared_database_and_scoped_resources" {
  command = plan
  assert {
    condition     = google_sql_database.ledger.instance == "jayburd-dev" && google_sql_database.ledger.name == "bookkeeping_prod"
    error_message = "Use only the separate database on the existing instance."
  }
  assert {
    condition     = length(google_sql_user.component) == 5 && alltrue([for u in google_sql_user.component : u.instance == "jayburd-dev" && u.type == "CLOUD_IAM_SERVICE_ACCOUNT"])
    error_message = "Only dedicated Production IAM SQL logins."
  }
  assert {
    condition     = google_iam_workload_identity_pool.github.disabled && google_iam_workload_identity_pool_provider.build.disabled
    error_message = "Build federation must remain disabled pending exact review."
  }
  assert {
    condition     = length(google_secret_manager_secret.provider) == 10 && length(google_secret_manager_secret_iam_member.webhook_reader) == 2 && length(google_secret_manager_secret_iam_member.reader) == 20
    error_message = "Bounded static/slot destinations and component-specific readers only."
  }
  assert {
    condition     = alltrue([for b in google_storage_bucket.private : b.uniform_bucket_level_access && b.public_access_prevention == "enforced" && !b.force_destroy])
    error_message = "Private protected Production buckets only."
  }
  assert {
    condition     = google_kms_crypto_key_iam_member.sync.role == "roles/cloudkms.cryptoKeyDecrypter" && length(google_kms_crypto_key_iam_member.api) == 2
    error_message = "Sync cannot use the hosted-link key or encrypt bank credentials."
  }
  assert {
    condition     = toset(google_project_iam_custom_role.secret_sync.permissions) == toset(["secretmanager.secrets.get", "secretmanager.versions.add", "secretmanager.versions.list"]) && length(google_secret_manager_secret_iam_member.secret_sync) == 10
    error_message = "Secret delivery cannot read payloads or administer destinations."
  }
}
run "reject_other_project" {
  command = plan
  variables { project_id = "fictional-production" }
  expect_failures = [var.project_id]
}
run "reject_mixed_numeric_project" {
  command = plan
  variables { project_number = "123456789012" }
  expect_failures = [var.project_number]
}
