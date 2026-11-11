import type { Locator, Page } from "@playwright/test";

import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  frame,
  grabRows,
  openFromHero,
  orchard,
  searched,
  sheet,
  useChoice,
  workId,
} from "./manual-search.ts";

test.use({ scenario: "manual-search" });
test.describe.configure({ timeout: 180_000 });

const bluray = "Bluray-1080p · FKW · 9.8 GB";
const webdl = "WEBDL-1080p · FKW · 8.2 GB";
const cmrg = "WEBDL-1080p · CMRG · 8.1 GB";
const both = "It gets you 1080p (standard) for The Quiet Orchard, and HD (standard) as an upgrade.";

/** Tamarack's Bluray for 1080p, and the WEB-DL every automatic Indexer lists for HD. */
async function readsTwoGrabs(opened: Locator): Promise<void> {
  await expect(opened).toContainText("2 Grabs get you everything.");
  const rows = grabRows(opened);
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText(bluray);
  await expect(rows.nth(0)).toContainText("It gets you 1080p (standard) for The Quiet Orchard.");
  await expect(rows.nth(1)).toContainText(webdl);
  await expect(rows.nth(1)).toContainText(
    "It gets you HD (standard) for The Quiet Orchard as an upgrade.",
  );
}

/** Once the grab's cells leave the plan, each row grabbed stays on the sheet saying so. */
async function keepsGrabbed(opened: Locator, words: readonly string[]): Promise<void> {
  await expect(opened).toContainText("Every Target here has its file or a Grab downloading one.");
  const rows = grabRows(opened);
  await expect(rows).toHaveCount(words.length);
  await expect(rows).toContainText([...words]);
  await expect(rows.getByText("Grabbed.")).toHaveCount(words.length);
}

/** Opened again, the sheet keeps no row it grabbed, and each cell its Grabs hold names their Release. */
async function reopensHeld(page: Page, opened: Locator, lines: readonly string[]): Promise<void> {
  await page.keyboard.press("Escape");
  await openFromHero(page, orchard);
  const held = opened.getByRole("list", { name: "Cells a Grab is downloading" });
  await expect(held.getByRole("listitem")).toHaveText([...lines]);
  await expect(grabRows(opened)).toHaveCount(0);
}

test("the plan reads as automatic search would grab, a swapped row holds while Indexers answer, and Grab takes the ticked rows", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  await page.goto(`/works/${movie}`);
  await openFromHero(page, orchard);
  const opened = sheet(page);
  const rows = grabRows(opened);
  await readsTwoGrabs(opened);
  await expect(opened.getByRole("button", { name: "Grab 2" })).toBeEnabled();
  await rows.nth(1).getByRole("checkbox", { name: webdl }).uncheck();
  await expect(opened.getByRole("button", { name: "Grab 1" })).toBeEnabled();
  await rows.nth(1).getByRole("checkbox", { name: webdl }).check();
  await expectAccessible(page);
  await frame(page, testInfo, "plan");

  await expect(opened).toContainText("3 have responded.");
  await useChoice(rows.nth(0), webdl);
  await expect(opened).toContainText("1 Grab gets you everything.");
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText(webdl);
  await expect(rows.nth(0)).toContainText(both);
  await expect(
    rows.nth(0).getByRole("list", { name: "Other choices" }).locator(":scope > li").first(),
  ).toContainText(`${bluray}Use thisFrom Tamarack, which automatic search doesn't use.`);

  await searched(opened);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText(webdl);
  await expect(opened).toContainText("1 Grab gets you everything.");

  await world.fw.fault.set({ id: "swarm-stalled", target: "seeders", fault: "stall" });
  await opened.getByRole("button", { name: "Grab 1" }).click();
  await keepsGrabbed(opened, [webdl]);
  await expect(opened.getByRole("list", { name: "Cells a Grab is downloading" })).toHaveCount(0);
  await expectAccessible(page);
  await frame(page, testInfo, "plan-grabbed");

  await reopensHeld(page, opened, [
    `A Grab is downloading ${webdl} for 1080p (standard).`,
    `A Grab is downloading ${webdl} for HD (standard).`,
  ]);

  await world.fw.fault.clear("swarm-stalled");
  await expect(opened).toContainText("Every Target here already has its file.", {
    timeout: 60_000,
  });
});

test("a Release pinned in the HD row that also fills 1080p gets the HD row alone, and Grab takes both rows", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  await page.goto(`/works/${movie}`);
  await openFromHero(page, orchard);
  const opened = sheet(page);
  const rows = grabRows(opened);
  await searched(opened);
  await readsTwoGrabs(opened);

  await useChoice(rows.nth(1), cmrg);
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText(bluray);
  await expect(rows.nth(1)).toContainText(cmrg);
  await expect(rows.nth(1)).toContainText(
    "It gets you HD (standard) for The Quiet Orchard as an upgrade.",
  );

  await world.fw.fault.set({ id: "swarm-stalled", target: "seeders", fault: "stall" });
  await opened.getByRole("button", { name: "Grab 2" }).click();
  await keepsGrabbed(opened, [bluray, cmrg]);
  await frame(page, testInfo, "plan-pinned-grabbed");

  await reopensHeld(page, opened, [
    `A Grab is downloading ${bluray} for 1080p (standard).`,
    `A Grab is downloading ${cmrg} for HD (standard).`,
  ]);
  await world.fw.fault.clear("swarm-stalled");
});
