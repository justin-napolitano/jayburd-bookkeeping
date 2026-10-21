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
    error_message = "Verified project number required."
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
variable "runtime" {
  description = "Explicit immutable release metadata; runtime never owns bootstrap IAM or scheduler activation."
  nullable    = false
  type        = object({ image = string, binding_id = string, publisher_policy = optional(string, "binding-v1"), client_id_version = string, client_secret_version = string })
  validation {
    condition = var.runtime == null ? true : (
      can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/worker@sha256:[a-f0-9]{64}$", var.runtime.image)) &&
      contains(["binding-v1", "shared-v1"], var.runtime.publisher_policy) && (var.runtime.publisher_policy == "shared-v1" ? var.runtime.binding_id == "" : (can(regex("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", var.runtime.binding_id)) && var.runtime.binding_id != "00000000-0000-0000-0000-000000000000")) &&
      can(regex("^[1-9][0-9]{0,18}$", var.runtime.client_id_version)) && try(tonumber(var.runtime.client_id_version) <= 9223372036854775807, false) && can(regex("^[1-9][0-9]{0,18}$", var.runtime.client_secret_version)) && try(tonumber(var.runtime.client_secret_version) <= 9223372036854775807, false)
    )
    error_message = "Runtime needs an immutable worker image, matching publisher policy/binding and exact numeric secret versions."
  }
}
