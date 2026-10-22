output "webhook_url" { value = var.runtime == null ? null : "${local.origin}/webhooks/plaid" }
output "service_name" { value = var.runtime == null ? null : "jayburd-dev-webhook" }
