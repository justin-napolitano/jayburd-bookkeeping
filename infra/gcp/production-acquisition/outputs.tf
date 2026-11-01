output "worker_origins" { value = var.runtime == null ? null : local.origins }
output "webhook_origin" { value = var.runtime == null ? null : local.webhook_origin }
output "scheduler_name" { value = var.runtime == null ? null : google_cloud_scheduler_job.publisher[0].name }
output "queue_name" { value = var.runtime == null ? null : google_cloud_tasks_queue.acquisition[0].name }
