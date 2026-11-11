import type { Page, TestInfo } from "@playwright/test";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import {
  blocklisted,
  frame,
  grabRows,
  lanternKeepers,
  openBlocklist,
  orchard,
  searched,
  sheet,
  workId,
} from "./manual-search.ts";
import { openFromHero } from "./manual-search-entries.ts";
import { episodeRow, openLevel, reveal } from "./series.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "manual-search" });
test.describe.configure({ timeout: 300_000 });

const webdlTitle = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW";
const e06 = "The.Lantern.Keepers.S02E06.1080p.WEB-DL.H264-FKW";

/** The episode sheet's Manual search, its row's Details in place, and a Grab, in one sheet at a time. */
async function grabFromEpisodeSheet(page: Page, world: World, testInfo: TestInfo): Promise<void> {
  const series = await workId(world, lanternKeepers);
  await page.goto(`/works/${series}`);
  await openLevel(page, "Season 2");
  await clickWhenStill(await reveal(page, episodeRow(page, "S02E06")));
  await page
    .getByRole("dialog", { name: "S02E06 Shoal Water" })
    .getByRole("button", { name: "Manual search" })
    .click();
  const manual = sheet(page, "S02E06 of The Lantern Keepers");
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await searched(manual);
  const row = grabRows(manual).first();
  await expect(row).toContainText("It gets you 1080p for E06.");
  await row.getByRole("button", { name: "Details" }).click();
  await expect(row).toContainText(e06);
  await expect(row).toContainText("IndexersFixture Site, Heron");
  const sendTo = await row.getByRole("combobox", { name: "Send to" }).boundingBox();
  expect(
    (sendTo?.x ?? Infinity) + (sendTo?.width ?? 0),
    "Send to fits the phone's width",
  ).toBeLessThanOrEqual(page.viewportSize()?.width ?? 0);
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await expectAccessible(page);
  await frame(page, testInfo, "phone-sheet");

  await manual.getByRole("button", { name: "Grab 1" }).click();
  await expect(manual).toContainText("Every Target here already has its file.", {
    timeout: 60_000,
  });
  await expectAccessible(page);
  await frame(page, testInfo, "phone-grabbed");
  await page.keyboard.press("Escape");
}

test("on a phone Manual search opens from an episode's sheet, its Details open in place, it grabs, and the Blocklist lists a canceled Release @phone", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await grabFromEpisodeSheet(page, world, testInfo);

  const movie = await workId(world, orchard);
  await blocklisted(world, movie, webdlTitle);
  await page.goto(`/works/${movie}`);
  await openFromHero(page, orchard);
  const orchardSheet = sheet(page);
  await searched(orchardSheet);
  const first = grabRows(orchardSheet).first();
  await first.getByRole("button", { name: "Other choices" }).click();
  await expect(first.getByRole("list", { name: "Other choices" })).toContainText(
    "It's been on the Blocklist since",
  );
  await expectAccessible(page);
  await frame(page, testInfo, "phone-other-choices");
  await page.keyboard.press("Escape");

  await openBlocklist(page);
  await expect(page.getByRole("main")).toContainText(webdlTitle);
  await expect(page.getByRole("main")).toContainText(
    "Blocklisted through the API key when its Grab was canceled.",
  );
  await expectAccessible(page);
  await frame(page, testInfo, "phone-blocklist");
});
