import type { Page } from "@playwright/test";

import { expect } from "./fixtures.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

const label = "The Quiet Orchard";

// On phone the matrix is one card per Edition, a list rather than a table.
const matrix = (page: Page) =>
  isPhone(page)
    ? page.getByRole("list", { name: "Editions and Targets" })
    : page.getByRole("table", { name: "Editions and Targets" });

// Every scenario Profile wants Any cut, so the Work's cells sit in that one Edition's row.
export const orchardCells = (page: Page) =>
  isPhone(page)
    ? matrix(page).getByRole("listitem", { name: "Any cut" })
    : matrix(page).getByRole("row", { name: /^Any cut/u });

// The phone cards have no column headers, only each cell's Target label.
export async function expectColumns(page: Page, targets: readonly string[]): Promise<void> {
  if (!isPhone(page)) {
    await expect(matrix(page).getByRole("columnheader")).toHaveText([
      "Edition",
      ...targets.map((target) => `${target} Downloaded`),
    ]);
  }
}

/** The hero's Search for the Work. */
export const workSearch = (page: Page) =>
  page.getByRole("main").getByRole("button", { name: "Search", exact: true });

/** Picks an action from the hero's "…" menu, an action sheet on phone. */
export async function workAction(page: Page, name: string, work = label): Promise<void> {
  await page.getByRole("button", { name: `More actions for ${work}` }).click();
  await clickWhenStill(
    page.getByRole(isPhone(page) ? "button" : "menuitem", { name, exact: true }),
  );
}
