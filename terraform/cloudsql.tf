resource "google_project_service" "sqladmin" {
  project            = var.project_id
  service            = "sqladmin.googleapis.com"
  disable_on_destroy = false
}

resource "random_password" "db" {
  for_each = toset(["api-gateway","market-service","wallet-service","settlement-service","transaction-service"]) 
  length   = 24
  special  = true
}

resource "google_secret_manager_secret" "db_password" {
  for_each  = random_password.db
  secret_id = "db-password-${each.key}"
  replication {
    user_managed {
      replicas { location = var.region }
    }
  }
  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "db_password" {
  for_each    = random_password.db
  secret      = google_secret_manager_secret.db_password[each.key].id
  secret_data = each.value.result
  depends_on  = [google_project_service.secretmanager]
}

# Compose full DATABASE_URL and store as a secret per service
resource "google_secret_manager_secret" "db_url" {
  for_each  = random_password.db
  secret_id = "db-url-${each.key}"
  replication {
    user_managed {
      replicas { location = var.region }
    }
  }
  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "db_url" {
  for_each = random_password.db
  secret   = google_secret_manager_secret.db_url[each.key].id
  secret_data = format(
    "postgres://%s:%s@%s:%d/%s?sslmode=disable",
    google_sql_user.service[each.key].name,
    urlencode(random_password.db[each.key].result),
    google_sql_database_instance.service[each.key].ip_address[0].ip_address,
    5432,
    google_sql_database.service[each.key].name,
  )
  depends_on = [google_project_service.secretmanager]
}

locals {
  db_config = {
    "api-gateway"       = { tier = "db-custom-1-3840" }
    "market-service"    = { tier = "db-custom-1-3840" }
    "wallet-service"    = { tier = "db-custom-1-3840" }
    "settlement-service"= { tier = "db-custom-1-3840" }
    "transaction-service" = { tier = "db-custom-1-3840" }
  }
}

resource "google_sql_database_instance" "service" {
  for_each         = local.db_config
  name             = "${replace(each.key, "_", "-")}-db"
  database_version = "POSTGRES_15"
  region           = var.region
  deletion_protection = false
  settings {
    tier = each.value.tier
    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.aegis.id
    }
  }
  depends_on = [google_service_networking_connection.private_vpc_connection, google_project_service.sqladmin]

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_sql_database" "service" {
  for_each = local.db_config
  name     = "app"
  instance = google_sql_database_instance.service[each.key].name
}

resource "google_sql_user" "service" {
  for_each = local.db_config
  instance = google_sql_database_instance.service[each.key].name
  name     = "appuser"
  password = random_password.db[each.key].result
}
