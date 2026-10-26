import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";

// A trace records every frame of the scroll, which would add to the tasks it measures.
test.use({ scenario: "scale", launchSeconds: 1500, trace: "off" });
test.describe.configure({ timeout: 300_000 });

declare global {
  interface Window {
    firstPoster: number | undefined;
    longTasks: number[];
  }
}

// The spec's budget, which this test measures and reports.
const budget = { libraryMs: 500, firstPosterMs: 1500, longestTaskMs: 50 };

// Records when the first poster image loads and every main-thread task over 50 ms.
function watch() {
  window.longTasks = [];
  new PerformanceObserver((list) => {
    window.longTasks.push(...list.getEntries().map((entry) => entry.duration));
  }).observe({ type: "longtask", buffered: true });
  document.addEventListener(
    "load",
    (event) => {
      if (event.target instanceof HTMLImageElement && event.target.closest("[data-poster]")) {
        window.firstPoster ??= performance.now();
      }
    },
    true,
  );
}

// Scrolls the window in steps a wheel would take, one frame apart, until a
// step no longer moves it.
async function scrollToEnd(page: Page) {
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        const step = () => {
          const before = window.scrollY;
          window.scrollBy(0, 600);
          if (window.scrollY === before) {
            resolve();
            return;
          }
          requestAnimationFrame(step);
        };
        requestAnimationFrame(step);
      }),
  );
}

// Chromium alone reports long tasks.
test("a Library of 20,000 Works paints its first poster and scrolls end to end @cdp", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const started = performance.now();
  const library = await world.api("GET", "/api/v1/library");
  const libraryMs = performance.now() - started;
  expect(library.status).toBe(200);

  await page.addInitScript(watch);
  await page.goto("/library/movie");
  await expect(page.getByRole("main").getByRole("heading", { level: 2 }).first()).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => window.firstPoster), { timeout: 30_000 })
    .toBeDefined();
  const firstPosterMs = await page.evaluate(() => window.firstPoster ?? Number.NaN);

  await page.evaluate(() => {
    window.longTasks = [];
  });
  await scrollToEnd(page);
  const longTasks = await page.evaluate(() => window.longTasks);
  const measured = {
    libraryMs: Math.round(libraryMs),
    firstPosterMs: Math.round(firstPosterMs),
    longTasks: longTasks.length,
    longestTaskMs: Math.round(Math.max(0, ...longTasks)),
  };
  await testInfo.attach("scale", {
    body: JSON.stringify({ budget, measured }, null, 2),
    contentType: "application/json",
  });
  for (const [key, value] of Object.entries(measured)) {
    testInfo.annotations.push({ type: key, description: String(value) });
  }
  await expect(page.getByRole("main").getByRole("listitem").last()).toBeVisible();
});
