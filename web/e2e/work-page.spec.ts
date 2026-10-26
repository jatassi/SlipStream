import type { Page } from "@playwright/test";
import { z } from "zod";

import { orchardCells, workAction } from "./cells.ts";
import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "targets" });
test.describe.configure({ timeout: 120_000 });

const Added = z.object({ work: z.object({ id: z.string() }) });
const Cells = z.object({ cells: z.array(z.object({ state: z.string() })) });

async function add(world: World, key: string): Promise<string> {
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key,
    monitor: true,
  });
  expect(added.status, `TMDB ${key} is added`).toBe(200);
  return Added.parse(added.body).work.id;
}

async function searched(world: World, workId: string, state: string): Promise<void> {
  const search = await world.api("POST", `/api/v1/works/${workId}/search`);
  expect(search.status, "its search starts").toBe(202);
  await expect
    .poll(
      async () =>
        Cells.parse((await world.api("GET", `/api/v1/works/${workId}/cells`)).body).cells.every(
          (cell) => cell.state === state,
        ),
      { message: `every cell is ${state}`, timeout: 60_000 },
    )
    .toBe(true);
}

const notifications = (page: Page) => page.getByRole("region", { name: "Notifications" });

async function openEditions(page: Page) {
  if (isPhone(page)) {
    await workAction(page, "Edit Editions");
  } else {
    await page.getByRole("button", { name: /^Editions /u }).click();
  }
  return page.getByRole("dialog", { name: "Editions of The Quiet Orchard" });
}

test("a failed download shows under Needs you, and Search again retries it", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await world.fw.fault.set({
    id: "download-refused",
    target: "fixture-site",
    node: "home",
    route: "download",
    fault: "status=500",
  });
  const workId = await add(world, "9000001");
  await searched(world, workId, "failed");

  await page.goto(`/works/${workId}`);
  const needsYou = page.getByRole("region", { name: "Needs you" });
  const card = needsYou.getByRole("article");
  await expect(card).toHaveCount(1);
  await expect(card).toContainText("Failed at");
  await expect(card).toHaveAccessibleName("1080p failed at Downloading");
  await expect(page.getByRole("list", { name: "Cells" })).toContainText("failed");
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("needs-you.png"), fullPage: true });

  await world.fw.fault.clear("download-refused");
  await card.getByRole("button", { name: "Search again" }).click();
  await expect(notifications(page)).toContainText("Search started");
  await expect(orchardCells(page)).toContainText("Upgrade available", { timeout: 60_000 });
  await expect(card).toHaveCount(0);
  await expect(needsYou.getByRole("list", { name: "Upgrades available" })).toContainText(
    "Upgrade available for 1080p and HD",
  );
});

test("an Admin sets the Work's own Editions through the Impact, and an unseen cut waits in its own row", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const workId = await add(world, "9000001");
  await searched(world, workId, "upgradable");
  await page.goto(`/works/${workId}`);
  await expect(orchardCells(page)).toContainText("From Profile Films");

  const editions = await openEditions(page);
  await expect(
    editions.getByRole("heading", { name: "Editions from Profile Films" }),
  ).toBeVisible();
  await editions.getByRole("radio", { name: "Choose cuts" }).click();
  await editions.getByRole("button", { name: "Add cut" }).click();
  await clickWhenStill(
    page.getByRole(isPhone(page) ? "button" : "menuitem", { name: "Extended", exact: true }),
  );
  await expect(editions.getByRole("heading", { name: "Editions set for this Work" })).toBeVisible();
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("editions.png") });
  await editions.getByRole("button", { name: "Save", exact: true }).click();

  const impact = page.getByRole("dialog", { name: "Save the Editions of The Quiet Orchard" });
  await expect(impact.getByRole("list", { name: "What changes" })).toContainText("1 Work changes.");
  await impact.getByRole("button", { name: /^Save/u }).click();

  await expect(notifications(page)).toContainText("Saved the Editions of The Quiet Orchard.");
  const matrix = isPhone(page)
    ? page.getByRole("list", { name: "Editions and Targets" }).getByRole("listitem")
    : page
        .getByRole("table", { name: "Editions and Targets" })
        .getByRole("row")
        .filter({ hasNot: page.getByRole("columnheader") });
  await expect(matrix).toHaveCount(2);
  await expect(matrix.first()).toContainText("StandardSet for this Work");
  await expect(matrix.nth(1)).toContainText(
    "Not seen yet. It's wanted once a release of it is found.",
  );
  const wants = await world.api("GET", `/api/v1/works/${workId}/wants`);
  expect(wants.body).toMatchObject({
    from: "work",
    editions: { kind: "cuts", cuts: ["standard", "extended"] },
  });
});
