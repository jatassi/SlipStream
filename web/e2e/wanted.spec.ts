import type { Locator, Page } from "@playwright/test";

import { type World, expect, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "auto-search" });
test.describe.configure({ timeout: 120_000 });

/** Adds both series and the three movies, none of them searched. */
async function addEverything(world: World): Promise<void> {
  const works = [
    { contentType: "movie", key: "9000001" },
    { contentType: "movie", key: "9000005" },
    { contentType: "movie", key: "9000006" },
    { contentType: "series", key: "1001", preset: "all" },
    { contentType: "series", key: "1002", preset: "all" },
  ];
  const adds = await Promise.all(
    works.map(async (work) => ({ work, add: await world.api("POST", "/api/v1/works", work) })),
  );
  for (const { work, add } of adds) {
    expect(add.status, `${work.contentType} ${work.key} is added`).toBe(200);
  }
}

const status = (page: Page) =>
  page.getByRole("main").getByRole("paragraph").filter({ hasText: "Next Backlog search" });
const notifications = (page: Page) => page.getByRole("region", { name: "Notifications" });

async function openWanted(page: Page): Promise<void> {
  await page.goto("/library");
  await page.getByRole("list", { name: "Wanted" }).getByText("5 wanted.").click();
  await expect(page.getByRole("heading", { name: "Wanted", level: 1 })).toBeVisible();
}

/** The Work's row: a table row on desktop, a list row that opens an action sheet on phone. */
const row = (page: Page, work: string): Locator =>
  isPhone(page)
    ? page
        .getByRole("list", { name: "Wanted" })
        .getByRole("button", { name: new RegExp(`^${work}`, "u") })
    : page.getByRole("table", { name: "Wanted" }).getByRole("row").filter({ hasText: work });

test("Search all queues every Work, the rest wait for Heron's limit, and Cancel queued searches empties the queue", async ({
  adminPage: page,
  world,
}) => {
  await addEverything(world);
  await openWanted(page);
  await expect(status(page)).toContainText(/^Nothing queued/u);

  await page.getByRole("button", { name: "Search 5 Works" }).click();
  await expect(notifications(page)).toContainText("Queued 5 Works for search.");
  await expect(status(page)).toContainText("4 queued");
  await expect(
    status(page).getByRole("link", { name: /^Waiting for Heron's search limit until /u }),
  ).toHaveAttribute("href", /^\/settings\/indexers\//u);

  await page.getByRole("button", { name: "Cancel queued searches" }).click();
  await expect(notifications(page)).toContainText("Canceled 4 queued searches.");
  await expect(status(page)).not.toContainText("queued");
  await expect(page.getByRole("button", { name: "Cancel queued searches" })).toBeHidden();
});

test("a row's Search grabs that Work's Release, and the Grab takes it off Wanted", async ({
  adminPage: page,
  world,
}) => {
  await addEverything(world);
  await openWanted(page);
  if (isPhone(page)) {
    await clickWhenStill(row(page, "Meadowlark"));
    await page
      .getByRole("dialog", { name: "Actions for Meadowlark" })
      .getByRole("button", { name: "Search", exact: true })
      .click();
  } else {
    await row(page, "Meadowlark").getByRole("button", { name: "Search Meadowlark" }).click();
  }
  await expect(notifications(page)).toContainText("Searching Meadowlark");
  await expect(row(page, "Meadowlark")).toBeHidden({ timeout: 30_000 });
  await expect(row(page, "The Quiet Orchard")).toBeVisible();
});

test("Items read as episode codes, a movie lists none, and a row never searched waits for the next Backlog search", async ({
  adminPage: page,
  world,
}) => {
  await addEverything(world);
  await openWanted(page);
  if (isPhone(page)) {
    await expect(row(page, "The Lantern Keepers")).toContainText(
      /Missing\. S01E01 to S02E03\. Next search .+ at \d{2}:\d{2}\./u,
    );
    await expect(row(page, "Meadowlark")).toContainText(
      /Missing\. Next search .+ at \d{2}:\d{2}\./u,
    );
    return;
  }
  const cells = (work: string) => row(page, work).getByRole("cell");
  await expect(cells("The Lantern Keepers").nth(1)).toHaveText("S01E01 to S02E03");
  await expect(cells("Meadowlark").nth(1)).toHaveText("");
  await expect(cells("The Lantern Keepers").nth(4)).toHaveText(/^.+ at \d{2}:\d{2}$/u);
  await expect(cells("Meadowlark").nth(4)).toHaveText(/^.+ at \d{2}:\d{2}$/u);
});
