resource "google_project_service" "redis" {
  project            = var.project_id
  service            = "redis.googleapis.com"
  disable_on_destroy = false
}

resource "google_redis_instance" "aegis" {
  name               = "aegis-redis"
  tier               = "BASIC"
  memory_size_gb     = 1
  region             = var.region
  authorized_network = google_compute_network.aegis.id
  depends_on         = [google_project_service.redis]
}

