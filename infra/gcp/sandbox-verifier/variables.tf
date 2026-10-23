variable "project_id" {
  type = string
  validation {
    condition     = var.project_id == "jayburd-bookkeeping-dev"
    error_message = "Only the reviewed development project is supported."
  }
}
variable "region" {
  type    = string
  default = "us-central1"
  validation {
    condition     = var.region == "us-central1"
    error_message = "Only the reviewed development region is supported."
  }
}
variable "enabled" {
  type    = bool
  default = false
}
# Metadata only; all three bounded files are uploaded separately, never Terraform-managed.
variable "run" {
  type    = object({ image = string, run_id = string })
  default = null
  validation {
    condition = var.run == null ? true : (
      var.enabled &&
      can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/worker@sha256:[a-f0-9]{64}$", var.run.image)) &&
      can(regex("^verify-[a-z0-9]{8,40}$", var.run.run_id))
    )
    error_message = "An enabled verifier, exact worker digest and bounded run ID are required."
  }
}
