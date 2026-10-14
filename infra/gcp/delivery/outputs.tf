# Public identity identifiers only; no secret versions or state contents are exported.
output "service_account" { value = google_service_account.delivery.email }
output "job_name" { value = var.canary == null ? null : google_cloud_run_v2_job.canary[0].name }
