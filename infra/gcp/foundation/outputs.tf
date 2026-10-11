# These outputs are identifiers only; agents must not retrieve Terraform state or plans.
output "state_bucket_names" {
  description = "Separate bootstrap/application state destinations for owner-operated backend setup."
  value       = { for key, bucket in google_storage_bucket.state : key => bucket.name }
}
output "service_account_names" {
  description = "Separated component identities; runtime privileges are added by their owning slices."
  value       = { for key, account in google_service_account.component : key => account.email }
}
output "image_repository" {
  description = "Artifact Registry image prefix; promotion must use immutable digests."
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.repository_id}"
}
output "build_identity_provider" {
  description = "Disabled-by-default GitHub provider; requires canonical resource-name audience."
  value       = google_iam_workload_identity_pool_provider.build.name
}
