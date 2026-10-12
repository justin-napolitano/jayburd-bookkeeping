# This root alone owns API activation; it does not apply the foundation/runtime roots.
resource "google_project_service" "identity" {
  project            = var.project_id
  service            = "identitytoolkit.googleapis.com"
  disable_on_destroy = false
}

# Computed hash configuration can enter state: use the owner-only foundation bucket.
resource "google_identity_platform_config" "development" {
  project = var.project_id
  authorized_domains = [
    "${var.project_id}.firebaseapp.com",
    "jayburd-dev-web-${var.project_number}.us-central1.run.app",
  ]
  sign_in {
    allow_duplicate_emails = false
    email {
      enabled           = true
      password_required = true
    }
    anonymous { enabled = false }
    phone_number { enabled = false }
  }
  client {
    permissions {
      disabled_user_signup   = true
      disabled_user_deletion = true
    }
  }
  # ENABLED permits first-factor enrollment; the API separately requires a TOTP token.
  mfa {
    state             = "ENABLED"
    enabled_providers = []
    provider_configs {
      state = "ENABLED"
      totp_provider_config { adjacent_intervals = 1 }
    }
  }
  multi_tenant { allow_tenants = false }
  monitoring {
    request_logging { enabled = false }
  }
  lifecycle { prevent_destroy = true }
  depends_on = [google_project_service.identity]
}
