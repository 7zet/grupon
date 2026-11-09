import { expect, test } from '@playwright/test';
import { createHmac } from 'node:crypto';
import { mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';
const SHOT_DIR = process.env.ROLE_SHOTS ?? fileURLToPath(new URL('../../grupon-artifacts/2026-11-09/roles', import.meta.url));
const VIEWPORTS = [
  { width: 390, height: 844 },
  { width: 1440, height: 900 },
];

// Seeded demo accounts (see docs/DEMO.md). Each role is a separate person.
const PHONES = {
  buyer: '+998900000001',
  owner: '+998900000002',
  manager: '+998910000010',
  cashier: '+998900000003',
  moderator: '+998910000011',
  support: '+998910000012',
  finance: '+998910000013',
  superadmin: '+998900000004',
};

/** Expected landing page and the workspace menu labels each role sees (Uzbek, the default locale). */
const ROLES = {
  buyer: { home: /\/uz$/, workspace: false },
  owner: { home: /\/uz\/merchant\/dashboard$/, workspace: true, shows: ['Mening aksiyalarim', "To'lovlar", 'Xodimlar', 'Sozlamalar'] },
  manager: { home: /\/uz\/merchant\/dashboard$/, workspace: true, shows: ['Mening aksiyalarim'], hides: ["To'lovlar", 'Xodimlar', 'Sozlamalar'] },
  cashier: { home: /\/uz\/merchant\/redeem$/, workspace: false, hides: ['Mening aksiyalarim', "To'lovlar"] },
  moderator: { home: /\/uz\/admin\/moderation$/, workspace: true, admin: true, shows: ['Aksiyalar', 'Bizneslar', 'Firibgarlik signallari'], hides: ['Moliya', 'Audit jurnali'] },
  support: { home: /\/uz\/admin\/finance$/, workspace: true, admin: true, shows: ['Moliya'], hides: ['Aksiyalar', 'Sharh shikoyatlari', 'Bizneslar'] },
  finance: { home: /\/uz\/admin\/finance$/, workspace: true, admin: true, shows: ['Moliya'], hides: ['Aksiyalar', 'Sharh shikoyatlari'] },
  superadmin: { home: /\/uz\/admin$/, workspace: true, admin: true, shows: ['Umumiy ko\'rinish', 'Moliya', 'Audit jurnali', 'Aksiyalar'] },
};

const NO_ACCESS = "Bu bo'limga kirish yo'q";

function totp(secret = 'JBSWY3DPEHPK3PXP') {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = '';
  for (const char of secret) bits += alphabet.indexOf(char).toString(2).padStart(5, '0');
  const key = Buffer.from(bits.match(/.{1,8}/g).map((b) => String.fromCharCode(parseInt(b.padEnd(8, '0'), 2))).join(''), 'binary');
  const counter = Buffer.alloc(8);
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30000)));
  const sum = createHmac('sha1', key).update(counter).digest();
  const offset = sum[19] & 15;
  return String((sum.readUInt32BE(offset) & 0x7fffffff) % 1000000).padStart(6, '0');
}

/** Signs in through the API inside the browser context, so the page gets the same session cookies. */
async function signIn(context, phone) {
  const sent = await context.request.post(`${API}/auth/phone/code`, { data: { phone } });
  expect(sent.status()).toBe(202);
  const { dev_code: code } = await sent.json();
  let res = await context.request.post(`${API}/auth/phone/verify`, { data: { phone, code } });
  if (res.status() === 202) {
    const { challenge_token } = await res.json();
    res = await context.request.post(`${API}/auth/2fa/verify`, { data: { challenge_token, code: totp() } });
  }
  expect(res.status()).toBe(200);
  return (await res.json()).access_token;
}

/** The profile control that is visible at this width: the bottom bar on shop pages below 1024 px, the header elsewhere. */
async function profileControl(page) {
  const bottomBar = page.locator('nav.fixed summary[aria-label="Profil"]').first();
  if (await bottomBar.isVisible().catch(() => false)) return bottomBar;
  const header = page.locator('header summary[aria-label="Profil"]').first();
  await header.waitFor({ state: 'visible' });
  return header;
}

