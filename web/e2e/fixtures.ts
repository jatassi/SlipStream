import { execFile } from "node:child_process";
import { readFile, rm } from "node:fs/promises";
import { join } from "node:path";
import { env } from "node:process";

import { AxeBuilder } from "@axe-core/playwright";
import {
  type Browser,
  type BrowserContext,
  type Page,
  test as base,
  expect,
} from "@playwright/test";
import { z } from "zod";

import { type Manifest, type ManifestNode, manifestSchema } from "./manifest.ts";

export { expect };

export type Scenario = "single-machine" | "first-run";

export type World = {
  manifest: Manifest;
  node: ManifestNode;
  baseURL: string;
  admin: () => NonNullable<Manifest["admin"]>;
  setupCode: () => Promise<string>;
  api: (
    method: string,
    path: `/api/v1/${string}`,
    body?: unknown,
  ) => Promise<{ status: number; body: unknown }>;
};

type Fixtures = {
  scenario: Scenario;
  manifestPath: string;
  world: World;
  newContext: () => Promise<BrowserContext>;
  adminPage: Page;
};

type WorkerFixtures = {
  workerBrowser: Browser | null;
};

declare global {
  interface Window {
    reportCspViolation: (violation: string) => Promise<void>;
  }
}

const launchTimeout = 60;
const wcag = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22a", "wcag22aa"];

function binary(variable: "TEST_SLIPSTREAM_BINARY" | "TEST_FAKEWORLD_BINARY"): string {
  const path = env[variable];
  if (path === undefined) {
    throw new Error(`${variable} is not set; run the browser tier with release test -tier browser`);
  }
  return path;
}

function fakeworld(
  args: readonly string[],
): Promise<{ ok: boolean; stdout: string; stderr: string }> {
  return new Promise((resolve) => {
    execFile(binary("TEST_FAKEWORLD_BINARY"), args, (error, stdout, stderr) => {
      resolve({ ok: error === null, stdout, stderr });
    });
  });
}

const SetupCodeRecord = z.object({ msg: z.literal("setup code"), code: z.string() });

async function readSetupCode(log: string): Promise<string> {
  for (const line of (await readFile(log, "utf8")).split("\n")) {
    const record = line === "" ? null : SetupCodeRecord.safeParse(JSON.parse(line));
    if (record?.success === true) {
      return record.data.code;
    }
  }
  throw new Error(`${log} has no setup code record`);
}

export async function expectAccessible(page: Page): Promise<void> {
  const { violations } = await new AxeBuilder({ page }).withTags(wcag).analyze();
  const found = violations.map(
    (violation) =>
      `${violation.id}: ${violation.nodes.map((node) => node.target.join(" ")).join(", ")}`,
  );
  expect(found, `axe on ${page.url()}`).toEqual([]);
}

async function guard(context: BrowserContext, origin: string): Promise<() => Promise<void>> {
  const foreign: string[] = [];
  const violations: string[] = [];
  const errors: string[] = [];
  context.on("request", (request) => {
    const url = new URL(request.url());
    if (url.protocol !== "data:" && url.protocol !== "blob:" && url.origin !== origin) {
      foreign.push(request.url());
    }
  });
  context.on("weberror", (error) => {
    errors.push(String(error.error()));
  });
  await context.exposeBinding("reportCspViolation", (_source, violation: string) => {
    violations.push(violation);
  });
  // The page's CSP doesn't apply to an init script.
  await context.addInitScript(() => {
    document.addEventListener("securitypolicyviolation", (event) => {
      void window.reportCspViolation(
        `${event.violatedDirective} ${event.blockedURI} at ${event.sourceFile}:${event.lineNumber}`,
      );
    });
  });
  return async () => {
    expect.soft(foreign, "requests to an origin other than the Node's").toEqual([]);
    expect.soft(violations, "CSP violations").toEqual([]);
    expect.soft(errors, "page errors").toEqual([]);
    await Promise.all(context.pages().map((page) => expectAccessible(page)));
  };
}

