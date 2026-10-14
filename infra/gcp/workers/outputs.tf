output "worker_origins" { value = local.origins }
output "scheduler_enabled" { value = false }
output "provider_secret_containers" { value = sort([for secret in google_secret_manager_secret.provider : secret.secret_id]) }
output "credential_registry_bucket" { value = google_storage_bucket.private["credential-registry"].name }
output "acquisition_evidence_bucket" { value = google_storage_bucket.private["acquisition-evidence"].name }
output "sql_worker_logins" { value = { for component, user in google_sql_user.worker : component => user.name } }
