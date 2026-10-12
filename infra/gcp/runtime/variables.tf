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
variable "project_number" {
  type = string
  validation {
    condition     = can(regex("^[0-9]{6,20}$", var.project_number))
    error_message = "Verified project number required for deterministic origins."
  }
}
variable "images" {
  description = "Reviewed API/web images from successful build receipts; mutable tags forbidden."
  type        = object({ api = string, web = string })
  validation {
    condition     = alltrue([for component, image in var.images : can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/${component}@sha256:[a-f0-9]{64}$", image))])
    error_message = "Each image must be a matching component digest in the project's repository."
  }
}
variable "firebase_api_key" {
  description = "Public browser identifier, never a service credential."
  type        = string
  validation {
    condition     = can(regex("^[A-Za-z0-9_-]{1,128}$", var.firebase_api_key))
    error_message = "Public Firebase API identifier required."
  }
}
