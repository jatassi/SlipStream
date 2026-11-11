import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  failedGrab,
  frame,
  grabRows,
  itemId,
  lanternKeepers,
  openGrabDetails,
  orchard,
  searched,
  sheet,
  useChoice,
  workId,
} from "./manual-search.ts";
import { needsYouAction, openFromHero } from "./manual-search-entries.ts";
import { navigate } from "./shell.ts";

test.use({ scenario: "manual-search" });
test.describe.configure({ timeout: 180_000 });

const lq = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-LQ";

test("a Manual Grab's Details read who grabbed it, past what, where it went and its stages, from Activity and the Work page", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  await page.goto(`/works/${movie}`);
  await openFromHero(page, orchard);
  const manual = sheet(page);
  await searched(manual);
  await useChoice(grabRows(manual).first(), "WEBDL-1080p · LQ · 8.0 GB");
  await manual.getByRole("button", { name: "Grab 1" }).click();
  await page.keyboard.press("Escape");

  await navigate(page, "Activity");
  const fromActivity = await openGrabDetails(page, page.getByRole("table", { name: "Grabs" }), lq);
  await expect(fromActivity).toHaveAccessibleName("Grab of The Quiet Orchard");
  await expect(fromActivity).toContainText(/Manual search by Ada Okafor, .+\./u);
  await expect(fromActivity).toContainText(
    "Ada Okafor grabbed it although score -5 is below the minimum of 0 at 1080p.",
  );
  await expect(fromActivity).toContainText(
    "Ada Okafor grabbed it although score -5 is below the minimum of 0 at HD.",
  );
  await expect(fromActivity).toContainText("IndexerFixture Site");
  await expect(fromActivity).toContainText("TrackerFixture Site");
  await expect(fromActivity).toContainText("ClientBuilt-in client on home");
  await expect(fromActivity).toContainText("For1080p, HD");
  await expect(fromActivity).toContainText("Fetching");
  await expect(fromActivity).toContainText("Downloading");
  await expectAccessible(page);
  await frame(page, testInfo, "details-activity");
  await fromActivity.getByRole("button", { name: "Close" }).click();

  await page.goto(`/works/${movie}`);
  const fromWork = await openGrabDetails(
    page,
    page.getByRole("main").getByRole("table", { name: "Grabs" }),
    lq,
  );
  await expect(fromWork).toContainText(/Manual search by Ada Okafor, .+\./u);
  await expect(fromWork).toContainText(lq);
});

test("a failed automatic Grab's Details open from its Needs you card", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const series = await workId(world, lanternKeepers);
  await failedGrab(world, series, await itemId(world, series, "S02E10"));
  await page.goto(`/works/${series}`);
  await needsYouAction(page, "Details");
  const record = page.getByRole("dialog", { name: "Grab of The Lantern Keepers" });
  await expect(record).toContainText(/Automatic search, .+\./u);
  await expect(record).toContainText("The.Lantern.Keepers.S02E10.1080p.WEB-DL.H264-NTb");
  await expect(record).toContainText("Failed");
  await expect(record).toContainText(
    "SlipStream can't read the Release's .torrent file, or the metadata peers sent for its magnet link.",
  );
  await expect(record).toContainText("The Indexer sent a .torrent file SlipStream can't read.");
  await expect(record).not.toContainText("bencode");
  await expectAccessible(page);
  await frame(page, testInfo, "details-needs-you");
});
