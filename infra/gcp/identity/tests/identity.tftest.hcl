mock_provider "google" {}
variables {
  project_id     = "jayburd-synthetic-test"
  project_number = "123456789012"
}
run "invited_totp_identity" {
  command = plan
  assert {
    condition     = google_identity_platform_config.development.sign_in[0].email[0].enabled && google_identity_platform_config.development.sign_in[0].email[0].password_required && !google_identity_platform_config.development.sign_in[0].anonymous[0].enabled && !google_identity_platform_config.development.sign_in[0].phone_number[0].enabled && !google_identity_platform_config.development.sign_in[0].allow_duplicate_emails
    error_message = "Only unique email/password first-factor identities are enabled."
  }
  assert {
    condition     = google_identity_platform_config.development.client[0].permissions[0].disabled_user_signup && google_identity_platform_config.development.client[0].permissions[0].disabled_user_deletion
    error_message = "Account creation and deletion must remain operator-managed."
  }
  assert {
    condition     = google_identity_platform_config.development.mfa[0].state == "ENABLED" && length(google_identity_platform_config.development.mfa[0].enabled_providers) == 0 && google_identity_platform_config.development.mfa[0].provider_configs[0].state == "ENABLED" && google_identity_platform_config.development.mfa[0].provider_configs[0].totp_provider_config[0].adjacent_intervals == 1
    error_message = "Enable TOTP enrollment with one adjacent interval and no SMS fallback."
  }
  assert {
    condition     = toset(google_identity_platform_config.development.authorized_domains) == toset(["jayburd-synthetic-test.firebaseapp.com", "jayburd-dev-web-123456789012.us-central1.run.app"]) && !google_identity_platform_config.development.multi_tenant[0].allow_tenants && !google_identity_platform_config.development.monitoring[0].request_logging[0].enabled
    error_message = "Use explicit hosted domains, project identities and no request payload logging."
  }
  assert {
    condition     = !google_project_service.identity.disable_on_destroy && google_project_service.identity.service == "identitytoolkit.googleapis.com"
    error_message = "Preserve identity API activation."
  }
}
run "reject_invalid_project" {
  command = plan
  variables { project_id = "invalid/project" }
  expect_failures = [var.project_id]
}
run "reject_invalid_number" {
  command = plan
  variables { project_number = "not-a-number" }
  expect_failures = [var.project_number]
}
