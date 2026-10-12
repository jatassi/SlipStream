import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { argv, execPath } from "node:process";

import { engineOf, targetProjects } from "./playwright.config.ts";

const flags = argv.slice(2);
const unknown = flags.filter((flag) => flag !== "--with-deps");
if (unknown.length > 0) {
  throw new Error(`e2e:install takes only --with-deps, not ${unknown.join(" ")}`);
}
const engines = [...new Set(targetProjects().map((name) => engineOf(name)))];
const cli = fileURLToPath(import.meta.resolve("@playwright/test/cli"));
execFileSync(execPath, [cli, "install", ...flags, ...engines], { stdio: "inherit" });
