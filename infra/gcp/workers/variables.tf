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
    condition     = can(regex("^[1-9][0-9]{5,18}$", var.project_number)) && try(tonumber(var.project_number) <= 9223372036854775807, false)
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
variable "worker_services_exist" {
  description = "Owner confirms both services exist and Scheduler is paused before installing private invocation IAM."
  type        = bool
  default     = false
}
variable "infisical_sync_service_account" {
  description = "Existing verified Infisical impersonation connection identity; null grants no sync access."
  type        = string
  default     = null
  validation {
    condition     = var.infisical_sync_service_account == null ? true : (can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]@${var.project_id}[.]iam[.]gserviceaccount[.]com$", var.infisical_sync_service_account)) && !contains([for component in ["api", "sync", "publisher", "consumer", "frontend", "builder", "deployer", "worker-deployer", "migrator", "scheduler", "task-invoker"] : "jayburd-dev-${component}@${var.project_id}.iam.gserviceaccount.com"], var.infisical_sync_service_account))
    error_message = "Supply the verified same-project Infisical connection account, never an external principal or key."
  }
}
