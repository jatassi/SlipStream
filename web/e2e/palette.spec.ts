import type { Page } from "@playwright/test";
import { z } from "zod";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

const Candidates = z.object({ candidates: z.array(z.object({ key: z.string() })).min(1) });
const Added = z.object({ work: z.object({ id: z.string() }) });

async function openPalette(page: Page): Promise<void> {
  if (isPhone(page)) {
    await page
      .getByRole("navigation", { name: "Tabs" })
      .getByRole("button", { name: "Search" })
      .click();
    return;
  }
  await expect(page.getByRole("navigation", { name: "Main" })).toBeVisible();
  await page.keyboard.press("ControlOrMeta+k");
}

test("the palette opens a Work by its name", async ({ adminPage: page, world }) => {
  const found = await world.api("GET", "/api/v1/content-types/movie/catalog?query=quiet+orchard");
  const [candidate] = Candidates.parse(found.body).candidates;
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: candidate?.key,
    monitor: true,
  });
  const { id } = Added.parse(added.body).work;

  await openPalette(page);
  const palette = page.getByRole("dialog", { name: "Search Works or run a command" });
  await expect(palette.getByRole("combobox")).toBeFocused();

  await page.keyboard.type("quiet orch");
  await expect(palette.getByRole("group", { name: "Works" }).getByRole("option")).toHaveText([
    "The Quiet Orchard2021 · Movie · Missing",
  ]);
  await expectAccessible(page);

  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(new RegExp(`/works/${id}$`, "u"));
  await expect(page.getByRole("heading", { name: "The Quiet Orchard", exact: true })).toBeVisible();
  await expect(palette).toHaveCount(0);
});

// Esc starts the close animation, and the palette reopens before it ends.
async function closeAndReopen(page: Page): Promise<void> {
  const palette = page.getByRole("dialog", { name: "Search Works or run a command" });
  await openPalette(page);
  await expect(palette.getByRole("combobox")).toBeFocused();
  await page.keyboard.press("Escape");
  await openPalette(page);
  await page.keyboard.type("go to act");
  await expect(palette.getByRole("combobox")).toHaveValue("go to act");
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/activity$/u);
  await expect(palette).toHaveCount(0);
}

test("the palette reopens mid-close and keeps every key typed", async ({ adminPage: page }) => {
  for (let attempt = 0; attempt < 10; attempt += 1) {
    // oxlint-disable-next-line no-await-in-loop -- each attempt starts from the last one's page
    await closeAndReopen(page);
  }
});
