output "job_name" {
  description = "Public job identifier for the explicit owner-operated proof."
  value       = google_cloud_run_v2_job.probe.name
}
