variable "project_id" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Explicit reviewed project required."
  }
}
variable "project_number" {
  type = string
  validation {
    condition     = can(regex("^[1-9][0-9]{5,18}$", var.project_number)) && try(tonumber(var.project_number) <= 9223372036854775807, false)
    error_message = "Canonical numeric project required."
  }
}
variable "region" {
  type    = string
  default = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Only reviewed us-central1 is permitted."
  }
}
variable "enabled" {
  type        = bool
  default     = false
  description = "Owner bootstrap only after fixed service, runtime state and public approval already exist."
}
variable "workflow_sha" {
  type        = string
  default     = ""
  description = "Exact reviewed main commit containing deployment workflow and manifest; owner updates the pin."
  validation {
    condition     = var.enabled ? can(regex("^[a-f0-9]{40}$", var.workflow_sha)) : var.workflow_sha == ""
    error_message = "Enabled federation requires an explicit exact workflow SHA; disabled is inert."
  }
}
variable "federation_active" {
  type        = bool
  default     = false
  description = "Independent owner activation; deactivation preserves identity/grants but does not revoke existing short-lived tokens."
  validation {
    condition     = !var.federation_active || var.enabled
    error_message = "Federation cannot be active while delivery resources are disabled."
  }
}
