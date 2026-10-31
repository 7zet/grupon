import { expect, test } from '@playwright/test';

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';
const PASSWORD = 'Password123!';
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
  'base64',
);
const text = (value) => ({ uz: value, ru: value, en: value });

async function login(email) {
  const response = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD }),
  });
  return (await response.json()).access_token;
}

function call(token, method, path, body, headers = {}) {
  return fetch(`${API}${path}`, {
    method,
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', ...headers },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

/** Creates a deal under review through the API, the way a merchant's submitted draft looks. */
async function submitDeal(owner, title) {
  const config = await (await fetch(`${API}/config`)).json();
  const category = config.categories.find((item) => item.subcategories?.length);
  const locations = await (await call(owner, 'GET', '/merchant/locations')).json();
  const media = [];
  for (let index = 0; index < 3; index += 1) {
    const upload = await (await call(owner, 'POST', '/merchant/media', { filename: 'a.png', content_type: 'image/png', size: PNG.length })).json();
    await fetch(new URL(upload.upload_url, API), { method: 'PUT', headers: { Authorization: `Bearer ${owner}`, 'Content-Type': 'image/png' }, body: PNG });
    media.push({ url: upload.asset.url, alt: text('Rasm'), sort_order: index, width: 1, height: 1, blurhash: '' });
  }
  const created = await call(owner, 'POST', '/merchant/deals', {
    slug: `ui-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
    category_id: category.id,
    subcategory_id: category.subcategories[0].id,
    kind: 'voucher',
    title: text(title),
    short_description: text('Qisqa'),
    description: text('Tavsif'),
    highlights: [text('a'), text('b'), text('c')],
    whats_included: [],
    terms: text('Shartlar'),
    faq: [],
    redemption_method: 'qr_or_code',
    redemption_instructions: text('Ko‘rsating'),
    booking_required: false,
    is_giftable: false,
    starts_at: new Date(Date.now() - 3600_000).toISOString(),
    ends_at: new Date(Date.now() + 30 * 86_400_000).toISOString(),
    voucher_valid_days: 30,
    max_per_user: 2,
    location_ids: [locations.items[0].id],
    options: [{ title: text('Standart'), original_price: 200000, deal_price: 100000, stock_total: 5, sort_order: 0, is_active: true }],
    media,
  });
  const deal = await created.json();
  await call(owner, 'POST', `/merchant/deals/${deal.id}/submit`);
  return deal;
}

async function signIn(page, email) {
  await page.goto('/uz/login');
  await page.getByLabel(/Elektron pochta/).fill(email);
  await page.getByLabel(/Parol/).fill(PASSWORD);
  await page.getByRole('button', { name: /^Kirish$/ }).click();
  await page.waitForURL((url) => !url.pathname.endsWith('/login'));
}

test('admin approves and rejects deals in the moderation queue', async ({ page }) => {
  const owner = await login('owner@local.test');
  const first = await submitDeal(owner, 'Moderatsiya approve');
  const second = await submitDeal(owner, 'Moderatsiya reject');

  await signIn(page, 'admin@local.test');
  await page.goto('/uz/admin/moderation');
  await expect(page.getByText('Moderatsiya approve')).toBeVisible();
  await page.getByText('Moderatsiya approve').locator('xpath=ancestor::li').getByRole('button', { name: /^Tasdiqlash$/ }).click();
  await expect(page.getByText('Moderatsiya approve')).toHaveCount(0);

  await page.getByText('Moderatsiya reject').locator('xpath=ancestor::li').getByRole('button', { name: /^Rad etish$/ }).click();
  await page.getByLabel(/Rad etish sababi/).fill('Rasmlar tavsifga mos emas');
  await page.getByRole('button', { name: /Rad etishni yuborish/ }).click();
  await expect(page.getByText('Moderatsiya reject')).toHaveCount(0);
  expect(first.id).toBeTruthy();
  expect(second.id).toBeTruthy();
});

test('buyers cannot open the moderation queue', async ({ page }) => {
  await signIn(page, 'user@local.test');
  await page.goto('/uz/admin/moderation');
  await expect(page.getByRole('heading', { name: /Moderatsiya navbati/ })).toHaveCount(0);
});

test('merchant uploads a photo and opens the deal wizard', async ({ page }) => {
  await signIn(page, 'owner@local.test');
  await page.goto('/uz/merchant/media');
  await page.locator('input[type=file]').setInputFiles({ name: 'photo.png', mimeType: 'image/png', buffer: PNG });
  await expect(page.locator('main img').first()).toBeVisible({ timeout: 15000 });

  await page.goto('/uz/merchant/deals/new');
  await expect(page.getByRole('heading', { name: /Yangi aksiya/ })).toBeVisible();
  await expect(page.getByText(/Asosiy/).first()).toBeVisible();
});
