variable "project_id" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Explicit project ID required."
  }
}
variable "region" {
  type    = string
  default = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Only the reviewed development region is allowed."
  }
}
variable "project_number" {
  type = string
  validation {
    condition     = can(regex("^[1-9][0-9]{0,18}$", var.project_number)) && try(tonumber(var.project_number) <= 9223372036854775807, false)
    error_message = "Canonical positive signed-int64 project number required for deterministic origins."
  }
}
variable "images" {
  description = "Reviewed API/web images from successful build receipts; mutable tags forbidden."
  type        = object({ api = string, web = string })
  validation {
    condition     = alltrue([for component, image in var.images : can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/${component}@sha256:[a-f0-9]{64}$", image))])
    error_message = "Each image must be a matching component digest in the project's repository."
  }
}
variable "firebase_api_key" {
  description = "Public browser identifier, never a service credential."
  type        = string
  validation {
    condition     = can(regex("^[A-Za-z0-9_-]{1,128}$", var.firebase_api_key))
    error_message = "Public Firebase API identifier required."
  }
}

# Optional metadata only; null preserves the existing disabled release profile.
variable "connections" {
  description = "Owner-reviewed Sandbox connection metadata; contains identifiers, never credentials."
  type = object({
    infisical_origin      = string
    infisical_project     = string
    infisical_identity    = string
    link_events_qualified = bool
    client_id_version     = string
    client_secret_version = string
    consent_expires_at    = string
    webhook_url           = optional(string, "")
  })
  default = null
  validation {
    condition = var.connections == null ? true : (
      contains(["https://us.infisical.com", "https://eu.infisical.com", "https://app.infisical.com"], var.connections.infisical_origin) &&
      alltrue([for id in [var.connections.infisical_project, var.connections.infisical_identity] : can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id)) && id != "00000000-0000-0000-0000-000000000000"]) &&
      can(regex("^projects/${var.project_number}/secrets/jayburd-dev-bank-sync-client-id/versions/[1-9][0-9]{0,18}$", var.connections.client_id_version)) &&
      can(regex("^projects/${var.project_number}/secrets/jayburd-dev-bank-sync-client-secret/versions/[1-9][0-9]{0,18}$", var.connections.client_secret_version)) &&
      alltrue([for version in [var.connections.client_id_version, var.connections.client_secret_version] : try(tonumber(reverse(split("/", version))[0]) <= 9223372036854775807, false)]) &&
      can(formatdate("YYYY-MM-DD", var.connections.consent_expires_at)) &&
      (var.connections.webhook_url == "" || (var.project_id == "jayburd-bookkeeping-dev" && var.project_number == "9088794002" && var.connections.webhook_url == "https://jayburd-dev-webhook-9088794002.us-central1.run.app/webhooks/plaid"))
    )
    error_message = "Connections require approved Infisical metadata and exact numeric static secret versions."
  }
}
