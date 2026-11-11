import type { Page } from "@playwright/test";

import { workAction } from "./cells.ts";
import { episodeRow, reveal } from "./series.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

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

/** Chooses an action from the menu that label names, an action sheet on phone. */
export async function menuAction(
  page: Page,
  label: string | RegExp,
  action: string,
): Promise<void> {
  await clickWhenStill(
    isPhone(page)
      ? page.getByRole("dialog", { name: label }).getByRole("button", { name: action, exact: true })
      : page
          .getByRole("menu", { name: label })
          .getByRole("menuitem", { name: action, exact: true }),
  );
}

/** The Manual search of a Target's cell: beside its Search on desktop, in the Search menu on phone. */
export async function openFromCell(page: Page, target: string): Promise<void> {
  const cell = (words: string) => new RegExp(`^${words} for ${target}, `, "u");
  if (isPhone(page)) {
    await page.getByRole("button", { name: cell("Search") }).click();
    await menuAction(page, cell("Actions"), "Manual search");
    return;
  }
  await page.getByRole("button", { name: cell("Manual search") }).click();
}

/** An episode's Manual search: in its row's menu on desktop, in its sheet on phone. */
export async function openFromEpisode(page: Page, code: string): Promise<void> {
  if (isPhone(page)) {
    await clickWhenStill(await reveal(page, episodeRow(page, code)));
    await page
      .getByRole("dialog", { name: new RegExp(`^${code} `, "u") })
      .getByRole("button", { name: "Manual search" })
      .click();
    return;
  }
  await (
    await reveal(page, page.getByRole("button", { name: `More actions for ${code}` }))
  ).click();
  await menuAction(page, `Actions for ${code}`, "Manual search");
}

/** Presses an action of the Needs you card: on its face on desktop, under More on phone. */
export async function needsYouAction(
  page: Page,
  action: "Manual search" | "Details",
): Promise<void> {
  const needsYou = page.getByRole("region", { name: "Needs you" });
  if (isPhone(page)) {
    await needsYou.getByRole("button", { name: "More", exact: true }).click();
    await menuAction(page, "More actions", action);
    return;
  }
  await needsYou.getByRole("button", { name: action }).click();
}
