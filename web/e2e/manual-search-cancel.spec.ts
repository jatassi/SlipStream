import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  downloading,
  frame,
  grabAction,
  grabStage,
  manualSearched,
  openGrabDetails,
  orchard,
  workId,
} from "./manual-search.ts";
import { navigate } from "./shell.ts";

test.use({ scenario: "manual-search" });
test.describe.configure({ timeout: 180_000 });

const webdlTitle = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW";
const byHand = "Ada Okafor blocklisted it when canceling its Grab.";

test("a downloading Grab canceled with Also blocklist lists its Release on Activity's Blocklist, where Remove takes it off", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  const grabId = await downloading(world, movie, webdlTitle);
  await page.goto("/activity");
  await grabAction(page, page.getByRole("table", { name: "Grabs" }), webdlTitle, "Cancel Grab");
  const confirm = page.getByRole("dialog", { name: "Cancel this Grab?" });
  await expect(confirm).toContainText(webdlTitle);
  await confirm.getByRole("checkbox", { name: "Also blocklist this Release" }).check();
  await expectAccessible(page);
  await frame(page, testInfo, "cancel-downloading");
  await confirm.getByRole("button", { name: "Cancel Grab" }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Canceled the Grab of The Quiet Orchard.",
  );
  await expect.poll(() => grabStage(world, grabId)).toBe("failed");
  await world.fw.fault.clear("swarm-stalled");

  await page
    .getByRole("navigation", { name: "Activity" })
    .getByRole("link", { name: "Blocklist" })
    .click();
  const blocklist = page.getByRole("table", { name: "Blocklist" });
  const entry = blocklist.getByRole("row").filter({ hasText: webdlTitle });
  await expect(entry).toContainText(byHand);
  await expect(entry).toContainText("Fixture Site");
  await expect(entry.getByRole("link", { name: orchard })).toBeVisible();
  await expectAccessible(page);
  await frame(page, testInfo, "blocklist");

  await entry.getByRole("checkbox", { name: `Pick ${webdlTitle}` }).check();
  await page
    .getByRole("toolbar", { name: "Picked entries" })
    .getByRole("button", { name: "Remove" })
    .click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Removed 1 Release from the Blocklist.",
  );
  await expect(blocklist.getByRole("row").filter({ hasText: webdlTitle })).toHaveCount(0);
});

test("a fetching Grab cancels from its Details without blocklisting, and its cell reads missing with nothing in Needs you", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  const search = await manualSearched(world, movie);
  const pick = search.plan.rows.find((row) => row.pick?.release.title === webdlTitle)?.pick;
  if (pick === undefined || pick === null) {
    throw new Error("the plan doesn't pick the WEB-DL");
  }
  await world.fw.fault.set({
    id: "torrent-slow",
    target: "fixture-site",
    route: "torznab.download",
    fault: "delay=120s",
  });
  const grabbed = await world.api("POST", `/api/v1/works/${movie}/manual-search/grabs`, {
    manualSearchId: search.id,
    picks: [{ releaseId: pick.release.id, lifts: [] }],
  });
  expect(grabbed.status, "the grab is taken").toBe(200);

  await navigate(page, "Library");
  await page.goto(`/works/${movie}`);
  const record = await openGrabDetails(
    page,
    page.getByRole("main").getByRole("table", { name: "Grabs" }),
    webdlTitle,
  );
  await expect(record).toContainText("Manual search by the API key");
  await record.getByRole("button", { name: "Cancel Grab" }).click();
  const confirm = page.getByRole("dialog", { name: "Cancel this Grab?" });
  await expect(
    confirm.getByRole("checkbox", { name: "Also blocklist this Release" }),
  ).not.toBeChecked();
  await expectAccessible(page);
  await frame(page, testInfo, "cancel-fetching");
  await confirm.getByRole("button", { name: "Cancel Grab" }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Canceled the Grab of The Quiet Orchard.",
  );
  await world.fw.fault.clear("torrent-slow");
  await expect(page.getByRole("region", { name: "Needs you" })).not.toContainText("failed at");
  const listed = await world.api("GET", "/api/v1/blocklist");
  expect(listed.body, "nothing is blocklisted").toMatchObject({ total: 0 });
});
