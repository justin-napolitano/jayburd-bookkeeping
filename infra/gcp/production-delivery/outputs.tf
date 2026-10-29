output "delivery_identity" {
  description = "Public identity to bind narrowly in the isolated Infisical project; never a credential."
  value       = google_service_account.delivery.email
}
output "job_name" {
  value = var.delivery == null ? null : google_cloud_run_v2_job.delivery[0].name
}
