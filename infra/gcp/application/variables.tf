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
