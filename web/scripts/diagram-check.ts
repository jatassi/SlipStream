import { spawn } from "node:child_process";
import { mkdirSync } from "node:fs";
import { join } from "node:path";
import { env, exit } from "node:process";
import { fileURLToPath } from "node:url";

import { type Page, chromium } from "playwright";
import { createServer } from "vite";

import { LINE_ENDS } from "../../.claude/skills/anatomy/scripts/line-ends.mjs";
import { FIXTURE_SIZES } from "../src/diagram/fixtures";
import { say } from "./diagram-record";

// The Diagram's checks against the dev-only harness page: every texture line
// ends on something, the anatomy audit passes with the page's own depth
// keys, a frame of the 8-Node fixture stays inside its budget, and 4x
// close-ups land in .verify/diagram for a person to read.

const WEB = fileURLToPath(new URL("..", import.meta.url));
const ROOT = join(WEB, "..");
const CLOSE_UPS = join(ROOT, ".verify/diagram");
const FRAME_BUDGET_MS = 3;
const BUDGET_FIXTURE = 8;
const SETTLE_MS = 6000;
const LINES_SETTLE_MS = 2700;

env["DEV_API_ORIGIN"] ??= "http://127.0.0.1:1";

const server = await createServer({
  root: WEB,
  logLevel: "warn",
  server: { host: "127.0.0.1", port: 0 },
});
await server.listen();
const origin = server.resolvedUrls?.local[0];
if (origin === undefined) {
  throw new Error("the harness server has no local address");
}
const pageOf = (size: number): string =>
  `${origin}src/diagram/harness/index.html?fixture=${String(size)}`;

const failures: string[] = [];

/** Runs a program to its end without blocking this process, which serves the harness page meanwhile. */
function runOf(command: string, args: readonly string[]): Promise<number | null> {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd: WEB, stdio: "inherit" });
    child.on("error", reject);
    child.on("close", resolve);
  });
}

type LineEnds = {
  readonly problems: readonly unknown[];
  readonly ends: number;
  readonly lines: number;
};

// The anatomy skill's own line-ends check (what its lines.mjs runs), read
// through Playwright: lines.mjs's driver aborts its navigation on Windows runners.
async function lines(page: Page, size: number) {
  await page.goto(pageOf(size), { waitUntil: "networkidle" });
  await page.waitForTimeout(LINES_SETTLE_MS);
  const result: LineEnds = await page.evaluate(`(${LINE_ENDS})({ select: ".diagram" })`);
  say(
    `fixture ${String(size)}: line ends: ${String(result.problems.length)} of ${String(result.ends)} ends on ${String(result.lines)} lines float`,
  );
  if (result.problems.length > 0) {
    failures.push(`line ends on fixture ${String(size)}`);
  }
}

async function audit() {
  if ((await runOf("bun", [join(WEB, "scripts/diagram-audit.ts")])) !== 0) {
    failures.push("anatomy audit");
  }
}

const percentile = (values: readonly number[], share: number): number =>
  values.toSorted((a, b) => a - b)[
    Math.min(values.length - 1, Math.floor(values.length * share))
  ] ?? 0;

async function frameBudget(page: Page) {
  await page.goto(pageOf(BUDGET_FIXTURE), { waitUntil: "networkidle" });
  await page.waitForTimeout(SETTLE_MS);
  const ticks: number[] = await page.evaluate("window.__diagramTickMs ?? []");
  const p95 = percentile(ticks, 0.95);
  say(
    `frame: p95 ${p95.toFixed(2)} ms over ${String(ticks.length)} frames (budget ${String(FRAME_BUDGET_MS)} ms)`,
  );
  if (ticks.length === 0 || p95 >= FRAME_BUDGET_MS) {
    failures.push("frame budget");
  }
}

const CLOSE_UP_PARTS = [
  ["controller", "[data-controller]"],
  ["chips", "[data-controller] [data-chip]"],
  ["piece-board", "[data-controller] [data-role='downloading']"],
  ["link", "[data-link]"],
] as const;

type Box = {
  readonly x: number;
  readonly y: number;
  readonly width: number;
  readonly height: number;
};

async function clipOf(page: Page, selector: string): Promise<Box> {
  const boxes: Box[] = await page.locator(selector).evaluateAll((elements) =>
    elements.map((element) => {
      const { x, y, width, height } = element.getBoundingClientRect();
      return { x, y, width, height };
    }),
  );
  const x = Math.min(...boxes.map((box) => box.x)) - 8;
  const y = Math.min(...boxes.map((box) => box.y)) - 8;
  return {
    x,
    y,
    width: Math.max(...boxes.map((box) => box.x + box.width)) - x + 8,
    height: Math.max(...boxes.map((box) => box.y + box.height)) - y + 8,
  };
}

async function closeUps(page: Page) {
  mkdirSync(CLOSE_UPS, { recursive: true });
  await page.goto(pageOf(2), { waitUntil: "networkidle" });
  await page.waitForTimeout(2000);
  await Promise.all(
    CLOSE_UP_PARTS.map(async ([name, selector]) => {
      const clip = await clipOf(page, selector);
      await page.screenshot({ path: join(CLOSE_UPS, `${name}.png`), clip });
    }),
  );
  say(`close-ups: ${CLOSE_UPS}`);
}

try {
  const browser = await chromium.launch();
  const linesPage = await browser.newPage({ viewport: { width: 1440, height: 1100 } });
  await FIXTURE_SIZES.reduce(async (prior, size) => {
    await prior;
    await lines(linesPage, size);
  }, Promise.resolve());
  // An open page keeps animating and would share the frame budget's CPU.
  await linesPage.close();
  await audit();
  await frameBudget(await browser.newPage({ viewport: { width: 1400, height: 1000 } }));
  await closeUps(
    await browser.newPage({ viewport: { width: 1400, height: 1000 }, deviceScaleFactor: 4 }),
  );
  await browser.close();
} finally {
  await server.close();
}
if (failures.length > 0) {
  say(`diagram check failed: ${failures.join(", ")}`);
  exit(1);
}
say("diagram check passed");
