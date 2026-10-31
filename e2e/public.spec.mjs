import { expect, test } from '@playwright/test';

for (const locale of ['uz', 'ru', 'en']) {
  test(`${locale} public routes render`, async ({ page }) => {
    for (const path of [
      '',
      '/tashkent',
      '/tashkent/beauty-and-spas',
      '/coupons',
      '/business/atlas-wellness',
      '/seasonal-deals',
      '/collections/navruz',
      '/merchant',
      '/help',
      '/terms',
      '/privacy',
    ]) {
      const response = await page.goto(`/${locale}${path}`, { waitUntil: 'domcontentloaded' });
      expect(response?.status()).toBe(200);
      await expect(page.locator('html')).toHaveAttribute('lang', locale);
    }
  });
}

for (const viewport of [
  { width: 360, height: 640 },
  { width: 390, height: 844 },
  { width: 1440, height: 900 },
]) {
  test(`deal first screen ${viewport.width}x${viewport.height}`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.goto('/uz/deal/beauty-and-spas-special-2', { waitUntil: 'domcontentloaded' });
    const buy =
      viewport.height === 640
        ? page.getByTestId('sticky-buy')
        : page.locator('button:visible').filter({ hasText: 'Hozir xarid qilish' }).first();
    await expect(buy).toBeVisible();
    const box = await buy.boundingBox();
    expect(box).not.toBeNull();
    const unobscuredBottom = viewport.width < 1024 ? viewport.height - 64 : viewport.height;
    expect(box.y + box.height).toBeLessThanOrEqual(unobscuredBottom);
    await page.screenshot({
      path: `docs/phase-2/screenshots/deal-${viewport.width}x${viewport.height}.png`,
      fullPage: false,
    });
  });
}

test('listing filters remain in the URL', async ({ page }) => {
  await page.goto('/en/tashkent/beauty-and-spas');
  await page.getByLabel('Minimum rating').selectOption('4.5');
  await expect(page).toHaveURL(/rating=4.5/);
  await page.getByLabel('Sort').selectOption('price-asc');
  await expect(page).toHaveURL(/sort=price-asc/);
});
