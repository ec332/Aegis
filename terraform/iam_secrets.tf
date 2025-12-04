resource "google_project_iam_member" "secret_accessor_svc" {
  for_each = google_service_account.svc
  project  = var.project_id
  role     = "roles/secretmanager.secretAccessor"
  member   = "serviceAccount:${each.value.email}"
}

resource "google_project_iam_member" "secret_accessor_single" {
  count   = var.enable_single_service && local.service_account_email != "" ? 1 : 0
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${local.service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "db_url_access" {
  for_each = google_service_account.svc
  secret_id = google_secret_manager_secret.db_url[each.key].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${each.value.email}"
}

resource "google_secret_manager_secret_iam_member" "db_password_access" {
  for_each = google_service_account.svc
  secret_id = google_secret_manager_secret.db_password[each.key].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${each.value.email}"
}
