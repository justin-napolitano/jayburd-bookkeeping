terraform {
  required_version = "= 1.15.7"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "= 8.6.0"
    }
  }
}

# Explicit owner-operated root; no credential or provider input discovery.
provider "google" {
  project = var.project_id
  region  = var.region
}
