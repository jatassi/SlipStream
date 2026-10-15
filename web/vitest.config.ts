import { playwright } from "@vitest/browser-playwright";
import { msw } from "msw/vite";
import { defineConfig, mergeConfig } from "vitest/config";
import type { BrowserCommand } from "vitest/node";

import presets from "./harness/presets.json" with { type: "json" };
import viteConfig from "./vite.config.ts";

const emulateReducedMotion: BrowserCommand<[reduce: boolean]> = async ({ page }, reduce) => {
  await page.emulateMedia({ reducedMotion: reduce ? "reduce" : "no-preference" });
};

export default defineConfig((env) =>
  mergeConfig(viteConfig(env), {
    plugins: [msw({ mode: "worker-only" })],
    server: { fs: { allow: [".."] } },
    // Scanning every source file up front keeps Vite from finding a dependency
    // mid-run and reloading the browser, which fails the files it was running.
    // A kept cache skips the scan, so an import added since the cache was built
    // (a rebase, a new kit component) is still found mid-run: force the scan.
    optimizeDeps: { entries: ["src/**/*.{ts,tsx}"], force: true },
    test: {
      include: ["src/**/*.test.{ts,tsx}"],
      setupFiles: ["src/test/setup.ts"],
      expect: { poll: { timeout: 3000 } },
      browser: {
        enabled: true,
        headless: true,
        provider: playwright(),
        instances: [{ browser: "chromium" }],
        commands: { emulateReducedMotion },
        viewport: { width: presets.desktop.width, height: presets.desktop.height },
      },
    },
  }),
);
