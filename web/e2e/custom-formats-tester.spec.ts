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
    "SlipStream reads this as WEBDL-1080p from FKW. It matches The Quiet Orchard, and an automatic search would grab it for 1080p and HD.",
    "SlipStream reads this as WEBDL-720p from GRP. It matches nothing in your library. 1080p and HD accept it.",
    "SlipStream reads this as Bluray-2160p from FKW. It matches The Quiet Orchard, but no Target would grab it. 1080p doesn't accept Bluray-2160p.",
  ]);
  const first = results.first();
  await expect(first).toContainText(webdl);
  await first.getByRole("button", { name: "Details" }).click();
  await expect(first.getByRole("definition")).toHaveText([
    "The Quiet Orchard",
    "2021",
    "WEBDL-1080p",
    "WEB-DL",
    "1080p",
    "FKW",
    "x264",
  ]);
  await expect(first.getByRole("list", { name: "Targets" }).locator(":scope > li")).toHaveText([
    /^1080pScore 0Would grab/u,
    /^HDScore 0Would grab/u,
  ]);
  await expect(first).toContainText("No Custom Formats yet.");

  const third = results.nth(2);
  await third.getByRole("button", { name: "Details" }).click();
  await expect(third.getByRole("list", { name: "Targets" })).toContainText(
    "1080pScore 0Wouldn't grab1080p doesn't accept Bluray-2160p.",
  );
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("tester.png"));
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
    "SlipStream reads this as WEBDL-1080p from FKW. It matches The Quiet Orchard, and an automatic search would grab it for 1080p and HD.",
  );
  await result.getByRole("button", { name: "Details" }).click();
  await expect(result.getByRole("list", { name: "Custom Formats it matches" })).toHaveText(
    /^GoldenDraftMatchesGoldenPassesGildedFails$/u,
  );
  await expect(result.getByRole("list", { name: "Targets" }).locator(":scope > li")).toHaveText([
    /^1080pScore 25Would grab/u,
    /^HDScore 25Would grab/u,
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
  ).toHaveText([/^1080pScore 0Would grab/u, /^HDScore 0Would grab/u]);
});
