import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderApp } from "../test/render-app";
import { searchSettingsWorld } from "../test/search-settings";

const path = "/settings/automatic-search";

function openSection() {
  const world = searchSettingsWorld();
  renderApp({ path, handlers: world.handlers });
  return {
    world,
    section: page.getByRole("region", { name: "Automatic search" }),
  };
}

test("the section leads with what Backlog search and RSS sync do, and the lead follows the switches", async () => {
  const { world, section } = openSection();
  await expect
    .element(
      section.getByText(
        "SlipStream searches for what's wanted every 8 hours and reads new uploads every 15 minutes. Searches never go past an Indexer's limits.",
      ),
    )
    .toBeVisible();
  await userEvent.click(section.getByRole("switch", { name: "Backlog search", exact: true }));
  await expect
    .element(
      section.getByText(
        "SlipStream reads new uploads every 15 minutes and searches only when you ask. Searches never go past an Indexer's limits.",
      ),
    )
    .toBeVisible();
  expect(world.writes.at(-1)).toEqual({
    values: { "search.backlogOn": false },
  });
  await userEvent.click(section.getByRole("switch", { name: "RSS sync" }));
  await expect
    .element(
      section.getByText(
        "SlipStream searches only when you ask. Searches never go past an Indexer's limits.",
      ),
    )
    .toBeVisible();
  expect(world.writes.at(-1)).toEqual({ values: { "search.rssOn": false } });
});

test("the machinery sits under Details, each value saved on its own", async () => {
  const { world, section } = openSection();
  await expect
    .element(
      section.getByRole("switch", {
        name: "Backlog search looks for upgrades",
      }),
    )
    .toBeVisible();
  const wait = section.getByRole("textbox", {
    name: "Wait for each Indexer up to",
  });
  expect(wait.query()).toBeNull();
  await userEvent.click(section.getByRole("button", { name: "Details" }));
  await expect
    .element(
      section.getByRole("textbox", {
        name: "Search a Want that keeps coming up empty at least every",
      }),
    )
    .toBeVisible();
  await expect
    .element(
      section.getByRole("textbox", {
        name: "Don't space out searches for Items released in the last",
      }),
    )
    .toBeVisible();
  await userEvent.fill(wait, "5");
  await userEvent.click(section.getByRole("button", { name: "Save" }));
  await expect.poll(() => world.writes.at(-1)).toEqual({ values: { "indexers.searchWait": 300 } });
  await expect.element(section.getByRole("button", { name: "Reset to default" })).toBeVisible();
});

test("the stop reads as one line and saves its count and its window apart", async () => {
  const { world, section } = openSection();
  await userEvent.click(section.getByRole("button", { name: "Details" }));
  await userEvent.click(
    section.getByRole("button", {
      name: "Stop searching again after: 3 failed Grabs",
    }),
  );
  await userEvent.fill(page.getByRole("textbox", { name: "Stop searching again after" }), "1");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.poll(() => world.writes.at(-1)).toEqual({ values: { "search.stopCount": 1 } });
  await expect
    .element(
      section.getByRole("button", {
        name: "Stop searching again after: 1 failed Grab",
      }),
    )
    .toBeVisible();
  await userEvent.click(
    section.getByRole("button", {
      name: "Count failed Grabs in the last: 7 days",
    }),
  );
  await userEvent.fill(page.getByRole("textbox", { name: "Count failed Grabs in the last" }), "2");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => world.writes.at(-1))
    .toEqual({ values: { "search.stopWindow": 2 * 24 * 3600 } });
});

test("Backlog search's interval sits beside its switches, and a longest gap shorter than it is refused on the field that moved", async () => {
  const { world, section } = openSection();
  await userEvent.click(section.getByRole("button", { name: "Details" }));
  await userEvent.fill(
    section.getByRole("textbox", {
      name: "Search a Want that keeps coming up empty at least every",
    }),
    "1",
  );
  await userEvent.click(section.getByRole("button", { name: "Save" }));
  await expect.poll(() => world.writes.at(-1)).toEqual({ values: { "search.longestGap": 86_400 } });
  await userEvent.fill(section.getByRole("textbox", { name: "Backlog search runs every" }), "48");
  await userEvent.click(section.getByRole("button", { name: "Save" }));
  await expect
    .element(
      section
        .getByRole("alert")
        .getByText("Use a longest gap of at least the Backlog search interval, 2 days."),
    )
    .toBeVisible();
  expect(world.writes.at(-1)).toEqual({ values: { "tasks.backlog-search.cadence": 172_800 } });
  await userEvent.fill(section.getByRole("textbox", { name: "Backlog search runs every" }), "24");
  await userEvent.click(section.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => world.writes.at(-1))
    .toEqual({ values: { "tasks.backlog-search.cadence": 86_400 } });
  expect(section.getByRole("alert").query()).toBeNull();
});

test("RSS sync's interval sits beside its switch, and the lead reads the new interval", async () => {
  const { world, section } = openSection();
  await userEvent.fill(section.getByRole("textbox", { name: "RSS sync runs every" }), "30");
  await userEvent.click(section.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => world.writes.at(-1))
    .toEqual({ values: { "tasks.rss-sync.cadence": 1800 } });
  await expect
    .element(
      section.getByText(
        "SlipStream searches for what's wanted every 8 hours and reads new uploads every 30 minutes. Searches never go past an Indexer's limits.",
      ),
    )
    .toBeVisible();
});
