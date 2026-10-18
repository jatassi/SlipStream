import { playwright } from "@vitest/browser-playwright";
import { msw } from "msw/vite";
import { defineConfig, mergeConfig } from "vitest/config";
import type { BrowserCommand } from "vitest/node";

import presets from "./harness/presets.json" with { type: "json" };
import viteConfig from "./vite.config.ts";

const emulateReducedMotion: BrowserCommand<[reduce: boolean]> = async ({ page }, reduce) => {
  await page.emulateMedia({ reducedMotion: reduce ? "reduce" : "no-preference" });
};

// A pointer drag in steps, as a hand moves, from one element's centre to
// another's. Playwright's own drag jumps there in one move.
const drag: BrowserCommand<[from: string, to: string]> = async (context, from, to) => {
  const frame = await context.frame();
  const centre = async (selector: string) => {
    const box = await frame.locator(selector).boundingBox();
    if (box === null) {
      throw new Error(`${selector} is not shown`);
    }
    return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
  };
  const start = await centre(from);
  const end = await centre(to);
  const { mouse } = context.page;
  await mouse.move(start.x, start.y);
  await mouse.down();
  await mouse.move(end.x, end.y, { steps: 20 });
  await mouse.up();
};

export default defineConfig((env) =>
  mergeConfig(viteConfig(env), {
    plugins: [msw({ mode: "worker-only" })],
    server: { fs: { allow: [".."] } },
    // Scanning every source file up front keeps Vite from finding a dependency
    // mid-run and reloading the browser, which fails the files it was running.
    // A kept cache skips the scan, so an import added since the cache was built
    // (a rebase, a new kit component) is still found mid-run: force the scan.
    optimizeDeps: { entries: ["src/**/*.{ts,tsx}", "!src/**/*.d.ts"], force: true },
    test: {
      include: ["src/**/*.test.{ts,tsx}"],
      setupFiles: ["src/test/setup.ts"],
      expect: { poll: { timeout: 3000 } },
      browser: {
        enabled: true,
        headless: true,
        // Windows runners turn animations off, which Chromium reports as
        // prefers-reduced-motion: reduce. Tests that want it emulate it.
        provider: playwright({ contextOptions: { reducedMotion: "no-preference" } }),
        instances: [{ browser: "chromium" }],
        commands: { emulateReducedMotion, drag },
        viewport: { width: presets.desktop.width, height: presets.desktop.height },
      },
    },
  }),
);
