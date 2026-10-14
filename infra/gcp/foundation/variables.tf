variable "project_id" {
  description = "Existing owner-selected development project; never inferred from CLI context."
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Supply a valid existing GCP project ID."
  }
}
variable "project_number" {
  description = "Verified numeric ID of the same project, required for federation and budget scope."
  type        = string
  validation {
    condition     = can(regex("^[1-9][0-9]{5,18}$", var.project_number)) && try(tonumber(var.project_number) <= 9223372036854775807, false)
    error_message = "Supply the verified project number."
  }
}
variable "billing_account_id" {
  description = "Existing billing account identifier; this module does not create or link billing."
  type        = string
  validation {
    condition     = can(regex("^[A-Z0-9]{6}-[A-Z0-9]{6}-[A-Z0-9]{6}$", var.billing_account_id))
    error_message = "Supply a billing account identifier, not credentials."
  }
}
variable "region" {
  description = "Development region; us-central1 is the accepted provisional default."
  type        = string
  default     = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Review regional cost and architecture before changing the initial region."
  }
}
variable "enable_github_build_identity" {
  description = "Owner-controlled opt-in after promotion review; no deployment federation is granted."
  type        = bool
  default     = false
}

variable "enable_github_deploy_identity" {
  description = "Owner opt-in after reviewing runtime rollout authority; separate from build federation."
  type        = bool
  default     = false
}

variable "enable_github_worker_deploy_identity" {
  description = "Owner opt-in for the isolated, exact worker deployment workflow."
  type        = bool
  default     = false
}
