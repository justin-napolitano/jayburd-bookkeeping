output "enabled" { value = var.enabled }
output "federation_active" { value = var.enabled && var.federation_active }
output "workflow_sha" { value = var.enabled ? var.workflow_sha : null }
