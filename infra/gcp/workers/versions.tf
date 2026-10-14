terraform {
  required_version = "= 1.15.7"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "= 8.6.0"
    }
  }
}

# No credentials are accepted here; owner-operated apply uses short-lived identity.
provider "google" {
  project = var.project_id
  region  = var.region
}
