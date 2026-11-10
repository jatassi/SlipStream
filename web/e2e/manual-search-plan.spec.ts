import { expect, expectAccessible, test } from "./fixtures.ts";
import {
  frame,
  grabRows,
  openFromHero,
  orchard,
  searched,
  sheet,
  useChoice,
  workId,
} from "./manual-search.ts";

test.use({ scenario: "manual-search" });
test.describe.configure({ timeout: 180_000 });

const bluray = "Bluray-1080p · FKW · 9.8 GB";
const webdl = "WEBDL-1080p · FKW · 8.2 GB";
const webdlTitle = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW";
const both = "It gets you 1080p (standard) for The Quiet Orchard, and HD (standard) as an upgrade.";

test("the plan reads as automatic search would grab, a swapped row holds while Indexers answer, and Grab takes the ticked rows", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const movie = await workId(world, orchard);
  await page.goto(`/works/${movie}`);
  await openFromHero(page, orchard);
  const opened = sheet(page);
  await expect(opened).toContainText("2 Grabs get you everything.");
  const rows = grabRows(opened);
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText(bluray);
  await expect(rows.nth(0)).toContainText("It gets you 1080p (standard) for The Quiet Orchard.");
  await expect(rows.nth(1)).toContainText(webdl);
  await expect(rows.nth(1)).toContainText(
    "It gets you HD (standard) for The Quiet Orchard as an upgrade.",
  );
  await expect(opened.getByRole("button", { name: "Grab 2" })).toBeEnabled();
  await rows.nth(1).getByRole("checkbox", { name: webdl }).uncheck();
  await expect(opened.getByRole("button", { name: "Grab 1" })).toBeEnabled();
  await rows.nth(1).getByRole("checkbox", { name: webdl }).check();
  await expectAccessible(page);
  await frame(page, testInfo, "plan");

  await expect(opened).toContainText("3 have responded.");
  await useChoice(rows.nth(0), webdl);
  await expect(opened).toContainText("1 Grab gets you everything.");
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText(webdl);
  await expect(rows.nth(0)).toContainText(both);
  await expect(
    rows.nth(0).getByRole("list", { name: "Other choices" }).locator(":scope > li").first(),
  ).toContainText(`${bluray}From Tamarack, which automatic search doesn't use.`);

  await searched(opened);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText(webdl);
  await expect(opened).toContainText("1 Grab gets you everything.");

  await world.fw.fault.set({ id: "swarm-stalled", target: "seeders", fault: "stall" });
  await opened.getByRole("button", { name: "Grab 1" }).click();
  const held = opened.getByRole("list", { name: "Cells a Grab is downloading" });
  await expect(held.getByRole("listitem")).toHaveText([
    `A Grab is downloading ${webdlTitle} for 1080p (standard).`,
    `A Grab is downloading ${webdlTitle} for HD (standard).`,
  ]);
  await expect(rows).toHaveCount(0);
  await expectAccessible(page);
  await frame(page, testInfo, "plan-grabbed");

  await world.fw.fault.clear("swarm-stalled");
  await expect(opened).toContainText("Every Target here already has its file.", {
    timeout: 60_000,
  });
});
