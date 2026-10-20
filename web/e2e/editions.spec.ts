import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "targets" });
test.describe.configure({ timeout: 120_000 });

const superfan = "The.Quiet.Orchard.2021.Superfan.1080p.WEB-DL.H264-FKW";

const Named = z.object({ id: z.string(), name: z.string() });
const Profiles = z.object({ profiles: z.array(Named) });
const Added = z.object({ work: z.object({ id: z.string() }) });
const Cells = z.object({ cells: z.array(z.object({ edition: z.string(), state: z.string() })) });

async function openEditions(page: Page) {
  await navigate(page, "Settings");
  const sections = isPhone(page)
    ? page.getByRole("list", { name: "Settings" })
    : page.getByRole("navigation", { name: "Settings" });
  await sections.getByRole("link", { name: "Editions" }).click();
  await expect(page).toHaveURL(/\/settings\/editions$/u);
}

// On phone the table stacks each row as a card, so a row is found by its text.
const cutRow = (page: Page, name: string) =>
  page.getByRole("table", { name: "Cuts" }).getByRole("row").filter({ hasText: name });

async function pick(page: Page, button: string, item: string) {
  await page.getByRole("button", { name: button }).click();
  await clickWhenStill(
    page.getByRole(isPhone(page) ? "button" : "menuitem", { name: item, exact: true }),
  );
}

async function filmsId(world: World) {
  const { profiles } = Profiles.parse((await world.api("GET", "/api/v1/profiles")).body);
  const films = profiles.find((profile) => profile.name === "Films");
  if (films === undefined) {
    throw new Error("the targets scenario has no Films Profile");
  }
  return films.id;
}

// The Quiet Orchard, added and searched the way a script would, once fixture-site lists its Superfan cut.
async function importedSuperfan(world: World): Promise<string> {
  await world.fw.cue("fixture-site-list", { release: superfan });
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
        Cells.parse((await world.api("GET", `/api/v1/works/${workId}/cells`)).body).cells.map(
          (cell) => `${cell.edition} ${cell.state}`,
        ),
      { message: "the Superfan WEB-DL fills both Targets", timeout: 60_000 },
    )
    .toEqual(["superfan upgradable", "superfan upgradable"]);
  return workId;
}

async function addSuperfan(page: Page) {
  await page.getByRole("button", { name: "Add cut" }).click();
  const adding = page.getByRole("dialog", { name: "Add a cut" });
  await adding.getByRole("textbox", { name: "Name" }).fill("Superfan");
  await adding.getByRole("textbox", { name: "Also read as" }).fill("Fan Cut");
  await adding.getByRole("button", { name: "Add cut" }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Added Superfan.",
  );
  await expect(adding).toBeHidden();
  await expect(cutRow(page, "Superfan")).toContainText("Superfan, Fan Cut");
  await expect(cutRow(page, "Superfan")).toContainText("Custom");
}

// Films lists Superfan alone, so it wants that cut outright.
async function filmsWantSuperfan(page: Page, world: World) {
  const films = await filmsId(world);
  await page.goto(`/settings/profiles/${films}`);
  await page.getByRole("radio", { name: "Choose cuts" }).click();
  await pick(page, "Add cut", "Superfan");
  await page.getByRole("button", { name: "Remove Standard" }).click();
  await expect(page.getByText(/^Every Item wants Superfan outright/u)).toBeVisible();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText("Saved Films.");
  const saved = await world.api("GET", `/api/v1/profiles/${films}`);
  expect(saved.body).toMatchObject({ editions: { kind: "cuts", cuts: ["superfan"] } });
}

test("an Admin adds a custom cut, lists it in a Profile, and renames it and its file through the Impact", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await openEditions(page);
  await expect(cutRow(page, "Standard")).toContainText("Any title that names no cut");
  await expect(cutRow(page, "Extended")).toContainText("Built in");
  await addSuperfan(page);
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("editions.png"), fullPage: true });

  await filmsWantSuperfan(page, world);
  const workId = await importedSuperfan(world);
  await page.goto(`/works/${workId}`);
  const files = page.getByRole("list", { name: "Files" });
  await expect(files).toContainText("The Quiet Orchard (2021) - {edition-Superfan} 1080p.mkv");

  await openEditions(page);
  await page.getByRole("button", { name: "Actions for Superfan" }).click();
  await clickWhenStill(page.getByRole(isPhone(page) ? "button" : "menuitem", { name: "Edit" }));
  const editing = page.getByRole("dialog", { name: "Edit Superfan" });
  await editing.getByRole("textbox", { name: "Name" }).fill("Superfan Edition");
  await editing.getByRole("button", { name: "Save" }).click();
  const impact = page.getByRole("dialog", { name: "Save Superfan" });
  await expect(impact.getByRole("list", { name: "What changes" }).getByRole("listitem")).toHaveText(
    ["1 Work changes.", "1 file is renamed."],
  );
  await expect(impact.getByRole("radiogroup")).toHaveCount(0);
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("rename-impact.png") });
  await impact.getByRole("button", { name: "Save", exact: true }).click();

  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Saved Superfan Edition.",
  );
  await expect(editing).toBeHidden();
  await expect(cutRow(page, "Superfan Edition")).toContainText("Superfan Edition, Fan Cut");
  await page.goto(`/works/${workId}`);
  await expect(files).toContainText(
    "The Quiet Orchard (2021) - {edition-Superfan Edition} 1080p.mkv",
  );
});
