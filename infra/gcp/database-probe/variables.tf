variable "project_id" {
  type = string
  validation {
    condition     = var.project_id == "jayburd-bookkeeping-dev"
    error_message = "Only the approved development project is allowed."
  }
}
variable "region" {
  type    = string
  default = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Only the development region is allowed."
  }
}
variable "image" {
  type = string
  validation {
    condition     = can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/api@sha256:[a-f0-9]{64}$", var.image))
    error_message = "Exact reviewed API image digest required."
  }
}
