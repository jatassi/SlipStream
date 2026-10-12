import { env } from "node:process";

import babel from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { type ServerOptions, defineConfig } from "vite";

function devServer(command: string, mode: string): ServerOptions {
  if (command !== "serve" || mode === "test") {
    return {};
  }
  const target = env["DEV_API_ORIGIN"];
  if (target === undefined || target === "") {
    throw new Error("DEV_API_ORIGIN is not set; start the dev server with make dev");
  }
  return { proxy: { "/api": { target, changeOrigin: false } } };
}

export default defineConfig(({ command, mode }) => ({
  plugins: [
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    react(),
    babel({ presets: [reactCompilerPreset()] }),
    tailwindcss(),
  ],
  // src/zod-jitless.ts must run before any module in another chunk builds a
  // Zod schema.
  build: { rolldownOptions: { output: { strictExecutionOrder: true } } },
  server: devServer(command, mode),
}));
