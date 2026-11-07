import type { Locator, Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import { expectMonitoring, treeOf, workIdOf } from "./series.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "series" });
test.describe.configure({ timeout: 120_000 });

const results = (page: Page) => page.getByRole("list", { name: "Results" });

async function search(page: Page, query: string) {
  await page.getByRole("searchbox", { name: "Title" }).fill(query);
  await page.getByRole("main").getByRole("button", { name: "Search", exact: true }).click();
}

async function choose(page: Page, sheet: Locator, preset: RegExp) {
  await sheet.getByRole("combobox", { name: "Episodes to monitor" }).click();
  await clickWhenStill(page.getByRole("option", { name: preset }));
  await expect(sheet.getByRole("combobox", { name: "Episodes to monitor" })).toHaveText(preset);
}

const sentence = (sheet: Locator) => sheet.getByText(/^Adds /u);

async function expectEachPreset(page: Page, sheet: Locator) {
  await sheet.getByRole("combobox", { name: "Episodes to monitor" }).click();
  await expect(page.getByRole("option")).toHaveText([
    "All · 9 episodes",
    "Future · 0 episodes",
    "Missing · 9 episodes",
    "Existing · 0 episodes",
    "First season · 3 episodes",
    "Latest season · 3 episodes",
    "None · 0 episodes",
  ]);
  await page.keyboard.press("Escape");
  const monitoring = [
    [/^Future/u, "monitoring only new episodes as they air"],
    [/^Missing/u, "monitoring all 9 episodes and new ones as they air"],
    [/^Existing/u, "monitoring no episodes"],
    [/^First season/u, "monitoring 3 episodes and no new ones"],
    [/^None/u, "monitoring no episodes"],
    [/^Latest season/u, "monitoring 3 episodes and new ones as they air"],
  ] as const;
  for (const [preset, words] of monitoring) {
    // oxlint-disable-next-line no-await-in-loop -- each choice reads the sentence the last one left
    await choose(page, sheet, preset);
    // oxlint-disable-next-line no-await-in-loop -- as above
    await expect(sentence(sheet)).toContainText(`, ${words}, `);
  }
}

async function expectRemembered(page: Page) {
  await page.goto("/add?contentType=series&query=lantern");
  await expect(
    page.getByRole("link", { name: /^The Lantern Keepers .*In the library$/u }),
  ).toBeVisible();
  await page.goto("/add?contentType=series&query=harbor");
  await page.getByRole("button", { name: "Add Harbor Nights" }).click();
  const next = page.getByRole("dialog", { name: "Add Harbor Nights" });
  await expect(next.getByRole("combobox", { name: "Episodes to monitor" })).toHaveText(
    "Latest season · 2 episodes",
  );
  await expect(next.getByRole("switch", { name: "Include specials" })).toBeChecked();
  await expect(next.getByRole("switch", { name: "Season folders" })).not.toBeChecked();
  await expect(sentence(next)).toContainText(", monitoring 2 episodes and new ones as they air, ");
}

test("a series found by its alternative title adds with the preset, specials and folders chosen, and Remember keeps them", async ({
  adminPage: page,
  world,
}) => {
  await page.goto("/library/series");
  await expect(page.getByText("No Series in the library yet.")).toBeVisible();
  await page.getByRole("main").getByRole("button", { name: "Add a Series" }).last().click();
  await expect(page.getByRole("heading", { name: "Add a Series", level: 1 })).toBeVisible();

  await search(page, "Nights at the Harbor");
  await expect(results(page).getByRole("listitem")).toHaveCount(1);
  await expect(results(page)).toContainText("Harbor Nights2015 · Harbor TV");

  await search(page, "lantern");
  await page.getByRole("button", { name: "Add The Lantern Keepers" }).click();
  const sheet = page.getByRole("dialog", { name: "Add The Lantern Keepers" });
  await expect(sheet).toContainText("3 seasons · 9 episodes");
  await expect(sentence(sheet)).toHaveText(
    /^Adds The Lantern Keepers \(2020\) to TV on .+[/\\]tv, monitoring all 9 episodes and new ones as they air, wanting TV's 1 Target\.$/u,
  );
  await expectEachPreset(page, sheet);

  const specials = sheet.getByRole("switch", { name: "Include specials" });
  await expect(specials).not.toBeChecked();
  await specials.click();
  await expect(sentence(sheet)).toContainText(", monitoring 5 episodes and new ones as they air, ");
  const folders = sheet.getByRole("switch", { name: "Season folders" });
  await expect(folders).toBeChecked();
  await folders.click();
  await expect(sheet.getByRole("switch", { name: "Remember these choices" })).toBeChecked();
  await sheet.getByRole("button", { name: "Add Series" }).click();

  await expect(page.getByRole("heading", { name: "The Lantern Keepers", level: 1 })).toBeVisible();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Added The Lantern Keepers",
  );
  const workId = workIdOf(page);
  await expectMonitoring(page, world, workId, [5, 11]);
  expect(await treeOf(world, workId)).toMatchObject({
    monitorNew: true,
    monitorAsides: true,
    levelFolders: false,
  });

  await expectRemembered(page);
});

test("while TVDB search fails, TMDB's results show, and one TVDB doesn't list can't be added", async ({
  adminPage: page,
  world,
}) => {
  await world.fw.fault.set({
    id: "tvdb-search",
    target: "tvdb",
    route: "search",
    fault: "status=500",
  });
  await page.goto("/add?contentType=series&query=harbor");
  const unlisted = results(page).getByRole("listitem").filter({ hasText: "Harbor Lights" });
  await expect(unlisted).toContainText(
    "TVDB doesn't list this series. SlipStream needs TVDB to keep it up to date.",
  );
  await expect(unlisted.getByRole("button")).toHaveCount(0);

  await world.fw.fault.clear("tvdb-search");
  await page.goto(`/add?contentType=series&query=${encodeURIComponent("Lantern Keepers")}`);
  await expect(page.getByRole("button", { name: "Add The Lantern Keepers" })).toBeVisible();
});
