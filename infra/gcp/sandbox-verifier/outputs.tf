output "login" {
  value = var.enabled ? trimsuffix(local.login, ".gserviceaccount.com") : null
}
output "job" {
  value = var.run == null ? null : google_cloud_run_v2_job.compare[0].name
}
output "input_bucket" {
  value = var.enabled ? google_storage_bucket.inputs[0].name : null
}
