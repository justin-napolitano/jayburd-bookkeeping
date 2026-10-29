output "instance_connection_name" {
  description = "Nonsecret connector target, never a password-bearing URL."
  value       = "jayburd-bookkeeping-dev:us-central1:jayburd-dev"
}
output "database_name" {
  description = "Database requiring owner bootstrap before migrations/runtime readiness."
  value       = google_sql_database.ledger.name
}

# Identifiers only; no credentials, Terraform state or private data in outputs.
output "service_account_names" {
  value = { for k, account in google_service_account.component : k => account.email }
}
output "state_bucket_names" {
  value = { for k, bucket in google_storage_bucket.state : k => bucket.name }
}
output "image_repository" {
  value = "${var.region}-docker.pkg.dev/${var.project_id}/jayburd-prod"
}
output "build_identity_provider" {
  value = google_iam_workload_identity_pool_provider.build.name
}
