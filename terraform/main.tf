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

        dynamic "env" {
          for_each = var.env_vars
          content {
            name  = env.key
            value = env.value
          }
        }

        resources {
          limits = {
            cpu    = var.cpu
            memory = var.memory
          }
        }
      }

      # vpc_access block removed due to dynamic block issue
      # dynamic "vpc_access" {
      #   for_each = var.vpc_connector != null && var.vpc_connector != "" ? [1] : []
      #   content {
      #     connector = var.vpc_connector
      #   }
      # }
    }

    metadata {
      annotations = merge(
        {
          "autoscaling.knative.dev/maxScale" = var.max_instances != null ? tostring(var.max_instances) : "100"
          "autoscaling.knative.dev/minScale" = var.min_instances != null ? tostring(var.min_instances) : "0"
          "run.googleapis.com/ingress"       = var.ingress
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

  depends_on = [
    google_project_service.run,
    google_project_service.containerregistry,
  ]
}

resource "google_service_account" "main" {
  count        = var.service_account_email == "" ? 1 : 0
  account_id   = "${var.service_name}-sa"
  display_name = "Service account for ${var.service_name}"
  project      = var.project_id
}

locals {
  service_account_email = var.service_account_email != "" ? var.service_account_email : google_service_account.main[0].email
}
