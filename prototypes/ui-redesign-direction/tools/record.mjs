// PROTOTYPE tool: record one screencast per direction from dist/index.html.
// node tools/record.mjs [--out dir] [A B C]
// Each tour is a list of steps: go to a route, move or click, wait. Frames come from Chrome's
// screencast as high-quality JPEGs, timed by their own timestamps, and ffmpeg encodes them to
// <key>.mp4 (H.264, 30 fps). Playwright's built-in video is too lossy for the smoke in A.
import { chromium } from 'playwright-core';
import { mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
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
  const context = await browser.newContext({ viewport: size, deviceScaleFactor: 1 });
  const page = await context.newPage();
  const frames = join(out, `${key}-frames`);
  rmSync(frames, { recursive: true, force: true });
  mkdirSync(frames, { recursive: true });
  const shots = [];
  let recording = false;
  const cdp = await context.newCDPSession(page);
  cdp.on('Page.screencastFrame', async ({ data, metadata, sessionId }) => {
    if (recording) {
      const file = join(frames, `${String(shots.length).padStart(5, '0')}.jpg`);
      writeFileSync(file, Buffer.from(data, 'base64'));
      shots.push({ file, t: metadata.timestamp });
    }
    await cdp.send('Page.screencastFrameAck', { sessionId }).catch(() => {});
  });
  page.on('pageerror', (e) => console.log(`[${key} pageerror] ${e.message}`));
  await page.goto(pathToFileURL(join(here, '..', 'dist', 'index.html')).href + `?bar=0&variant=${key}`);
  await page.waitForFunction(() => document.documentElement.dataset.ready === '1', null, { timeout: 30000 });
  await page.waitForTimeout(400);
  await cdp.send('Page.startScreencast', { format: 'jpeg', quality: 92, maxWidth: size.width, maxHeight: size.height, everyNthFrame: 1 });
  recording = true;
  for (const step of TOURS[key]) await run(page, step, key);
  recording = false;
  await cdp.send('Page.stopScreencast');
  await context.close();
  // The screencast only sends a frame when the page paints, so each frame lasts until the next.
  const list = shots.map((f, i) => `file '${f.file}'\nduration ${Math.max(0.001, ((shots[i + 1]?.t ?? f.t + 0.5) - f.t)).toFixed(4)}`).join('\n') + `\nfile '${shots.at(-1).file}'\n`;
  writeFileSync(join(frames, 'list.txt'), list);
  const file = join(out, `${key}.mp4`);
  execFileSync('ffmpeg', ['-loglevel', 'error', '-y', '-f', 'concat', '-safe', '0', '-i', join(frames, 'list.txt'), '-vf', 'fps=30,format=yuv420p', '-c:v', 'libx264', '-preset', 'slow', '-crf', '20', '-movflags', '+faststart', file]);
  rmSync(frames, { recursive: true, force: true });
  console.log(`${file} · ${shots.length} frames`);
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
