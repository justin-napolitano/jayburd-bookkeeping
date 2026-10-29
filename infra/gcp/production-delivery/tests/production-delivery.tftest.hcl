mock_provider "google" {}
variables {
  project_id     = "jayburd-bookkeeping-dev"
  project_number = "9088794002"
}
run "inert_default" {
  command = plan
  assert {
    condition     = length(google_cloud_run_v2_job.delivery) == 0 && length(google_storage_bucket_iam_member.registry) == 0 && length(google_secret_manager_secret_iam_member.add_version) == 0
    error_message = "No job, registry access or destination write by default."
  }
}
run "lookup_without_destination_write" {
  command = plan
  variables {
    delivery = {
      image              = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      key                = "PLAID_PRODUCTION_SECRET"
      infisical_project  = "00000000-0000-4000-8000-000000000001"
      infisical_identity = "00000000-0000-4000-8000-000000000002"
      source_version     = "7"
      expires_at         = "2030-01-01T00:00:00Z"
    }
  }
  assert {
    condition     = length(google_secret_manager_secret_iam_member.add_version) == 0 && google_cloud_run_v2_job.delivery[0].template[0].task_count == 1 && google_cloud_run_v2_job.delivery[0].template[0].parallelism == 1 && google_cloud_run_v2_job.delivery[0].template[0].template[0].max_retries == 0 && google_cloud_run_v2_job.delivery[0].template[0].template[0].timeout == "60s"
    error_message = "One bounded lookup task, no retries or destination write."
  }
  assert {
    condition     = contains(google_cloud_run_v2_job.delivery[0].template[0].template[0].containers[0].args, "-lookup") && contains(google_cloud_run_v2_job.delivery[0].template[0].template[0].containers[0].args, "-environment=production") && !contains(google_cloud_run_v2_job.delivery[0].template[0].template[0].containers[0].args, "-execute")
    error_message = "Only explicit Production lookup mode."
  }
}
run "execute_one_static_destination" {
  command = plan
  variables {
    delivery = {
      image              = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      key                = "PLAID_PRODUCTION_SECRET"
      infisical_project  = "00000000-0000-4000-8000-000000000001"
      infisical_identity = "00000000-0000-4000-8000-000000000002"
      source_version     = "7"
      source_id          = "00000000-0000-4000-8000-000000000003"
      mode               = "execute"
      expires_at         = "2030-01-01T00:00:00Z"
    }
  }
  assert {
    condition     = length(google_secret_manager_secret_iam_member.add_version) == 1 && google_secret_manager_secret_iam_member.add_version[0].secret_id == "jayburd-prod-bank-sync-client-secret" && google_secret_manager_secret_iam_member.add_version[0].role == "roles/secretmanager.secretVersionAdder"
    error_message = "Only addVersion on the one selected Production destination."
  }
  assert {
    condition     = strcontains(google_storage_bucket_iam_member.registry[0].condition[0].expression, "/objects/production-static-delivery-v1/")
    error_message = "Immutable delivery metadata cannot access API registry objects."
  }
}
run "reject_sandbox_key" {
  command = plan
  variables {
    delivery = {
      image              = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      key                = "PLAID_SECRET"
      infisical_project  = "00000000-0000-4000-8000-000000000001"
      infisical_identity = "00000000-0000-4000-8000-000000000002"
      source_version     = "7"
      expires_at         = "2030-01-01T00:00:00Z"
    }
  }
  expect_failures = [var.delivery]
}
