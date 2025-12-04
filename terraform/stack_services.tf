locals {
  service_images = {
    api-gateway        = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/api-gateway:latest"
    market-service     = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/market-service:latest"
    wallet-service     = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/wallet-service:latest"
    settlement-service = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/settlement-service:latest"
    transaction-service= "${var.region}-docker.pkg.dev/${var.project_id}/aegis/transaction-service:latest"
  }
}

resource "google_service_account" "svc" {
  for_each    = local.service_images
  account_id  = each.key
  display_name= "Service account for ${each.key}"
  project     = var.project_id
}

resource "google_cloud_run_service" "svc" {
  for_each = local.service_images
  name     = each.key
  location = var.region

  template {
    spec {
      container_concurrency = var.concurrency
      service_account_name  = google_service_account.svc[each.key].email

      containers {
        image = each.value

        env {
          name  = "DB_HOST"
          value = google_sql_database_instance.service[each.key].ip_address[0].ip_address
        }
        env {
          name  = "DB_PORT"
          value = "5432"
        }
        env {
          name  = "DB_NAME"
          value = google_sql_database.service[each.key].name
        }
        env {
          name  = "DB_USER"
          value = google_sql_user.service[each.key].name
        }

        env {
          name  = "REDIS_HOST"
          value = google_redis_instance.aegis.host
        }
        env {
          name  = "REDIS_PORT"
          value = tostring(google_redis_instance.aegis.port)
        }

        resources {
          limits = {
            cpu    = var.cpu
            memory = var.memory
          }
        }
      }
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = var.max_instances != null ? tostring(var.max_instances) : "100"
        "autoscaling.knative.dev/minScale" = var.min_instances != null ? tostring(var.min_instances) : "0"
        "run.googleapis.com/ingress"       = var.ingress
        "run.googleapis.com/secrets"       = "DB_PASSWORD=db-password-${each.key}:latest"
        "run.googleapis.com/vpc-access-connector" = google_vpc_access_connector.aegis.name
      }
      labels = var.labels
    }
  }

  traffic {
    percent         = var.traffic_percent
    latest_revision = true
  }
  autogenerate_revision_name = true

  depends_on = [
    google_project_service.run,
    google_project_service.artifactregistry,
    google_project_service.compute,
    google_project_service.vpcaccess,
    google_project_service.sqladmin,
    google_project_service.redis,
    google_project_service.secretmanager,
    google_artifact_registry_repository.aegis,
    google_vpc_access_connector.aegis,
  ]
}

resource "google_cloud_run_service_iam_member" "api_public" {
  service  = google_cloud_run_service.svc["api-gateway"].name
  location = google_cloud_run_service.svc["api-gateway"].location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

