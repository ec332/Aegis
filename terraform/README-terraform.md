# Terraform Configuration for Aegis Cloud Run Deployment

## Prerequisites

1. Install Google Cloud SDK: `brew install --cask google-cloud-sdk` and init with `gcloud init`
2. Ensure you have appropriate GCP permissions for Cloud Run, IAM, VPC, Cloud SQL, and Redis
3. Have your container image ready in Artifact Registry

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
     -var 'region=asia-southeast1' \
     -var 'service_name=my-service' \
     -var 'image=asia-southeast1-docker.pkg.dev/my-gcp-project/aegis/my-image:latest' \
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

## Region Selection (Singapore)

- Use `asia-southeast1` for Cloud Run, Cloud SQL, Redis, and Artifact Registry.
- Image URIs: `asia-southeast1-docker.pkg.dev/<PROJECT_ID>/aegis/<service>:latest`.

## Before You Start

- Set credentials for Terraform using your local JSON:
  ```bash
  export GOOGLE_APPLICATION_CREDENTIALS="/absolute/path/to/your-service-account.json"
  ```
- Ensure the service account used by Terraform has roles:
  - `roles/run.admin`, `roles/iam.serviceAccountAdmin`
  - `roles/artifactregistry.reader`, `roles/compute.networkAdmin`, `roles/sql.admin`, `roles/redis.admin`

## Build and Push Container Images (Local alternative)

```bash
gcloud auth configure-docker asia-southeast1-docker.pkg.dev
docker build -t asia-southeast1-docker.pkg.dev/<PROJECT_ID>/aegis/api-gateway:latest -f api-gateway/Dockerfile .
docker push asia-southeast1-docker.pkg.dev/<PROJECT_ID>/aegis/api-gateway:latest
```

## What to Expect

- Resources created: VPC, VPC connector, per‑service Cloud SQL (private IP), Redis, five Cloud Run services. Artifact Registry repos are managed outside of Terraform (CI/Console).
- Verify the gateway:
  ```bash
  curl https://<api-gateway-url>/health
  ```

## Troubleshooting

- Image not found: ensure AR image exists and tag matches.
- Permission denied: verify service account roles listed above.
- Unauthenticated errors: set `allow_unauthenticated=true` for public endpoints or grant `roles/run.invoker` to callers.
- API enablement: enable required APIs once via Console or CLI:
  ```bash
  gcloud services enable run.googleapis.com compute.googleapis.com sqladmin.googleapis.com redis.googleapis.com secretmanager.googleapis.com artifactregistry.googleapis.com
  ```
