# Cloud Run service resource
resource "google_cloud_run_service" "main" {
  name     = var.service_name
  location = var.region

  template {
    spec {
      container_concurrency = var.concurrency
      service_account_name  = local.service_account_email

      containers {
        image = var.image

        env {
          dynamic "block" {
            for_each = var.env_vars
            content {
              name  = block.key
              value = block.value
            }
          }
        }

        resources {
          limits = {
            cpu    = var.cpu
            memory = var.memory
          }
        }
      }

      dynamic "vpc_access" {
        for_each = var.vpc_connector != null ? [1] : []
        content {
          connector = var.vpc_connector
        }
      }
    }

    metadata {
      annotations = merge(
        {
          "autoscaling.knative.dev/maxScale" = var.max_instances != null ? tostring(var.max_instances) : "100"
          "autoscaling.knative.dev/minScale" = var.min_instances != null ? tostring(var.min_instances) : "0"
          "run.googleapis.com/ingress"         = var.ingress
        },
        var.annotations
      )
      labels = var.labels
    }
  }

  traffic {
    percent         = var.traffic_percent
    latest_revision = true
  }

  autogenerate_revision_name = true

  lifecycle {
    ignore_changes = [
      template[0].metadata[0].annotations["run.googleapis.com/client-name"],
      template[0].metadata[0].annotations["run.googleapis.com/client-version"],
    ]
  }
}

locals {
  service_account_email = var.service_account_email != "" ? var.service_account_email : google_service_account.main[0].email
}