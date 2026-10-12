terraform {
  required_version = "= 1.15.7"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "= 8.6.0"
    }
  }
}

# Owner ADC/workload identity only; no credential arguments or user resources.
provider "google" {
  project = var.project_id
}
