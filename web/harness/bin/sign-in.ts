import { mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { argv, stdout } from "node:process";

import { chromium } from "playwright";

import { node, identity as pick, read } from "../manifest.ts";
import { declinePasskeyPrompt, submitPin } from "../sign-in.ts";
import { startupPlayedScript } from "../startup.ts";

const usage = "usage: bun web/harness/bin/sign-in.ts <run> <identity> <shell> [<node>]";

const [run, identity, shell, name, ...rest] = argv.slice(2);
if (run === undefined || identity === undefined || shell === undefined || rest.length > 0) {
  throw new Error(usage);
}
const manifest = await read(run);
const { presets, hostResolverRules } = manifest.browser;
const preset = presets[shell];
if (preset === undefined) {
  throw new Error(`shell ${shell} is not one of ${Object.keys(presets).join(", ")}`);
}
const { width, height, ...device } = preset;
const person = pick(manifest, identity);
// A Node named on the line takes the sign-in, such as a new machine a
// Backup restored onto, which the manifest still lists as Unclaimed.
const target = name === undefined ? manifest.nodes.find((n) => n.claim === "controller") : node(manifest, name);
if (target === undefined) {
  throw new Error(`the manifest of ${run} has no Controller`);
}

const browser = await chromium.launch({
  args: [`--host-resolver-rules=${hostResolverRules}`],
});
try {
  const context = await browser.newContext({
    ...device,
    viewport: { width, height },
    baseURL: target.browserUrl,
  });
  await context.addInitScript({ path: startupPlayedScript });
  // The page after sign-in may be the Library, the re-home step of a
  // restore or the Superseded page, so only leaving /sign-in is awaited.
  const page = await context.newPage();
  await page.goto("/sign-in");
  const answer = await submitPin(page, person.email, person.pin);
  if (!answer.ok()) {
    throw new Error(`${person.email} could not sign in to ${target.name}: ${answer.status()}`);
  }
  await declinePasskeyPrompt(page, answer);
  await page.waitForURL((url) => url.pathname !== "/sign-in");
  const out = join(run, "auth", `${identity}-${shell}.json`);
  await mkdir(dirname(out), { recursive: true });
  await context.storageState({ path: out });
  stdout.write(`${out}\n`);
} finally {
  await browser.close();
}
