import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  frame,
  grabRows,
  openGrabDetails,
  orchard,
  searched,
  sheet,
  useChoice,
  workId,
} from "./manual-search.ts";
import { openFromHero } from "./manual-search-entries.ts";
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
  await expect(fromActivity).toContainText(/Found by Ada Okafor in Manual search at .+\./u);
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
  await expect(fromWork).toContainText(/Found by Ada Okafor in Manual search at .+\./u);
  await expect(fromWork).toContainText(lq);
});
