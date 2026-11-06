import { expect, test } from '@playwright/test';

test('buyer stays signed in when opening the account menu', async ({ page }) => {
  await page.goto('/uz/login');
  await expect(page.getByText('Demo akkauntlar')).toBeVisible();
  await page.getByRole('button', { name: /Xaridor.*998900000001/ }).click();
  await expect(page.getByLabel(/Telefon raqami/)).toHaveValue('900000001');
  await page.getByRole('button', { name: /Kod yuborish/ }).click();
  const hint = page.getByText(/Test rejimi kodi: \d{6}/);
  await expect(hint).toBeVisible();
  const code = (await hint.textContent()).match(/\d{6}/)[0];
  await page.getByLabel(/Tasdiqlash kodi/).fill(code);
  await page.getByRole('button', { name: /^Kirish$/ }).click();
  await expect(page).toHaveURL(/\/uz$/);

  await page.getByLabel('Profil').click();
  await expect(page.getByTestId('role-badge')).toHaveText(/Xaridor/);
  await expect(page.getByText('Aziz — xaridor')).toBeVisible();
  await expect(page).not.toHaveURL(/\/login/);
});
