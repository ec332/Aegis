variable "project_id" {
  type        = string
  description = "GCP project ID"
}

variable "region" {
  type        = string
  description = "GCP region for Cloud Run (e.g., us-central1)"
}

variable "service_name" {
  type        = string
  description = "Cloud Run service name"
  default     = ""
}

variable "image" {
  type        = string
  description = "Container image URI (e.g., gcr.io/my-project/my-image:tag)"
  default     = ""
}

variable "allow_unauthenticated" {
  type        = bool
  description = "Whether the service is publicly invokable"
  default     = false
}

variable "cpu" {
  type        = string
  description = "CPU allocation (e.g., '1', '2')"
  default     = "1"
}

variable "memory" {
  type        = string
  description = "Memory allocation (e.g., '512Mi', '1Gi')"
  default     = "512Mi"
}

variable "concurrency" {
  type        = number
  description = "Request concurrency per container"
  default     = 80
}

variable "env_vars" {
  type        = map(string)
  description = "Environment variables for container"
  default     = {}
}

variable "service_account_email" {
  type        = string
  description = "Service account to run the service as (optional)"
  default     = ""
}

variable "max_instances" {
  type        = number
  description = "Maximum number of instances (optional)"
  default     = null
}

variable "min_instances" {
  type        = number
  description = "Minimum number of instances (optional)"
  default     = null
}

variable "vpc_connector" {
  type        = string
  description = "Name of Serverless VPC Connector to attach (optional)"
  default     = null
}

variable "ingress" {
  type        = string
  description = "Ingress settings: 'all', 'internal', 'internal-and-cloud-load-balancing'"
  default     = "all"
}

variable "traffic_percent" {
  type        = number
  description = "Traffic split percentage (0-100)"
  default     = 100
}

variable "labels" {
  type        = map(string)
  description = "Labels to apply to the service"
  default     = {}
}

variable "annotations" {
  type        = map(string)
  description = "Annotations to apply to the service"
  default     = {}
}

variable "enable_single_service" {
  type        = bool
  description = "Enable single-service deployment via main.tf"
  default     = false
}

 