export const test = base.extend<Fixtures, WorkerFixtures>({
  scenario: ["single-machine", { option: true }],

  manifestPath: [
    async ({ scenario }, use, testInfo) => {
      const runDir = testInfo.outputPath("run");
      const start = await fakeworld([
        "start",
        "--binary",
        binary("TEST_SLIPSTREAM_BINARY"),
        "--scenario",
        scenario,
        "--run-dir",
        runDir,
        "--timeout",
        `${launchTimeout}s`,
      ]);
      if (!start.ok) {
        throw new Error(`fakeworld start: ${start.stderr}`);
      }
      await use(start.stdout.trim());
      const stop = await fakeworld(["stop", runDir, "--timeout", `${launchTimeout}s`]);
      await testInfo.attach("fakeworld stop", {
        body: stop.stdout,
        contentType: "application/json",
      });
      if (!stop.ok) {
        throw new Error(`fakeworld stop: ${stop.stderr}`);
      }
      // Only a failed test keeps its run dir, as in the Go tier.
      if (testInfo.status === testInfo.expectedStatus) {
        await rm(runDir, { recursive: true });
      }
    },
    { timeout: (launchTimeout + 10) * 1000 },
  ],

  world: async ({ manifestPath, scenario, browserName }, use) => {
    const manifest = manifestSchema.parse(JSON.parse(await readFile(manifestPath, "utf8")));
    const [node] = manifest.nodes;
    const app = await fetch(node.url);
    if (!app.ok) {
      throw new Error("TEST_SLIPSTREAM_BINARY has no embedded web app; run release build");
    }
    await use({
      manifest,
      node,
      baseURL: browserName === "chromium" ? node.browserUrl : node.url,
      admin: () => {
        if (manifest.admin === undefined) {
          throw new Error(`Scenario ${scenario} has no Admin`);
        }
        return manifest.admin;
      },
      setupCode: () => readSetupCode(node.log),
      api: async (method, path, body) => {
        const response = await fetch(new URL(path, node.url), {
          method,
          headers: manifest.apiKey === undefined ? {} : { "X-Api-Key": manifest.apiKey },
          ...(body === undefined
            ? {}
            : { body: new Blob([JSON.stringify(body)], { type: "application/json" }) }),
        });
        return {
          status: response.status,
          body: response.status === 204 ? null : ((await response.json()) as unknown),
        };
      },
    });
  },

  // A fresh Firefox on Windows can stall its main thread for seconds while it
  // starts up, so engines that take nothing from a world at launch share one
  // browser per worker.
  workerBrowser: [
    async ({ playwright, browserName, headless, launchOptions }, use) => {
      if (browserName === "chromium") {
        await use(null);
        return;
      }
      const browser = await playwright[browserName].launch({ ...launchOptions, headless });
      await use(browser);
      await browser.close();
    },
    { scope: "worker" },
  ],

  // Host resolver rules are launch flags and a world's names exist only once
  // it is up, so Chromium launches a browser per test.
  newContext: async (
    { playwright, headless, launchOptions, contextOptions, world, workerBrowser },
    use,
  ) => {
    const hostRules = world.manifest.nodes
      .map((node) => `MAP ${node.host} ${node.address}`)
      .join(",");
    const browser =
      workerBrowser ??
      (await playwright.chromium.launch({
        ...launchOptions,
        headless,
        args: [...(launchOptions.args ?? []), `--host-resolver-rules=${hostRules}`],
      }));
    const origin = new URL(world.baseURL).origin;
    const contexts: BrowserContext[] = [];
    const checks: (() => Promise<void>)[] = [];
    await use(async () => {
      const context = await browser.newContext({ ...contextOptions, baseURL: world.baseURL });
      contexts.push(context);
      checks.push(await guard(context, origin));
      return context;
    });
    await Promise.all(checks.map((check) => check()));
    // Playwright saves a context's trace when the context closes.
    await Promise.all(contexts.map((context) => context.close()));
    if (workerBrowser === null) {
      await browser.close();
    }
  },

  context: async ({ newContext }, use) => {
    await use(await newContext());
  },

  adminPage: async ({ page, world }, use) => {
    const admin = world.admin();
    await page.goto("/sign-in");
    await page.getByLabel("Email").fill(admin.email);
    await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially(admin.pin);
    await expect(
      page.getByRole("heading", { name: `Signed in as ${admin.displayName}` }),
    ).toBeVisible();
    await page.context().storageState({ path: join(world.manifest.runDir, "admin.json") });
    await use(page);
  },
});

export function skipFlaky(issue: `https://github.com/jatassi/SlipStream/issues/${number}`): void {
  test.skip(true, issue);
}
