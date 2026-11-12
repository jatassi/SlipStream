import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  closeSheet,
  failedGrab,
  frame,
  itemId,
  lanternKeepers,
  orchard,
  searched,
  sheet,
  workId,
} from "./manual-search.ts";
import {
  menuAction,
  needsYouAction,
  openFromCell,
  openFromEpisode,
  openFromHero,
} from "./manual-search-entries.ts";
import { openLevel, reveal, seasons } from "./series.ts";

test.use({ scenario: "manual-search" });
test.describe.configure({ timeout: 180_000 });

const freeleech = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-CMRG";

test("Manual search from the hero shows each Indexer's Releases as it responds, the slow one last", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await page.goto(`/works/${await workId(world, orchard)}`);
  await openFromHero(page, orchard);
  const opened = sheet(page);
  await expect(opened).toContainText("Searching 4 Indexers. 3 have responded.");
  await expect(opened).toContainText("2 Grabs get you everything.");
  await expect(opened.getByRole("list", { name: "Grabs" })).toContainText(
    "It gets you 1080p (standard) for The Quiet Orchard.",
  );
  await opened.getByRole("button", { name: "Details" }).first().click();
  const indexers = opened.getByRole("list", { name: "Indexers" });
  await expect(indexers.getByRole("listitem").filter({ hasText: "Heron" })).toContainText(
    "Searching",
  );
  await expect(indexers.getByRole("listitem").filter({ hasText: "Tamarack" })).toContainText(
    "Found 1 Release",
  );
  await expectAccessible(page);
  await frame(page, testInfo, "open-streaming");

  await searched(opened);
  await expect(indexers.getByRole("listitem").filter({ hasText: "Heron" })).toContainText(
    /^Heron\s*Found \d+ Releases/u,
  );
  await opened.getByText(/^All Releases \(\d+\)$/u).click();
  const all = opened.getByRole("list", { name: "All Releases" });
  const late = all.locator(":scope > li").filter({ hasText: "CMRG" });
  await expect(late).toContainText("Freeleech");
  await late.getByRole("button", { name: "Details" }).click();
  await expect(late).toContainText(freeleech);
  await expect(late).toContainText("Heron");
  await expectAccessible(page);
  await frame(page, testInfo, "open-answered");
});

test("Manual search opens from a cell, a season, an episode and a Needs you card, each titled with its scope", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  await page.goto(`/works/${movie}`);
  await openFromCell(page, "1080p");
  await expect(sheet(page)).toContainText("It gets you 1080p (standard) for The Quiet Orchard.");
  await expect(sheet(page)).not.toContainText("as an upgrade");
  await closeSheet(page);

  const series = await workId(world, lanternKeepers);
  const e10 = await itemId(world, series, "S02E10");
  await failedGrab(world, series, e10);

  await page.goto(`/works/${series}`);
  await needsYouAction(page, "Manual search");
  await expect(sheet(page, "S02E10 of The Lantern Keepers")).toContainText(
    "It gets you 1080p for E10.",
  );
  await expectAccessible(page);
  await frame(page, testInfo, "open-needs-you");
  await closeSheet(page);

  await reveal(page, seasons(page));
  await page.getByRole("button", { name: "More actions for Season 2" }).click();
  await menuAction(page, "Actions for Season 2", "Manual search");
  const season = sheet(page, lanternKeepers);
  await searched(season);
  await expect(season).toContainText("5 Grabs get you everything.");
  await expect(season).toContainText("It gets you 1080p for E05.");
  await expect(season).toContainText(
    "It gets you 1080p for E09 and E10. E01 to E04 keep their files.",
  );
  await expectAccessible(page);
  await frame(page, testInfo, "open-season");
  await closeSheet(page);

  await openLevel(page, "Season 2");
  await openFromEpisode(page, "S02E06");
  const episode = sheet(page, "S02E06 of The Lantern Keepers");
  await expect(episode.getByRole("searchbox", { name: "Query" })).toBeVisible();
  await expect(episode).toContainText("It gets you 1080p for E06.");
  await expect(episode).not.toContainText("for E05");
});
