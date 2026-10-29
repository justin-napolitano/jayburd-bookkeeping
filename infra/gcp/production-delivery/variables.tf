variable "project_id" {
  type = string
  validation {
    condition     = var.project_id == "jayburd-bookkeeping-dev"
    error_message = "Use the reviewed shared project."
  }
}
variable "project_number" {
  type = string
  validation {
    condition     = var.project_number == "9088794002"
    error_message = "Use the reviewed numeric project."
  }
}
variable "region" {
  type    = string
  default = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Use the reviewed region."
  }
}
variable "delivery" {
  description = "Reviewed metadata only. Null creates no job or resource-access grants; lookup cannot add destination versions."
  type = object({
    image              = string
    key                = string
    infisical_project  = string
    infisical_identity = string
    source_version     = string
    source_id          = optional(string, "")
    mode               = optional(string, "lookup")
    expires_at         = string
  })
  default = null
  validation {
    condition = var.delivery == null ? true : (
      can(regex("^us-central1-docker\\.pkg\\.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:[a-f0-9]{64}$", var.delivery.image)) &&
      contains(["PLAID_CLIENT_ID", "PLAID_PRODUCTION_SECRET"], var.delivery.key) &&
      can(regex("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", var.delivery.infisical_project)) && var.delivery.infisical_project != "00000000-0000-0000-0000-000000000000" &&
      can(regex("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", var.delivery.infisical_identity)) && var.delivery.infisical_identity != "00000000-0000-0000-0000-000000000000" &&
      can(regex("^[1-9][0-9]{0,18}$", var.delivery.source_version)) && try(tonumber(var.delivery.source_version) <= 9223372036854775807, false) &&
      contains(["lookup", "execute"], var.delivery.mode) &&
      (var.delivery.mode == "lookup" ? var.delivery.source_id == "" : (can(regex("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", var.delivery.source_id)) && var.delivery.source_id != "00000000-0000-0000-0000-000000000000")) &&
      can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$", var.delivery.expires_at)) && can(timeadd(var.delivery.expires_at, "0s"))
    )
    error_message = "Pin the Production image, one static key, exact source metadata and UTC deadline."
  }
}
