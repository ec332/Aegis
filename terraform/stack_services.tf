locals {
  service_images = {
    api-gateway        = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/api-gateway:latest"
    market-service     = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/market:latest"
    wallet-service     = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/wallet:latest"
    settlement-service = "${var.region}-docker.pkg.dev/${var.project_id}/aegis/settlement:latest"
    transaction-service= "${var.region}-docker.pkg.dev/${var.project_id}/aegis/transaction-service:latest"
  }
  service_ports = {
    api-gateway        = 8080
    market-service     = 50051
    wallet-service     = 50052
    settlement-service = 50053
    transaction-service= 50052
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

        ports {
          container_port = local.service_ports[each.key]
        }


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
          name = "DATABASE_URL"
          value_from {
            secret_key_ref {
              name = "db-url-${each.key}"
              key  = "latest"
            }
          }
        }
        env {
          name = "DB_PASSWORD"
          value_from {
            secret_key_ref {
              name = "db-password-${each.key}"
              key  = "latest"
            }
          }
        }
        env {
          name = "WALLET_DATABASE_URL"
          value_from {
            secret_key_ref {
              name = "db-url-${each.key}"
              key  = "latest"
            }
          }
        }
        env {
          name = "APP_DB_PASSWORD"
          value_from {
            secret_key_ref {
              name = "db-password-${each.key}"
              key  = "latest"
            }
          }
        }

        env {
          name  = "GRPC_PORT"
          value = tostring(local.service_ports[each.key])
        }
        env {
          name  = "WALLET_SERVICE_PORT"
          value = tostring(local.service_ports[each.key])
        }
        env {
          name  = "APP_GRPC_PORT"
          value = tostring(local.service_ports[each.key])
        }

        env {
          name  = "REDIS_HOST"
          value = google_redis_instance.aegis.host
        }
        env {
          name  = "REDIS_PORT"
          value = tostring(google_redis_instance.aegis.port)
        }
        env {
          name  = "REDIS_URL"
          value = format("redis://%s:%s", google_redis_instance.aegis.host, tostring(google_redis_instance.aegis.port))
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
      annotations = merge(
        {
          "run.googleapis.com/vpc-access-connector" = "projects/${var.project_id}/locations/${var.region}/connectors/${google_vpc_access_connector.aegis.name}"
          "run.googleapis.com/vpc-access-egress"    = "private-ranges-only"
        },
        (
          var.max_instances != null
          ? { "autoscaling.knative.dev/maxScale" = tostring(var.max_instances) }
          : {}
        ),
        (
          var.min_instances != null
          ? { "autoscaling.knative.dev/minScale" = tostring(var.min_instances) }
          : {}
        )
      )
      labels = var.labels
    }
  }

  metadata {
    annotations = {
      "run.googleapis.com/ingress" = var.ingress
    }
    labels = var.labels
  }

  traffic {
    percent         = var.traffic_percent
    latest_revision = true
  }
  autogenerate_revision_name = true

  depends_on = [
    google_project_service.run,
    google_project_service.compute,
    google_project_service.vpcaccess,
    google_project_service.sqladmin,
    google_project_service.redis,
    google_project_service.secretmanager,
    google_vpc_access_connector.aegis,
  ]
}

resource "google_cloud_run_service_iam_member" "api_public" {
  service  = google_cloud_run_service.svc["api-gateway"].name
  location = google_cloud_run_service.svc["api-gateway"].location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
