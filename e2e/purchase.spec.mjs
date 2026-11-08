import { expect, test } from '@playwright/test';

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';

/** Finds a voucher deal with at least one sellable unit, using the public API. */
async function sellableVoucherDeal() {
  const response = await fetch(`${API}/deals?page_size=100&sort=popular`);
  const page = await response.json();
  for (const deal of page.items) {
    if (deal.status !== 'active') continue;
    const optionIndex = deal.options.findIndex((item) => item.is_active && item.stock_total - item.stock_sold - item.stock_reserved > 0);
    if (optionIndex >= 0) {
      const option = deal.options[optionIndex];
      return { slug: deal.slug, title: deal.title.uz, optionIndex, optionName: option.name.uz, unitPrice: option.deal_price.amount };
    }
  }
  throw new Error('no sellable voucher deal in seed data');
}

/** A fresh Uzbekistan number for each run, so every buyer is a new account. */
function freshDigits() {
  const tail = String(Math.floor(Math.random() * 1e7)).padStart(7, '0');
  return `91${tail}`;
}

test('browse → login mid-flow → buy → voucher visible', async ({ page }) => {
  const deal = await sellableVoucherDeal();
  const digits = freshDigits();

  // Browse the deal page without an account and start buying.
  await page.goto(`/uz/deal/${deal.slug}`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('radio').nth(deal.optionIndex).check();
  await page.getByRole('button', { name: '+' }).first().click();
  await page.getByRole('button', { name: /Hozir xarid qilish/ }).click();

  // Checkout requires a session, so the visitor lands on login with the return path.
  await expect(page).toHaveURL(/\/uz\/login\?next=%2Fcheckout|\/uz\/login\?next=\/checkout/);
  await page.getByLabel(/Telefon raqami/).fill(digits);
  await page.getByRole('button', { name: /Kod yuborish/ }).click();

  // Development shows the code on the page; a real phone would receive it by SMS.
  const hint = page.getByText(/Test rejimi kodi: \d{6}/);
  await expect(hint).toBeVisible();
  const code = (await hint.textContent()).match(/\d{6}/)[0];
  await page.getByLabel(/Tasdiqlash kodi/).fill(code);
  await page.getByRole('button', { name: /^Kirish$/ }).click();

  // Back on checkout with the cart preserved.
  await expect(page).toHaveURL(/\/uz\/checkout$/);
  // The deal, the chosen option and the quantity survive the login detour.
  const summary = page.locator('#summary-heading').locator('xpath=..');
  await expect(summary).toContainText(deal.title);
  await expect(summary).toContainText(deal.optionName);
  await expect(summary).toContainText(/Miqdor:\s*2/);
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
