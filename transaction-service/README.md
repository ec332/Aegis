# Transaction Service

A standalone Go service for managing transactions in the Aegis platform.

## Overview

This service provides RESTful APIs for transaction management including:
- Create transactions
- List all transactions
- Get transaction by ID
- Update transactions
- Delete transactions

## Architecture

The service follows a clean architecture pattern with:
- **HTTP Handlers**: REST API endpoints
- **Service Layer**: Business logic
- **Repository Layer**: Data access abstraction
- **Models**: Domain entities

## Storage Options

The service supports two storage backends:
- **PostgreSQL**: For production use
- **In-Memory**: For testing and development

## Configuration

The service uses environment variables for configuration:
- `APP_HTTP_PORT`: HTTP server port (default: 5555)
- `APP_DB_HOST`: Database host (default: localhost)
- `APP_DB_PORT`: Database port (default: 5432)
- `APP_DB_NAME`: Database name (default: transaction)
- `APP_DB_USER`: Database user (default: postgres)
- `APP_DB_PASSWORD`: Database password (default: postgres)

## API Endpoints

- `GET /transactions` - List all transactions
- `POST /transactions` - Create a new transaction
- `GET /transactions/{id}` - Get transaction by ID
- `PUT /transactions/{id}` - Update transaction
- `DELETE /transactions/{id}` - Delete transaction

## Running the Service

```bash
go mod download
go run cmd/transaction-service/main.go
```