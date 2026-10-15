# Isolated one-shot proof; no SQL grants, secrets, public service or scheduler.
resource "google_cloud_run_v2_job" "probe" {
  project             = var.project_id
  name                = "jayburd-dev-database-probe"
  location            = var.region
  deletion_protection = true
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = "jayburd-dev-api@${var.project_id}.iam.gserviceaccount.com"
      timeout         = "120s"
      max_retries     = 0
      containers {
        image   = var.image
        command = ["/database-probe"]
        args    = ["-action=probe", "-instance=${var.project_id}:${var.region}:jayburd-dev"]
        resources { limits = { cpu = "1", memory = "512Mi" } }
      }
    }
  }
  lifecycle { prevent_destroy = true }
}
