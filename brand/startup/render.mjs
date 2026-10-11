import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const FPS = 60;
const SHARDS = 4;
const here = fileURLToPath(new URL(".", import.meta.url));
const url = pathToFileURL(join(here, "startup.html")).href + "?still";
const frames = mkdtempSync(join(tmpdir(), "slipstream-startup-"));

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

execFileSync("ffmpeg", [
  "-loglevel", "error", "-y", "-framerate", String(FPS), "-i", join(frames, "f%04d.png"),
  "-vf", "scale=1920:1080:flags=lanczos,format=yuv420p",
  "-c:v", "libx264", "-preset", "slow", "-crf", "14", "-tune", "grain", "-movflags", "+faststart",
  join(here, "startup.mp4"),
], { stdio: "inherit" });
rmSync(frames, { recursive: true });
