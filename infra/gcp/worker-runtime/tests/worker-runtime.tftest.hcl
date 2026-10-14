mock_provider "google" {
  mock_resource "google_service_account" {
    defaults = {
      email = "jayburd-dev-sync@jayburd-synthetic-test.iam.gserviceaccount.com"
    }
  }
  mock_data "google_service_account" {
    defaults = {
      name      = "projects/jayburd-synthetic-test/serviceAccounts/jayburd-dev-task-invoker@jayburd-synthetic-test.iam.gserviceaccount.com"
      unique_id = "123456789012345678901"
    }
  }
}
variables {
  project_id     = "jayburd-synthetic-test"
  project_number = "123456789012"
  runtime = {
    image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    binding_id            = "00000000-0000-4000-8000-000000000001"
    client_id_version     = "7"
    client_secret_version = "23"
  }
}
run "runtime_is_private_bounded_and_paused" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "7"
      client_secret_version = "23"
    }
  }
  assert {
    condition     = length(google_cloud_run_v2_service.worker) == 2 && alltrue([for s in google_cloud_run_v2_service.worker : !s.invoker_iam_disabled && s.deletion_protection && s.template[0].max_instance_request_concurrency == 1 && s.template[0].timeout == "90s" && s.template[0].scaling[0].max_instance_count == 1])
    error_message = "Runtime must retain IAM invocation, bounded capacity, deadline and paused scheduling."
  }
  assert {
    condition     = contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-project-number=123456789012") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-credential-mapping-prefix=provider-credentials-v1") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-plaid-client-id-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/7") && contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-plaid-client-secret-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/23")
    error_message = "Consumer uses canonical numeric resources and the frozen immutable mapping prefix."
  }

}
run "reject_mutable_image" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker:latest"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "1"
      client_secret_version = "1"
    }
  }
  expect_failures = [var.runtime]
}
run "reject_alias_version" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "latest"
      client_secret_version = "1"
    }
  }
  expect_failures = [var.runtime]
}
run "reject_unbounded_version" {
  command = plan
  variables {
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "9223372036854775808"
      client_secret_version = "1"
    }
  }
  expect_failures = [var.runtime]
}
run "canonical_int64_upper_boundary" {
  command = plan
  variables {
    project_number = "9223372036854775807"
    runtime = {
      image                 = "us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      binding_id            = "00000000-0000-4000-8000-000000000001"
      client_id_version     = "9223372036854775807"
      client_secret_version = "9223372036854775807"
    }
  }
  assert {
    condition     = contains(google_cloud_run_v2_service.worker["sync"].template[0].containers[0].args, "-project-number=9223372036854775807")
    error_message = "Manifest canonical int64 bounds must survive Terraform validation."
  }
}
run "reject_project_overflow" {
  command = plan
  variables { project_number = "9223372036854775808" }
  expect_failures = [var.project_number]
}
run "reject_unreviewed_region" {
  command = plan
  variables { region = "europe-west1" }
  expect_failures = [var.region]
}
