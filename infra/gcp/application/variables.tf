variable "project_id" {
  description = "Owner-selected project from the foundation; never inherited from CLI context."
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "A valid existing project ID is required."
  }
}
variable "region" {
  description = "Reviewed development region."
  type        = string
  default     = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Review architecture and cost before changing region."
  }
}
variable "bootstrap_operator_email" {
  description = "Explicit owner IAM administrator for SQL bootstrap; null provisions no administrator. Never an application service account."
  type        = string
  default     = null
  validation {
    condition = var.bootstrap_operator_email == null ? true : (
      can(regex("^[a-z0-9][a-z0-9._+-]*@[a-z0-9.-]+\\.[a-z]{2,}$", var.bootstrap_operator_email)) &&
      !endswith(var.bootstrap_operator_email, ".gserviceaccount.com") &&
      !endswith(var.bootstrap_operator_email, ".iam") &&
      length(var.bootstrap_operator_email) <= 63
    )
    error_message = "Select a lowercase human IAM email (at most 63 characters), never an application identity."
  }
}
