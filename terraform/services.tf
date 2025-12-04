resource "google_project_service" "run" {
  project            = var.project_id
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "containerregistry" {
  project            = var.project_id
  service            = "containerregistry.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "artifactregistry" {
  project            = var.project_id
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "secretmanager" {
  project            = var.project_id
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "aegis" {
  location      = var.region
  repository_id = "aegis"
  format        = "DOCKER"
}

resource "google_artifact_registry_repository" "aegis_cache" {
  location      = var.region
  repository_id = "aegis-cache"
  format        = "DOCKER"
}
