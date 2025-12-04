# IAM binding for unauthenticated access (if enabled)
resource "google_cloud_run_service_iam_member" "public_invoker" {
  count    = var.allow_unauthenticated && var.enable_single_service ? 1 : 0
  service  = google_cloud_run_service.main[0].name
  location = google_cloud_run_service.main[0].location
  role     = "roles/run.invoker"
  member   = "allUsers"

  depends_on = [google_cloud_run_service.main]
}
