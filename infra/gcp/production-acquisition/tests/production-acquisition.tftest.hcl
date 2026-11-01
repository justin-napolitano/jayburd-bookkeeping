mock_provider "google" {}
override_data {
  target = data.google_service_account.caller["scheduler"]
  values = {
    unique_id = "111111111111111111111"
    name      = "projects/jayburd-bookkeeping-dev/serviceAccounts/jayburd-prod-scheduler@jayburd-bookkeeping-dev.iam.gserviceaccount.com"
  }
}
override_data {
  target = data.google_service_account.caller["task-invoker"]
  values = {
    unique_id = "222222222222222222222"
    name      = "projects/jayburd-bookkeeping-dev/serviceAccounts/jayburd-prod-task-invoker@jayburd-bookkeeping-dev.iam.gserviceaccount.com"
  }
}
variables {
  project_id     = "jayburd-bookkeeping-dev"
  project_number = "9088794002"
  runtime = {
    image                 = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
    binding_id            = "00000000-0000-4000-8000-000000000011"
    client_id_version     = "7"
    client_secret_version = "9"
    approval_expires_at   = timeadd(timestamp(), "24h")
  }
}
run "inert_default" {
  command = plan
  variables { runtime = null }
  assert {
    condition     = length(google_cloud_run_v2_service.worker) == 0 && length(google_cloud_run_v2_service.webhook) == 0 && length(google_cloud_tasks_queue.acquisition) == 0 && length(google_cloud_scheduler_job.publisher) == 0 && length(google_cloud_run_v2_service_iam_binding.worker) == 0 && length(google_cloud_run_v2_service_iam_binding.webhook) == 0 && length(data.google_service_account.caller) == 0
    error_message = "Omission creates no resources, caller lookups or public/private invocation grants."
  }
}
run "private_runtime" {
  command = plan
  assert {
    condition     = length(google_cloud_run_v2_service.worker) == 2 && google_cloud_run_v2_service.webhook[0].name == "jayburd-prod-webhook" && length(google_cloud_run_v2_service_iam_binding.webhook) == 0
    error_message = "Separate Production services; receiver stays private without explicit selection."
  }
  assert {
    condition     = google_cloud_run_v2_service_iam_binding.worker["publisher"].members == toset(["serviceAccount:jayburd-prod-scheduler@jayburd-bookkeeping-dev.iam.gserviceaccount.com"]) && google_cloud_run_v2_service_iam_binding.worker["sync"].members == toset(["serviceAccount:jayburd-prod-task-invoker@jayburd-bookkeeping-dev.iam.gserviceaccount.com"]) && contains(google_cloud_run_v2_service.worker["publisher"].template[0].containers[0].args, "-caller-subject=111111111111111111111") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-caller-subject=222222222222222222222")
    error_message = "Private workers authenticate exact distinct caller subjects as well as IAM identities."
  }
  assert {
    condition     = contains(google_cloud_run_v2_service.worker["publisher"].template[0].containers[0].args, "-binding-id=00000000-0000-4000-8000-000000000011") && contains(google_cloud_run_v2_service.worker["publisher"].template[0].containers[0].args, "-publisher-policy=binding-v1") && contains(google_cloud_run_v2_service.worker["publisher"].template[0].containers[0].args, "-queue-name=jayburd-prod-acquisition") && !contains(google_cloud_run_v2_service.worker["publisher"].template[0].containers[0].args, "-bank-credential-key=${local.bank_key}")
    error_message = "Single binding publisher has only dispatch metadata, no bank unwrap flags."
  }
  assert {
    condition     = contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-bank-credential-key=projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/bank-credentials") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-plaid-client-id-version=projects/9088794002/secrets/jayburd-prod-bank-sync-client-id/versions/7") && contains(google_cloud_run_v2_service.webhook[0].template[0].containers[0].args, "-plaid-client-secret-version=projects/9088794002/secrets/jayburd-prod-bank-sync-client-secret/versions/9") && google_cloud_run_v2_service.webhook[0].template[0].containers[0].command == tolist(["/webhook"])
    error_message = "Exact Production keys/static references and dedicated receiver executable."
  }
  assert {
    condition     = alltrue([for s in concat(values(google_cloud_run_v2_service.worker), google_cloud_run_v2_service.webhook) : s.scaling[0].min_instance_count == 0 && s.scaling[0].max_instance_count == 1 && s.template[0].scaling[0].max_instance_count == 1 && !s.invoker_iam_disabled && s.template[0].containers[0].resources[0].cpu_idle && !s.template[0].containers[0].resources[0].startup_cpu_boost && contains(s.template[0].containers[0].args, "-provider-environment=production") && contains(s.template[0].containers[0].args, "-provider-approval-expires-at=${var.runtime.approval_expires_at}") && contains(s.template[0].containers[0].args, "-cloud-sql-instance=jayburd-bookkeeping-dev:us-central1:jayburd-dev")])
    error_message = "All runtimes use fixed Production routing, same finite authority and bounded capacity."
  }
  assert {
    condition     = google_cloud_scheduler_job.publisher[0].paused && google_cloud_scheduler_job.publisher[0].retry_config[0].retry_count == 0 && google_cloud_scheduler_job.publisher[0].http_target[0].oidc_token[0].audience == "https://jayburd-prod-acquisition-publisher-9088794002.us-central1.run.app" && google_cloud_tasks_queue.acquisition[0].rate_limits[0].max_concurrent_dispatches == 1 && google_cloud_tasks_queue.acquisition[0].retry_config[0].max_attempts == 5
    error_message = "Always-paused exact publisher scheduling and bounded queue dispatch/retry."
  }
  assert {
    condition     = google_cloud_tasks_queue_iam_member.publisher[0].member == "serviceAccount:jayburd-prod-publisher@jayburd-bookkeeping-dev.iam.gserviceaccount.com" && google_service_account_iam_member.task_act_as[0].member == "serviceAccount:jayburd-prod-publisher@jayburd-bookkeeping-dev.iam.gserviceaccount.com" && google_service_account_iam_member.task_act_as[0].service_account_id == "projects/jayburd-bookkeeping-dev/serviceAccounts/jayburd-prod-task-invoker@jayburd-bookkeeping-dev.iam.gserviceaccount.com"
    error_message = "Only publisher enqueues and acts as the exact task identity."
  }
}
run "explicit_public_receiver_only" {
  command = plan
  variables { runtime = merge(var.runtime, { public_webhook_approved = true }) }
  assert {
    condition     = google_cloud_run_v2_service_iam_binding.webhook[0].members == toset(["allUsers"]) && google_cloud_run_v2_service_iam_binding.webhook[0].name == "jayburd-prod-webhook" && alltrue([for b in google_cloud_run_v2_service_iam_binding.worker : !contains(b.members, "allUsers")]) && google_cloud_scheduler_job.publisher[0].paused
    error_message = "Explicit receiver exposure never opens workers or resumes scheduling."
  }
}
run "reject_sandbox_image" {
  command = plan
  variables { runtime = merge(var.runtime, { image = "us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" }) }
  expect_failures = [var.runtime]
}
run "reject_secret_alias" {
  command = plan
  variables { runtime = merge(var.runtime, { client_id_version = "latest" }) }
  expect_failures = [var.runtime]
}
run "reject_secret_overflow" {
  command = plan
  variables { runtime = merge(var.runtime, { client_secret_version = "9223372036854775808" }) }
  expect_failures = [var.runtime]
}
run "reject_empty_binding" {
  command = plan
  variables { runtime = merge(var.runtime, { binding_id = "00000000-0000-0000-0000-000000000000" }) }
  expect_failures = [var.runtime]
}
run "reject_expired_authority" {
  command = plan
  variables { runtime = merge(var.runtime, { approval_expires_at = "2020-01-01T00:00:00Z" }) }
  expect_failures = [var.runtime]
}
run "reject_beyond_seven_days" {
  command = plan
  variables { runtime = merge(var.runtime, { approval_expires_at = timeadd(timestamp(), "192h") }) }
  expect_failures = [var.runtime]
}
run "reject_wrong_project" {
  command = plan
  variables {
    runtime    = null
    project_id = "fictional-project"
  }
  expect_failures = [var.project_id]
}
run "reject_wrong_project_number" {
  command = plan
  variables { project_number = "9088794003" }
  expect_failures = [var.project_number]
}
