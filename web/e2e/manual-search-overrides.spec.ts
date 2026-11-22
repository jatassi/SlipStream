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
  await expect(row).toContainText(
    "Its title names the Extended Edition, which isn't one of this movie's Editions. SlipStream grabs it as the Standard Edition.",
  );
  await row.getByRole("button", { name: "Details" }).last().click();
  await expect(row).toContainText("The.Quiet.Orchard.2021.EXTENDED.1080p.WEB-DL.H264-FKW");
  await expect(row).toContainText(
    "It meets 1080p (standard). This movie's Editions at 1080p don't include the extended cut.",
  );
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-edition");

  await manual.getByRole("button", { name: "Grab 1" }).click();
  await expect(manual).toContainText(filled, { timeout: 60_000 });
  const record = await recordOf(page, "The.Quiet.Orchard.2021.EXTENDED.1080p.WEB-DL.H264-FKW");
  await expect(record).toContainText("Ada Okafor grabbed it as the Standard Edition.");
});

test("a row and the whole plan can be sent to a client Tracker routing wouldn't choose", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const manual = await opened(page, world);
  const rows = grabRows(manual);
  await sendAllTo(page, manual, "Built-in client on home");
  await expect(rows.nth(0)).toContainText("SlipStream sends it to the built-in client on home.");
  await expect(rows.nth(1)).toContainText("SlipStream sends it to the built-in client on home.");

  await rows.nth(1).getByRole("button", { name: "Details" }).last().click();
  const sendTo = rows.nth(1).getByRole("combobox", { name: "Send to" });
  await sendTo.click();
  await page.getByRole("option", { name: /^Where Tracker routing sends it/u }).click();
  await expect(rows.nth(1)).toContainText(
    "Tracker routing sends it to the built-in client on home.",
  );
  await expect(rows.nth(0)).toContainText("SlipStream sends it to the built-in client on home.");
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-client");

  await rows.nth(1).getByRole("checkbox").uncheck();
  await manual.getByRole("button", { name: "Grab 1" }).click();
  const record = await recordOf(page, "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW");
  await expect(record).toContainText("Ada Okafor sent it to the built-in client on home.");
  await expect(record).not.toContainText("Tracker routing would have sent it");
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
  await expect(choice).toContainText(
    /It's been on the Blocklist since .+ because an app using the API key blocklisted it when canceling its Grab\./u,
  );
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-blocklist");

  await choice.getByRole("button", { name: "Remove from Blocklist and grab" }).click();
  await expect(manual).toContainText("1 Grab downloads every file you're missing.");
  await expect(first).toContainText("Grabbing it removes it from the Blocklist.");
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
  await expect(record).toContainText("Ada Okafor removed it from the Blocklist to grab it.");
  const listed = await world.api("GET", "/api/v1/blocklist");
  expect(listed.body, "the entry is gone").toMatchObject({ total: 0 });
});
