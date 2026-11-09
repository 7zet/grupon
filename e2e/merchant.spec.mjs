import { expect, test } from '@playwright/test';
import { createHmac } from 'node:crypto';

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';
const WEB = 'http://127.0.0.1:3000';
const PASSWORD = 'Password123!';
// A 1×1 PNG: small enough for the media upload and accepted by the format check.
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
  'base64',
);

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
const SEEDED_TOTP_SECRET = 'JBSWY3DPEHPK3PXP';

function totp(secret = SEEDED_TOTP_SECRET) {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = '';
  for (const char of secret) bits += alphabet.indexOf(char).toString(2).padStart(5, '0');
  const key = Buffer.from(bits.match(/.{1,8}/g).map((byte) => String.fromCharCode(parseInt(byte.padEnd(8, '0'), 2))).join(''), 'binary');
  const counter = Buffer.alloc(8); counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30000)));
  const sum = createHmac('sha1', key).update(counter).digest(); const offset = sum[19] & 15;
  return String((sum.readUInt32BE(offset) & 0x7fffffff) % 1000000).padStart(6, '0');
}

/** Signs in with the seeded phone: request a code, then verify it (development returns the code). */
async function login(email) {
  const phone = SEEDED_PHONES[email];
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
  if (response.status === 202) {
    const { challenge_token } = await response.json();
    const verified = await fetch(`${API}/auth/2fa/verify`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ challenge_token, code: totp() }) });
    expect(verified.status).toBe(200);
    return (await verified.json()).access_token;
  }
  expect(response.status).toBe(200);
  return (await response.json()).access_token;
}

/** The business-level price proof that deal submission requires; uploading again replaces it. */
async function uploadPriceProof(owner) {
  const response = await fetch(`${API}/merchant/documents/price_proof`, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${owner}`, 'Content-Type': 'image/png' },
    body: PNG,
  });
  expect(response.status).toBe(204);
}

function call(token, method, path, body, headers = {}) {
  return fetch(`${API}${path}`, {
    method,
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', ...headers },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

const text = (value) => ({ uz: value, ru: value, en: value });

test('cashier cannot create deals', async () => {
  const cashier = await login('cashier-b@local.test');
  const response = await call(cashier, 'POST', '/merchant/deals', {});
  expect(response.status).toBe(403);
});

test('owner creates, admin approves, buyer buys, cashier redeems and sees own history', async ({ browser }) => {
  const owner = await login('owner-b@local.test');
  const admin = await login('admin-b@local.test');
  const buyer = await login('user@local.test');

  const config = await (await fetch(`${API}/config`)).json();
  const category = config.categories.find((item) => item.subcategories?.length);
  const subcategory = category.subcategories[0];
  const locations = await (await call(owner, 'GET', '/merchant/locations')).json();
  const locationId = locations.items[0].id;

  // Three photos through the media library: create the asset, then upload its bytes.
  const media = [];
  for (let index = 0; index < 3; index += 1) {
    const created = await call(owner, 'POST', '/merchant/media', {
      filename: `e2e-${index}.png`,
      content_type: 'image/png',
      size: PNG.length,
    });
    expect(created.status).toBe(201);
    const upload = await created.json();
    const stored = await fetch(new URL(upload.upload_url, API), {
      method: 'PUT',
      headers: { Authorization: `Bearer ${owner}`, 'Content-Type': 'image/png' },
      body: PNG,
    });
    expect(stored.status).toBe(204);
    media.push({ url: upload.asset.url, alt: text('Rasm'), sort_order: index, width: 1, height: 1, blurhash: '' });
  }

  const slug = `e2e-deal-${Date.now()}`;
  const created = await call(owner, 'POST', '/merchant/deals', {
    slug,
    category_id: category.id,
    subcategory_id: subcategory.id,
    title: text('E2E aksiya'),
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
    location_ids: [locationId],
    options: [
      { title: text('Standart'), original_price: 200000, deal_price: 100000, stock_total: 5, sort_order: 0, is_active: true },
    ],
    media,
  });
  expect(created.status).toBe(201);
  const deal = await created.json();

  await uploadPriceProof(owner);
  const submitted = await call(owner, 'POST', `/merchant/deals/${deal.id}/submit`);
  expect(submitted.status).toBe(200);

  const approved = await call(admin, 'POST', `/admin/deals/${deal.id}/approve`);
  expect(approved.status).toBe(200);
  expect((await approved.json()).status).toBe('active');

  // Buyer pays through the mock provider; the webhook runs asynchronously, so poll for the voucher.
  const order = await call(buyer, 'POST', '/orders', { items: [{ deal_option_id: deal.options[0].id, quantity: 1 }] }, {
    'Idempotency-Key': crypto.randomUUID(),
  });
  expect(order.status).toBe(201);
  const orderBody = await order.json();
  const payment = await (await call(buyer, 'POST', `/orders/${orderBody.id}/pay`, { provider: 'mock', terms_accepted: true })).json();
  expect((await call(buyer, 'POST', `/payments/mock/${payment.payment_id}/complete`, { outcome: 'succeeded' })).status).toBe(202);

  let voucher;
  for (let attempt = 0; attempt < 20 && !voucher; attempt += 1) {
    const page = await (await call(buyer, 'GET', '/me/vouchers')).json();
    voucher = page.items.find((item) => item.deal.id === deal.id);
    if (!voucher) await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  expect(voucher, 'voucher issued after webhook').toBeTruthy();

  // Cashier B signs in with their own account and redeems at the counter on a phone-sized screen.
  const context = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await context.newPage();
  await page.goto(`${WEB}/uz/login`);
  await page.getByLabel(/Telefon raqami/).fill(SEEDED_PHONES['cashier-b@local.test'].slice(4));
  await page.getByRole('button', { name: /Kod yuborish/ }).click();
  const hint = page.getByText(/Test rejimi kodi: \d{6}/);
  await expect(hint).toBeVisible();
  await page.getByLabel(/Tasdiqlash kodi/).fill((await hint.textContent()).match(/\d{6}/)[0]);
  await page.getByRole('button', { name: /^Kirish$/ }).click();
  await page.waitForURL((url) => !url.pathname.endsWith('/login'));
  await page.goto(`${WEB}/uz/merchant/redeem`);
  await page.getByLabel(/Vaucher kodi/).fill(voucher.code);
  await page.getByRole('button', { name: /^Tekshirish$/ }).click();
  await expect(page.getByRole('heading', { name: 'E2E aksiya' })).toBeVisible();
  await page.getByRole('button', { name: /Ishlatilganini tasdiqlash/ }).click();
  await expect(page.getByRole('status')).toHaveText(/Vaucher ishlatildi/);
  // The cashier screen lists today's redemptions as cards, not a table.
  await expect(page.getByRole('heading', { name: /Bugungi tasdiqlashlar/ })).toBeVisible();
  await expect(page.locator('li').filter({ hasText: 'E2E aksiya' }).first()).toBeVisible();

  // Cashiers have no payouts or platform-finance access, including direct URLs.
  await page.goto(`${WEB}/uz/merchant/payouts`);
  await expect(page.getByRole('heading', { name: /Bu bo'limga kirish yo'q/ })).toBeVisible();
  await page.goto(`${WEB}/uz/admin/finance`);
  await expect(page.getByRole('heading', { name: /Bu bo'limga kirish yo'q/ })).toBeVisible();

  // The same voucher cannot be used twice.
  const cashier = await login('cashier-b@local.test');
  const again = await call(cashier, 'POST', `/merchant/vouchers/${voucher.id}/redeem`, { location_id: locationId });
  expect(again.status).toBe(409);
  await context.close();
});
