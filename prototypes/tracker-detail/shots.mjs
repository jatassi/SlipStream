// PROTOTYPE · screenshots of the Tracker detail variants. Serve prototypes/ first: cd prototypes && python3 -m http.server 4175
// Poster art 404s are filtered: ui-direction/art is a local, uncommitted copy (see torrent-client/run.sh) and may be partial.
// Usage: node tracker-detail/shots.mjs [baseUrl]
import { createRequire } from 'node:module';
import { mkdirSync, existsSync } from 'node:fs';

const require = createRequire(import.meta.url);
let pw;
for (const p of ['playwright-core', '/opt/node-tools/node_modules/playwright-core', '/tmp/claude-0/pw/node_modules/playwright-core']) {
  try { pw = require(p); break; } catch { /* try the next */ }
}
const BASE = process.argv[2] || 'http://localhost:4175/tracker-detail/';
const OUT = new URL('./shots/', import.meta.url).pathname;
mkdirSync(OUT, { recursive: true });

const SHOTS = [
  ['a-wide', 'variant=A&tracker=beacon'],
  ['b-wide', 'variant=B&tracker=beacon'],
  ['b-wide-seeding', 'variant=B&tracker=beacon&tab=seeding'],
  ['c-wide', 'variant=C&tracker=beacon'],
  ['a-phone', 'variant=A&tracker=beacon&device=phone'],
  ['b-phone', 'variant=B&tracker=beacon&device=phone'],
  ['c-phone', 'variant=C&tracker=beacon&device=phone'],
  ['aurora-a-wide', 'variant=A&tracker=aurora'],
  ['public-c-wide', 'variant=C&tracker=public'],
  ...(process.env.ALL ? ['A', 'B', 'C'].flatMap((v) => ['aurora', 'cinder', 'public'].flatMap((t) => [[`x-${t}-${v}-wide`, `variant=${v}&tracker=${t}`], [`x-${t}-${v}-phone`, `variant=${v}&tracker=${t}&device=phone`]])) : []),
];

// The sandbox's proxy only tunnels HTTPS, so serve lucide from a local copy when one exists (curl it to /tmp/claude-0/pw/lucide.min.js).
const LUCIDE = '/tmp/claude-0/pw/lucide.min.js';
const browser = await pw.chromium.launch({ executablePath: '/opt/pw-browsers/chromium' });
const errors = [];
for (const [name, q] of SHOTS) {
  const phone = q.includes('device=phone');
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: phone ? 2 : 1, ignoreHTTPSErrors: true });
  if (existsSync(LUCIDE)) await page.route('https://unpkg.com/**', (r) => r.fulfill({ path: LUCIDE, contentType: 'text/javascript' }));
  page.on('console', (m) => { if (m.type() === 'error' && !m.text().includes('status of 404')) errors.push(`${name}: ${m.text()}`); });
  page.on('pageerror', (e) => errors.push(`${name}: ${e.message}`));
  await page.goto(`${BASE}?${q}`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(700);
  if (phone) {
    const h = await page.evaluate(() => document.getElementById('app').scrollHeight);
    await page.setViewportSize({ width: 430, height: 900 });
    await page.addStyleTag({ content: `body.device-phone { padding: 0 0 20px !important; display: block !important; min-height: 0 !important; } body.device-phone #frame { height: ${h}px !important; margin: 0 auto; } body.device-phone #app { overflow: visible !important; height: auto !important; } #proto-bar { position: static !important; transform: none !important; width: 390px; box-sizing: border-box; margin: 10px auto; flex-wrap: wrap; justify-content: center; border-radius: 20px; }` });
    await page.evaluate(() => document.body.prepend(document.getElementById('proto-bar')));
    await page.waitForTimeout(300);
    await page.screenshot({ path: `${OUT}${name}.png`, fullPage: true });
  } else {
    const ph = await page.evaluate(() => document.documentElement.scrollHeight);
    await page.addStyleTag({ content: `#proto-bar { position: absolute !important; bottom: auto !important; top: ${ph - 64}px !important; }` });
    await page.screenshot({ path: `${OUT}${name}.png`, fullPage: true });
  }
  await page.close();
}
await browser.close();
console.log(errors.length ? `Console errors:\n${errors.join('\n')}` : 'No console errors');
