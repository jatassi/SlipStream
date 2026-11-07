import type { Locator, Page } from "@playwright/test";

import { type World, expect, test } from "./fixtures.ts";
import { expectColumns, orchardCells } from "./cells.ts";
import { add, orchard } from "./library.ts";
import { addSeries, harborNights, lanternKeepers } from "./series.ts";
import { isPhone, navigate } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test("an Admin adds a movie from the catalog and sees its missing cell", async ({
  adminPage: page,
}) => {
  await expect(
    page.getByText("Nothing in the library yet. Add your first Work to start."),
  ).toBeVisible();
  await page.getByRole("main").getByRole("button", { name: "Add a Movie" }).click();
  await page.getByRole("searchbox", { name: "Title" }).fill("quiet orchard");
  await page.getByRole("main").getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.getByRole("list", { name: "Results" })).toHaveText("The Quiet Orchard2021Add");

  await page.getByRole("button", { name: "Add The Quiet Orchard" }).click();
  await page
    .getByRole("dialog", { name: "Add The Quiet Orchard" })
    .getByRole("button", { name: "Add Movie" })
    .click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard", exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Added The Quiet Orchard",
  );
  await expect(page.getByText("2021 · Movie · 1h 44m")).toBeVisible();
  await expect(page.getByRole("region", { name: "About" }).getByRole("definition")).toHaveText([
    "The Quiet Orchard",
    "en",
    "April 16, 2021",
  ]);
  await expect(orchardCells(page)).toContainText("Missing");
  await expectColumns(page, ["1080p"]);

  await navigate(page, "Library");
  await expect(page.getByRole("heading", { name: "Q 1", level: 2 })).toBeVisible();
  await expect(page.getByRole("main").getByRole("listitem")).toHaveText(
    "The Quiet Orchard2021Missing",
  );
  await page.goto("/add?contentType=movie&query=quiet+orchard");
  await expect(
    page.getByRole("link", { name: "The Quiet Orchard 2021 In the library" }),
  ).toBeVisible();
});

// A phone shows a menu as a sheet of switches.
const menuChoice = (page: Page, options: { name: string }) =>
  page.getByRole(isPhone(page) ? "switch" : "menuitemcheckbox", options);

async function seedBoth(world: World) {
  await addSeries(world, lanternKeepers, { preset: "first" });
  await addSeries(world, harborNights, { preset: "all" });
  await add(world, "9000001", "all");
}

async function expectAvailabilityWords(page: Page, types: Locator) {
  await page.getByRole("button", { name: /^Sort: /u }).click();
  await expect(menuChoice(page, { name: "Next airing" })).toBeVisible();
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Library options" }).click();
  await clickWhenStill(menuChoice(page, { name: "Table" }));
  await expect(menuChoice(page, { name: "Next airing" })).toBeVisible();
  await page.keyboard.press("Escape");

  await types.getByRole("link", { name: "All" }).click();
  await page.getByRole("button", { name: "Library options" }).click();
  await clickWhenStill(menuChoice(page, { name: "Table" }));
  await expect(menuChoice(page, { name: "Available" })).toBeVisible();
  await expect(menuChoice(page, { name: "Next airing" })).toHaveCount(0);
}

test.describe("with movies and TV", () => {
  test.use({ scenario: "series" });

  test("the Content Types switch moves between All, Movies and TV, each keeping its filters, and TV filters by its own fields", async ({
    adminPage: page,
    world,
  }) => {
    await seedBoth(world);

    await page.goto("/library");
    const types = page.getByRole("navigation", { name: "Content Types" });
    await expect(types.getByRole("link")).toHaveText(["All", "Movies", "TV"]);
    await expect(types.getByRole("link", { name: "All" })).toHaveAttribute("aria-current", "page");
    const posters = page.getByRole("main").getByRole("listitem");
    await expect(posters).toHaveCount(3);
    await expect(posters.filter({ hasText: orchard })).toContainText("Missing");

    await types.getByRole("link", { name: "TV" }).click();
    await expect(page).toHaveURL(/\/library\/series/u);
    await expect(page.getByRole("heading", { name: "TV", level: 1 })).toBeVisible();
    await expect(posters).toHaveCount(2);
    await expect(posters.filter({ hasText: "The Lantern Keepers" })).toContainText(
      "Missing · 0 of 3 met",
    );
    await expect(posters.filter({ hasText: "Harbor Nights" })).toContainText(
      "Missing · 0 of 4 met",
    );

    await page.getByRole("button", { name: "Filter" }).click();
    const filters = page.getByRole("dialog", { name: "Filters" });
    await expect(filters.getByRole("group", { name: "Network" })).toBeVisible();
    await filters
      .getByRole("group", { name: "Production status" })
      .getByRole("checkbox", { name: "Ended (1)" })
      .click();
    await page.keyboard.press("Escape");
    await expect(posters).toHaveText([/^Harbor Nights/u]);

    await types.getByRole("link", { name: "Movies" }).click();
    await expect(posters).toHaveText([new RegExp(`^${orchard}`, "u")]);
    await types.getByRole("link", { name: "TV" }).click();
    await expect(posters).toHaveText([/^Harbor Nights/u]);

    await expectAvailabilityWords(page, types);
  });
});
