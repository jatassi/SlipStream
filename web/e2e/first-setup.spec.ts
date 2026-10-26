import type { BrowserContext, Page } from "@playwright/test";
import { z } from "zod";

import { addVirtualAuthenticator } from "./authenticator.ts";
import { type World, expect, test as base } from "./fixtures.ts";

// The project's preset sizes the context, so the locale joins it rather than
// replacing it.
const test = base.extend({
  contextOptions: async ({ contextOptions }, use) => {
    await use({ ...contextOptions, locale: "en-GB" });
  },
});

test.use({ scenario: "first-run" });

// The claim answers with the session cookie, so it runs in the page, where
// the browser reaches the Node's name.
async function claim(page: Page, world: World): Promise<void> {
  await page.goto("/setup");
  const status = await page.evaluate(
    async (setupCode) => {
      const response = await fetch("/api/v1/setup/controller", {
        method: "POST",
        headers: { "X-Setup-Code": setupCode, "Content-Type": "application/json" },
        body: JSON.stringify({
          email: "grace@example.test",
          displayName: "Grace Hopper",
          pin: "1357",
        }),
      });
      return response.status;
    },
    await world.setupCode(),
  );
  expect(status).toBe(201);
}

const read = (page: Page, path: `/api/v1/${string}`) =>
  page.evaluate(async (url) => (await fetch(url)).json() as Promise<unknown>, path);

const MetadataSettings = z.object({
  settings: z.array(z.object({ key: z.string(), value: z.unknown() })),
});

async function metadataValues(page: Page): Promise<Record<string, unknown>> {
  const { settings } = MetadataSettings.parse(await read(page, "/api/v1/metadata/settings"));
  return Object.fromEntries(settings.map((setting) => [setting.key, setting.value]));
}

const heading = (page: Page, name: string) => page.getByRole("heading", { name, level: 1 });

async function addressPasskeyAndLanguage(
  page: Page,
  context: BrowserContext,
  world: World,
): Promise<void> {
  await expect(heading(page, "Where people reach SlipStream")).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/setup/address`);
  await expect(page.getByText("Step 1 of 6")).toBeVisible();
  await expect(
    page.getByText(
      "After this: Passkey, Language and region, Library roots, Downloads and What SlipStream grabs",
    ),
  ).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Address" })).toHaveValue(world.baseURL);
  await page.getByRole("button", { name: "Continue" }).click();

  await expect(heading(page, "Add a passkey")).toBeVisible();
  await addVirtualAuthenticator(context, page);
  await page.getByRole("button", { name: "Add a passkey" }).click();

  await expect(heading(page, "Language and region")).toBeVisible();
  await expect(page.getByRole("combobox", { name: "Metadata language" })).toHaveText(
    "English (United Kingdom)",
  );
  await expect(page.getByRole("combobox", { name: "Metadata region" })).toHaveText(
    "United Kingdom",
  );
  await page.getByRole("button", { name: "Continue" }).click();
}

// A reload and an app URL both come back to the step the Admin is on.
async function rootsAfterReload(page: Page, world: World): Promise<void> {
  await expect(heading(page, "Library roots")).toBeVisible();
  await page.reload();
  await expect(heading(page, "Library roots")).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/setup/roots`);
  await page.goto("/settings/address");
  await expect(heading(page, "Library roots")).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/setup/roots`);
  await expect(
    page.getByRole("list", { name: "Library roots" }).getByRole("listitem"),
  ).toContainText(["MoviesNo folder yet"]);
  await page.getByRole("button", { name: "Skip" }).click();
}

async function downloadsAndTargets(page: Page): Promise<void> {
  await expect(heading(page, "Downloads")).toBeVisible();
  await expect(
    page.getByRole("list", { name: "Downloading Nodes" }).getByRole("listitem", { name: "home" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Continue" }).click();

  await expect(heading(page, "What SlipStream grabs")).toBeVisible();
  await expect(page.getByRole("listitem", { name: "Movies" }).getByText(/^Grabs /u)).toBeVisible();
  await page.getByRole("button", { name: "Continue" }).click();
}

async function doneScreen(page: Page, world: World): Promise<void> {
  await expect(heading(page, "First setup")).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/setup/done`);
  await expect(page.getByRole("list", { name: "First setup" }).getByRole("listitem")).toHaveText([
    "AddressSet",
    "PasskeySet",
    "Language and regionSet",
    "Library rootsSet it in SettingsSkipped",
    "DownloadsSet",
    "What SlipStream grabsSet",
  ]);
}

test("a new Admin walks first setup step by step, resumes it after a reload and lands in the Library @cdp", async ({
  page,
  context,
  world,
}) => {
  await claim(page, world);
  await page.goto("/");
  await addressPasskeyAndLanguage(page, context, world);
  await rootsAfterReload(page, world);
  await downloadsAndTargets(page);
  await doneScreen(page, world);
  expect(await metadataValues(page)).toMatchObject({
    "metadata.language": "en-GB",
    "metadata.region": "GB",
  });

  await page.getByRole("button", { name: "Go to Library" }).click();
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/library`);
  await page.goto("/setup/roots");
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/library`);
  expect(await read(page, "/api/v1/first-run")).toMatchObject({ open: false });
});

test("the passkey step shows only where the browser can hold a passkey", async ({
  page,
  world,
  browserName,
}) => {
  await claim(page, world);
  await page.goto("/setup/passkey");
  if (browserName === "chromium") {
    await expect(heading(page, "Add a passkey")).toBeVisible();
    await expect(page.getByText("Step 2 of 6")).toBeVisible();
  } else {
    await expect(heading(page, "Where people reach SlipStream")).toBeVisible();
    expect(page.url()).toBe(`${world.baseURL}/setup/address`);
    await expect(page.getByText("Step 1 of 5")).toBeVisible();
  }
});