for (const viewport of VIEWPORTS) {
  for (const [role, expected] of Object.entries(ROLES)) {
    test(`${role} lands on its own home and sees only its menu (${viewport.width}px)`, async ({ browser }) => {
      mkdirSync(SHOT_DIR, { recursive: true });
      const context = await browser.newContext({ viewport });
      await signIn(context, PHONES[role]);
      const page = await context.newPage();
      await page.goto('/uz');
      await expect(page).toHaveURL(expected.home);
      await page.waitForLoadState('networkidle');

      // Only the marketplace has no workspace navigation; every other role gets its own.
      // Attribute match, not role match: the admin console keeps its menu in a closed <details> on phones.
      const workspaceNav = page.locator('nav[aria-label="Ish maydoni"]');
      if (expected.workspace) {
        await expect(workspaceNav.first()).toBeAttached();
      } else {
        await expect(workspaceNav).toHaveCount(0);
      }
      if (viewport.width >= 1024) {
        for (const label of expected.shows ?? []) await expect(page.getByRole('link', { name: label }).first()).toBeVisible();
        for (const label of expected.hides ?? []) await expect(page.getByRole('link', { name: label, exact: true })).toHaveCount(0);
      }
      await page.screenshot({ path: join(SHOT_DIR, `${role}-${viewport.width}.png`), fullPage: true, animations: 'disabled' });
      await context.close();
    });

    test(`${role} profile icon opens the menu, never the login page (${viewport.width}px)`, async ({ browser }) => {
      const context = await browser.newContext({ viewport });
      await signIn(context, PHONES[role]);
      const page = await context.newPage();
      await page.goto('/uz');
      await page.waitForLoadState('networkidle');
      await (await profileControl(page)).click();
      await expect(page).not.toHaveURL(/\/login/);
      await expect(page.getByRole('button', { name: 'Chiqish' })).toBeVisible();
      await context.close();
    });
  }
}

test('manager cannot open payouts, staff or settings (1440px)', async ({ browser }) => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await signIn(context, PHONES.manager);
  const page = await context.newPage();
  for (const path of ['/uz/merchant/payouts', '/uz/merchant/staff', '/uz/merchant/settings']) {
    await page.goto(path);
    await expect(page.getByRole('heading', { name: NO_ACCESS })).toBeVisible();
  }
  await context.close();
});

test('cashier cannot open the dashboard or payouts; buyer cannot open /merchant or /admin (390px)', async ({ browser }) => {
  const viewport = { width: 390, height: 844 };
  const cashier = await browser.newContext({ viewport });
  await signIn(cashier, PHONES.cashier);
  const cashierPage = await cashier.newPage();
  for (const path of ['/uz/merchant/dashboard', '/uz/merchant/payouts']) {
    await cashierPage.goto(path);
    await expect(cashierPage.getByRole('heading', { name: NO_ACCESS })).toBeVisible();
  }
  await cashier.close();

  const buyer = await browser.newContext({ viewport });
  await signIn(buyer, PHONES.buyer);
  const buyerPage = await buyer.newPage();
  for (const path of ['/uz/merchant/dashboard', '/uz/admin', '/uz/admin/finance']) {
    await buyerPage.goto(path);
    await expect(buyerPage.getByRole('heading', { name: NO_ACCESS })).toBeVisible();
  }
  await buyer.close();
});

test('moderator cannot open finance; support cannot open moderation (1440px)', async ({ browser }) => {
  const viewport = { width: 1440, height: 900 };
  const moderator = await browser.newContext({ viewport });
  await signIn(moderator, PHONES.moderator);
  const modPage = await moderator.newPage();
  await modPage.goto('/uz/admin/finance');
  await expect(modPage.getByRole('heading', { name: NO_ACCESS })).toBeVisible();
  await moderator.close();

  const support = await browser.newContext({ viewport });
  await signIn(support, PHONES.support);
  const supportPage = await support.newPage();
  await supportPage.goto('/uz/admin/moderation');
  await expect(supportPage.getByRole('heading', { name: NO_ACCESS })).toBeVisible();
  await support.close();
});

