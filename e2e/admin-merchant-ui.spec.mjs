import { expect, test } from '@playwright/test';

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';
const PASSWORD = 'Password123!';
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
  'base64',
);
const text = (value) => ({ uz: value, ru: value, en: value });

const SEEDED_PHONES = {
  'user@local.test': '+998900000001',
  'owner@local.test': '+998900000002',
  'cashier@local.test': '+998900000003',
  'admin@local.test': '+998900000004',
  'owner-b@local.test': '+998900000005',
  'owner-c@local.test': '+998900000006',
  'owner-d@local.test': '+998900000007',
  'admin-b@local.test': '+998900000008',
  'cashier-b@local.test': '+998900000009',
};

/** Signs in with the seeded phone: request a code, then verify it (development returns the code). */
async function login(email) {
  return loginPhone(SEEDED_PHONES[email]);
}

async function loginPhone(phone) {
  const sent = await fetch(`${API}/auth/phone/code`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone }),
  });
  expect(sent.status).toBe(202);
  const { dev_code: code } = await sent.json();
  const response = await fetch(`${API}/auth/phone/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone, code }),
  });
  expect(response.status).toBe(200);
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
  return signInPhone(page, SEEDED_PHONES[email]);
}

async function signInPhone(page, phone) {
  await page.goto('/uz/login');
  await page.getByLabel(/Telefon raqami/).fill(phone.slice(4));
  await page.getByRole('button', { name: /Kod yuborish/ }).click();
  const hint = page.getByText(/Test rejimi kodi: \d{6}/);
  await expect(hint).toBeVisible();
  await page.getByLabel(/Tasdiqlash kodi/).fill((await hint.textContent()).match(/\d{6}/)[0]);
  await page.getByRole('button', { name: /^Kirish$/ }).click();
  await page.waitForURL((url) => !url.pathname.endsWith('/login'));
}

test('merchant attaches price proof before submission and admin sees it', async ({ page }) => {
  const suffix = String(Date.now()).slice(-7);
  const phone = `+99890${suffix}`;
  const owner = await loginPhone(phone);
  const config = await (await fetch(`${API}/config`)).json();
  const category = config.categories.find((item) => item.deal_creation_enabled);
  const city = config.cities[0];
  const displayName = `Price proof ${suffix}`;
  const registered = await call(owner, 'POST', '/merchants', {
    legal_name: `${displayName} MChJ`, display_name: displayName,
    category_id: category.id, city_id: city.id, phone, address: 'Test kochasi 1',
    lat: 41.3, lng: 69.2, stir: `4${suffix}0`.slice(0, 9), org_form: 'mchj',
    tax_regime: 'vat_payer', bank_mfo: '00014', bank_account: `2020800090010${suffix}`.padEnd(20, '0').slice(0, 20),
    responsible_name: 'Alisher Usmonov', contract_version: 'v1.1', contract_accepted: true,
  });
  expect(registered.status).toBe(201);

  await signInPhone(page, phone);
  await page.goto('/uz/merchant/review');
  await page.waitForLoadState('networkidle');
  for (let index = 0; index < 3; index += 1) {
    const field = page.locator('input[type=file]').nth(index);
    const uploaded = page.waitForResponse((response) => response.url().includes('/api/bff/merchant/documents/'));
    await field.setInputFiles({ name: 'proof.png', mimeType: 'image/png', buffer: PNG });
    expect((await uploaded).status()).toBe(204);
    await expect(field.locator('xpath=..').getByText('Yuklangan')).toBeVisible();
  }
  await page.getByRole('button', { name: /Tekshiruvga yuborish/ }).click();
  await expect(page.getByRole('status')).toContainText(/Ariza tekshiruvga yuborildi/);

  await signIn(page, 'admin@local.test');
  await page.goto('/uz/admin/merchants');
  const merchant = page.getByText(displayName).locator('xpath=ancestor::li');
  await expect(merchant.getByRole('link', { name: /Narx tasdig'i/ })).toBeVisible();
});

test('admin approves and rejects deals in the moderation queue', async ({ page }) => {
  const owner = await login('owner-c@local.test');
  const stamp = Date.now();
  const approveTitle = `Moderatsiya approve ${stamp}`;
  const rejectTitle = `Moderatsiya reject ${stamp}`;
  const first = await submitDeal(owner, approveTitle);
  const second = await submitDeal(owner, rejectTitle);

  await signIn(page, 'admin@local.test');
  await page.goto('/uz/admin/moderation');
  await expect(page.getByText(approveTitle)).toBeVisible();
  await page.getByText(approveTitle).locator('xpath=ancestor::li').getByRole('button', { name: /^Tasdiqlash$/ }).click();
  await expect(page.getByText(approveTitle)).toHaveCount(0);

  await page.getByText(rejectTitle).locator('xpath=ancestor::li').getByRole('button', { name: /^Rad etish$/ }).click();
  await page.getByLabel(/Rad etish sababi/).fill('Rasmlar tavsifga mos emas');
  await page.getByRole('button', { name: /Rad etishni yuborish/ }).click();
  await expect(page.getByText(rejectTitle)).toHaveCount(0);
  expect(first.id).toBeTruthy();
  expect(second.id).toBeTruthy();
});

test('buyers cannot open the moderation queue', async ({ page }) => {
  await signIn(page, 'user@local.test');
  await page.goto('/uz/admin/moderation');
  await expect(page.getByRole('heading', { name: /Moderatsiya navbati/ })).toHaveCount(0);
});

test('merchant completes all six deal-wizard steps and submits for review', async ({ page }) => {
  await signIn(page, 'owner-c@local.test');
  await page.goto('/uz/merchant/media');
  await page.waitForLoadState('networkidle');
  const existing = await page.locator('main img').count();
  for (let index = 0; index < 3; index += 1) {
    await page.locator('input[type=file]').setInputFiles({ name: `wizard-${Date.now()}-${index}.png`, mimeType: 'image/png', buffer: PNG });
    await page.getByRole('button', { name: /Qirqib yuklash/ }).click();
    await expect(page.locator('main img')).toHaveCount(existing + index + 1, { timeout: 15000 });
  }

  await page.goto('/uz/merchant/deals/new');
  await expect(page.getByRole('heading', { name: /Yangi aksiya/ })).toBeVisible();
  await page.getByRole('button', { name: /^Keyingi$/ }).click();
  await expect(page.getByRole('alert').first()).toBeVisible();

  const fillGroup = async (name, value) => {
    const fields = page.getByRole('group', { name });
    for (const input of await fields.locator('input').all()) await input.fill(value);
  };
  const title = `Wizard aksiya ${Date.now()}`;
  await page.getByLabel(/Manzil \(slug\)/).fill(`wizard-${Date.now()}`);
  await page.locator('main select').nth(0).selectOption({ index: 1 });
  await page.locator('main select').nth(1).selectOption({ index: 1 });
  await fillGroup('Sarlavha', title);
  await fillGroup('Qisqa tavsif', 'Qisqa tavsif');
  await fillGroup('Tavsif', 'To\'liq tavsif');
  await expect(page.getByLabel('Ko‘rinish').getByText(title)).toBeVisible();
  await page.getByRole('button', { name: /^Keyingi$/ }).click();

  await fillGroup('Variant nomi', 'Standart');
  await page.getByLabel(/Asl narx/).fill('200000');
  await page.getByLabel(/Chegirmali narx/).fill('100000');
  await page.getByLabel(/Zaxira/).fill('10');
  await page.getByRole('button', { name: /^Keyingi$/ }).click();

  const mediaChoices = page.locator('input[type=checkbox]');
  for (let index = 0; index < 3; index += 1) await mediaChoices.nth(index).check({ force: true });
  await page.getByRole('button', { name: /^Keyingi$/ }).click();

  for (let index = 1; index <= 3; index += 1) await fillGroup(`Afzalliklar ${index}`, `Afzallik ${index}`);
  await fillGroup('Shartlar', 'Foydalanish shartlari');
  await fillGroup('Foydalanish yo‘riqnomasi', 'Kassirga kodni ko‘rsating');
  await page.getByRole('button', { name: /^Keyingi$/ }).click();

  await page.locator('input[type=checkbox]').first().check();
  await page.getByLabel(/^Boshlanish$/).fill('2027-01-01T10:00');
  await page.getByLabel(/^Tugash$/).fill('2027-02-01T10:00');
  await page.getByRole('button', { name: /^Keyingi$/ }).click();

  await expect(page.getByText(title).last()).toBeVisible();
  const created = page.waitForResponse((response) => response.url().endsWith('/api/bff/merchant/deals') && response.request().method() === 'POST');
  await page.getByRole('button', { name: /Tekshiruvga yuborish/ }).click();
  const createdResponse = await created;
  expect(createdResponse.status(), `${await createdResponse.text()}\n${createdResponse.request().postData()}`).toBe(201);
  await expect(page.getByRole('alert').first()).toContainText('Tekshiruvga yuborildi');
  await page.goto('/uz/merchant/deals');
  const row = page.getByRole('row').filter({ hasText: title });
  await expect(row).toContainText(/Ko'rib chiqilmoqda/);
});
