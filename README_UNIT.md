# Unit Tests

## Overview

Unit tests cover individual functions and components across backend (Go) and frontend (Next.js/TypeScript). They run fast, in-memory, and enforce coverage thresholds.

## Prerequisites

- Go 1.22+
- Node.js 20+
- npm

## Backend (Go)

- Run all unit tests:
  - `go test ./...`
- With coverage report:
  - `go test -coverprofile=coverage.out ./...`
  - `go tool cover -func=coverage.out`
- Thresholds (CI):
  - Minimum total coverage: `70%` (build fails below threshold)
- Artifacts (CI):
  - `coverage.out`
  - `coverage.txt` summary

## Frontend (Vitest)

- Install deps:
  - `cd aegis-fe && npm ci`
- Run unit tests:
  - `npm test`
- Coverage report (thresholds enforced):
  - `npm run test:coverage`
  - HTML report: `aegis-fe/coverage/index.html`
- Thresholds:
  - Lines, functions, statements: `>= 80%`
  - Branches: `>= 70%`

## Common Troubleshooting

- Ensure environment variables are not required for unit tests; mocks are used.
- If coverage fails, inspect low-coverage packages and add focused tests.