test('cashier on a phone: enter code, confirm, see it in today, cancel within 10 minutes', async ({ browser }) => {
  // Buying, paying and redeeming take several round trips; the default 30 s is too short for the whole flow.
  test.setTimeout(120000);
  const viewport = { width: 390, height: 844 };
  // A buyer buys a Lotus Spa voucher through the API so the cashier has something to redeem.
  const buyer = await browser.newContext();
  const buyerToken = await signIn(buyer, PHONES.buyer);
  const auth = { Authorization: `Bearer ${buyerToken}`, 'Content-Type': 'application/json' };
  const deals = await (await buyer.request.get(`${API}/deals?page_size=100`)).json();
  const lotus = (item) => item.merchant?.slug === 'seed-merchant-01' || item.merchant?.display_name === 'Lotus Spa';
  const available = (option) => option.stock_total - option.stock_sold - option.stock_reserved;
  // Take the Lotus Spa deal with the most stock left, so repeated runs do not exhaust it.
  const deal = deals.items
    .filter((item) => lotus(item) && item.status === 'active')
    .sort((x, y) => Math.max(...y.options.map(available)) - Math.max(...x.options.map(available)))[0];
  expect(deal, 'a sellable Lotus Spa deal').toBeTruthy();
  const option = deal.options.filter((item) => item.is_active && available(item) > 0).sort((x, y) => available(y) - available(x))[0];
  const order = await buyer.request.post(`${API}/orders`, {
    headers: { ...auth, 'Idempotency-Key': crypto.randomUUID() },
    data: { items: [{ deal_option_id: option.id, quantity: 1 }] },
  });
  expect(order.status()).toBe(201);
  const orderBody = await order.json();
  const payment = await (await buyer.request.post(`${API}/orders/${orderBody.id}/pay`, { headers: auth, data: { provider: 'mock', terms_accepted: true } })).json();
  expect((await buyer.request.post(`${API}/payments/mock/${payment.payment_id}/complete`, { headers: auth, data: { outcome: 'succeeded' } })).status()).toBe(202);
  let voucher;
  for (let attempt = 0; attempt < 20 && !voucher; attempt += 1) {
    const mine = await (await buyer.request.get(`${API}/me/vouchers`, { headers: auth })).json();
    // IDs are time-ordered (UUIDv7), so the largest ID is the voucher bought just now, not an old one.
    voucher = mine.items
      .filter((item) => item.deal.slug === deal.slug && item.status === 'active')
      .sort((x, y) => y.id.localeCompare(x.id))[0];
    if (!voucher) await new Promise((resolve) => setTimeout(resolve, 500));
  }
  expect(voucher, 'the bought voucher').toBeTruthy();
  await buyer.close();

  const context = await browser.newContext({ viewport });
  const cashierToken = await signIn(context, PHONES.cashier);
  const page = await context.newPage();
  await page.goto('/uz/merchant/redeem');
  await expect(page.locator('nav[aria-label="Ish maydoni"]')).toHaveCount(0);
  await page.getByLabel(/Vaucher kodi/).fill(voucher.code);
  await page.getByRole('button', { name: /^Tekshirish$/ }).click();
  await expect(page.getByText(/Xaridor:/).first()).toBeVisible();
  await page.getByRole('button', { name: /Ishlatilganini tasdiqlash/ }).click();
  await expect(page.getByRole('status')).toHaveText(/./);

  // Find this redemption's id from the cashier's own history, then act on that exact row.
  await expect(page.getByRole('heading', { name: 'Bugungi tasdiqlashlar' })).toBeVisible();
  // The history is ordered newest first, so this redemption is among the first entries. Read it again a few
  // times if it is not there yet; the failure message shows what the cashier's history holds.
  let history;
  let redemption;
  for (let attempt = 0; attempt < 10 && !redemption; attempt += 1) {
    history = await (await context.request.get(`${API}/merchant/redemptions`, { headers: { Authorization: `Bearer ${cashierToken}` } })).json();
    redemption = history.items.slice(0, 5).find((item) => item.voucher_id === voucher.id);
    if (!redemption) await new Promise((resolve) => setTimeout(resolve, 500));
  }
  expect(redemption, `redemption of voucher ${voucher.id} in ${JSON.stringify(history.items.slice(0, 5).map((item) => item.voucher_id))}`).toBeTruthy();
  const row = page.locator(`li[data-redemption-id="${redemption.id}"]`);
  await expect(row).toBeVisible();
  const cancel = row.getByRole('button', { name: 'Bekor qilish' });
  await expect(cancel).toBeVisible();
  const [undo] = await Promise.all([page.waitForResponse((response) => response.url().includes('/reverse')), cancel.click()]);
  expect(undo.status(), 'the undo request succeeds').toBeLessThan(300);

  // After the undo the same row stays in the list, marked reversed, without a cancel button.
  await page.reload();
  await expect(page.locator(`li[data-redemption-id="${redemption.id}"]`)).toBeVisible();
  await expect(page.locator(`li[data-redemption-id="${redemption.id}"]`).getByRole('button', { name: 'Bekor qilish' })).toHaveCount(0);
  await page.screenshot({ path: join(SHOT_DIR, 'cashier-redeem-390-after-cancel.png'), fullPage: true, animations: 'disabled' });
  await context.close();
});
