import { expect, test } from '@playwright/test';

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';

/** Finds a voucher deal with at least one sellable unit, using the public API. */
async function sellableVoucherDeal() {
  const response = await fetch(`${API}/deals?page_size=100&sort=popular`);
  const page = await response.json();
  for (const deal of page.items) {
    if (deal.kind !== 'voucher' || deal.status !== 'active') continue;
    const option = deal.options.find((item) => item.is_active && item.stock_total - item.stock_sold - item.stock_reserved > 0);
    if (option) return { slug: deal.slug, title: deal.title.uz };
  }
  throw new Error('no sellable voucher deal in seed data');
}

test('browse → login mid-flow → buy → voucher visible', async ({ page, request }) => {
  const deal = await sellableVoucherDeal();
  const email = `e2e${Date.now()}@example.com`;
  const password = 'Passw0rd!long';

  // A user who exists before the buyer arrives; they sign in only at checkout.
  const registered = await request.post(`${API}/auth/register`, {
    data: { email, password, name: 'E2E Buyer', locale: 'uz', currency: 'UZS' },
  });
  expect(registered.status()).toBe(201);

  // Browse the deal page without an account and start buying.
  await page.goto(`/uz/deal/${deal.slug}`);
  await page.getByRole('button', { name: /Hozir xarid qilish/ }).click();

  // Checkout requires a session, so the visitor lands on login with the return path.
  await expect(page).toHaveURL(/\/uz\/login\?next=%2Fcheckout|\/uz\/login\?next=\/checkout/);
  await page.getByLabel(/Elektron pochta/).fill(email);
  await page.getByLabel(/Parol/).fill(password);
  await page.getByRole('button', { name: /^Kirish$/ }).click();

  // Back on checkout with the cart preserved.
  await expect(page).toHaveURL(/\/uz\/checkout$/);
  await page.getByRole('checkbox').check();
  await page.getByRole('button', { name: /To.lovga o.tish/ }).click();

  // Test payment succeeds; the success page polls until the webhook marks the order paid.
  // Creating the order and the payment session can take a few seconds on a cold server.
  await expect(page).toHaveURL(/\/uz\/checkout\/mock\//, { timeout: 20000 });
  await page.getByRole('button', { name: /Muvaffaqiyatli to.lash/ }).click();
  await expect(page.getByRole('heading', { name: /Xarid muvaffaqiyatli/ })).toBeVisible({ timeout: 20000 });

  // The voucher, with its human code, is visible in the account.
  await page.getByRole('link', { name: /Mening vauchelarim/ }).click();
  await expect(page.getByText(/[A-Z0-9]{4}-[A-Z0-9]{4}/).first()).toBeVisible();
});
