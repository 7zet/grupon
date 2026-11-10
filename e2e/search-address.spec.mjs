import { expect, test } from '@playwright/test';

// Address search: the suggestion list offers street addresses, and a search by street finds the deals
// on that street. Each deal card shows the business address under its title.

test('typing a street offers addresses, and the results show the address on each card', async ({ page }) => {
  await page.goto('/uz', { waitUntil: 'load' });
  const box = page.locator('header input[name="q"]:visible').first();
  await box.fill('Amir Temur');
  const suggestion = page.locator('header form a').filter({ hasText: /Amir Temur/ }).first();
  await expect(suggestion).toBeVisible({ timeout: 10000 });
  await box.press('Enter');
  await expect(page).toHaveURL(/\/uz\/search\?q=Amir/);
  await expect(page.locator('article').first()).toBeVisible();
  await expect(page.locator('article p[title]').first()).toBeVisible();
});
