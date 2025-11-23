# IAM binding for unauthenticated access (if enabled)
resource "google_cloud_run_service_iam_member" "public_invoker" {
  count    = var.allow_unauthenticated ? 1 : 0
  service  = google_cloud_run_service.main.name
  location = google_cloud_run_service.main.location
  role     = "roles/run.invoker"
  member   = "allUsers"

  depends_on = [google_cloud_run_service.main]
}