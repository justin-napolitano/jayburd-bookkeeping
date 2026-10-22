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
    error_message = "Canonical numeric project identifier required."
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
  description = "Null is inert. Exact reviewed worker image and numeric secret versions; bootstrap and public IAM are separate."
  type        = object({ image = string, client_id_version = string, client_secret_version = string })
  default     = null
  validation {
    condition = var.runtime == null ? true : (
      can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/worker@sha256:[a-f0-9]{64}$", var.runtime.image)) &&
      can(regex("^[1-9][0-9]{0,18}$", var.runtime.client_id_version)) && try(tonumber(var.runtime.client_id_version) <= 9223372036854775807, false) &&
      can(regex("^[1-9][0-9]{0,18}$", var.runtime.client_secret_version)) && try(tonumber(var.runtime.client_secret_version) <= 9223372036854775807, false)
    )
    error_message = "Runtime requires the project worker digest and canonical bounded numeric versions."
  }
}
