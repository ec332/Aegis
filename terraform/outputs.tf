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