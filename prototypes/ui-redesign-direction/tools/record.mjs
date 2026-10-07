// PROTOTYPE tool: record one screencast per direction from dist/index.html.
// node tools/record.mjs [--out dir] [A B C]
// Each tour is a list of steps: go to a route, move or click, wait. Videos land as <key>.webm,
// then convert with ffmpeg to mp4 for the PR (see README).
import { chromium } from 'playwright-core';
import { mkdirSync, renameSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { TOURS } from './tours.mjs';

const here = dirname(fileURLToPath(import.meta.url));
const args = process.argv.slice(2);
let out = join(here, '..', 'media', 'raw');
const keys = [];
for (let i = 0; i < args.length; i++) args[i] === '--out' ? (out = resolve(args[++i])) : keys.push(args[i].toUpperCase());
mkdirSync(out, { recursive: true });

const size = { width: 1440, height: 900 };
const browser = await chromium.launch({ channel: 'chrome' });
for (const key of keys.length ? keys : Object.keys(TOURS)) {
  const context = await browser.newContext({ viewport: size, deviceScaleFactor: 1, recordVideo: { dir: out, size } });
  const page = await context.newPage();
  page.on('pageerror', (e) => console.log(`[${key} pageerror] ${e.message}`));
  await page.goto(pathToFileURL(join(here, '..', 'dist', 'index.html')).href + `?bar=0&variant=${key}`);
  await page.waitForFunction(() => document.documentElement.dataset.ready === '1', null, { timeout: 30000 });
  await page.waitForTimeout(400);
  for (const step of TOURS[key]) await run(page, step, key);
  const video = page.video();
  await context.close();
  const file = join(out, `${key}.webm`);
  renameSync(await video.path(), file);
  console.log(file);
}
await browser.close();

async function run(page, s, key) {
  if (s.go) await page.evaluate(([k, sh, v]) => window.__ss.go(k, sh, v), [key, s.go[0], s.go[1] || '']);
  if (s.hover) { const el = await page.$(s.hover); if (el) { const b = await el.boundingBox(); await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 18 }); } else console.log(`[${key}] missing ${s.hover}`); }
  if (s.click) { const el = await page.$(s.click); if (el) { const b = await el.boundingBox(); await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 14 }); await page.mouse.down(); await page.waitForTimeout(90); await page.mouse.up(); } else console.log(`[${key}] missing ${s.click}`); }
  if (s.key) await page.keyboard.press(s.key);
  if (s.type) await page.keyboard.type(s.type, { delay: 70 });
  if (s.scroll) { await page.mouse.wheel(0, s.scroll); }
  if (s.move) await page.mouse.move(s.move[0], s.move[1], { steps: 20 });
  if (s.drag) { const [x0, y0, x1, y1] = s.drag; await page.mouse.move(x0, y0); await page.mouse.down(); await page.mouse.move(x1, y1, { steps: 24 }); await page.mouse.up(); }
  await page.waitForTimeout(s.wait ?? 800);
}
