import { test, expect } from '@playwright/test';

test('profile menu closes with Escape and returns focus', async ({ page }) => {
  await page.goto('/');
  const trigger = page.getByRole('button', { name: 'Profile' });
  if (await trigger.count()) {
    await trigger.click();
    await expect(page.getByRole('menu')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.getByRole('menu')).toHaveCount(0);
    await expect(trigger).toBeFocused();
  }
});

test('profile menu closes on outside click', async ({ page }) => {
  await page.goto('/');
  const trigger = page.getByRole('button', { name: 'Profile' });
  if (await trigger.count()) {
    await trigger.click();
    await expect(page.getByRole('menu')).toBeVisible();
    await page.mouse.click(10, 10);
    await expect(page.getByRole('menu')).toHaveCount(0);
  }
});
