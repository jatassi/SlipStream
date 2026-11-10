import type { Page } from "@playwright/test";

import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  blocklisted,
  frame,
  grabRows,
  openFromHero,
  openGrabDetails,
  orchard,
  searched,
  sheet,
  useChoice,
  workId,
} from "./manual-search.ts";

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

test("a Release under a Target's minimum score is grabbed past it, the row saying so first", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const manual = await opened(page, world);
  const row = grabRows(manual).first();
  await useChoice(row, "WEBDL-1080p · LQ · 8.0 GB");
  await expect(manual).toContainText("1 Grab gets you everything.");
  await expect(row).toContainText(
    "It gets you 1080p (standard) for The Quiet Orchard, and HD (standard) as an upgrade.",
  );
  await expect(row.getByText(/^Score -5 is below/u)).toHaveText([
    "Score -5 is below the minimum of 0 at 1080p.",
    "Score -5 is below the minimum of 0 at HD.",
  ]);
  await expectAccessible(page);
  await row.evaluate((element) => {
    element.scrollIntoView({ block: "start" });
  });
  await frame(page, testInfo, "overrides-soft-refusal");

  await manual.getByRole("button", { name: "Grab 1" }).click();
  const upgrade = grabRows(manual).first();
  await expect(upgrade).toContainText(
    "It gets you 1080p (standard) and HD (standard) for The Quiet Orchard as an upgrade.",
    { timeout: 60_000 },
  );
  await expect(upgrade.getByRole("checkbox", { name: "WEBDL-1080p · FKW · 8.2 GB" })).toBeChecked();
  await expect(manual.getByRole("button", { name: "Grab 1" })).toBeEnabled();
  const record = await recordOf(page, "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-LQ");
  await expect(record).toContainText(
    "Ada Okafor grabbed it although score -5 is below the minimum of 0 at 1080p.",
  );
});

test("a Release of a cut the Work doesn't want is grabbed as its Standard Edition", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const manual = await opened(page, world);
  const row = grabRows(manual).first();
  await useChoice(row, "WEBDL-1080p · FKW · 8.9 GB");
  await row.getByRole("button", { name: "Details" }).last().click();
  await expect(row).toContainText("The.Quiet.Orchard.2021.EXTENDED.1080p.WEB-DL.H264-FKW");
  await expect(row).toContainText(
    "It meets 1080p (standard). The Work doesn't want the extended cut at 1080p.",
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
  await manual.getByRole("combobox", { name: "Send all to" }).click();
  await page.getByRole("option", { name: "Built-in client on home" }).click();
  await expect(rows.nth(0)).toContainText("SlipStream sends it to the built-in client on home.");
  await expect(rows.nth(1)).toContainText("SlipStream sends it to the built-in client on home.");

  await rows.nth(1).getByRole("button", { name: "Details" }).last().click();
  const sendTo = rows.nth(1).getByRole("combobox", { name: "Send to" });
  await sendTo.click();
  await page.getByRole("option", { name: /^Where Tracker routing sends it/u }).click();
  await expect(rows.nth(1)).toContainText(
    "Where Tracker routing sends it (the built-in client on home)",
  );
  await expect(rows.nth(0)).toContainText("SlipStream sends it to the built-in client on home.");
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-client");

  await rows.nth(1).getByRole("checkbox").uncheck();
  await manual.getByRole("button", { name: "Grab 1" }).click();
  const record = await recordOf(page, "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW");
  await expect(record).toContainText(
    "Ada Okafor sent it to the built-in client on home. Tracker routing would have sent it to the built-in client on home.",
  );
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
    /It's been on the Blocklist since .+ because it was blocklisted through the API key when its Grab was canceled\./u,
  );
  await expectAccessible(page);
  await frame(page, testInfo, "overrides-blocklist");

  await choice.getByRole("button", { name: "Remove from Blocklist and grab" }).click();
  await expect(manual).toContainText("1 Grab gets you everything.");
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
