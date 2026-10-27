import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";

test.use({ scenario: "catalog-sites" });
test.describe.configure({ timeout: 180_000 });

const Indexers = z.object({
  indexers: z.array(z.object({ name: z.string(), kind: z.string(), sentence: z.string() })),
});
const CaptchaShown = z.object({ text: z.string() });

const saved = async (world: World) =>
  Indexers.parse((await world.api("GET", "/api/v1/indexers")).body).indexers;

const field = (page: Page, name: string) => page.getByRole("textbox", { name, exact: true });

test("an Admin adds a Torznab feed, testing it first and naming the Tracker its items don't", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const torznab = await world.fw.site("torznab");
  if (torznab.feed === undefined || torznab.apiKey === undefined) {
    throw new Error("fakeworld names no Torznab feed");
  }
  await page.goto("/settings/indexers/new");
  await expect(page.getByRole("option", { name: /^Quarry/u })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("picker.png") });
  await page.getByRole("link", { name: /^Torznab feed/u }).click();
  await expect(page).toHaveURL(/\/settings\/indexers\/new\/torznab$/u);

  await field(page, "Name").fill("Fixture feed");
  await field(page, "Feed address").fill(torznab.feed);
  await page.getByLabel("API key", { exact: true }).fill("wrong");
  await page.getByRole("button", { name: "Test" }).click();
  await expect(
    page
      .getByRole("alert")
      .getByText(
        "Fixture feed refused the API key with 'Invalid API Key'. Check the account, then test again.",
      ),
  ).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("test-failure.png") });

  await page.getByLabel("API key", { exact: true }).fill(torznab.apiKey);
  await page.getByRole("button", { name: "Test" }).click();
  await expect(page.getByText(/^The feed passed the Test and listed \d+ items?\.$/u)).toBeVisible();
  await expect(field(page, "Tracker")).toHaveValue("Fixture feed");
  await field(page, "Tracker").fill("Fixture Tracker");
  await expectAccessible(page);
  await page.getByRole("button", { name: "Save" }).click();

  await expect(
    page.getByRole("heading", { name: "Fixture feed", exact: true, level: 1 }),
  ).toBeVisible();
  expect(await saved(world)).toContainEqual(
    expect.objectContaining({ name: "Fixture feed", kind: "torznab" }),
  );
});

// Larkspur's form, picked from the catalog with the keyboard.
async function pickLarkspur(page: Page) {
  await page.goto("/settings/indexers/new");
  const search = page.getByRole("combobox", { name: "Search the Indexer catalog" });
  await search.fill("larkspur");
  await search.press("ArrowDown");
  await search.press("Enter");
  await expect(page.getByRole("heading", { name: "Add Larkspur", level: 1 })).toBeVisible();
}

test("an Admin adds Larkspur from the catalog, typing the code its captcha shows", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const larkspur = await world.fw.site("larkspur");
  const password = larkspur.settings?.["password"];
  if (password === undefined) {
    throw new Error("fakeworld names no password for Larkspur");
  }
  await pickLarkspur(page);
  await expect(page.getByText(/^Larkspur is a private Tracker\./u)).toBeVisible();

  await field(page, "Username").fill(larkspur.settings?.["username"] ?? "");
  await page.getByLabel("Password", { exact: true }).fill(`${password}-wrong`);
  await page.screenshot({ path: testInfo.outputPath("definition-form.png") });
  await page.getByRole("button", { name: "Continue" }).click();

  const captchas = world.fw.capture("captchas");
  const first = await captchas.wait({ where: { site: "larkspur" }, timeout: 30_000 });
  const image = page.getByRole("img", { name: "The captcha Larkspur shows" });
  await expect(image).toBeVisible();
  await page.getByRole("textbox", { name: "Code" }).fill(CaptchaShown.parse(first.data).text);
  await page.screenshot({ path: testInfo.outputPath("captcha.png") });
  await page.getByRole("button", { name: "Save" }).click();

  await expect(
    page
      .getByRole("alert")
      .getByText(
        /^Larkspur refused the sign-in with '.+'\. Check the username, password and any code, then test again\.$/u,
      ),
  ).toBeVisible({ timeout: 30_000 });
  const second = await captchas.wait({
    where: { site: "larkspur" },
    after: first.seq,
    timeout: 30_000,
  });
  await expect(page.getByRole("textbox", { name: "Code" })).toHaveValue("");
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("textbox", { name: "Code" }).fill(CaptchaShown.parse(second.data).text);
  await expectAccessible(page);
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page.getByRole("heading", { name: "Larkspur", exact: true, level: 1 })).toBeVisible({
    timeout: 30_000,
  });
  expect(await saved(world)).toContainEqual(
    expect.objectContaining({ name: "Larkspur", kind: "catalog" }),
  );
});
