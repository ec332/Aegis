# Terraform Configuration for Aegis Cloud Run Deployment

## Prerequisites

1. Install Google Cloud SDK: `gcloud auth application-default login`
2. Ensure you have appropriate GCP permissions for Cloud Run and IAM
3. Have your container image ready in GCR or Artifact Registry

## Quick Start

1. **Authenticate with GCP:**
   ```bash
   gcloud auth application-default login
   ```

2. **Initialize Terraform:**
   ```bash
   terraform init
   ```

3. **Apply the configuration:**
   ```bash
   terraform apply \
     -var 'project_id=my-gcp-project' \
     -var 'region=us-central1' \
     -var 'service_name=my-service' \
     -var 'image=gcr.io/my-gcp-project/my-image:latest' \
     -var 'allow_unauthenticated=true' \
     -var 'cpu=1' \
     -var 'memory=512Mi' \
     -var 'concurrency=80' \
     -var 'env_vars={ENV="prod",LOG_LEVEL="info"}'
   ```

## Notes

- The service account will be created automatically if not provided
- Minimal IAM roles are granted (logging and monitoring)
- Set `allow_unauthenticated=false` for private services
- Adjust CPU, memory, and concurrency based on your workload