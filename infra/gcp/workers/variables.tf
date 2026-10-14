variable "project_id" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Explicit development project required."
  }
}
variable "project_number" {
  type = string
  validation {
    condition     = can(regex("^[1-9][0-9]{5,17}$", var.project_number))
    error_message = "Verified project number required."
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
variable "runtime" {
  description = "Null provisions only private infrastructure. Exact image and SQL binding metadata enable services."
  type        = object({ image = string, binding_id = string, client_id_version = string, client_secret_version = string })
  default     = null
  validation {
    condition = var.runtime == null ? true : (
      can(regex("^us-central1-docker[.]pkg[.]dev/${var.project_id}/jayburd-dev/worker@sha256:[a-f0-9]{64}$", var.runtime.image)) &&
      can(regex("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", var.runtime.binding_id)) && var.runtime.binding_id != "00000000-0000-0000-0000-000000000000" &&
      can(regex("^[1-9][0-9]{0,17}$", var.runtime.client_id_version)) && can(regex("^[1-9][0-9]{0,17}$", var.runtime.client_secret_version))
    )
    error_message = "Runtime needs an immutable worker image, enrolled binding UUID and exact numeric secret versions."
  }
}
variable "scheduler_enabled" {
  type    = bool
  default = false
  validation {
    condition     = !var.scheduler_enabled || var.runtime != null
    error_message = "Scheduling requires an explicitly configured runtime."
  }
}
variable "infisical_sync_service_account" {
  description = "Existing verified Infisical impersonation connection identity; null grants no sync access."
  type        = string
  default     = null
  validation {
    condition     = var.infisical_sync_service_account == null ? true : (can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]@${var.project_id}[.]iam[.]gserviceaccount[.]com$", var.infisical_sync_service_account)) && !contains([for component in ["api", "sync", "publisher", "consumer", "frontend", "builder", "deployer", "migrator", "scheduler", "task-invoker"] : "jayburd-dev-${component}@${var.project_id}.iam.gserviceaccount.com"], var.infisical_sync_service_account))
    error_message = "Supply the verified same-project Infisical connection account, never an external principal or key."
  }
}
