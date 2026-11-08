import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, openSettingsSection } from "./shell.ts";
import { clickWhenStill } from "./still.ts";
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

async function openFormats(page: Page) {
  await openSettingsSection(page, "Custom Formats");
  await expect(page).toHaveURL(/\/settings\/custom-formats$/u);
}

const formatRow = (page: Page, name: string) =>
  page
    .getByRole("table", { name: "Custom Formats" })
    .getByRole("row")
    .filter({ has: page.getByRole("link", { name, exact: true }) });

const condition = (page: Page, index: number) =>
  page
    .getByRole("list", { name: "Conditions" })
    .getByRole("listitem", { name: `Condition ${String(index)}` });

async function pick(
  page: Page,
  scope: ReturnType<typeof condition>,
  field: string,
  option: string,
) {
  await scope.getByRole("combobox", { name: field }).click();
  await clickWhenStill(page.getByRole("option", { name: option, exact: true }));
  await expect(scope.getByRole("combobox", { name: field })).toHaveText(option);
}

async function made(world: World, name: string, pattern: string): Promise<string> {
  const reply = await world.api("POST", "/api/v1/custom-formats", {
    name,
    conditions: [
      {
        name,
        kind: "ReleaseTitleSpecification",
        negate: false,
        required: false,
        fields: { value: pattern },
      },
    ],
  });
  expect(reply.status, `${name} is made`).toBe(201);
  return Named.parse(reply.body).id;
}

async function draftDolbyVision(page: Page) {
  await page.getByRole("textbox", { name: "Name", exact: true }).fill("Dolby Vision");
  const first = condition(page, 1);
  await first.getByRole("textbox", { name: "Pattern" }).fill(String.raw`\b(DV`);
  await page.getByRole("button", { name: "Add condition" }).click();
  const second = condition(page, 2);
  await pick(page, second, "Kind", "Source");
  await pick(page, second, "Source", "WEB-DL");
  const sentence = page.getByText(/^Matches a Release when/u);
  await expect(sentence).toHaveText(
    String.raw`Matches a Release when its release title matches "\b(DV", and its source is WEB-DL.`,
  );
  return { first, second, sentence };
}

test("a Custom Format made in the editor says what it matches, refuses a broken pattern under its field, and lands in the list", async ({
  adminPage: page,
}, testInfo) => {
  await openFormats(page);
  await expect(page.getByText("No Custom Formats yet.")).toBeVisible();
  await page.getByRole("button", { name: "New Custom Format" }).click();
  await expect(page).toHaveURL(/\/settings\/custom-formats\/new$/u);

  const { first, second, sentence } = await draftDolbyVision(page);

  await page.getByRole("button", { name: "Create Custom Format" }).click();
  await expect(first.getByRole("textbox", { name: "Pattern" })).toHaveAccessibleDescription(
    /^The pattern in ".+" doesn't compile \(unterminated group\)\. Fix the pattern and save again\.$/u,
  );
  await first.getByRole("textbox", { name: "Pattern" }).fill(String.raw`\bDV\b`);
  await expect(first.getByRole("textbox", { name: "Pattern" })).toHaveAccessibleDescription(
    "A .NET regular expression, matched ignoring case.",
  );
  await second.getByRole("button", { name: "Details" }).click();
  await second.getByRole("checkbox", { name: "Negate" }).click();
  await expect(sentence).toHaveText(
    String.raw`Matches a Release when its release title matches "\bDV\b", and its source isn't WEB-DL.`,
  );
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("editor.png"));

  await page.getByRole("button", { name: "Create Custom Format" }).click();
  await expect(page).toHaveURL(/\/settings\/custom-formats$/u);
  const row = formatRow(page, "Dolby Vision");
  await expect(row).toContainText(
    String.raw`Matches a Release when its release title matches "\bDV\b", and its source isn't WEB-DL.`,
  );
  await expect(row).toContainText("None");

  await row.getByRole("link", { name: "Dolby Vision" }).click();
  await expect(page).toHaveURL(/\/settings\/custom-formats\/[^/]+$/u);
  await page.getByRole("textbox", { name: "Name", exact: true }).fill("DV");
  await condition(page, 2)
    .getByRole("button", { name: /^Remove/u })
    .click();
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page).toHaveURL(/\/settings\/custom-formats$/u);
  await expect(formatRow(page, "DV")).toContainText(
    String.raw`Matches a Release when its release title matches "\bDV\b".`,
  );
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("list.png"));
});

