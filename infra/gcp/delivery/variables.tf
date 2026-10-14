variable "project_id" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Explicit development project required."
  }
}
variable "project_number" {
  type = string
  validation {
    condition     = can(regex("^[1-9][0-9]{5,18}$", var.project_number)) && try(tonumber(var.project_number) <= 9223372036854775807, false)
    error_message = "Canonical project number required."
  }
}
variable "region" {
  type    = string
  default = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Only the reviewed region is allowed."
  }
}
variable "canary" {
  description = "Reviewed public metadata only. Null provisions identity/storage but no executable job."
  type        = object({ image = string, infisical_project = string, infisical_identity = string, source_id = optional(string), source_version = string, mode = optional(string, "lookup") })
  default     = null
  validation {
    condition = var.canary == null ? true : (
      can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/worker@sha256:[a-f0-9]{64}$", var.canary.image)) &&
      alltrue([for id in [var.canary.infisical_project, var.canary.infisical_identity] : can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id)) && id != "00000000-0000-0000-0000-000000000000"]) &&
      contains(["lookup", "execute"], var.canary.mode) &&
      (var.canary.mode == "lookup" ? var.canary.source_id == null : try(can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.canary.source_id)) && var.canary.source_id != "00000000-0000-0000-0000-000000000000", false)) &&
      can(regex("^[1-9][0-9]{0,18}$", var.canary.source_version)) && try(tonumber(var.canary.source_version) <= 9223372036854775807, false)
    )
    error_message = "Pinned reviewed worker image, source identity and exact numeric source version required."
  }
}
variable "enable_github_canary" {
  type        = bool
  default     = false
  description = "Owner enables only after reviewing canary job/image and Infisical identity scope."
  validation {
    condition     = !var.enable_github_canary || var.canary != null
    error_message = "A reviewed canary job is required before execution federation can be enabled."
  }
}
