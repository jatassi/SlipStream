import { expect, test } from "./fixtures.ts";
import {
  add,
  bar,
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
  await bar(page).getByRole("button", { name: "Monitor", exact: true }).click();
  await expect(toasts(page)).toContainText(`Monitored ${orchard} and ${saltRoad}.`);
  await expect
    .poll(async () => (await libraryRows(world)).map((row) => row.monitored))
    .toEqual(["all", "all"]);

  await bar(page).getByRole("button", { name: "Refresh metadata" }).click();
  await expect(toasts(page)).toContainText(`${orchard} and ${saltRoad}`);

  await bar(page).getByRole("button", { name: "Change Profile…" }).click();
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

test("an Admin moves two Works without files to another library root, then deletes them leaving files", async ({
  adminPage: page,
  world,
}) => {
  await rootAt(world, "movies");
  const archive = await rootAt(world, "archive");
  await add(world, "9000001", true);
  await add(world, "9000005", true);

  await page.goto("/library/movie");
  await pick(page, orchard, saltRoad);
  await bar(page).getByRole("button", { name: "Set library root…" }).click();
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

  await bar(page).getByRole("button", { name: "Delete…" }).click();
  const deleting = page.getByRole("dialog", { name: `Delete ${orchard} and ${saltRoad}?` });
  await deleting.getByRole("radio", { name: "Leave files on disk" }).click();
  await deleting.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(toasts(page)).toContainText(`Deleted ${orchard} and ${saltRoad}.`);
  await expect(toasts(page)).not.toContainText("files");
  await expect(page.getByText("No Movies in the library yet.")).toBeVisible();
  expect(await libraryRows(world)).toEqual([]);
});
