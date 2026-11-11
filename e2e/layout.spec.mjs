import { expect, test } from '@playwright/test';

// Fails when the stylesheet is missing or broken: a page with no layout styles has a tall header,
// unbounded content width, huge icons and horizontal scroll on phones.

const API = process.env.API_URL ?? 'http://127.0.0.1:8080/api/v1';
const VIEWPORTS = [
  { width: 390, height: 844 },
  { width: 1440, height: 900 },
];

async function signIn(context, phone) {
  const sent = await context.request.post(`${API}/auth/phone/code`, { data: { phone } });
  const { dev_code: code } = await sent.json();
  const res = await context.request.post(`${API}/auth/phone/verify`, { data: { phone, code } });
  if (res.status() === 202) {
    const { createHmac } = await import('node:crypto');
    const { challenge_token } = await res.json();
    const secret = 'JBSWY3DPEHPK3PXP';
    const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
    let bits = '';
    for (const char of secret) bits += alphabet.indexOf(char).toString(2).padStart(5, '0');
    const key = Buffer.from(bits.match(/.{1,8}/g).map((b) => String.fromCharCode(parseInt(b.padEnd(8, '0'), 2))).join(''), 'binary');
    const counter = Buffer.alloc(8);
    counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30000)));
    const sum = createHmac('sha1', key).update(counter).digest();
    const offset = sum[19] & 15;
    const totp = String((sum.readUInt32BE(offset) & 0x7fffffff) % 1000000).padStart(6, '0');
    await context.request.post(`${API}/auth/2fa/verify`, { data: { challenge_token, code: totp } });
  }
}

/** Measures what a visitor would see. Returns the numbers the assertions use. */
async function measure(page) {
  return page.evaluate(() => {
    // The top bar only: the category row under it is part of the header too.
    const topBar = document.querySelector('header > div');
    const headerH = topBar ? topBar.getBoundingClientRect().height : 0;
    // The widest page container: any element with a real max-width and side padding.
    const containers = [...document.querySelectorAll('body *')].map((el) => {
      const cs = getComputedStyle(el);
      return { maxW: parseFloat(cs.maxWidth), padX: parseFloat(cs.paddingLeft) + parseFloat(cs.paddingRight), display: cs.display };
    }).filter((c) => Number.isFinite(c.maxW) && c.maxW > 0);
    const bounded = containers.filter((c) => c.padX > 0);
    // Map tiles (Leaflet) are SVG too, but they are not icons; they have their own size.
    const icons = [...document.querySelectorAll('main svg')].filter((svg) => !svg.closest('.leaflet-container')).map((svg) => {
      const r = svg.getBoundingClientRect();
      return Math.max(r.width, r.height);
    });
    return {
      headerH,
      maxContainer: containers.length ? Math.max(...containers.map((c) => c.maxW)) : Infinity,
      hasBoundedPaddedContainer: bounded.length > 0,
      iconMax: icons.length ? Math.max(...icons) : 0,
      scrollW: document.documentElement.scrollWidth,
      vw: window.innerWidth,
    };
  });
}

for (const viewport of VIEWPORTS) {
  test.describe(`layout at ${viewport.width}px`, () => {
    test.use({ viewport });

    for (const [name, path] of [
      ['home', '/uz'],
      ['deal', '/uz/deal/seed-deal-001'],
    ]) {
      test(`${name}: header, container, icons and width are sane`, async ({ page }) => {
        await page.goto(path, { waitUntil: 'load' });
        // Not networkidle: map tiles from the internet can stay pending for a long time.
        await page.locator('header > div').first().waitFor();
        const m = await measure(page);
        expect(m.headerH, 'header height').toBeGreaterThan(0);
        expect(m.headerH, 'header height (72 px designed; collapsed styles give ~30 px)').toBeGreaterThanOrEqual(60);
        expect(m.headerH, 'header height').toBeLessThan(100);
        expect(m.hasBoundedPaddedContainer, 'a container with max-width and side padding').toBe(true);
        expect(m.maxContainer, 'widest container max-width').toBeLessThanOrEqual(1400);
        expect(m.iconMax, 'largest icon').toBeLessThanOrEqual(64);
        expect(m.scrollW, 'no horizontal scroll').toBeLessThanOrEqual(m.vw + 1);
      });
    }

    test(`business dashboard: layout is sane`, async ({ browser }) => {
      const context = await browser.newContext({ viewport });
      await signIn(context, '+998900000002');
      const page = await context.newPage();
      await page.goto('/uz/merchant/dashboard', { waitUntil: 'load' });
      await page.locator('header > div').first().waitFor();
      const m = await measure(page);
      expect(m.headerH).toBeGreaterThanOrEqual(60);
      expect(m.headerH).toBeLessThan(100);
      expect(m.hasBoundedPaddedContainer).toBe(true);
      expect(m.iconMax).toBeLessThanOrEqual(64);
      expect(m.scrollW).toBeLessThanOrEqual(m.vw + 1);
      await context.close();
    });
  });
}
