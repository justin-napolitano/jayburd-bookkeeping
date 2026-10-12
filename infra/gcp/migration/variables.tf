variable "project_id" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Explicit project ID required."
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
variable "image" {
  description = "Exact reviewed migrator digest; no tags or alternate repositories."
  type        = string
  validation {
    condition     = can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/migrate@sha256:[a-f0-9]{64}$", var.image))
    error_message = "Matching project migrator digest required."
  }
}
