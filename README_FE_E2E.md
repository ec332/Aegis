# Frontend E2E (Playwright)

## Overview

Browser tests cover key flows, accessibility checks (axe), and keyboard interactions. HTML reports are produced and uploaded in CI.

## Prerequisites

- Node.js 20+
- npm
- Playwright browsers

## Install & Build

- `cd aegis-fe && npm ci`
- Install browsers:
  - `npx playwright install --with-deps`
- Build app:
  - `npm run build`

## Run Tests

- Run all e2e tests with HTML report:
  - `npx playwright test --reporter=html`
- Open report locally:
  - `npx playwright show-report`

## Accessibility

- Axe scan included in `e2e/accessibility.spec.ts`
- Serious/Critical violations fail the test.

## Visual Regression

- Baseline-less snapshot test included (`visual.spec.ts`) and skipped in CI by default.
- To enable locally:
  - Remove `test.skip` and run `npx playwright test` to create and compare images.

## Key Test Files

- `e2e/home.spec.ts`: homepage render, modal keyboard/open/close, outside click.
- `e2e/profile.spec.ts`: profile menu Escape/outside-click and focus return.
- `e2e/accessibility.spec.ts`: axe-based checks.
- `e2e/visual.spec.ts`: visual snapshot example.

## Troubleshooting

- If tests fail to launch, ensure port `3000` is free.
- For flaky animations, increase timeouts or disable transitions in test env.

