import { env } from "node:process";
import { fileURLToPath } from "node:url";

import babel from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { type ServerOptions, defineConfig } from "vite";

const cacheGroups = {
  react: ["node_modules/react", "node_modules/react-dom", "node_modules/scheduler"],
  "base-ui": ["node_modules/@base-ui", "node_modules/@floating-ui"],
  motion: [
    "node_modules/motion",
    "node_modules/motion-dom",
    "node_modules/motion-utils",
    "node_modules/framer-motion",
  ],
  tanstack: ["node_modules/@tanstack"],
  vendor: ["node_modules"],
  api: ["src/api/gen"],
  nodes: ["src/diagram", "vendor/anatomy", "src/routes/_app/nodes"],
};

function underAny(dirs: readonly string[]): RegExp {
  const alternatives = dirs.map((dir) => dir.split("/").join(String.raw`[/\\]`));
  return new RegExp(String.raw`[/\\](?:${alternatives.join("|")})[/\\]`, "u");
}

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

// The React Compiler compiles only components, which return JSX, and hooks,
// named use*. Its default filter matches any capitalized word, which sends
// nearly every file through Babel.
const reactCompiler = reactCompilerPreset();
const compileComponentsAndHooks = {
  ...reactCompiler,
  rolldown: { ...reactCompiler.rolldown, filter: { code: /<\/|\/>|\buse[A-Z0-9]/u } },
};

export default defineConfig(({ command, mode }) => ({
  plugins: [
    // One bundle: a route whose code loads lazily can't open while the Node is
    // down, so the shell couldn't show the outage, and WebKit keeps the failed
    // import after the Node returns.
    tanstackRouter({ target: "react", autoCodeSplitting: false }),
    react(),
    // Vendored code runs as upstream wrote and tested it, not recompiled.
    babel({
      presets: [compileComponentsAndHooks],
      sourceMap: command === "serve",
      exclude: [/[/\\]node_modules[/\\]/u, /[/\\]vendor[/\\]/u],
    }),
    tailwindcss(),
  ],
  // TypeScript reads src/diagram/kit/'s declarations for @anatomy; the bundle
  // takes the vendored sources they describe.
  resolve: { alias: { "@anatomy": fileURLToPath(new URL("vendor/anatomy/kit", import.meta.url)) } },
  // src/zod-jitless.ts must run before any module in another chunk builds a
  // Zod schema.
  build: {
    chunkSizeWarningLimit: 1200,
    rolldownOptions: {
      checks: { bundlerTimings: false },
      output: {
        strictExecutionOrder: true,
        codeSplitting: {
          groups: Object.entries(cacheGroups).map(([name, dirs]) => ({
            name,
            test: underAny(dirs),
          })),
        },
      },
    },
  },
  // The startup sequence and its stage are brand/'s masters, imported as is.
  server: { ...devServer(command, mode), fs: { allow: [".", "../brand"] } },
}));
