output "service_url" {
  description = "URL of the deployed Cloud Run service"
  value       = google_cloud_run_service.main.status[0].url
}

output "service_name" {
  description = "Name of the Cloud Run service"
  value       = google_cloud_run_service.main.name
}

output "service_account_email" {
  description = "Email of the service account used by the Cloud Run service"
  value       = local.service_account_email
}

output "stack_service_urls" {
  description = "URLs for all Cloud Run services in the stack"
  value       = { for k, v in google_cloud_run_service.svc : k => v.status[0].url }
}

output "stack_service_accounts" {
  description = "Service account emails per service"
  value       = { for k, v in google_service_account.svc : k => v.email }
}

output "stack_db_private_ips" {
  description = "Private IPs for Cloud SQL instances per service"
  value       = { for k, v in google_sql_database_instance.service : k => v.private_ip_address }
}

output "stack_db_users" {
  description = "Database user names per service"
  value       = { for k, v in google_sql_user.service : k => v.name }
}

output "redis_endpoint" {
  description = "Redis host and port"
  value       = {
    host = google_redis_instance.aegis.host
    port = google_redis_instance.aegis.port
  }
}
