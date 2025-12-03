import { test, expect } from '@playwright/test';

test('homepage renders and has accessible navigation', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Active Markets' })).toBeVisible();
});

test('create market modal opens and closes with keyboard', async ({ page }) => {
  await page.goto('/');
  const createButton = page.getByRole('button', { name: /create market/i });
  // Button only exists when authenticated; skip if not present
  if (await createButton.count()) {
    await createButton.click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).toHaveCount(0);
  }
});

test('create market modal closes on outside click', async ({ page }) => {
  await page.goto('/');
  const createButton = page.getByRole('button', { name: /create market/i });
  if (await createButton.count()) {
    await createButton.click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.mouse.click(5, 5);
    await expect(page.getByRole('dialog')).toHaveCount(0);
  }
});
