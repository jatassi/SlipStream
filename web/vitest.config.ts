import { playwright } from "@vitest/browser-playwright";
import { msw } from "msw/vite";
import { defineConfig, mergeConfig } from "vitest/config";

import presets from "./harness/presets.json" with { type: "json" };
import viteConfig from "./vite.config.ts";

export default defineConfig((env) =>
  mergeConfig(viteConfig(env), {
    plugins: [msw({ mode: "worker-only" })],
    server: { fs: { allow: [".."] } },
    test: {
      include: ["src/**/*.test.{ts,tsx}"],
      setupFiles: ["src/test/setup.ts"],
      expect: { poll: { timeout: 3000 } },
      browser: {
        enabled: true,
        headless: true,
        provider: playwright(),
        instances: [{ browser: "chromium" }],
        viewport: { width: presets.desk.width, height: presets.desk.height },
      },
    },
  }),
);
