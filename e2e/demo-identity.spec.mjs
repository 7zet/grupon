import { expect, test } from '@playwright/test';

test('buyer stays signed in when opening the account menu', async ({ page }) => {
  await page.goto('/uz/login');
  await expect(page.getByText('Demo akkauntlar')).toHaveCount(0);
  await page.getByLabel(/Telefon raqami/).fill('900000001');
  await page.getByRole('button', { name: /Kod yuborish/ }).click();
  const hint = page.getByText(/Test rejimi kodi: \d{6}/);
  await expect(hint).toBeVisible();
  const code = (await hint.textContent()).match(/\d{6}/)[0];
  await page.getByLabel(/Tasdiqlash kodi/).fill(code);
  await page.getByRole('button', { name: /^Kirish$/ }).click();
  await expect(page).toHaveURL(/\/uz$/);

  // The header loads the signed-in user after the page opens; wait for it before using the menu.
  await expect(page.getByTestId('role-badge')).toHaveText(/Xaridor/);
  // Two profile controls exist (top bar and phone bar); exactly one is visible at this width.
  await page.locator('summary[aria-label="Profil"]:visible').click();
  await expect(page.getByText('Aziz — xaridor').first()).toBeVisible();
  await expect(page).not.toHaveURL(/\/login/);
});
