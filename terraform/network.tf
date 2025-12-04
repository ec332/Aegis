resource "google_project_service" "compute" {
  project            = var.project_id
  service            = "compute.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "vpcaccess" {
  project            = var.project_id
  service            = "vpcaccess.googleapis.com"
  disable_on_destroy = false
}

resource "google_compute_network" "aegis" {
  name                    = "aegis-net"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "aegis" {
  name          = "aegis-subnet"
  ip_cidr_range = "10.10.0.0/24"
  region        = var.region
  network       = google_compute_network.aegis.id
}

resource "google_compute_global_address" "private_range" {
  name          = "aegis-private-range"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.aegis.id
}

resource "google_project_service" "servicenetworking" {
  project            = var.project_id
  service            = "servicenetworking.googleapis.com"
  disable_on_destroy = false
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.aegis.id
  service                 = "services.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_range.name]
  depends_on              = [google_project_service.servicenetworking]
}

resource "google_vpc_access_connector" "aegis" {
  name          = "aegis-connector"
  region        = var.region
  network       = google_compute_network.aegis.name
  ip_cidr_range = "10.8.0.0/28"
  depends_on    = [google_project_service.vpcaccess]
}

