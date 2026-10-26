import { readdir } from "node:fs/promises";
import { join } from "node:path";

import { expect, test } from "./fixtures.ts";
import {
  add,
  bar,
  imported,
  libraryRows,
  orchard,
  pick,
  rootAt,
  saltRoad,
  toasts,
} from "./library.ts";

test.use({ scenario: "targets" });
test.describe.configure({ timeout: 120_000 });

const videosUnder = async (dir: string) =>
  (await readdir(dir, { recursive: true })).filter((name) => /\.(mkv|mp4)$/u.test(name));

test("a Work with files stays in its library root, and Delete takes its files from disk", async ({
  adminPage: page,
  world,
}) => {
  await imported(world);
  await add(world, "9000005", true);
  await rootAt(world, "archive");
  const media = join(world.manifest.scratchDir, "media");
  expect(await videosUnder(media)).toHaveLength(1);

  await page.goto("/library/movie");
  await pick(page, orchard, saltRoad);
  await bar(page).getByRole("button", { name: "Set library root…" }).click();
  const dialog = page.getByRole("dialog", {
    name: `Set the library root of ${orchard} and ${saltRoad}`,
  });
  await dialog.getByRole("combobox", { name: "Library root" }).click();
  await page.getByRole("option", { name: /archive/u }).click();
  const stays = `${orchard} has files, so it stays in its library root. Moving a Work with files arrives with Moves.`;
  await expect(dialog.getByText(stays)).toBeVisible();
  await dialog.getByRole("button", { name: "Set library root" }).click();
  await expect(toasts(page)).toContainText(`Set the library root of ${saltRoad}. ${stays}`);

  await bar(page).getByRole("button", { name: "Delete…" }).click();
  const deleting = page.getByRole("dialog", { name: `Delete ${orchard} and ${saltRoad}?` });
  await expect(deleting.getByRole("radio", { name: "Delete files from disk" })).toBeChecked();
  await deleting.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(toasts(page)).toContainText(`Deleted ${orchard} and ${saltRoad} and their files.`);
  await expect(page.getByText("No Movies in the library yet.")).toBeVisible();
  expect(await libraryRows(world)).toEqual([]);
  await expect.poll(() => videosUnder(media), { message: "the file leaves the disk" }).toEqual([]);
});
