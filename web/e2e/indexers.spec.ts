import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";

test.use({ scenario: "indexers" });
test.describe.configure({ timeout: 180_000 });

const Indexers = z.object({
  indexers: z.array(z.object({ id: z.string(), name: z.string(), priority: z.number() })),
});
const Nodes = z.object({ nodes: z.array(z.object({ id: z.string(), name: z.string() })) });
const Solver = z.object({ address: z.string().nullable() });
const CaptchaShown = z.object({ text: z.string() });

async function indexer(world: World, name: string) {
  const found = Indexers.parse((await world.api("GET", "/api/v1/indexers")).body).indexers.find(
    (row) => row.name === name,
  );
  if (found === undefined) {
    throw new Error(`the world has no Indexer ${name}`);
  }
  return found;
}

async function nodeID(world: World, name: string): Promise<string> {
  const found = Nodes.parse((await world.api("GET", "/api/v1/nodes")).body).nodes.find(
    (row) => row.name === name,
  );
  if (found === undefined) {
    throw new Error(`the world has no Node ${name}`);
  }
  return found.id;
}

async function openIndexer(page: Page, world: World, name: string) {
  await page.goto(`/settings/indexers/${(await indexer(world, name)).id}`);
  await expect(page.getByRole("heading", { name, exact: true, level: 1 })).toBeVisible();
}

const testButton = (page: Page) => page.getByRole("button", { name: "Test", exact: true });

test("an Admin edits Quarry's priority and tests it from its page", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await openIndexer(page, world, "Quarry");
  await page.getByRole("button", { name: "Edit", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("textbox", { name: "Priority" }).fill("5");
  await page.screenshot({ path: testInfo.outputPath("edit.png") });
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  await expect.poll(async () => (await indexer(world, "Quarry")).priority).toBe(5);

  await testButton(page).click();
  await expect(page.getByText(/^Quarry passed the Test and listed \d+ items?\.$/u)).toBeVisible({
    timeout: 30_000,
  });
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("tested.png") });
});

test("an Indexer stops when its site refuses the account and comes back once a Test passes", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await openIndexer(page, world, "Fixture Site");
  await world.fw.cue("fixture-site-refuse", { code: 101 });
  await testButton(page).click();
  await expect(
    page
      .getByRole("alert")
      .getByText(
        "Fixture Site refused the API key with 'Account suspended'. Check the account, then test again.",
      ),
  ).toBeVisible({ timeout: 30_000 });
  await expect(
    page.getByText("Stopped because Fixture Site refused the API key with 'Account suspended'."),
  ).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("stopped.png") });

  await world.fw.cue("fixture-site-refuse", { code: 0 });
  await testButton(page).click();
  await expect(
    page.getByText(/^Fixture Site passed the Test and listed \d+ items?\.$/u),
  ).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText("Stopped because Fixture Site refused the API key")).toBeHidden();
  await page.screenshot({ path: testInfo.outputPath("back.png") });
});

test("Larkspur's Test shows the captcha its sign-in meets, and the next Test answers its code", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await openIndexer(page, world, "Larkspur");
  await world.fw.cue("fixture-site-expire", { site: "larkspur" });
  await testButton(page).click();
  await expect(page.getByRole("img", { name: "The captcha Larkspur shows" })).toBeVisible({
    timeout: 30_000,
  });
  const shown = await world.fw.capture("captchas").list({ where: { site: "larkspur" } });
  const last = shown.at(-1);
  if (last === undefined) {
    throw new Error("Larkspur served no captcha");
  }
  await page.getByRole("textbox", { name: "Code" }).fill(CaptchaShown.parse(last.data).text);
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("captcha.png") });
  await testButton(page).click();
  await expect(page.getByText(/^Larkspur passed the Test and listed \d+ items?\.$/u)).toBeVisible({
    timeout: 30_000,
  });
});

test("moving Cinder, whose cookie was pasted for seedbox, asks for a cookie for home", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await openIndexer(page, world, "Cinder");
  await page.getByRole("button", { name: "Move", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("combobox", { name: "Search from" }).click();
  await page.getByRole("option", { name: "home" }).click();
  await expect(
    dialog.getByText(
      "The browser cookie pasted for Cinder works only from the Node it was pasted for. Paste the cookie of a browser signed in to Cinder on home's network.",
    ),
  ).toBeVisible();
  await dialog.getByRole("button", { name: "Move", exact: true }).click();
  await expect(
    dialog.getByText("Paste the cookie of a browser signed in to Cinder."),
  ).toBeVisible();
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("move-cookie.png") });
});

test("an Admin sets home's Challenge solver, which saves only once home reaches it", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const home = await nodeID(world, "home");
  const solver = Solver.parse(
    (await world.api("GET", `/api/v1/nodes/${await nodeID(world, "seedbox")}/challenge-solver`))
      .body,
  ).address;
  if (solver === null) {
    throw new Error("seedbox has no Challenge solver");
  }
  await page.goto(`/nodes/${home}/searching`);
  await expect(page.getByRole("heading", { name: "Challenge solver" })).toBeVisible();
  const address = page.getByRole("textbox", { name: "Address" });
  await address.fill("http://127.0.0.1:9");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(
    page.getByText(
      "home can't reach a Challenge solver at http://127.0.0.1:9. Check the address and that the solver is running beside home.",
    ),
  ).toBeVisible({ timeout: 30_000 });
  await page.screenshot({ path: testInfo.outputPath("solver-unreachable.png") });

  await address.fill(solver);
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("Saved home's Challenge solver")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole("button", { name: "Remove" })).toBeVisible();
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("solver-saved.png") });
});
