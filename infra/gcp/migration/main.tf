# This independent state lets a failed migration stop before either service configuration changes.
resource "google_cloud_run_v2_job" "migrate" {
  project             = var.project_id
  name                = "jayburd-dev-migrate"
  location            = var.region
  deletion_protection = true
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = "jayburd-dev-migrator@${var.project_id}.iam.gserviceaccount.com"
      timeout         = "660s"
      max_retries     = 0
      containers {
        image = var.image
        args  = ["-action=${var.action}", "-instance=${var.project_id}:${var.region}:jayburd-dev"]
        resources { limits = { cpu = "1", memory = "512Mi" } }
      }
    }
  }
  lifecycle { prevent_destroy = true }
}
