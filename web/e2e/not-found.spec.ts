import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, test } from "./fixtures.ts";

test.use({ scenario: "auto-search" });
test.describe.configure({ timeout: 120_000 });

const Added = z.object({ work: z.object({ id: z.string() }) });
const Library = z.object({ rows: z.array(z.object({ id: z.string(), needsYou: z.number() })) });

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

const notFound = (page: Page) =>
  page.getByRole("region", { name: "Needs you" }).getByRole("list", { name: "Not found" });
const notifications = (page: Page) => page.getByRole("region", { name: "Messages" });

test("a season whose search found nothing is listed as Not found, and its Search searches it again", async ({
  adminPage: page,
  world,
}) => {
  const workId = await added(world, "series", "1002");
  await page.goto(`/works/${workId}`);
  const row = notFound(page)
    .getByRole("listitem")
    .filter({ hasText: "S02E01 to S02E02 not found" });
  await expect(row).toContainText(
    /Found nothing today at \d{2}:\d{2}\. SlipStream searches again .+\./u,
    { timeout: 30_000 },
  );
  await expect(notFound(page).getByRole("listitem")).toHaveCount(1);

  await row.getByRole("button", { name: "Search", exact: true }).click();
  await expect(notifications(page)).toContainText(/^(Searching|Queued) 2 episodes/u);
  await expect(page.getByRole("list", { name: "Searches" })).toContainText(
    /^Found nothingSearch/u,
    {
      timeout: 30_000,
    },
  );
  await expect(row).toBeVisible();
});

test("a movie no Indexer lists is Not found on its page and leaves the Library's Needs you alone", async ({
  adminPage: page,
  world,
}) => {
  const workId = await added(world, "movie", "9000005");
  await page.goto(`/works/${workId}`);
  await expect(notFound(page)).toContainText("The Salt Road not found", { timeout: 30_000 });
  await expect(notFound(page).getByRole("button", { name: "Manual search" })).toBeVisible();

  const { rows } = Library.parse((await world.api("GET", "/api/v1/library")).body);
  expect(rows.find(({ id }) => id === workId)?.needsYou, "the Library counts no need").toBe(0);
});
