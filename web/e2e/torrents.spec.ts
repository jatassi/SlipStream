import type { Locator, Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "torrents" });
test.describe.configure({ timeout: 120_000 });

const work = "The Quiet Orchard";

const Added = z.object({ work: z.object({ id: z.string() }) });
const Rows = z.object({ rows: z.array(z.object({ state: z.string() })) });
const Grabs = z.object({
  grabs: z.array(z.object({ failure: z.nullable(z.object({ code: z.string() })) })),
});

// The Quiet Orchard, added and searched the way a script would, with its
// seeders held to a trickle so it stays downloading.
async function downloading(world: World): Promise<void> {
  await world.fw.fault.set({
    id: "swarm-hold",
    target: "seeders",
    node: "home",
    fault: "throttle=16KiB/s",
  });
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    monitor: true,
  });
  expect(added.status, "The Quiet Orchard is added").toBe(200);
  const workId = Added.parse(added.body).work.id;
  const search = await world.api("POST", `/api/v1/works/${workId}/search`);
  expect(search.status, "its search starts").toBe(202);
  await expect
    .poll(
      async () =>
        Rows.parse((await world.api("GET", "/api/v1/torrents")).body).rows.map((row) => row.state),
      { message: "its torrent downloads", timeout: 30_000 },
    )
    .toEqual(["downloading"]);
}

const table = (page: Page) => page.getByRole("table", { name: "Torrents" });
const row = (page: Page) => table(page).getByRole("row").filter({ hasText: work });

async function act(page: Page, action: string): Promise<void> {
  await row(page)
    .getByRole("button", { name: `${work} actions` })
    .click();
  const actions = isPhone(page)
    ? page.getByRole("dialog", { name: `${work} actions` }).getByRole("button", { name: action })
    : page.getByRole("menu", { name: `${work} actions` }).getByRole("menuitem", { name: action });
  await actions.click();
}

const header = (page: Page, name: string) => table(page).getByRole("columnheader", { name });
const wide = ["Ratio", "Speed", "Tracker", "Node · client"];

async function expectColumns(page: Page, shown: readonly string[], hidden: readonly string[]) {
  for (const name of shown) {
    // oxlint-disable-next-line no-await-in-loop -- one header at a time names the one that fails
    await expect(header(page, name), `${name} shows`).toBeVisible();
  }
  for (const name of hidden) {
    // oxlint-disable-next-line no-await-in-loop -- one header at a time names the one that fails
    await expect(header(page, name), `${name} hides`).toBeHidden();
  }
}

async function expectCards(page: Page) {
  await expect(header(page, "Work")).toBeHidden();
  await expect(row(page).getByRole("cell", { name: /^Speed Down /u })).toBeVisible();
  await expect(row(page).getByRole("cell", { name: /^Tracker 127\.0\.0\.1/u })).toBeVisible();
}

test("the table shows eight columns at 1280 px, four at 980 px and stacked cards at 620 px", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await downloading(world);
  await page.goto("/activity/torrents");
  await expect(row(page)).toContainText("Downloading");
  if (isPhone(page)) {
    await expectCards(page);
    await page.screenshot({ path: testInfo.outputPath("cards.png") });
    return;
  }
  await page.setViewportSize({ width: 1280, height: 900 });
  await expectColumns(page, ["Work", "State", "Progress", ...wide], []);
  await expect(row(page).getByRole("button", { name: `${work} actions` })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("1280.png") });

  await page.setViewportSize({ width: 980, height: 900 });
  await expectColumns(page, ["Work", "State", "Progress"], wide);
  await page.screenshot({ path: testInfo.outputPath("980.png") });

  await page.setViewportSize({ width: 620, height: 900 });
  await expectCards(page);
  await page.screenshot({ path: testInfo.outputPath("620.png") });
});

test("a resume inside its tracker's window waits, reading Resumes at, and the tracker refuses nothing", async ({
  adminPage: page,
  world,
}) => {
  await downloading(world);
  await page.goto("/activity/torrents");
  await expect(row(page)).toContainText("Downloading");

  await act(page, "Pause");
  await expect(row(page).getByRole("cell", { name: /^Paused/u })).toBeVisible();
  await act(page, "Resume");
  await expect(row(page).getByText(/^Resumes at \d{1,2}:\d{2}/u)).toBeVisible();
  await expect(row(page)).not.toContainText("left");
  await expectAccessible(page);

  const announces = world.fw.capture("announces");
  await announces.wait({ node: "home", where: { event: "stopped" }, after: 0, timeout: 15_000 });
  const started = await announces.list({ node: "home", where: { event: "started" } });
  expect(started, "only the grab's own started went out").toHaveLength(1);
  expect(await announces.list({ where: { accepted: "false" } }), "rejected announces").toEqual([]);
});

function inspectorOf(page: Page): Locator {
  return isPhone(page) ? page.getByRole("main") : page.getByRole("complementary", { name: work });
}

test("the Inspector reads one torrent, and a remove with its download copy cancels its Grab", async ({
  adminPage: page,
  world,
}, testInfo) => {
  // The live fills ease every tick, so a downloading page never holds still for the scan.
  await page.emulateMedia({ reducedMotion: "reduce" });
  await downloading(world);
  await page.goto("/activity/torrents");
  await row(page)
    .getByRole("button", { name: new RegExp(`^${work}(?! actions)`, "u") })
    .click();
  await expect(page).toHaveURL(/\/activity\/torrents\/[\da-f-]+$/u);
  const inspector = inspectorOf(page);
  await expect(inspector.getByText(/^Downloading from \d+ peers?/u)).toBeVisible();
  for (const section of ["Numbers", "Trackers", "Files", "Grabs", "Details"]) {
    // oxlint-disable-next-line no-await-in-loop -- one section at a time names the one missing
    await expect(inspector.getByRole("heading", { name: section, exact: true })).toBeVisible();
  }
  await expect(
    inspector.getByRole("definition").filter({ hasText: /[/\\]Movies[/\\]/u }),
  ).toHaveText(/[/\\]downloads[/\\]Movies[/\\][\da-f]{40}$/u);
  await page.screenshot({ path: testInfo.outputPath("inspector.png"), fullPage: true });

  await inspector.getByRole("button", { name: "Remove", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: `Remove ${work}?` });
  await expect(dialog.getByText("Files in your library stay.")).toBeVisible();
  await expect(dialog.getByText(/^It cancels this Grab, still downloading/u)).toBeVisible();
  const copy = dialog.getByRole("checkbox", { name: /^Also delete the download copy \(frees /u });
  await copy.check();
  await expectAccessible(page);
  await dialog.getByRole("button", { name: "Remove", exact: true }).click();
  await expect(dialog).toBeHidden();

  await world.fw.capture("announces").wait({
    node: "home",
    where: { event: "stopped" },
    after: 0,
    timeout: 30_000,
  });
  await expect
    .poll(async () => Rows.parse((await world.api("GET", "/api/v1/torrents")).body).rows, {
      message: "home confirms the remove",
      timeout: 30_000,
    })
    .toEqual([]);
  const { grabs } = Grabs.parse((await world.api("GET", "/api/v1/grabs")).body);
  expect(grabs.map((grab) => grab.failure?.code)).toEqual(["removed"]);
  await page.goto("/activity/torrents");
  await expect(page.getByText(/^No torrents yet\./u)).toBeVisible();
});
