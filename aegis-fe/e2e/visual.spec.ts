import { test, expect } from '@playwright/test';

test.skip(process.env.CI ? 'skip visual snapshot in CI' : 'home visual snapshot', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveScreenshot({ maxDiffPixelRatio: 0.02 });
});
