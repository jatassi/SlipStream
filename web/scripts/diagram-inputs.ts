import { relative } from "node:path";
import { stdout } from "node:process";
import { fileURLToPath } from "node:url";

// Prints every repo file the Diagram check's entrypoints import, one path
// per line from the repo root, so the release tool's lint can hold the
// check's change roots to what it reads.

const WEB = fileURLToPath(new URL("..", import.meta.url));
const ROOT = fileURLToPath(new URL("../..", import.meta.url));

const { metafile } = await Bun.build({
  entrypoints: [
    "src/diagram/harness/main.tsx",
    "scripts/diagram-check.ts",
    "scripts/diagram-audit.ts",
    "vite.config.ts",
  ].map((entry) => WEB + entry),
  target: "bun",
  packages: "external",
  metafile: true,
});
for (const input of Object.keys(metafile.inputs).toSorted()) {
  stdout.write(`${relative(ROOT, input).replaceAll("\\", "/")}\n`);
}
