import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test('homepage has no critical accessibility violations', async ({ page }) => {
  await page.goto('/');
  const accessibilityScanResults = await new AxeBuilder({ page }).analyze();
  // Allow minor issues but assert no serious or critical
  const serious = accessibilityScanResults.violations.filter(v => ['serious', 'critical'].includes(v.impact || 'minor'));
  expect(serious).toEqual([]);
});
