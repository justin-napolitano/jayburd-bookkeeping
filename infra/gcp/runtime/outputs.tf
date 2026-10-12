# Only public destinations and revision IDs belong in deployment receipts.
output "public_origin" { value = local.public_origin }
output "api_origin" { value = local.api_origin }
output "revisions" { value = { api = google_cloud_run_v2_service.api.latest_ready_revision, web = google_cloud_run_v2_service.web.latest_ready_revision } }
