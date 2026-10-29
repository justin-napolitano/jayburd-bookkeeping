mock_provider "google" {}
variables {
  project_id     = "jayburd-synthetic-test"
  project_number = "123456789012"
}
run "database_recovery_and_capacity" {
  command = plan
  assert {
    condition     = google_sql_database_instance.ledger.database_version == "POSTGRES_17" && google_sql_database_instance.ledger.settings[0].edition == "ENTERPRISE" && google_sql_database_instance.ledger.settings[0].tier == "db-f1-micro" && google_sql_database_instance.ledger.settings[0].availability_type == "ZONAL"
    error_message = "Keep the reviewed Production candidate database edition/version/tier."
  }
  assert {
    condition     = google_sql_database_instance.ledger.deletion_protection && google_sql_database_instance.ledger.settings[0].deletion_protection_enabled && google_sql_database_instance.ledger.settings[0].disk_size == 10 && google_sql_database_instance.ledger.settings[0].disk_autoresize_limit == 20
    error_message = "Require deletion protection and bounded disk growth."
  }
  assert {
    condition     = google_sql_database_instance.ledger.settings[0].backup_configuration[0].enabled && google_sql_database_instance.ledger.settings[0].backup_configuration[0].point_in_time_recovery_enabled && google_sql_database_instance.ledger.settings[0].backup_configuration[0].transaction_log_retention_days == 7 && google_sql_database_instance.ledger.settings[0].backup_configuration[0].backup_retention_settings[0].retained_backups == 8
    error_message = "Daily backups and eight backups and seven-day PITR are required."
  }
  assert {
    condition     = google_sql_database_instance.ledger.settings[0].connector_enforcement == "REQUIRED" && google_sql_database_instance.ledger.settings[0].ip_configuration[0].ssl_mode == "ENCRYPTED_ONLY" && length(google_sql_database_instance.ledger.settings[0].ip_configuration[0].authorized_networks) == 0
    error_message = "No direct network allowlists or unencrypted connections."
  }
  assert {
    condition     = alltrue([for u in google_sql_user.component : u.type == "CLOUD_IAM_SERVICE_ACCOUNT"]) && length(google_sql_user.component) == 5
    error_message = "Only separate Production runtime and migrator IAM users; no built-in password users."
  }
  assert {
    condition     = contains(google_sql_database_instance.ledger.settings[0].database_flags, { name = "max_connections", value = "50" }) && contains(google_sql_database_instance.ledger.settings[0].database_flags, { name = "cloudsql.iam_authentication", value = "on" })
    error_message = "IAM authentication and explicit server connection bounds are required."
  }
}
run "reject_missing_project" {
  command = plan
  variables { project_id = "" }
  expect_failures = [var.project_id]
}
run "bootstrap_disabled_by_default" {
  command = plan
  assert {
    condition     = length(google_sql_user.bootstrap_operator) == 0 && length(google_project_iam_member.bootstrap_operator) == 0
    error_message = "Default plans must not grant administrator access."
  }
}
run "separate_bootstrap_operator" {
  command = plan
  variables { bootstrap_operator_email = "synthetic-owner@example.test" }
  assert {
    condition     = google_sql_user.bootstrap_operator[0].type == "CLOUD_IAM_USER" && google_sql_user.bootstrap_operator[0].database_roles == tolist(["cloudsqlsuperuser"]) && google_sql_user.bootstrap_operator[0].name == "synthetic-owner@example.test"
    error_message = "Only the explicitly selected human operator receives the administrative role."
  }
  assert {
    condition     = length(google_project_iam_member.bootstrap_operator) == 2 && alltrue([for b in google_project_iam_member.bootstrap_operator : b.member == "user:synthetic-owner@example.test"])
    error_message = "Operator receives only connector/login IAM bindings."
  }
  assert {
    condition     = alltrue([for u in google_sql_user.component : u.database_roles == null || length(u.database_roles) == 0])
    error_message = "Application identities must not receive administrator memberships."
  }
}
run "reject_service_account_operator" {
  command = plan
  variables { bootstrap_operator_email = "synthetic-api@jayburd-synthetic-test.iam.gserviceaccount.com" }
  expect_failures = [var.bootstrap_operator_email]
}
run "reject_unreviewed_region" {
  command = plan
  variables { region = "europe-west1" }
  expect_failures = [var.region]
}
run "isolated_disabled_build_authority" {
  command = plan
  assert {
    condition     = google_iam_workload_identity_pool.github.disabled && google_iam_workload_identity_pool_provider.build.disabled
    error_message = "Build federation must be disabled by default."
  }
  assert {
    condition     = alltrue([for k, a in google_service_account.component : a.account_id == "jayburd-prod-${k}"]) && length(google_service_account.component) == 9
    error_message = "Production identities must be separate and bounded."
  }
  assert {
    condition     = google_artifact_registry_repository.images.repository_id == "jayburd-prod" && google_artifact_registry_repository.images.docker_config[0].immutable_tags && google_artifact_registry_repository_iam_member.builder.role == "roles/artifactregistry.writer"
    error_message = "Builder must publish only into the immutable Production registry."
  }
  assert {
    condition     = strcontains(google_iam_workload_identity_pool_provider.build.attribute_condition, "assertion.workflow_ref == 'justin-napolitano/jayburd-bookkeeping/.github/workflows/build-production-images.yml@refs/heads/main'") && strcontains(google_iam_workload_identity_pool_provider.build.attribute_condition, "assertion.actor_id == '55423093'") && strcontains(google_iam_workload_identity_pool_provider.build.attribute_condition, "assertion.repository_id == '1414515318'") && strcontains(google_iam_workload_identity_pool_provider.build.attribute_condition, "assertion.event_name == 'workflow_dispatch'")
    error_message = "Only the exact owner/main Production workflow can federate."
  }
  assert {
    condition     = alltrue([for b in google_storage_bucket.state : b.public_access_prevention == "enforced" && b.uniform_bucket_level_access && b.versioning[0].enabled && !b.force_destroy]) && google_sql_database_instance.ledger.name == "jayburd-prod" && google_sql_database_instance.ledger.settings[0].backup_configuration[0].location == "us"
    error_message = "Separate private state and explicit Production SQL/backup location required."
  }
}
run "reject_development_project" {
  command = plan
  variables { project_id = "jayburd-bookkeeping-dev" }
  expect_failures = [var.project_id]
}
run "reject_development_number" {
  command = plan
  variables { project_number = "9088794002" }
  expect_failures = [var.project_number]
}
run "explicit_build_optin" {
  command = plan
  variables { enable_github_build_identity = true }
  assert {
    condition     = !google_iam_workload_identity_pool.github.disabled && !google_iam_workload_identity_pool_provider.build.disabled
    error_message = "Only explicit reviewed input enables builder federation."
  }
}