const pastedText = JSON.stringify([
  {
    name: "x265",
    includeCustomFormatWhenRenaming: false,
    specifications: [
      {
        name: "x265",
        implementation: "ReleaseTitleSpecification",
        negate: false,
        required: false,
        fields: { value: String.raw`[xh][ .]?265|\bHEVC(\b|\d)` },
      },
    ],
  },
  {
    name: "Remux Tier 01",
    trash_id: "remux-tier-01",
    specifications: [
      {
        name: "FraMeSToR",
        implementation: "ReleaseGroupSpecification",
        negate: false,
        required: false,
        fields: { value: "^(FraMeSToR)$" },
      },
    ],
  },
]);

test("a paste asks which app it was written for, then keeps or replaces a format of the same name", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await made(world, "x265", "x265");
  await openFormats(page);
  await page.getByRole("button", { name: "Paste", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Paste Custom Formats" });
  await dialog.getByRole("textbox", { name: "JSON" }).fill(pastedText);
  await dialog.getByRole("button", { name: "Preview" }).click();
  await expect(
    dialog.getByText("Say whether this was written for Radarr or Sonarr."),
  ).toBeVisible();
  await dialog.getByRole("radio", { name: "Radarr" }).click();

  const adds = dialog.getByRole("list", { name: "What the paste adds" });
  await expect(adds.getByRole("listitem")).toHaveText([
    /^x265Adds it as x265 \(2\), beside the x265 you have\./u,
    /^Remux Tier 01Adds it\.$/u,
  ]);
  await expect(dialog.getByText("Read as Radarr formats.")).toBeVisible();
  await expect(dialog.getByRole("button", { name: "Add 2 Custom Formats" })).toBeVisible();
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("paste.png"));

  await dialog.getByRole("radio", { name: "Replace" }).click();
  await expect(adds.getByRole("listitem").first()).toContainText("Replaces the x265 you have.");
  await dialog.getByRole("button", { name: "Replace 1 and add 1" }).click();
  await expect(dialog).toBeHidden();

  await expect(formatRow(page, "x265")).toContainText("Radarr");
  await expect(formatRow(page, "x265")).toContainText(
    String.raw`matches "[xh][ .]?265|\bHEVC(\b|\d)"`,
  );
  await expect(formatRow(page, "Remux Tier 01")).toContainText(
    `Matches a Release when its release group matches "^(FraMeSToR)$".`,
  );
  await expect(page.getByRole("link", { name: "x265 (2)" })).toHaveCount(0);
});

test("a paste whose conditions settle the app reads it without asking", async ({
  adminPage: page,
}) => {
  await openFormats(page);
  await page.getByRole("button", { name: "Paste", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Paste Custom Formats" });
  await dialog.getByRole("textbox", { name: "JSON" }).fill(
    JSON.stringify({
      name: "Season pack",
      specifications: [
        {
          name: "Season pack",
          implementation: "ReleaseTypeSpecification",
          negate: false,
          required: false,
          fields: { value: 3 },
        },
      ],
    }),
  );
  await dialog.getByRole("button", { name: "Preview" }).click();
  await expect(dialog.getByText("Read as Sonarr formats.")).toBeVisible();
  await expect(dialog.getByText("Say whether this was written for Radarr or Sonarr.")).toHaveCount(
    0,
  );
  await dialog.getByRole("button", { name: "Add 1 Custom Format" }).click();
  await expect(formatRow(page, "Season pack")).toContainText("Sonarr");
});

// Every Target scores formatId.
async function scoredEverywhere(world: World, formatId: string): Promise<string[]> {
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
    expect(scored.status, `${name} scores the format`).toBe(200);
  }
  return targets.map((target) => target.name);
}

test("deleting a format names the Targets that score it and drops their scores", async ({
  adminPage: page,
  world,
}) => {
  const formatId = await made(world, "Golden", "GOLDEN");
  const names = await scoredEverywhere(world, formatId);

  await openFormats(page);
  await expect(formatRow(page, "Golden")).toContainText(names.join(", "));
  await formatRow(page, "Golden").getByRole("button", { name: "Actions for Golden" }).click();
  await clickWhenStill(
    page.getByRole(isPhone(page) ? "button" : "menuitem", { name: "Delete", exact: true }),
  );
  const dialog = page.getByRole("dialog", { name: "Delete Golden?" });
  await expect(dialog).toContainText(
    `${names.slice(0, -1).join(", ")} and ${names.at(-1) ?? ""} score it. Deleting it also deletes those scores.`,
  );
  await expectAccessible(page);
  await dialog.getByRole("button", { name: "Delete Golden" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByText("No Custom Formats yet.")).toBeVisible();

  const { targets } = z
    .object({ targets: z.array(z.object({ scores: z.object({ formats: z.array(z.unknown()) }) })) })
    .parse((await world.api("GET", "/api/v1/targets")).body);
  expect(
    targets.flatMap((target) => target.scores.formats),
    "no Target scores Golden",
  ).toEqual([]);
});
