import type { Locator, Page, TestInfo } from "@playwright/test";
import { z } from "zod";

import { type World, expect } from "./fixtures.ts";
import { workAction } from "./cells.ts";
import { isPhone, navigate } from "./shell.ts";

export const orchard = "The Quiet Orchard";
export const lanternKeepers = "The Lantern Keepers";

const Library = z.object({ rows: z.array(z.object({ id: z.string(), label: z.string() })) });
const Tree = z.object({
  items: z.array(
    z.object({
      id: z.string(),
      fields: z.array(z.object({ key: z.string(), text: z.string().optional() })),
    }),
  ),
});
const Grabs = z.object({
  results: z.array(z.object({ grabId: z.string().nullable(), problem: z.unknown() })),
});
const Grab = z.object({ stage: z.string() });
const Manual = z.object({
  id: z.string(),
  searching: z.boolean(),
  plan: z.object({
    rows: z.array(
      z.object({
        pick: z
          .object({
            release: z.object({ id: z.string(), title: z.string() }),
            cells: z.array(z.string()),
            lifts: z.array(z.object({ code: z.string(), targetId: z.string().optional() })),
          })
          .nullable(),
      }),
    ),
  }),
});
const Search = z.object({ searches: z.array(z.object({ outcome: z.string().nullable() })) });

/** The scanned Work of that title. */
export async function workId(world: World, label: string): Promise<string> {
  const rows = Library.parse((await world.api("GET", "/api/v1/library")).body).rows;
  const row = rows.find((candidate) => candidate.label === label);
  if (row === undefined) {
    throw new Error(`the library has no ${label}`);
  }
  return row.id;
}

/** The Item whose code field reads code, such as S02E10. */
export async function itemId(world: World, work: string, code: string): Promise<string> {
  const { items } = Tree.parse((await world.api("GET", `/api/v1/works/${work}`)).body);
  const item = items.find(({ fields }) =>
    fields.some((field) => field.key === "code" && field.text === code),
  );
  if (item === undefined) {
    throw new Error(`the Work has no ${code}`);
  }
  return item.id;
}

/** An automatic search of one Item, run to its outcome. */
export async function searchedItem(world: World, work: string, item: string): Promise<void> {
  const started = await world.api("POST", `/api/v1/works/${work}/search`, {
    levelIds: [],
    itemIds: [item],
  });
  expect(started.status, "the search starts").toBe(202);
  await expect
    .poll(
      async () =>
        Search.parse((await world.api("GET", `/api/v1/works/${work}/searches`)).body).searches[0]
          ?.outcome ?? null,
      { message: "the search finishes", timeout: 60_000 },
    )
    .not.toBeNull();
}

/** An automatic search of one Item whose Grab fails on a damaged .torrent, so Needs you lists it. */
export async function failedGrab(world: World, work: string, item: string): Promise<void> {
  await world.fw.fault.set({
    id: "torrent-damaged",
    target: "fixture-site",
    route: "torznab.download",
    fault: "malformed",
  });
  await searchedItem(world, work, item);
  await world.fw.fault.clear("torrent-damaged");
}

export async function grabStage(world: World, grabId: string): Promise<string> {
  return Grab.parse((await world.api("GET", `/api/v1/grabs/${grabId}`)).body).stage;
}

/** A Manual search of the whole Work through the API, once every Indexer has responded. */
export async function manualSearched(world: World, work: string) {
  const started = await world.api("POST", `/api/v1/works/${work}/manual-search`, { again: true });
  expect(started.status, "the Manual search starts").toBe(202);
  let search = Manual.parse((await world.api("GET", `/api/v1/works/${work}/manual-search`)).body);
  await expect
    .poll(
      async () => {
        search = Manual.parse((await world.api("GET", `/api/v1/works/${work}/manual-search`)).body);
        return search.searching;
      },
      { message: "every Indexer responds", timeout: 60_000 },
    )
    .toBe(false);
  return search;
}

type PlanPick = NonNullable<z.infer<typeof Manual>["plan"]["rows"][number]["pick"]>;

/** The grab body's pick for a plan row's pick, naming the cells and lifts the row showed. */
export const planPick = (pick: PlanPick) => ({
  releaseId: pick.release.id,
  cells: pick.cells,
  lifts: pick.lifts.map(({ code, targetId }) => ({ code, targetId })),
});

/**
 * Grabs the plan's pick of that Release through the API while the fake
 * seeders stall, so the Grab holds at downloading until the seeders are freed.
 */
