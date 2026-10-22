variable "project_id" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Explicit reviewed project required."
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
variable "enabled" {
  description = "Owner-reviewed bootstrap opt-in; default creates no resources."
  type        = bool
  default     = false
}
variable "public_invocation_approved" {
  description = "Separate explicit approval after service exists and security/release review; never defaults on."
  type        = bool
  default     = false
  validation {
    condition     = !var.public_invocation_approved || var.enabled
    error_message = "Public invocation cannot be approved with bootstrap disabled."
  }
}
