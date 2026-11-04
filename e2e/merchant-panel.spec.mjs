import { expect, test } from '@playwright/test';

const OWNER_PHONE = '+998900000007'; // owner-d: only this spec uses it

/** Signs in as the seeded merchant owner through the phone code form (development shows the code on the page). */
async function signInOwner(page) {
  await page.goto('/uz/login');
  await page.getByLabel(/Telefon raqami/).fill(OWNER_PHONE.slice(4));
  await page.getByRole('button', { name: /Kod yuborish/ }).click();
  const hint = page.getByText(/Test rejimi kodi: \d{6}/);
  await expect(hint).toBeVisible();
  await page.getByLabel(/Tasdiqlash kodi/).fill((await hint.textContent()).match(/\d{6}/)[0]);
  await page.getByRole('button', { name: /^Kirish$/ }).click();
  await page.waitForURL((url) => !url.pathname.endsWith('/login'));
}

// Each new panel section must render its heading, not the error boundary.
const SECTIONS = [
  { path: 'staff', heading: /Xodimlar/ },
  { path: 'payouts', heading: /To'lovlar/ },
  { path: 'orders', heading: /Buyurtmalar/ },
  { path: 'customers', heading: /Mijozlar/ },
  { path: 'analytics', heading: /Tahlil/ },
  { path: 'reviews', heading: /Sharhlar/ },
  { path: 'settings', heading: /Sozlamalar/ },
  { path: 'calculator', heading: /Aksiya kalkulyatori/ },
];

test('owner opens every merchant panel section without an error', async ({ page }) => {
  await signInOwner(page);
  for (const section of SECTIONS) {
    await page.goto(`/uz/merchant/${section.path}`);
    await expect(page.getByRole('heading', { name: section.heading })).toBeVisible();
    await expect(page.getByText('Something went wrong')).toHaveCount(0);
  }
});

test('calculator shows what the merchant receives per voucher', async ({ page }) => {
  await signInOwner(page);
  await page.goto('/uz/merchant/calculator');
  await page.getByLabel(/Asl narx/).fill('100000');
  await page.getByLabel(/Aksiya narxi/).fill('50000');
  await page.getByLabel(/Vaučerlar soni/).fill('10');
  await page.getByLabel(/Komissiya/).fill('10');
  await expect(page.getByText('Chegirma')).toBeVisible();
  await expect(page.getByText('50%')).toBeVisible();
});

test('settings can be saved', async ({ page }) => {
  await signInOwner(page);
  await page.goto('/uz/merchant/settings');
  await page.getByLabel(/Yangi buyurtmalar/).check();
  await page.getByRole('button', { name: /^Saqlash$/ }).click();
  await expect(page.getByText('Sozlamalar saqlandi.')).toBeVisible();
});
