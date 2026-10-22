output "runtime_identity" {
  value = var.enabled ? "jayburd-dev-webhook@${var.project_id}.iam.gserviceaccount.com" : null
}
output "database_login" {
  value = var.enabled ? "jayburd-dev-webhook@${var.project_id}.iam" : null
}
output "public_invocation_approved" { value = var.enabled && var.public_invocation_approved }
