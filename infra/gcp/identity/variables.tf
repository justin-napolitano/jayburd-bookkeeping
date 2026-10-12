variable "project_id" {
  type        = string
  description = "Existing billing-enabled development project."
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Use a valid GCP project ID."
  }
}

variable "project_number" {
  type        = string
  description = "Existing project number used for the deterministic Cloud Run web domain."
  validation {
    condition     = can(regex("^[0-9]{6,20}$", var.project_number))
    error_message = "Use a numeric GCP project number."
  }
}
