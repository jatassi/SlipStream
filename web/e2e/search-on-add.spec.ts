import type { Locator, Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "auto-search" });
test.describe.configure({ timeout: 120_000 });

const notifications = (page: Page) => page.getByRole("region", { name: "Messages" });
const searchLine = (page: Page) => page.getByRole("status", { name: "Search" });
const searches = (page: Page) => page.getByRole("list", { name: "Searches" });

async function openAdd(page: Page, contentType: string, query: string, label: string) {
  await page.goto(`/add?contentType=${contentType}&query=${encodeURIComponent(query)}`);
  await page.getByRole("button", { name: `Add ${label}` }).click();
  const sheet = page.getByRole("dialog", { name: `Add ${label}` });
  await expect(sheet).toBeVisible();
  return sheet;
}

async function choose(page: Page, sheet: Locator, field: string, option: RegExp | string) {
  await sheet.getByRole("combobox", { name: field }).click();
  await clickWhenStill(
    page.getByRole(
      "option",
      typeof option === "string" ? { name: option, exact: true } : { name: option },
    ),
  );
  await expect(sheet.getByRole("combobox", { name: field })).toHaveText(option);
}

test("a movie added with Search after adding grabs its Release, and the switch is remembered off", async ({
  adminPage: page,
}) => {
  const meadowlark = await openAdd(page, "movie", "meadowlark", "Meadowlark");
  const search = meadowlark.getByRole("switch", { name: "Search after adding" });
  await expect(search).toBeChecked();
  await search.click();
  await meadowlark.getByRole("button", { name: "Add Movie" }).click();
  await expect(page.getByRole("heading", { name: "Meadowlark", level: 1 })).toBeVisible();
  await expect(notifications(page)).toHaveText("Added Meadowlark");
  await expect(searches(page)).toHaveCount(0);

  const orchard = await openAdd(page, "movie", "quiet orchard", "The Quiet Orchard");
  const remembered = orchard.getByRole("switch", { name: "Search after adding" });
  await expect(remembered).not.toBeChecked();
  await remembered.click();
  await orchard.getByRole("button", { name: "Add Movie" }).click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard", level: 1 })).toBeVisible();
  await expect(notifications(page)).toContainText(/Added The Quiet Orchard.+Searching/u);
  await expect(searchLine(page)).toHaveText("Grabbed 1 release", { timeout: 30_000 });
  await expect(searches(page)).toContainText(/^Grabbed 1 releaseAdded/u);
});

test("a series' search choice warns when it misses what is monitored, and searches only its season", async ({
  adminPage: page,
}) => {
  const sheet = await openAdd(page, "series", "lantern", "The Lantern Keepers");
  await choose(page, sheet, "Episodes to monitor", /^Latest season/u);
  await choose(page, sheet, "Search after adding", "First season");
  const note = sheet.getByText(
    "Nothing in the first season is monitored, so SlipStream won't search after adding.",
  );
  await expect(note).toBeVisible();
  await choose(page, sheet, "Episodes to monitor", /^All/u);
  await expect(note).toBeHidden();

  await sheet.getByRole("button", { name: "Add Series" }).click();
  await expect(page.getByRole("heading", { name: "The Lantern Keepers", level: 1 })).toBeVisible();
  await expect(notifications(page)).toContainText(/Added The Lantern Keepers.+Searching/u);
  await expect(searchLine(page)).toHaveText("Grabbed 1 release", { timeout: 30_000 });
  await expect(searches(page).locator(":scope > li")).toHaveCount(1);
});
