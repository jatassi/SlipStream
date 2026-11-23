import type { Page } from "@playwright/test";

import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  blocklisted,
  frame,
  grabRows,
  openGrabDetails,
  orchard,
  searched,
  sendAllTo,
  sheet,
  useChoice,
  workId,
} from "./manual-search.ts";
import { openFromHero } from "./manual-search-entries.ts";

test.use({ scenario: "manual-search" });
test.describe.configure({ timeout: 180_000 });

const filled = "Every Target here already has its file.";
const webdl = "WEBDL-1080p · FKW · 8.2 GB";
const webdlTitle = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW";

async function opened(page: Page, world: Parameters<typeof workId>[0]) {
  await page.goto(`/works/${await workId(world, orchard)}`);
  await openFromHero(page, orchard);
  const sheetLocator = sheet(page);
  await searched(sheetLocator);
  return sheetLocator;
}

/** The Work page's Grab Details for the Release, once its Grab is in the Grabs section. */
async function recordOf(page: Page, title: string) {
  await page.keyboard.press("Escape");
  const section = page.getByRole("main").getByRole("table", { name: "Grabs" });
  return openGrabDetails(page, section, title);
}

test("a Release of a cut the Work doesn't want is grabbed as its Standard Edition", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const manual = await opened(page, world);
  const row = grabRows(manual).first();
  await useChoice(row, "WEBDL-1080p · FKW · 8.9 GB");
  await expect(row).toContainText(/Extended Edition.+Standard Edition/u);
  await row.getByRole("button", { name: "Details" }).last().click();
  await expect(row).toContainText("The.Quiet.Orchard.2021.EXTENDED.1080p.WEB-DL.H264-FKW");
  await expect(row).toContainText(/1080p \(standard\).+extended cut/u);
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-edition");

  await manual.getByRole("button", { name: "Grab 1" }).click();
  await expect(manual).toContainText(filled, { timeout: 60_000 });
  const record = await recordOf(page, "The.Quiet.Orchard.2021.EXTENDED.1080p.WEB-DL.H264-FKW");
  await expect(record).toContainText(/Ada Okafor.+Standard Edition/u);
});

test("a row and the whole plan can be sent to a client Tracker routing wouldn't choose", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const manual = await opened(page, world);
  const rows = grabRows(manual);
  await sendAllTo(page, manual, "Built-in client on home");
  await expect(rows.nth(0)).toContainText("built-in client on home");
  await expect(rows.nth(1)).toContainText("built-in client on home");

  await rows.nth(1).getByRole("button", { name: "Details" }).last().click();
  const sendTo = rows.nth(1).getByRole("combobox", { name: "Send to" });
  await sendTo.click();
  await page.getByRole("option", { name: /^Where Tracker routing sends it/u }).click();
  await expect(rows.nth(1)).toContainText(/Tracker routing.+built-in client on home/u);
  await expect(rows.nth(0)).toContainText("built-in client on home");
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-client");

  await rows.nth(1).getByRole("checkbox").uncheck();
  await manual.getByRole("button", { name: "Grab 1" }).click();
  const record = await recordOf(page, "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW");
  await expect(record).toContainText(/Ada Okafor.+built-in client on home/u);
  await expect(record).not.toContainText("Tracker routing");
});

test("a blocklisted Release says since when and why, and Remove from Blocklist and grab takes it back", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  await blocklisted(world, movie, webdlTitle);
  const manual = await opened(page, world);
  const rows = grabRows(manual);
  const first = rows.first();
  await first.getByRole("button", { name: "Other choices" }).click();
  const choice = first
    .getByRole("list", { name: "Other choices" })
    .locator(":scope > li")
    .filter({ hasText: webdl });
  await expect(choice).toContainText(/Blocklist since .+API key/u);
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-blocklist");

  await choice.getByRole("button", { name: "Remove from Blocklist and grab" }).click();
  await expect(manual).toContainText(/1 Grab\b.+every file/u);
  await expect(first).toContainText(/removes it from the Blocklist/u);
  await manual.getByRole("button", { name: "Grab 1" }).click();
  await page.keyboard.press("Escape");
  await expect(
    page
      .getByRole("main")
      .getByRole("table", { name: "Grabs" })
      .getByRole("row")
      .filter({ hasText: webdlTitle }),
  ).toHaveCount(2);
  const record = await recordOf(page, webdlTitle);
  await expect(record).toContainText(/Ada Okafor.+Blocklist/u);
  const listed = await world.api("GET", "/api/v1/blocklist");
  expect(listed.body, "the entry is gone").toMatchObject({ total: 0 });
});
