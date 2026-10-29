variable "project_id" {
  description = "Owner-selected project from the foundation; never inherited from CLI context."
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id)) && var.project_id == "jayburd-bookkeeping-dev"
    error_message = "A valid existing project ID is required."
  }
}
variable "region" {
  description = "Reviewed Production candidate region."
  type        = string
  default     = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Review architecture and cost before changing region."
  }
}
variable "project_number" {
  description = "Verified numeric identifier of the owner-selected shared project."
  type        = string
  validation {
    condition     = can(regex("^[1-9][0-9]{5,18}$", var.project_number)) && var.project_number == "9088794002" && try(tonumber(var.project_number) <= 9223372036854775807, false)
    error_message = "Use the exact reviewed shared project number."
  }
}
variable "enable_github_build_identity" {
  description = "Explicit opt-in after owner review of Production target and effective IAM."
  type        = bool
  default     = false
}
