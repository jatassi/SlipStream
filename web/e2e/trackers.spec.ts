import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, openSettingsSection, pageAction } from "./shell.ts";
import { untilStill } from "./still.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "indexers" });
test.describe.configure({ timeout: 180_000 });

const Rule = z.object({
  clears: z.string(),
  ratio: z.number().nullable(),
  seedSeconds: z.number().nullable(),
  source: z.string(),
});
const Goal = z.object({
  combine: z.string(),
  ratio: z.number().nullable(),
  seedSeconds: z.number().nullable(),
  action: z.string().nullable(),
});
const Tracker = z.object({
  id: z.string(),
  name: z.string(),
  privacy: z.string(),
  rule: Rule.nullable(),
  goal: Goal.nullable(),
});
const Trackers = z.object({ trackers: z.array(Tracker) });

async function trackers(world: World) {
  return Trackers.parse((await world.api("GET", "/api/v1/trackers")).body).trackers;
}

async function capture(page: Page, path: string) {
  await untilStill(page);
  await screenshotWholePage(page, path);
}

async function openSeeding(page: Page, name: string) {
  if (isPhone(page)) {
    await page
      .getByRole("list", { name })
      .getByRole("link", { name: /^Seeding/u })
      .click();
  } else {
    await page
      .getByRole("navigation", { name })
      .getByRole("link", { name: /^Seeding/u })
      .click();
  }
  await expect(page).toHaveURL(/\/seeding$/u);
  await expect(page.getByText(`No torrent owes ${name} seeding.`)).toBeVisible();
  await expect(page.getByText("No hit and run window has ended yet.")).toBeVisible();
}

test("an Admin opens a Tracker from Settings > Trackers and sets its rule and goal", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const tracker = (await trackers(world)).find((each) => each.privacy !== "public");
  if (tracker === undefined) {
    throw new Error("the indexers world has no Tracker that may have a rule");
  }
  await openSettingsSection(page, "Trackers");
  await expect(page.getByRole("form", { name: "Seeding goal" })).toBeVisible();
  await expect(page.getByRole("term").filter({ hasText: "Seeding goals" })).toBeVisible();
  await capture(page, testInfo.outputPath("settings-trackers.png"));
  await expectAccessible(page);

  await page
    .getByRole("list", { name: "Trackers" })
    .getByRole("link", { name: tracker.name })
    .click();
  await expect(page).toHaveURL(new RegExp(`/settings/trackers/${tracker.id}$`, "u"));
  await expect(pageAction(page, `${tracker.name} actions`)).toBeVisible();
  await expect(page.getByRole("button", { name: "Details" })).toBeVisible();
  await capture(page, testInfo.outputPath("tracker-page.png"));
  await expectAccessible(page);

  await openSeeding(page, tracker.name);
  const rule = page.getByRole("form", { name: "Seeding rule" });
  await rule.getByRole("radio", { name: "Either one" }).click();
  await rule.getByRole("textbox", { name: "Ratio" }).fill("1");
  await rule.getByRole("textbox", { name: "Seeding time" }).fill("8");
  await rule.getByRole("button", { name: "Save rule" }).click();
  await expect(
    page
      .getByRole("region", { name: "Rule" })
      .getByText(`${tracker.name} requires a ratio of 1.0 or 8 days of seeding for each torrent.`),
  ).toBeVisible();
  const goal = page.getByRole("form", { name: "Seeding goal" });
  await goal.getByRole("textbox", { name: "Ratio" }).fill("3");
  await goal.getByRole("button", { name: "Save goal" }).click();
  await expect(page.getByText(`${tracker.name} has its own goal.`, { exact: false })).toBeVisible();
  await capture(page, testInfo.outputPath("seeding-tab.png"));
  await expectAccessible(page);

  await expect
    .poll(async () => (await trackers(world)).find((each) => each.id === tracker.id))
    .toMatchObject({
      rule: { clears: "either", ratio: 1, seedSeconds: 691_200, source: "admin" },
      goal: { combine: "first", ratio: 3, seedSeconds: 1_209_600, action: "remove-delete" },
    });
});

const Routes = z.object({
  routes: z.array(z.object({ homeNodeId: z.string(), stored: z.unknown().nullable() })),
});

async function storedRoutes(world: World, trackerId: string) {
  const { body } = await world.api("GET", `/api/v1/trackers/${trackerId}/routes`);
  return Routes.parse(body).routes.map((route) => route.stored);
}

async function openRouting(page: Page, name: string) {
  const sections = isPhone(page)
    ? page.getByRole("list", { name })
    : page.getByRole("navigation", { name });
  await sections.getByRole("link", { name: /^Routing/u }).click();
  await expect(page).toHaveURL(/\/routing$/u);
}

test("an Admin routes a private Tracker's Grabs to a client, which Download Clients lists, then uses the default again", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const tracker = (await trackers(world)).find((each) => each.privacy === "private");
  if (tracker === undefined) {
    throw new Error("the indexers world has no private Tracker");
  }
  await openSettingsSection(page, "Trackers");
  await page
    .getByRole("list", { name: "Trackers" })
    .getByRole("link", { name: tracker.name })
    .click();
  await openRouting(page, tracker.name);
  const row = page.getByRole("region", { name: /^Indexers on /u }).first();
  await expect(row.getByRole("combobox", { name: "Grabs go to" })).toBeVisible();
  await capture(page, testInfo.outputPath("routing-tab.png"));
  await expectAccessible(page);

  await row.getByRole("combobox", { name: "Grabs go to" }).click();
  await page.getByRole("option", { name: /^Built-in client on home/u }).click();
  await expect(
    row.getByText("These Grabs go to the built-in client on home, which you picked."),
  ).toBeVisible();
  await expect(
    row.getByText(/^The built-in client on home announces as libtorrent /u),
  ).toBeVisible();
  await expect
    .poll(() => storedRoutes(world, tracker.id))
    .toContainEqual({ builtInOn: expect.any(String) });
  await capture(page, testInfo.outputPath("routing-picked.png"));

  await openSettingsSection(page, "Download Clients");
  const routed = page
    .getByRole("article", { name: "Built-in client on home" })
    .getByRole("list", { name: "Trackers routed here" });
  await expect(routed.getByRole("link", { name: tracker.name })).toBeVisible();
  await expect(routed).not.toContainText("(default)");
  await capture(page, testInfo.outputPath("download-clients-routed.png"));
  await expectAccessible(page);

  await routed.getByRole("link", { name: tracker.name }).click();
  await expect(page).toHaveURL(new RegExp(`/settings/trackers/${tracker.id}/routing$`, "u"));
  await row.getByRole("button", { name: "Use the default" }).click();
  await expect(row.getByRole("button", { name: "Use the default" })).not.toBeVisible();
  await expect.poll(() => storedRoutes(world, tracker.id)).not.toContainEqual(expect.anything());
});
