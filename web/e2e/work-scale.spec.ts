import type { Page } from "@playwright/test";

import { z } from "zod";

import { type World, expect, test } from "./fixtures.ts";
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

const Rows = z.object({ rows: z.array(z.object({ id: z.string(), label: z.string() })) });
const Roots = z.object({ roots: z.array(z.object({ id: z.string() })) });
const Profiles = z.object({
  profiles: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      contentType: z.string(),
      targets: z.unknown(),
      editions: z.unknown(),
    }),
  ),
});

async function scannedSeries(world: World): Promise<string> {
  const [root] = Roots.parse((await world.api("GET", "/api/v1/roots")).body).roots;
  expect(root, "the Scenario has a TV root").toBeDefined();
  const scan = await world.api("POST", `/api/v1/roots/${root?.id ?? ""}/scan`);
  expect(scan.status, "the scan starts").toBeLessThan(300);
  let workId = "";
  await expect
    .poll(
      async () => {
        const { rows } = Rows.parse((await world.api("GET", "/api/v1/library")).body);
        workId = rows[0]?.id ?? "";
        return rows.length;
      },
      { timeout: 120_000 },
    )
    .toBe(1);
  return workId;
}

async function wantEnglish(world: World, workId: string) {
  const { profiles } = Profiles.parse((await world.api("GET", "/api/v1/profiles")).body);
  const profile = profiles.find(({ contentType }) => contentType === "series");
  expect(profile, "the library has a series Profile").toBeDefined();
  if (profile === undefined) {
    return;
  }
  const { id, name, contentType, targets, editions } = profile;
  const saved = await world.api("PUT", `/api/v1/profiles/${id}?confirm=0&confirmSubtitles=0`, {
    name,
    contentType,
    targets,
    editions,
    languages: [{ language: "en", forced: false, hearingImpaired: "allow", audio: "always" }],
  });
  expect(saved.status, "English is wanted").toBe(200);
  await expect
    .poll(
      async () => {
        const read = await world.api("GET", `/api/v1/works/${workId}/subtitles`);
        return z.object({ files: z.array(z.unknown()) }).parse(read.body).files.length;
      },
      { timeout: 120_000 },
    )
    .toBe(240);
}

test.describe("a series of 240 episode files", () => {
  test.use({ scenario: "series-files-scale", launchSeconds: 600 });

  // Chromium alone reports long tasks.
  test("the Files section folds 240 files per season and opens a season within its long-task budget @cdp @alone", async ({
    adminPage: page,
    world,
  }, testInfo) => {
    const workId = await scannedSeries(world);
    await wantEnglish(world, workId);
    await page.addInitScript(watch);

    await page.goto(`/works/${workId}`);
    const first = page.getByRole("region", { name: "Season 1" });
    await expect(first.getByRole("button", { name: /^Season 1/u })).toHaveAttribute(
      "aria-expanded",
      "true",
      { timeout: 30_000 },
    );
    await expect(first.getByRole("button", { name: /^Subtitles: English/u })).toHaveCount(30);
    const open = await takeLongTasks(page);

    await page
      .getByRole("region", { name: "Season 8" })
      .getByRole("button", { name: /^Season 8/u })
      .click();
    await expect(
      page
        .getByRole("region", { name: "Season 8" })
        .getByRole("button", { name: /^Subtitles: English/u }),
    ).toHaveCount(30);
    const fold = await takeLongTasks(page);

    const measured = { open: summary(open), fold: summary(fold) };
    await testInfo.attach("work-files-scale", {
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
});
