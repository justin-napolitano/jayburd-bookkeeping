output "api_origin" { value = var.runtime == null ? null : local.api_origin }
output "public_origin" { value = var.runtime == null ? null : local.public_origin }
output "readback_job" { value = var.runtime == null ? null : google_cloud_run_v2_job.readback[0].name }
