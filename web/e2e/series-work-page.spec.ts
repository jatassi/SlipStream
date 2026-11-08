import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import {
  addSeries,
  episodeRow,
  expectMonitoring,
  lanternKeepers,
  levelHeader,
  openLevel,
  refreshed,
  reveal,
  seasons,
} from "./series.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "series" });

// A phone row opens the episode's sheet, so only the desktop rows hold cells to hover.
async function expectCellPopover(page: Page) {
  const cell = seasons(page).getByRole("button", { name: "S01E01 1080p, Missing" });
  const detail = page.getByRole("dialog", { name: "S01E01 1080p" });
  await cell.hover();
  await expect(detail).toContainText("Missing");
  await cell.click();
  await page.getByRole("heading", { name: "The Lantern Keepers", level: 1 }).hover();
  await expect(detail).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(detail).toBeHidden();
}
test.describe.configure({ timeout: 120_000 });

test("a series reads as its seasons, Specials last, with what is wanted open, the rest quiet and each cell's detail on hover", async ({
  adminPage: page,
  world,
}) => {
  const workId = await addSeries(world, lanternKeepers, { preset: "first" });
  await page.goto(`/works/${workId}`);
  await expect(page.getByRole("heading", { name: "The Lantern Keepers", level: 1 })).toBeVisible();
  await expect(
    page.getByText("2020 to present · Series · Lantern One · Fantasy, Drama · Continuing"),
  ).toBeVisible();
  await expectMonitoring(page, world, workId, [3, 11]);

  await expect(seasons(page).getByRole("heading", { level: 3 })).toHaveText([
    /^Season 1/u,
    /^Season 2/u,
    /^Season 3/u,
    /^Specials/u,
  ]);
  await expect(levelHeader(page, "Season 1")).toContainText("0 of 3 met");
  await expect(
    seasons(page).getByRole("button", { name: "Season 1", exact: true }),
  ).toHaveAttribute("aria-expanded", "true");
  await expect(levelHeader(page, "Season 2")).toContainText("Not monitored");
  await expect(
    seasons(page).getByRole("button", { name: "Season 2", exact: true }),
  ).toHaveAttribute("aria-expanded", "false");
  await expect(episodeRow(page, "S01E01")).toContainText("First Flame");
  await expect(episodeRow(page, "S01E01")).not.toContainText("Not monitored");
  await expect(episodeRow(page, "S02E01")).toHaveCount(0);

  await openLevel(page, "Season 2");
  await expect(episodeRow(page, "S02E01")).toContainText("The Northern Lamp");
  await expect(episodeRow(page, "S02E01")).toContainText("Not monitored");
  if (!isPhone(page)) {
    await expect(episodeRow(page, "S02E01")).toContainText("January 8, 2021");
    await expectCellPopover(page);
  }
});

test("on a phone an episode's row opens a sheet with its cells, Monitor and Details @phone", async ({
  adminPage: page,
  world,
}) => {
  const workId = await addSeries(world, lanternKeepers, { preset: "first" });
  await page.goto(`/works/${workId}`);
  await clickWhenStill(episodeRow(page, "S01E01"));
  const sheet = page.getByRole("dialog", { name: "S01E01 First Flame" });
  await expect(sheet.getByRole("list", { name: "Cells" })).toContainText("Missing");
  await expect(sheet.getByRole("switch", { name: "Monitor" })).toBeChecked();
  await expect(sheet.getByRole("button", { name: "Details" })).toBeVisible();
});

test("announced episodes fold into one line, and an episode TVDB drops leaves its season", async ({
  adminPage: page,
  world,
}) => {
  const workId = await addSeries(world, lanternKeepers, { preset: "all" });
  await world.fw.cue("tvdb-add-season", { id: 1001 });
  await world.fw.cue("tvdb-remove-episode", { id: 100122 });
  await refreshed(world, workId);

  await page.goto(`/works/${workId}`);
  await expectMonitoring(page, world, workId, [11, 13]);
  await openLevel(page, "Season 4");
  await expect(seasons(page).getByText(/^Episodes 1 to 3 air /u)).toHaveText(
    /^Episodes 1 to 3 air \p{L}+ \d+ to \p{L}+ \d+$/u,
  );
  await expect(episodeRow(page, "S04E01")).toHaveCount(0);

  await openLevel(page, "Season 2");
  await expect(await reveal(page, episodeRow(page, "S02E01"))).toBeVisible();
  await expect(seasons(page).getByText("Fog Signal")).toHaveCount(0);
  await expect(levelHeader(page, "Season 2")).toContainText("0 of 2 met");
});
