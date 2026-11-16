import type { Page } from "@playwright/test";
import { z } from "zod";

import { orchardCells } from "./cells.ts";
import { type World, expect, test } from "./fixtures.ts";
import { levelHeader, seasons } from "./series.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "hold" });
test.describe.configure({ timeout: 120_000 });

const Added = z.object({ work: z.object({ id: z.string() }) });

/** Adds a Work that searches everything monitored once added, as the add sheet's default does. */
async function added(world: World, contentType: "movie" | "series", key: string): Promise<string> {
  const add = await world.api("POST", "/api/v1/works", {
    contentType,
    key,
    search: "monitored",
    ...(contentType === "series" && { preset: "all" }),
  });
  expect(add.status, `${contentType} ${key} is added`).toBe(200);
  return Added.parse(add.body).work.id;
}

const searchLine = (page: Page) => page.getByRole("status", { name: "Search" });
const searches = (page: Page) => page.getByRole("list", { name: "Searches" });
const notifications = (page: Page) => page.getByRole("region", { name: "Notifications" });

test("a search on add waits for a better Release, and the hero's Grab now grabs it", async ({
  adminPage: page,
  world,
}) => {
  const workId = await added(world, "movie", "9000001");
  await page.goto(`/works/${workId}`);
  await expect(searchLine(page)).toContainText(/^Waiting for a better release until /u, {
    timeout: 30_000,
  });
  await expect(searches(page)).toContainText(/^Waiting for a better releaseAdded/u);
  await expect(orchardCells(page)).toContainText(
    /1 release found so far\. SlipStream grabs the best one .+ unless a better one arrives\./u,
  );

  await searchLine(page).getByRole("button", { name: "Grab now" }).click();
  await expect(notifications(page)).toContainText("Grabbing 1 Release");
  await expect(searchLine(page)).toHaveText("Grabbed 1 Release");
  await expect(searches(page)).toContainText(/^Grabbed 1 ReleaseAdded/u);
  await expect(orchardCells(page)).not.toContainText("found so far");
});

test("a waiting season reads Waiting, and its menu's Grab now grabs the pack it holds", async ({
  adminPage: page,
  world,
}) => {
  const workId = await added(world, "series", "1001");
  await page.goto(`/works/${workId}`);
  await expect(levelHeader(page, "Season 1")).toContainText("Waiting", { timeout: 30_000 });

  await seasons(page).getByRole("button", { name: "More actions for Season 1" }).click();
  await clickWhenStill(
    page.getByRole(isPhone(page) ? "button" : "menuitem", { name: "Grab now", exact: true }),
  );
  await expect(notifications(page)).toContainText("Grabbing 1 Release");
  await expect(levelHeader(page, "Season 1")).not.toContainText("Waiting");
  await expect(searchLine(page)).toHaveText("Grabbed 1 Release");
});
