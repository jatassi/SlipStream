import type { Page } from "@playwright/test";
import { z } from "zod";

import { workSearch } from "./cells.ts";
import { type World, expect, test } from "./fixtures.ts";
import {
  addSeries,
  episodeRow,
  lanternKeepers,
  openLevel,
  seasons,
  setEpisodeMonitored,
} from "./series.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "series" });
test.describe.configure({ timeout: 180_000 });

const Cells = z.object({
  cells: z.array(z.object({ itemId: z.string(), file: z.object({ id: z.string() }).nullish() })),
});
const Items = z.object({ items: z.array(z.object({ id: z.string(), label: z.string() })) });

const notifications = (page: Page) => page.getByRole("region", { name: "Notifications" });
const grabbed = (page: Page) =>
  page.getByRole("list", { name: "Searches" }).getByText("Grabbed a Release");

/** The titles of the episodes that hold a library file. */
async function filed(world: World, workId: string): Promise<string[]> {
  const { items } = Items.parse((await world.api("GET", `/api/v1/works/${workId}`)).body);
  const { cells } = Cells.parse((await world.api("GET", `/api/v1/works/${workId}/cells`)).body);
  const held = new Set(cells.filter(({ file }) => file).map(({ itemId }) => itemId));
  return items.filter(({ id }) => held.has(id)).map(({ label }) => label);
}

// A phone row opens the episode's sheet, which holds its Search.
async function searchEpisode(page: Page, code: string): Promise<void> {
  if (!isPhone(page)) {
    await seasons(page)
      .getByRole("button", { name: `Search ${code}` })
      .click();
    return;
  }
  await clickWhenStill(episodeRow(page, code));
  await page
    .getByRole("dialog", { name: new RegExp(`^${code} `, "u") })
    .getByRole("button", { name: "Search", exact: true })
    .click();
}

test("the series' Search counts every wanted episode, and a season's only its monitored ones", async ({
  adminPage: page,
  world,
}) => {
  const workId = await addSeries(world, lanternKeepers, { preset: "first" });
  await page.goto(`/works/${workId}`);
  await setEpisodeMonitored(page, "S01E02", false);
  await workSearch(page).click();
  await expect(notifications(page)).toContainText("Searching 2 episodes");
  // The season pack fills both wanted episodes, so one Grab does.
  await expect(grabbed(page)).toHaveCount(1);

  await openLevel(page, "Season 2");
  await setEpisodeMonitored(page, "S02E01", true);
  await seasons(page).getByRole("button", { name: "Search Season 2" }).click();
  await expect(notifications(page)).toContainText("Searching 1 episode");
});

test("an episode's Search grabs its release, and the episode keeps its file once TVDB's order no longer lists it", async ({
  adminPage: page,
  world,
}) => {
  const workId = await addSeries(world, lanternKeepers, { preset: "none" });
  await page.goto(`/works/${workId}`);
  await openLevel(page, "Specials");
  await setEpisodeMonitored(page, "S00E02", true);
  await searchEpisode(page, "S00E02");
  await expect(notifications(page)).toContainText("Searching 1 episode");
  await expect(grabbed(page)).toHaveCount(1);
  await expect.poll(() => filed(world, workId), { timeout: 90_000 }).toEqual(["A Winter Light"]);

  const dvd = await world.api("PUT", `/api/v1/works/${workId}/options`, {
    type: { order: "dvd" },
    preview: false,
  });
  expect(dvd.status, "the order changes to DVD").toBe(200);
  await page.reload();
  await expect(seasons(page).getByRole("heading", { level: 3 })).toHaveText([
    /^Season 1/u,
    /^Season 2/u,
    /^Specials.*Not monitored$/u,
    /^No longer listed by TVDB.*1 episode$/u,
  ]);
  const gone = seasons(page).getByRole("button", { name: "No longer listed by TVDB", exact: true });
  await gone.click();
  await expect(gone).toHaveAttribute("aria-expanded", "true");
  await expect(episodeRow(page, "S00E02")).toContainText("No longer listed by TVDB");
  expect(await filed(world, workId)).toEqual(["A Winter Light"]);
});