export async function downloading(world: World, work: string, title: string): Promise<string> {
  const search = await manualSearched(world, work);
  const pick = search.plan.rows.find((row) => row.pick?.release.title === title)?.pick;
  if (pick === undefined || pick === null) {
    throw new Error(`the plan doesn't pick ${title}`);
  }
  await world.fw.fault.set({ id: "swarm-stalled", target: "seeders", fault: "stall" });
  const grabbed = await world.api("POST", `/api/v1/works/${work}/manual-search/grabs`, {
    manualSearchId: search.id,
    picks: [planPick(pick)],
  });
  const grabId = grabIdOf(grabbed.body);
  if (grabId === null) {
    throw new Error(`the grab of ${title} was refused: ${JSON.stringify(grabbed.body)}`);
  }
  await expect
    .poll(() => grabStage(world, grabId), { message: "the Grab downloads", timeout: 30_000 })
    .toBe("downloading");
  return grabId;
}

/** A Release on the Blocklist the way an Admin puts it there: Cancel with Also blocklist. */
export async function blocklisted(world: World, work: string, title: string): Promise<void> {
  const grabId = await downloading(world, work, title);
  const canceled = await world.api("POST", `/api/v1/grabs/${grabId}/cancel`, { blocklist: true });
  expect(canceled.status, "the Grab is canceled").toBe(204);
  await world.fw.fault.clear("swarm-stalled");
}

export const grabIdOf = (body: unknown): string | null =>
  Grabs.parse(body).results[0]?.grabId ?? null;

/** The Manual search sheet: a dialog on desktop and a sheet on phone, titled with the Work. */
export const sheet = (page: Page, title: string = orchard): Locator =>
  page.getByRole("dialog", { name: title, exact: true });

export const searchingLine = (sheetLocator: Locator): Locator =>
  sheetLocator.getByText(/^(Searching|SlipStream searched) /u).first();

/** Waits until every Indexer has responded, Heron last, 20 seconds late. */
export async function searched(sheetLocator: Locator): Promise<void> {
  await expect(sheetLocator).toContainText("SlipStream searched 4 Indexers.", { timeout: 60_000 });
}

export const grabRows = (sheetLocator: Locator): Locator =>
  sheetLocator.getByRole("list", { name: "Grabs" }).locator(":scope > li");

/** Opens a row's Other choices and presses Use this on the choice described by words. */
export async function useChoice(row: Locator, words: string): Promise<void> {
  await row.getByRole("button", { name: "Other choices" }).click();
  const choice = row
    .getByRole("list", { name: "Other choices" })
    .locator(":scope > li")
    .filter({ hasText: words });
  await choice
    .getByRole("button", { name: /^(Use this|Remove from Blocklist and grab)$/u })
    .click();
}

/** The hero's Manual search, from the phone's action sheet or beside Search. */
export async function openFromHero(page: Page, title: string): Promise<void> {
  if (isPhone(page)) {
    await workAction(page, "Manual search", title);
    return;
  }
  await page
    .getByRole("main")
    .locator("header")
    .getByRole("button", { name: "Manual search", exact: true })
    .click();
}

/**
 * Chooses an action from the menu of the newest Grab row that names the
 * Release, as on Activity or the Work page.
 */
export async function grabAction(
  page: Page,
  rows: Locator,
  title: string,
  action: "Details" | "Cancel Grab",
): Promise<void> {
  await rows
    .getByRole("row")
    .filter({ hasText: title })
    .first()
    .getByRole("button", { name: /actions$/u })
    .click();
  await page.getByRole("menuitem", { name: action }).click();
}

export async function openGrabDetails(page: Page, rows: Locator, title: string): Promise<Locator> {
  await grabAction(page, rows, title, "Details");
  return page.getByRole("dialog", { name: /^Grab of /u });
}

/** Activity's Blocklist view: a tab on desktop, a choice of View on phone. */
export async function openBlocklist(page: Page): Promise<void> {
  await navigate(page, "Activity");
  await (
    isPhone(page)
      ? page.getByRole("radio", { name: "Blocklist" })
      : page.getByRole("navigation", { name: "Activity" }).getByRole("link", { name: "Blocklist" })
  ).click();
}

export async function closeSheet(page: Page): Promise<void> {
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
}

/** Saves a viewport screenshot beside the test's other output, named for the story it shows. */
export async function frame(page: Page, testInfo: TestInfo, name: string): Promise<void> {
  await page.screenshot({ path: testInfo.outputPath(`${name}.png`) });
}
