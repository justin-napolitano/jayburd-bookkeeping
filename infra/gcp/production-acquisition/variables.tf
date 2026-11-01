variable "project_id" {
  type = string
  validation {
    condition     = var.project_id == "jayburd-bookkeeping-dev"
    error_message = "Use only the reviewed shared project."
  }
}
variable "project_number" {
  type = string
  validation {
    condition     = var.project_number == "9088794002"
    error_message = "Use its exact known numeric identity."
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
  description = "Reviewed public metadata only; null creates no worker, receiver, queue, scheduler or grants. Scheduling always remains paused."
  type = object({
    image                   = string
    binding_id              = string
    client_id_version       = string
    client_secret_version   = string
    approval_expires_at     = string
    public_webhook_approved = optional(bool, false)
  })
  default = null
  validation {
    condition = var.runtime == null ? true : (
      can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-prod/worker@sha256:[a-f0-9]{64}$", var.runtime.image)) &&
      can(regex("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", var.runtime.binding_id)) && var.runtime.binding_id != "00000000-0000-0000-0000-000000000000" &&
      alltrue([for version in [var.runtime.client_id_version, var.runtime.client_secret_version] : can(regex("^[1-9][0-9]{0,18}$", version)) && try(tonumber(version) <= 9223372036854775807, false)]) &&
      can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$", var.runtime.approval_expires_at)) &&
      try(timecmp(var.runtime.approval_expires_at, plantimestamp()) > 0 && timecmp(var.runtime.approval_expires_at, timeadd(plantimestamp(), "168h")) <= 0, false)
    )
    error_message = "Pin Production worker/binding/numeric versions and an explicit current UTC deadline within seven days."
  }
}
