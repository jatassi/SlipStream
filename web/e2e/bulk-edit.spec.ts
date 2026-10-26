import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import {
  add,
  bar,
  barAction,
  libraryRows,
  makeUHD,
  orchard,
  pick,
  rootAt,
  saltRoad,
  toasts,
} from "./library.ts";

test.use({ scenario: "movies" });
test.describe.configure({ timeout: 120_000 });

test("an Admin monitors, refreshes and changes the Profile of two picked Works together", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await rootAt(world, "movies");
  const uhd = await makeUHD(world);
  await add(world, "9000001", false);
  await add(world, "9000005", false);

  await page.goto("/library/movie");
  await pick(page, orchard, saltRoad);
  await expect(bar(page)).toContainText("2 Movies picked");
  await barAction(page, "Monitor");
  await expect(toasts(page)).toContainText(`Monitored ${orchard} and ${saltRoad}.`);
  await expect
    .poll(async () => (await libraryRows(world)).map((row) => row.monitored))
    .toEqual(["all", "all"]);

  await barAction(page, "Refresh metadata");
  await expect(toasts(page)).toContainText(`${orchard} and ${saltRoad}`);

  await barAction(page, "Change Profile…");
  const dialog = page.getByRole("dialog", {
    name: `Change the Profile of ${orchard} and ${saltRoad}`,
  });
  await dialog.getByRole("combobox", { name: "Profile" }).click();
  await page.getByRole("option", { name: "UHD" }).click();
  await dialog.getByRole("button", { name: "Next" }).click();
  const impact = page.getByRole("dialog", {
    name: `Give ${orchard} and ${saltRoad} the Profile UHD`,
  });
  await expect(impact.getByRole("list", { name: "What changes" })).toContainText("2 Works change.");
  await page.screenshot({ path: testInfo.outputPath("profile-impact.png") });
  await impact.getByRole("button", { name: /^Save/u }).click();
  await expect(toasts(page)).toContainText(`Changed the Profile of ${orchard} and ${saltRoad}.`);
  await expect
    .poll(async () => (await libraryRows(world)).map((row) => row.profileId))
    .toEqual([uhd, uhd]);
});

test("an Admin moves two Works without files to another library root, then deletes them", async ({
  adminPage: page,
  world,
}) => {
  await rootAt(world, "movies");
  const archive = await rootAt(world, "archive");
  await add(world, "9000001", true);
  await add(world, "9000005", true);

  await page.goto("/library/movie");
  await pick(page, orchard, saltRoad);
  await barAction(page, "Set library root…");
  const dialog = page.getByRole("dialog", {
    name: `Set the library root of ${orchard} and ${saltRoad}`,
  });
  await dialog.getByRole("combobox", { name: "Library root" }).click();
  await page.getByRole("option", { name: /archive/u }).click();
  await dialog.getByRole("button", { name: "Set library root" }).click();
  await expect(toasts(page)).toContainText(`Set the library root of ${orchard} and ${saltRoad}.`);
  await expect
    .poll(async () => (await libraryRows(world)).map((row) => row.rootId))
    .toEqual([archive, archive]);

  await barAction(page, "Delete…");
  const deleting = page.getByRole("dialog", { name: `Delete ${orchard} and ${saltRoad}?` });
  await expect(deleting.getByText(`${orchard} and ${saltRoad} have no files.`)).toBeVisible();
  await expect(deleting.getByRole("radiogroup", { name: "Files" })).toHaveCount(0);
  await deleting.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(toasts(page)).toContainText(`Deleted ${orchard} and ${saltRoad}.`);
  await expect(toasts(page)).not.toContainText("files");
  await expect(page.getByText("No Movies in the library yet.")).toBeVisible();
  expect(await libraryRows(world)).toEqual([]);
});

const documentTop = (page: Page, label: string) =>
  page
    .getByRole("link", { name: label })
    .first()
    .evaluate((link) => link.getBoundingClientRect().top + window.scrollY);

async function expectInView(page: Page, at: "top" | "bottom") {
  await page.evaluate((to) => {
    window.scrollTo(0, to === "top" ? 0 : document.documentElement.scrollHeight);
  }, at);
  const box = await bar(page).boundingBox();
  const height = page.viewportSize()?.height ?? 0;
  const tabs = page.getByRole("navigation", { name: "Tabs" });
  const floor = (await tabs.count()) === 0 ? height : ((await tabs.boundingBox())?.y ?? 0);
  expect(box, `the bar is shown at the ${at}`).not.toBeNull();
  expect(box?.y, `the bar's top is in view at the ${at}`).toBeGreaterThanOrEqual(0);
  expect(
    (box?.y ?? 0) + (box?.height ?? 0),
    `the bar sits above the tabs at the ${at}`,
  ).toBeLessThanOrEqual(floor);
}

test("the picked bar stays in view without moving the grid", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await rootAt(world, "movies");
  await add(world, "9000001", false);
  await add(world, "9000005", false);

  await page.goto("/library/movie");
  const before = await documentTop(page, orchard);
  await pick(page, orchard, saltRoad);
  await expect(bar(page)).toContainText("2 Movies picked");
  expect(await documentTop(page, orchard), "the grid holds still as the bar opens").toBe(before);
  await expectInView(page, "bottom");
  await page.screenshot({ path: testInfo.outputPath("bulk-bar.png") });

  const viewport = page.viewportSize();
  await page.setViewportSize({ width: viewport?.width ?? 0, height: 420 });
  await expectInView(page, "top");
  await expectInView(page, "bottom");
  const box = await bar(page).boundingBox();
  expect(box?.height, "the bar is one row").toBeLessThan(72);
});
