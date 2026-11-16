import type { Page } from "@playwright/test";
import { z } from "zod";

import { expectColumns, orchardCells } from "./cells.ts";
import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, openSettingsSection } from "./shell.ts";
import { clickWhenStill } from "./still.ts";
import { screenshotWholePage } from "./whole-page.ts";

const Made = z.object({ id: z.string() });

// A second Target beside the starter 1080p one, made the way a script would.
async function make2160p(world: World): Promise<string> {
  const reply = await world.api("POST", "/api/v1/targets", {
    name: "2160p",
    tiers: [["remux-2160p", "bluray-2160p"], ["webdl-2160p"]],
    cutoff: "remux-2160p",
  });
  expect(reply.status, "the 2160p Target is created").toBe(201);
  return Made.parse(reply.body).id;
}

async function openProfiles(page: Page) {
  await openSettingsSection(page, "Profiles");
  await expect(page).toHaveURL(/\/settings\/profiles$/u);
}

async function pick(page: Page, button: string, item: string) {
  await page.getByRole("button", { name: button }).click();
  await clickWhenStill(
    page.getByRole(isPhone(page) ? "button" : "menuitem", { name: item, exact: true }),
  );
}

// On phone the table stacks each row as a card, labels inline, so a row is read by its parts.
const profileRow = (page: Page, name: string) =>
  page
    .getByRole("table", { name: "Movies Profiles" })
    .getByRole("row")
    .filter({ has: page.getByRole("link", { name, exact: true }) });

const ranked = (page: Page, list: string) =>
  page.getByRole("list", { name: list, exact: true }).getByRole("listitem");

async function chooseCutsWithoutStandard(page: Page) {
  const sentence = page.getByText(/^SlipStream (keeps|always|searches) /u);
  await page.getByRole("radio", { name: "Choose cuts" }).click();
  await expect(sentence).toHaveText("SlipStream always gets Standard.");
  await pick(page, "Add cut", "Extended");
  await expect(sentence).toHaveText(
    "SlipStream always gets Standard. It also gets Extended once a release or file of that cut turns up.",
  );
  await page.getByRole("button", { name: "Remove Standard" }).click();
  await expect(page.getByText("No Standard")).toBeVisible();
  await expect(sentence).toHaveText(
    "SlipStream searches for Extended every time, even if no release of it has turned up. It skips releases that name no cut, because they count as Standard.",
  );
}

test("a Profile takes two Targets reordered by keyboard, and warns once its cuts drop Standard", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await make2160p(world);
  await openProfiles(page);
  const movies = profileRow(page, "Movies");
  await expect(movies).toContainText("Default");
  await expect(movies).toContainText("1080p");
  await page.getByRole("button", { name: "New Profile" }).click();
  await expect(page).toHaveURL(/\/settings\/profiles\/new$/u);
  await page.getByRole("textbox", { name: "Name" }).fill("Both");
  await pick(page, "Add Target", "1080p");
  await pick(page, "Add Target", "2160p");
  await expect(ranked(page, "Targets")).toHaveText([
    "Rank 11080pNo estimate yet",
    "Rank 22160pNo estimate yet",
  ]);

  await page.getByRole("button", { name: "Move 2160p", exact: true }).focus();
  await page.keyboard.press("ArrowUp");
  await expect(ranked(page, "Targets")).toHaveText([
    "Rank 12160pNo estimate yet",
    "Rank 21080pNo estimate yet",
  ]);
  await expect(page.getByRole("button", { name: "Move 2160p", exact: true })).toBeFocused();

  await chooseCutsWithoutStandard(page);
  // The phone's glass tab bar over the amber line would read as low contrast, so the line sits mid-screen.
  await page.getByText("No Standard").evaluate((chip) => {
    chip.scrollIntoView({ block: "center" });
  });
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("editor.png"));

  await page.getByRole("radio", { name: "Any cut" }).click();
  await page.getByRole("button", { name: "Create Profile" }).click();
  await expect(page).toHaveURL(/\/settings\/profiles$/u);
  const both = profileRow(page, "Both");
  await expect(both).toContainText("2160p, 1080p");
  await expect(both).not.toContainText("Default");
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("list.png"));
});

test("an Admin adds a movie on a Profile other than the default and sees its Targets wanted", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const uhd = await make2160p(world);
  const reply = await world.api("POST", "/api/v1/profiles", {
    name: "Both",
    contentType: "movie",
    targets: [{ targetId: uhd, sourcing: "download" }],
    editions: { kind: "any" },
  });
  expect(reply.status, "the Both Profile is created").toBe(201);

  await page.goto("/add?contentType=movie&query=quiet+orchard");
  await page.getByRole("button", { name: "Add The Quiet Orchard" }).click();
  const sheet = page.getByRole("dialog", { name: "Add The Quiet Orchard" });
  const picker = sheet.getByRole("combobox", { name: "Profile" });
  await expect(picker).toHaveText("Movies (default)");
  await picker.click();
  await page.getByRole("option", { name: "Both" }).click();
  await expect(picker).toHaveText("Both");
  await expect(sheet.getByRole("switch", { name: "Monitor" })).toBeChecked();
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("add-sheet.png") });
  await sheet.getByRole("button", { name: "Add Movie" }).click();

  await expect(page.getByRole("heading", { name: "The Quiet Orchard", exact: true })).toBeVisible();
  await expect(orchardCells(page)).toContainText("From Profile Both");
  await expect(orchardCells(page)).toContainText("Missing");
  await expectColumns(page, ["2160p"]);
  const workId = /\/works\/([^/]+)$/u.exec(page.url())?.[1];
  const wants = await world.api("GET", `/api/v1/works/${String(workId)}/wants`);
  expect(wants.body).toMatchObject({
    profileName: "Both",
    from: "profile",
    editions: { kind: "any" },
  });
});
