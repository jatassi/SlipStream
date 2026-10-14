import { mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { argv, stdout } from "node:process";

import { chromium } from "playwright";

import { hostResolverRules, read } from "../manifest.ts";
import presets from "../presets.json" with { type: "json" };
import { signIn } from "../sign-in.ts";

const usage = "usage: bun web/harness/bin/sign-in.ts <run> <identity> <shell>";

const [run, identity, shell, ...rest] = argv.slice(2);
if (run === undefined || identity === undefined || shell === undefined || rest.length > 0) {
  throw new Error(usage);
}
const isShell = (name: string): name is keyof typeof presets => Object.hasOwn(presets, name);
if (!isShell(shell)) {
  throw new Error(`shell ${shell} is not one of ${Object.keys(presets).join(", ")}`);
}
const { width, height, ...device } = presets[shell];

const manifest = await read(run);
if (identity !== "admin" || manifest.admin === undefined) {
  throw new Error(`the manifest of ${run} names no identity ${identity}`);
}
const controller = manifest.nodes.find((n) => n.claim === "controller");
if (controller === undefined) {
  throw new Error(`the manifest of ${run} has no Controller`);
}

const browser = await chromium.launch({
  args: [`--host-resolver-rules=${hostResolverRules(manifest)}`],
});
try {
  const context = await browser.newContext({
    ...device,
    viewport: { width, height },
    baseURL: controller.browserUrl,
  });
  await signIn(await context.newPage(), manifest.admin);
  const out = join(run, "auth", `${identity}-${shell}.json`);
  await mkdir(dirname(out), { recursive: true });
  await context.storageState({ path: out });
  stdout.write(`${out}\n`);
} finally {
  await browser.close();
}
