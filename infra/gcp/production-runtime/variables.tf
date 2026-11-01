variable "project_id" {
  type = string
  validation {
    condition     = var.project_id == "jayburd-bookkeeping-dev"
    error_message = "Use the reviewed shared project only."
  }
}
variable "project_number" {
  type = string
  validation {
    condition     = var.project_number == "9088794002"
    error_message = "Use its exact reviewed numeric identity."
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
variable "runtime" {
  description = "Public reviewed metadata only. Null creates no application, readback job or runtime IAM. Provider access remains expired."
  type = object({
    images                = object({ api = string, web = string, worker = string })
    firebase_api_key      = string
    client_id_version     = string
    client_secret_version = string
    readback_expires_at   = string
  })
  default = null
  validation {
    condition = var.runtime == null ? true : (
      alltrue([for component, image in var.runtime.images : can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-prod/${component}@sha256:[a-f0-9]{64}$", image))]) &&
      can(regex("^[A-Za-z0-9_-]{1,128}$", var.runtime.firebase_api_key)) &&
      alltrue([for version in [var.runtime.client_id_version, var.runtime.client_secret_version] : can(regex("^[1-9][0-9]{0,18}$", version)) && try(tonumber(version) <= 9223372036854775807, false)]) &&
      can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$", var.runtime.readback_expires_at)) && can(timeadd(var.runtime.readback_expires_at, "0s"))
    )
    error_message = "Pin matching Production image digests, public Firebase identifier, numeric static versions and canonical UTC readback deadline."
  }
}

variable "pilot" {
  description = "Separate explicit reviewed bank authority; null retains permanently expired recovery. Readback expiry cannot activate it."
  type        = object({ approval_expires_at = string, webhook_url = string })
  default     = null
  validation {
    condition = var.pilot == null ? true : (
      var.runtime != null &&
      var.pilot.webhook_url == "https://jayburd-prod-webhook-${var.project_number}.${var.region}.run.app/webhooks/plaid" &&
      can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$", var.pilot.approval_expires_at)) &&
      try(timecmp(var.pilot.approval_expires_at, plantimestamp()) > 0 && timecmp(var.pilot.approval_expires_at, timeadd(plantimestamp(), "168h")) <= 0, false)
    )
    error_message = "Pilot requires configured Production runtime, exact matching receiver and current UTC authority within seven days."
  }
}
