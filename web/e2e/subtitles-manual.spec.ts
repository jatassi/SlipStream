import { expect, expectAccessible, test } from "./fixtures.ts";
import { frame, workId } from "./manual-search.ts";
import {
  amber,
  englishOf,
  keyProviders,
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

const file = "Amber.Signal.2020.720p.WEB-DL.H264-FKW.mkv";
const release = "Amber.Signal.2020.720p.WEB-DL.H264-OTHER";

test("Manual subtitle search ranks a film's candidates, a failing pick stays open with its timing match, and a passing one adds the subtitle", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await keyProviders(world);
  await searchOff(world);
  await wantEnglish(world, "Movies");
  await scanned(world, amber);

  await page.goto(`/works/${await workId(world, amber)}`);
  await rowAction(page, "Manual search");
  const sheet = subtitleSheet(page, file);
  const failing = sheet.getByRole("button", {
    name: `Pick ${release} from SubDL`,
  });
  const passing = sheet.getByRole("button", {
    name: `Pick ${release} from SubSource`,
  });
  await expect(failing).toBeVisible({ timeout: 60_000 });
  await expect(passing).toBeVisible();
  await expect(sheet).toContainText("Best");
  await expectAccessible(page);
  await frame(page, testInfo, "candidates");

  await failing.click();
  await expect(sheet).toContainText("Its timing didn't match this file. Its timing match is", {
    timeout: 60_000,
  });
  await expect(sheet).toBeVisible();
  await frame(page, testInfo, "failed-pick");

  await passing.click();
  await expect(toast(page, "English subtitle added.")).toBeVisible({
    timeout: 60_000,
  });
  await expect(sheet).toBeHidden();
  await expect.poll(() => englishOf(world, amber)).toBe("met");
  await expect(subtitleRows(page)).toContainText("SubSource");
  await frame(page, testInfo, "added");
});
