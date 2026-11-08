import type { Locator, Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { openSettingsSection } from "./shell.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "targets" });

const Named = z.object({ id: z.string(), name: z.string() });
const Target = Named.extend({
  tiers: z.array(z.array(z.string()).nullable()),
  cutoff: z.string(),
  filters: z.array(z.unknown()),
  sizes: z.array(z.unknown()),
  propers: z.string(),
});

const webdl = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW";
const uhd = "The.Quiet.Orchard.2021.2160p.BluRay.x265-FKW";
const heat = "Heat.1995.720p.WEB-DL-GRP";

// The Quiet Orchard, added the way a script would, with nothing held.
async function added(world: World): Promise<void> {
  const reply = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    preset: "all",
  });
  expect(reply.status, "The Quiet Orchard is added").toBe(200);
}

const titleCondition = (name: string, pattern: string) => ({
  name,
  kind: "ReleaseTitleSpecification",
  negate: false,
  required: false,
  fields: { value: pattern },
});

// A format that every Target scores 25, whose patterns match none of the names here.
async function scoredGolden(world: World): Promise<string> {
  const made = await world.api("POST", "/api/v1/custom-formats", {
    name: "Golden",
    conditions: [titleCondition("Golden", "GOLDEN"), titleCondition("Gilded", "GILDED")],
  });
  expect(made.status, "Golden is made").toBe(201);
  const formatId = Named.parse(made.body).id;
  const { targets } = z
    .object({ targets: z.array(Target) })
    .parse((await world.api("GET", "/api/v1/targets")).body);
  for (const { id, name, tiers, cutoff, filters, sizes, propers } of targets) {
    // oxlint-disable-next-line no-await-in-loop -- each Target's change converges its Works before the next
    const scored = await world.api("PUT", `/api/v1/targets/${id}`, {
      name,
      tiers,
      cutoff,
      filters,
      sizes,
      propers,
      scores: { minimum: 0, upgradeUntil: 0, increment: 1, formats: [{ formatId, score: 25 }] },
    });
    expect(scored.status, `${name} scores Golden`).toBe(200);
  }
  return formatId;
}

const pattern = (page: Page, index: number) =>
  page
    .getByRole("list", { name: "Conditions" })
    .getByRole("listitem", { name: `Condition ${String(index)}` })
    .getByRole("textbox", { name: "Pattern" });

const resultsIn = (scope: Page | Locator) =>
  scope.getByRole("list", { name: "Results" }).locator(":scope > li");

async function testNames(scope: Page | Locator, label: string, names: readonly string[]) {
  await scope.getByRole("textbox", { name: label }).fill(names.join("\n"));
  await scope.getByRole("button", { name: "Test", exact: true }).click();
}

test("the tester reads each pasted name, matches it to a Work by its title and says whether a search would grab it", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await added(world);
  await openSettingsSection(page, "Custom Formats");
  await testNames(page, "Release names", [webdl, heat, uhd]);

  const results = resultsIn(page);
  await expect(results.locator(":scope > div > p")).toHaveText([
    "Reads as WEBDL-1080p from FKW. It matches The Quiet Orchard and would fill 1080p and HD.",
    "Reads as WEBDL-720p from GRP. It matches no Work. 1080p and HD accept it.",
    "Reads as Bluray-2160p from FKW. It matches The Quiet Orchard and would fill no Target. 1080p doesn't accept Bluray-2160p.",
  ]);
  const first = results.first();
  await expect(first).toContainText(webdl);
  await first.getByRole("button", { name: "Details" }).click();
  await expect(first.getByRole("definition")).toHaveText([
    "The Quiet Orchard",
    "2021",
    "webdl-1080p",
    "webdl",
    "1080p",
    "FKW",
    "x264",
  ]);
  await expect(first.getByRole("list", { name: "Targets" }).locator(":scope > li")).toHaveText([
    /^1080pScore 0Would fill/u,
    /^HDScore 0Would fill/u,
  ]);
  await expect(first).toContainText("No Custom Formats yet.");

  const third = results.nth(2);
  await third.getByRole("button", { name: "Details" }).click();
  await expect(third.getByRole("list", { name: "Targets" })).toContainText(
    "1080pScore 0Wouldn't fill1080p doesn't accept Bluray-2160p.",
  );
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("tester.png"));
});

