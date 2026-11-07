import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import { addSeries, expectMonitoring, seasons } from "./series.ts";

// A trace records every frame of the scroll, which would add to the tasks it measures.
test.use({ scenario: "series-scale", launchSeconds: 600, trace: "off" });
test.describe.configure({ timeout: 300_000 });

declare global {
  interface Window {
    longTasks: number[];
  }
}

// The series-scale Scenario's 12 seasons of 100 episodes.
const longSeries = "60000001";

// The spec's target for the longest main-thread task, which this test measures and reports.
const targetMs = 50;

function watch() {
  window.longTasks = [];
  new PerformanceObserver((list) => {
    window.longTasks.push(...list.getEntries().map((entry) => entry.duration));
  }).observe({ type: "longtask", buffered: true });
}

const takeLongTasks = (page: Page) =>
  page.evaluate(() => {
    const taken = window.longTasks;
    window.longTasks = [];
    return taken;
  });

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

const summary = (tasks: readonly number[]) => ({
  longTasks: tasks.length,
  longestTaskMs: Math.round(Math.max(0, ...tasks)),
});

// Chromium alone reports long tasks.
test("a series of 1,200 episodes opens, scrolls and takes a Monitor toggle, reporting its long tasks @cdp @alone", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const workId = await addSeries(world, longSeries, { preset: "all" });
  await page.addInitScript(watch);

  await page.goto(`/works/${workId}`);
  const first = seasons(page).getByRole("switch", { name: "Monitor S01E01" });
  await expect(first).toBeVisible({ timeout: 30_000 });
  await expectMonitoring(page, world, workId, [1200, 1200]);
  const open = await takeLongTasks(page);

  await scrollToEnd(page);
  await expect(seasons(page).getByRole("switch", { name: "Monitor S12E100" })).toBeVisible();
  const scroll = await takeLongTasks(page);

  await page.evaluate(() => {
    window.scrollTo(0, 0);
  });
  await expect(first).toBeVisible();
  await takeLongTasks(page);
  await first.click();
  await expect(first).toHaveAttribute("aria-checked", "false");
  await expectMonitoring(page, world, workId, [1199, 1200]);
  const toggle = await takeLongTasks(page);

  const measured = { open: summary(open), scroll: summary(scroll), toggle: summary(toggle) };
  await testInfo.attach("work-scale", {
    body: JSON.stringify({ targetMs, measured }, null, 2),
    contentType: "application/json",
  });
  for (const [phase, { longTasks, longestTaskMs }] of Object.entries(measured)) {
    testInfo.annotations.push({
      type: phase,
      description: `${longTasks} long tasks, longest ${longestTaskMs} ms against ${targetMs} ms`,
    });
  }
});
