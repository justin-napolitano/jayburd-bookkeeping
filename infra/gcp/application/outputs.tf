output "instance_connection_name" {
  description = "Nonsecret connector target, never a password-bearing URL."
  value       = google_sql_database_instance.ledger.connection_name
}
output "database_name" {
  description = "Database requiring owner bootstrap before migrations/runtime readiness."
  value       = google_sql_database.ledger.name
}
