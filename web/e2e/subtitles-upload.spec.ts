import type { Page, TestInfo } from "@playwright/test";

import { catalogSubtitle } from "../harness/fakeworld.ts";
import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { frame, workId } from "./manual-search.ts";
import {
  brick,
  cinder,
  cinderSidecar,
  englishOf,
  exists,
  keyProviders,
  providerOff,
  rowAction,
  scanned,
  searchOff,
  subtitleRows,
  subtitleSheet,
  toast,
  wantEnglish,
} from "./subtitles.ts";

test.use({ scenario: "subtitles" });
test.describe.configure({ timeout: 300_000 });

/** Uploads text as the file's English subtitle from the Work's English row. */
async function upload(page: Page, world: World, label: string, file: string, text: string) {
  await page.goto(`/works/${await workId(world, label)}`);
  await rowAction(page, "Upload");
  const sheet = subtitleSheet(page, file);
  await sheet.locator("input[type=file]").setInputFiles({
    name: `${file.replace(/\.mkv$/u, "")}.en.srt`,
    mimeType: "application/x-subrip",
    buffer: Buffer.from(text),
  });
  return sheet;
}

/** Marks the kept upload bad from its row, which deletes it from the disk and searches for another. */
async function markBad(page: Page, testInfo: TestInfo, sidecar: string): Promise<void> {
  await expect.poll(() => exists(sidecar)).toBe(true);
  await expect(subtitleRows(page)).toContainText("English");
  await rowAction(page, "Mark bad…");
  const question = page
    .getByRole("alertdialog")
    .or(page.getByRole("dialog"))
    .filter({ hasText: "bad?" });
  await expect(question).toContainText(/deletes your file/u);
  await frame(page, testInfo, "mark-bad");
  await question.getByRole("button", { name: "Mark bad", exact: true }).click();
  await expect(
    toast(page, "English subtitle marked bad. SlipStream is searching for another."),
  ).toBeVisible({ timeout: 60_000 });
  await expect.poll(() => exists(sidecar)).toBe(false);
}

test("an upload that follows the film's speech is added, and one that doesn't is kept anyway from Details, then marked bad off the disk", async ({
  adminPage: page,
  world,
  runDir,
}, testInfo) => {
  await keyProviders(world);
  await searchOff(world);
  await wantEnglish(world, "Movies");
  await scanned(world, brick);
  await scanned(world, cinder);
  const speech = await catalogSubtitle(runDir, "brick-speech");

  const brickFile = "Brick.Lane.Echo.2019.1080p.WEB-DL.H264-FKW.mkv";
  const passing = await upload(page, world, brick, brickFile, speech);
  await expect(toast(page, "English subtitle added.")).toBeVisible({
    timeout: 60_000,
  });
  await expect(passing).toBeHidden();
  await expect.poll(() => englishOf(world, brick)).toBe("met");
  await frame(page, testInfo, "upload-added");

  const cinderFile = "Cinder.Bloom.2017.1080p.WEB-DL.H264-FKW.mkv";
  const failing = await upload(page, world, cinder, cinderFile, speech);
  await expect(failing).toContainText(
    "This subtitle's timing doesn't match the file, so SlipStream didn't keep it.",
    { timeout: 60_000 },
  );
  await failing.getByRole("button", { name: "Details" }).click();
  await expect(failing).toContainText("SlipStream keeps subtitles at");
  await expectAccessible(page);
  await frame(page, testInfo, "upload-failed");
  await failing.getByRole("button", { name: "Keep anyway" }).click();
  await expect(failing).toBeHidden({ timeout: 60_000 });
  await expect.poll(() => englishOf(world, cinder)).toBe("met");

  await providerOff(world, "subdl");
  await markBad(page, testInfo, cinderSidecar(world));
  await expect.poll(() => englishOf(world, cinder)).toMatch(/^missing/u);
});
