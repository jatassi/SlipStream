// PROTOTYPE tool: screenshot variant × shell × view combinations from dist/index.html.
// node tools/shoot.mjs [--out dir] [--wait ms] [--hidpi] [--still] A:wide:home B:phone C:portal:work/halcyon-drift
// Phone shells are captured as the device frame; wide shells at 1440×900. Console errors print.
import { chromium } from 'playwright-core';
import { mkdirSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const args = process.argv.slice(2);
const opt = { out: join(here, '..', 'media', 'shots'), wait: 1400, hidpi: false, still: false, width: 1440, height: 900 };
const specs = [];
for (let i = 0; i < args.length; i++) {
  const a = args[i];
  if (a === '--out') opt.out = resolve(args[++i]);
  else if (a === '--wait') opt.wait = +args[++i];
  else if (a === '--hidpi') opt.hidpi = true;
  else if (a === '--still') opt.still = true;
  else if (a === '--size') [opt.width, opt.height] = args[++i].split('x').map(Number);
  else specs.push(a);
}
if (!specs.length) { console.error('usage: node tools/shoot.mjs [--out dir] A:wide:home ...'); process.exit(1); }
mkdirSync(opt.out, { recursive: true });

const browser = await chromium.launch({ channel: 'chrome' });
const page = await browser.newPage({ viewport: { width: opt.width, height: opt.height }, deviceScaleFactor: opt.hidpi ? 2 : 1 });
page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') console.log(`[${m.type()}] ${m.text()}`); });
page.on('pageerror', (e) => console.log(`[pageerror] ${e.message}\n${e.stack || ''}`));
const url = pathToFileURL(join(here, '..', 'dist', 'index.html')).href + `?bar=0${opt.still ? '&still=1' : ''}`;
await page.goto(url);
await page.waitForFunction(() => document.documentElement.dataset.ready === '1', null, { timeout: 30000 });

for (const spec of specs) {
  const [variant, shell = 'wide', ...rest] = spec.split(':');
  const view = rest.join(':');
  await page.evaluate(([v, s, w]) => window.__ss.go(v, s, w), [variant, shell, view]);
  await page.waitForTimeout(opt.wait);
  const file = join(opt.out, `${variant}-${shell}${view ? '-' + view.replace(/[^a-z0-9]+/gi, '_') : ''}.png`);
  const device = await page.$('.device');
  if (device) await device.screenshot({ path: file });
  else await page.screenshot({ path: file });
  console.log(file);
}
await browser.close();