test("the tester reads a file path with its folder's name when the file's gives no title", async ({
  adminPage: page,
  world,
}) => {
  await added(world);
  await openSettingsSection(page, "Custom Formats");
  await page.getByRole("radio", { name: "File paths" }).click();
  await testNames(page, "File paths", [
    String.raw`D:\Films\The Quiet Orchard (2021)\1080p.BluRay-FKW.mkv`,
  ]);
  await expect(resultsIn(page).locator(":scope > div > p")).toHaveText([
    "Reads as Bluray-1080p from FKW. It matches The Quiet Orchard and would fill 1080p and HD.",
  ]);
  await expectAccessible(page);
});

test("Try it scores the names with the editor's draft before it is saved", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await added(world);
  const formatId = await scoredGolden(world);
  await page.goto(`/settings/custom-formats/${formatId}`);
  await pattern(page, 1).fill(String.raw`\bFKW\b`);
  await page.getByRole("button", { name: "Try it" }).click();

  const dialog = page.getByRole("dialog", { name: "Tester" });
  await expect(
    dialog.getByText("Golden is matched as it stands in the editor, before it's saved."),
  ).toBeVisible();
  await testNames(dialog, "Release names", [webdl]);
  const result = resultsIn(dialog).first();
  await expect(result.locator(":scope > div > p")).toHaveText(
    "Reads as WEBDL-1080p from FKW. It matches The Quiet Orchard and would fill 1080p and HD.",
  );
  await result.getByRole("button", { name: "Details" }).click();
  await expect(result.getByRole("list", { name: "Custom Formats it matches" })).toHaveText(
    /^GoldenDraftMatchesGoldenPassesGildedFails$/u,
  );
  await expect(result.getByRole("list", { name: "Targets" }).locator(":scope > li")).toHaveText([
    /^1080pScore 25Would fill/u,
    /^HDScore 25Would fill/u,
  ]);
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("try-it.png"));
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();

  const saved = z
    .object({ conditions: z.array(z.object({ fields: z.object({ value: z.string() }) })) })
    .parse((await world.api("GET", `/api/v1/custom-formats/${formatId}`)).body);
  expect(saved.conditions[0]?.fields.value, "the draft isn't saved").toBe("GOLDEN");

  await openSettingsSection(page, "Custom Formats");
  await testNames(page, "Release names", [webdl]);
  await resultsIn(page).first().getByRole("button", { name: "Details" }).click();
  await expect(
    resultsIn(page).first().getByRole("list", { name: "Targets" }).locator(":scope > li"),
  ).toHaveText([/^1080pScore 0Would fill/u, /^HDScore 0Would fill/u]);
});

test("Try it with a pattern that doesn't compile says so in the tester and under the pattern", async ({
  adminPage: page,
}) => {
  await openSettingsSection(page, "Custom Formats");
  await page.getByRole("button", { name: "New Custom Format" }).click();
  await page.getByRole("textbox", { name: "Name", exact: true }).fill("Dolby Vision");
  // A second condition keeps the phone's glass tab bar off the white Save button, which axe reads as low contrast.
  await page.getByRole("button", { name: "Add condition" }).click();
  await pattern(page, 1).fill(String.raw`\b(DV`);
  await pattern(page, 2).fill(String.raw`\bDoVi\b`);
  await page.getByRole("button", { name: "Try it" }).click();

  const dialog = page.getByRole("dialog", { name: "Tester" });
  await testNames(dialog, "Release names", [webdl]);
  const refusal = String.raw`The pattern in "\b(DV" doesn't compile (unterminated group). Fix the pattern and try it again.`;
  await expect(dialog.getByRole("alert")).toHaveText(refusal);
  await expectAccessible(page);
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(pattern(page, 1)).toHaveAccessibleDescription(refusal);
});
