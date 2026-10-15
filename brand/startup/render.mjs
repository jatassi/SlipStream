import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { createServer } from "node:http";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { extname, join } from "node:path";
import { fileURLToPath } from "node:url";

const { chromium } = createRequire(new URL("../../web/package.json", import.meta.url))("playwright");

const FPS = 60;
const SHARDS = 4;
const here = fileURLToPath(new URL(".", import.meta.url));
const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".png": "image/png" };
const frames = mkdtempSync(join(tmpdir(), "slipstream-startup-"));

// Chromium refuses module scripts from file:// pages.
const server = createServer((req, res) => {
  const name = new URL(req.url, "http://localhost").pathname.slice(1);
  const type = types[extname(name)];
  if (type === undefined || name.includes("/")) {
    res.writeHead(404).end();
    return;
  }
  res.writeHead(200, { "Content-Type": type }).end(readFileSync(join(here, name)));
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const url = `http://127.0.0.1:${server.address().port}/startup.html?still`;

const browser = await chromium.launch();
async function shard(index) {
  const page = await browser.newPage({ viewport: { width: 1920, height: 1080 }, deviceScaleFactor: 2 });
  await page.goto(url);
  await page.waitForFunction(() => window.READY);
  const duration = await page.evaluate(() => window.SLIPSTREAM.DURATION);
  const count = Math.round(duration * FPS) + 1;
  for (let i = index; i < count; i += SHARDS) {
    await page.evaluate(t => window.SLIPSTREAM.render(t), i / FPS);
    await page.screenshot({ path: join(frames, `f${String(i).padStart(4, "0")}.png`) });
  }
}
await Promise.all(Array.from({ length: SHARDS }, (_, i) => shard(i)));
await browser.close();
server.close();

execFileSync("ffmpeg", [
  "-loglevel", "error", "-y", "-framerate", String(FPS), "-i", join(frames, "f%04d.png"),
  "-vf", "scale=1920:1080:flags=lanczos,format=yuv420p",
  "-c:v", "libx264", "-preset", "slow", "-crf", "14", "-tune", "grain", "-movflags", "+faststart",
  join(here, "startup.mp4"),
], { stdio: "inherit" });
rmSync(frames, { recursive: true });
