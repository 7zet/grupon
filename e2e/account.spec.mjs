import { expect, test } from '@playwright/test';

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';

/** Every buyer account page must load without the generic error screen (regression: /me/reviews returned 501). */
const PAGES = ['/uz/account', '/uz/account/coupons', '/uz/account/orders', '/uz/account/reviews', '/uz/account/profile'];

test('buyer account pages all load without an error screen', async ({ browser }) => {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const sent = await context.request.post(`${API}/auth/phone/code`, { data: { phone: '+998900000001' } });
  const { dev_code: code } = await sent.json();
  const verified = await context.request.post(`${API}/auth/phone/verify`, { data: { phone: '+998900000001', code } });
  expect(verified.status()).toBe(200);
  const page = await context.newPage();
  for (const path of PAGES) {
    const response = await page.goto(path);
    expect(response?.status(), path).toBe(200);
    await expect(page.getByRole('heading', { name: /Xatolik yuz berdi/ }), path).toHaveCount(0);
  }
  await context.close();
});
