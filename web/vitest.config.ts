import { playwright } from "@vitest/browser-playwright";
import { msw } from "msw/vite";
import { defineConfig, mergeConfig } from "vitest/config";

import viteConfig from "./vite.config.ts";

const desk = { width: 1440, height: 900 };

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
        viewport: desk,
      },
    },
  }),
);
