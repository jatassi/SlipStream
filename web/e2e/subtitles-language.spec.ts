import { readdir } from "node:fs/promises";
import { join } from "node:path";

import type { Page } from "@playwright/test";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { frame, workId } from "./manual-search.ts";
import { isPhone } from "./shell.ts";
import {
  cinder,
  cinderSidecar,
  englishOf,
  exists,
  keyProviders,
  profileId,
  rowAction,
  scanned,
  subtitleRows,
} from "./subtitles.ts";

test.use({ scenario: "subtitles" });
test.describe.configure({ timeout: 300_000 });

/** Adds English to the open Profile's Subtitles, from a menu on desktop and a sheet on phone. */
async function addEnglish(page: Page): Promise<void> {
  const section = page.getByRole("region", { name: "Subtitles" });
  await section.getByRole("button", { name: "Add language" }).click();
  await (
    isPhone(page)
      ? page.getByRole("dialog").getByRole("button", { name: "English", exact: true })
      : page.getByRole("menuitem", { name: "English", exact: true })
  ).click();
  await expect(
    page
      .getByRole("list", { name: "Subtitle languages" })
      .getByLabel("Always get English subtitles, full, hearing impaired allowed."),
  ).toBeVisible();
}

test("adding English to a Profile scans each film's audio, and a search lands its subtitle beside the video", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await keyProviders(world);
  await page.goto(`/settings/profiles/${await profileId(world, "Movies")}`);
  await addEnglish(page);
  await page.getByRole("button", { name: "Save", exact: true }).click();
  const impact = page.getByRole("dialog", { name: "Save Movies" });
  await expect(impact.getByRole("list", { name: "What changes" })).toContainText(
    "Saving searches English subtitles for 3 files. SlipStream scans their audio first.",
  );
  await expectAccessible(page);
  await frame(page, testInfo, "impact");
  await impact.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByText("Saved Movies.")).toBeVisible();

  await scanned(world, cinder);
  const maps = join(world.node.dataDir, "speechmaps");
  await expect
    .poll(async () => (await readdir(maps).catch(() => [])).length, {
      message: "each film's Speech Map is on home's disk",
    })
    .toBe(3);

  await page.goto(`/works/${await workId(world, cinder)}`);
  await rowAction(page, "Search");
  await expect.poll(() => englishOf(world, cinder), { timeout: 120_000 }).toBe("met");
  await expect(subtitleRows(page)).toContainText("SubDL", { timeout: 30_000 });
  await expect.poll(() => exists(cinderSidecar(world))).toBe(true);
  await expectAccessible(page);
  await frame(page, testInfo, "landed");
});
