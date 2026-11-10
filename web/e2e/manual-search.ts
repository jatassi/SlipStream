import type { Locator, Page, TestInfo } from "@playwright/test";
import { z } from "zod";

import { type World, expect } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

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
    await page.getByRole("button", { name: `More actions for ${title}` }).click();
    await page.getByRole("menuitem", { name: "Manual search" }).click();
    return;
  }
  await page
    .getByRole("main")
    .locator("header")
    .getByRole("button", { name: "Manual search", exact: true })
    .click();
}

export async function closeSheet(page: Page): Promise<void> {
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
}

/** Saves a viewport screenshot beside the test's other output, named for the story it shows. */
export async function frame(page: Page, testInfo: TestInfo, name: string): Promise<void> {
  await page.screenshot({ path: testInfo.outputPath(`${name}.png`) });
}
