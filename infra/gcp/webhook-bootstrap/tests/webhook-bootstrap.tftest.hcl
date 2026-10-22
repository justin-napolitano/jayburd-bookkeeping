mock_provider "google" {
  override_during = plan
  mock_resource "google_service_account" {
    defaults = { email = "jayburd-dev-webhook@jayburd-synthetic-test.iam.gserviceaccount.com" }
  }
}
variables { project_id = "jayburd-synthetic-test" }

run "default_is_inert" {
  command = plan
  assert {
    condition     = length(google_service_account.webhook) == 0 && length(google_project_iam_member.sql) == 0 && length(google_sql_user.webhook) == 0 && length(google_secret_manager_secret_iam_member.client) == 0 && length(google_cloud_run_v2_service_iam_binding.public) == 0
    error_message = "Default bootstrap must create no identity, SQL user, secret grant or public binding."
  }
}
run "dedicated_identity_and_two_static_reads" {
  command = plan
  variables { enabled = true }
  assert {
    condition     = google_service_account.webhook[0].account_id == "jayburd-dev-webhook" && google_sql_user.webhook[0].name == "jayburd-dev-webhook@jayburd-synthetic-test.iam" && google_sql_user.webhook[0].type == "CLOUD_IAM_SERVICE_ACCOUNT" && google_sql_user.webhook[0].instance == "jayburd-dev" && google_sql_user.webhook[0].password == null
    error_message = "Bootstrap must derive the webhook IAM SQL user without a password or alternate identity."
  }
  assert {
    condition     = toset(keys(google_project_iam_member.sql)) == toset(["roles/cloudsql.client", "roles/cloudsql.instanceUser"]) && alltrue([for grant in google_project_iam_member.sql : grant.member == "serviceAccount:jayburd-dev-webhook@jayburd-synthetic-test.iam.gserviceaccount.com"])
    error_message = "Only the dedicated webhook identity receives client/login IAM."
  }
  assert {
    condition     = toset(keys(google_secret_manager_secret_iam_member.client)) == toset(["jayburd-dev-bank-sync-client-id", "jayburd-dev-bank-sync-client-secret"]) && alltrue([for grant in google_secret_manager_secret_iam_member.client : grant.role == "roles/secretmanager.secretAccessor" && grant.member == "serviceAccount:jayburd-dev-webhook@jayburd-synthetic-test.iam.gserviceaccount.com"])
    error_message = "Exactly two existing static secret containers may be read at secret scope."
  }
  assert {
    condition     = length(google_cloud_run_v2_service_iam_binding.public) == 0
    error_message = "Identity bootstrap alone must not expose the service."
  }
}
run "public_approval_is_one_fixed_service" {
  command = plan
  variables {
    enabled                    = true
    public_invocation_approved = true
  }
  assert {
    condition     = length(google_cloud_run_v2_service_iam_binding.public) == 1 && google_cloud_run_v2_service_iam_binding.public[0].name == "jayburd-dev-webhook" && google_cloud_run_v2_service_iam_binding.public[0].role == "roles/run.invoker" && google_cloud_run_v2_service_iam_binding.public[0].members == toset(["allUsers"])
    error_message = "Public invocation must cover exactly the fixed dedicated host, never private API/workers."
  }
}
run "reject_public_with_disabled_bootstrap" {
  command = plan
  variables { public_invocation_approved = true }
  expect_failures = [var.public_invocation_approved]
}
run "reject_unreviewed_region" {
  command = plan
  variables { region = "europe-west1" }
  expect_failures = [var.region]
}
run "reject_project_interpolation" {
  command = plan
  variables { project_id = "synthetic/project" }
  expect_failures = [var.project_id]
}
