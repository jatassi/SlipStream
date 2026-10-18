import { playwright } from "@vitest/browser-playwright";
import { msw } from "msw/vite";
import { defineConfig, mergeConfig } from "vitest/config";
import type { BrowserCommand } from "vitest/node";

import presets from "./harness/presets.json" with { type: "json" };
import viteConfig from "./vite.config.ts";

const emulateReducedMotion: BrowserCommand<[reduce: boolean]> = async ({ page }, reduce) => {
  await page.emulateMedia({ reducedMotion: reduce ? "reduce" : "no-preference" });
};

// A pointer drag in steps, as a hand moves, from one element's centre to
// another's. Playwright's own drag jumps there in one move.
const drag: BrowserCommand<[from: string, to: string]> = async (context, from, to) => {
  const frame = await context.frame();
  const centre = async (selector: string) => {
    const box = await frame.locator(selector).boundingBox();
    if (box === null) {
      throw new Error(`${selector} is not shown`);
    }
    return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
  };
  const start = await centre(from);
  const end = await centre(to);
  const { mouse } = context.page;
  await mouse.move(start.x, start.y);
  await mouse.down();
  await mouse.move(end.x, end.y, { steps: 20 });
  await mouse.up();
};

/** A point on an element, as shares of its box from its top left corner. */
type At = { readonly selector: string; readonly at: readonly [number, number] };

async function pointOf(context: Parameters<BrowserCommand>[0], { selector, at }: At) {
  const box = await (await context.frame()).locator(selector).first().boundingBox();
  if (box === null) {
    throw new Error(`${selector} is not shown`);
  }
  return { x: box.x + box.width * at[0], y: box.y + box.height * at[1] };
}

type Fingers = readonly (readonly [number, number])[];

const STEPS = 12;

/** Touches the screen with each finger, moves every one to its end in steps, and lifts them. */
async function touch(context: Parameters<BrowserCommand>[0], from: Fingers, to: Fingers) {
  const cdp = await context.page.context().newCDPSession(context.page);
  const send = (type: "touchStart" | "touchMove" | "touchEnd", fingers: Fingers) =>
    cdp.send("Input.dispatchTouchEvent", {
      type,
      touchPoints: fingers.map(([x, y], id) => ({ x, y, id })),
    });
  await send("touchStart", from);
  for (let step = 1; step <= STEPS; step += 1) {
    // oxlint-disable-next-line no-await-in-loop -- a hand's moves come one after another
    await send(
      "touchMove",
      from.map(([x, y], finger) => {
        const [ex, ey] = to[finger] ?? [x, y];
        return [x + ((ex - x) * step) / STEPS, y + ((ey - y) * step) / STEPS] as const;
      }),
    );
  }
  await send("touchEnd", []);
  await cdp.detach();
}

// A press on a point of an element that moves by a distance in steps, with
// the mouse or one finger.
const swipe: BrowserCommand<[from: At, by: readonly [number, number], finger: boolean]> = async (
  context,
  from,
  [dx, dy],
  finger,
) => {
  const { x, y } = await pointOf(context, from);
  if (finger) {
    await touch(context, [[x, y]], [[x + dx, y + dy]]);
    return;
  }
  const { mouse } = context.page;
  await mouse.move(x, y);
  await mouse.down();
  await mouse.move(x + dx, y + dy, { steps: STEPS });
  await mouse.up();
};

// The mouse wheel turned over a point of an element.
const wheel: BrowserCommand<[over: At, deltaY: number]> = async (context, over, deltaY) => {
  const { x, y } = await pointOf(context, over);
  await context.page.mouse.move(x, y);
  await context.page.mouse.wheel(0, deltaY);
};

// Two fingers either side of a point of an element, spreading from one
// distance apart to another.
const pinch: BrowserCommand<[centre: At, from: number, to: number]> = async (
  context,
  centre,
  from,
  to,
) => {
  const { x, y } = await pointOf(context, centre);
  await touch(
    context,
    [
      [x - from / 2, y],
      [x + from / 2, y],
    ],
    [
      [x - to / 2, y],
      [x + to / 2, y],
    ],
  );
};

export default defineConfig((env) =>
  mergeConfig(viteConfig(env), {
    plugins: [msw({ mode: "worker-only" })],
    server: { fs: { allow: [".."] } },
    // Scanning every source file up front keeps Vite from finding a dependency
    // mid-run and reloading the browser, which fails the files it was running.
    // A kept cache skips the scan, so an import added since the cache was built
    // (a rebase, a new kit component) is still found mid-run: force the scan.
    optimizeDeps: { entries: ["src/**/*.{ts,tsx}", "!src/**/*.d.ts"], force: true },
    test: {
      include: ["src/**/*.test.{ts,tsx}"],
      setupFiles: ["src/test/setup.ts"],
      expect: { poll: { timeout: 3000 } },
      browser: {
        enabled: true,
        headless: true,
        // Windows runners turn animations off, which Chromium reports as
        // prefers-reduced-motion: reduce. Tests that want it emulate it.
        provider: playwright({ contextOptions: { reducedMotion: "no-preference" } }),
        instances: [{ browser: "chromium" }],
        commands: { emulateReducedMotion, drag, swipe, pinch, wheel },
        viewport: { width: presets.desktop.width, height: presets.desktop.height },
      },
    },
  }),
);
