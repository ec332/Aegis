# Coverage & Reports

## Overview

Coverage thresholds and HTML reports are enforced for both backend and frontend. CI uploads artifacts for inspection.

## Backend (Go)

- Generate coverage locally:
  - `go test -coverprofile=coverage.out ./...`
  - `go tool cover -func=coverage.out | tee coverage.txt`
- Threshold (CI): `>= 70%` total coverage required.
- Artifacts (CI): `coverage.out`, `coverage.txt`.

## Frontend (Vitest)

- Coverage thresholds in `vitest.config.ts`:
  - Lines/functions/statements: `>= 80%`, branches: `>= 70%`.
- Run locally:
  - `cd aegis-fe && npm run test:coverage`
- HTML report path:
  - `aegis-fe/coverage/index.html`

## Playwright Reports

- HTML e2e report uploaded in CI as `playwright-report`.
- Local open:
  - `cd aegis-fe && npx playwright show-report`

## Tips

- Focus tests on untested core logic to increase coverage.
- Mock external dependencies and use table-driven tests for breadth.

