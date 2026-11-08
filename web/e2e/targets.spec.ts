import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, openSettingsSection } from "./shell.ts";
import { clickWhenStill } from "./still.ts";
import { screenshotWholePage } from "./whole-page.ts";

async function openTargets(page: Page) {
  await openSettingsSection(page, "Targets");
  await expect(page).toHaveURL(/\/settings\/targets$/u);
}

async function choose(page: Page, popup: string, item: string) {
  await clickWhenStill(
    page
      .getByRole(isPhone(page) ? "dialog" : "menu", { name: popup })
      .getByRole(isPhone(page) ? "button" : "menuitem", { name: item }),
  );
}

async function untick(page: Page, qualities: readonly string[]) {
  for (const quality of qualities) {
    // oxlint-disable-next-line no-await-in-loop -- each tick lands before the next, as a person's does
    await page.getByRole("checkbox", { name: quality, exact: true }).click();
  }
}

async function tickAcross(page: Page) {
  const ranked = page.getByRole("list", { name: "Qualities", exact: true });
  const unaccepted = page.getByRole("list", { name: "Not accepted" });
  await expect(unaccepted.getByRole("listitem")).toHaveText([
    "Remux-2160p",
    "Bluray-2160p",
    "WEBDL-2160p",
    "WEBRip-2160p",
    "HDTV-2160p",
    "WEBRip-480p",
    "DVD",
    "SDTV",
  ]);
  await expect(ranked.getByRole("checkbox", { name: "Remux-2160p" })).toHaveCount(0);
  await unaccepted.getByRole("checkbox", { name: "DVD" }).click();
  await expect(ranked.getByRole("listitem").last()).toHaveText("Rank 5DVD");
  await expect(ranked.getByRole("checkbox", { name: "DVD" })).toBeFocused();
  await ranked.getByRole("checkbox", { name: "DVD" }).click();
  await expect(unaccepted.getByRole("checkbox", { name: "DVD" })).toBeFocused();

  const lines = await ranked.locator("label").evaluateAll((labels) =>
    labels.map((label) => {
      const range = document.createRange();
      range.selectNode(label.lastChild ?? label);
      return new Set([...range.getClientRects()].map((rect) => Math.round(rect.top))).size;
    }),
  );
  expect(lines, "every ranked quality's name sits on one line").toEqual(lines.map(() => 1));
}

async function moveCutoffDown(page: Page) {
  if (isPhone(page)) {
    await page.getByRole("button", { name: "Actions for Cutoff" }).click();
    await choose(page, "Actions for Cutoff", "Move down");
  } else {
    await page.getByRole("button", { name: "Move Cutoff", exact: true }).focus();
    await page.keyboard.press("ArrowDown");
  }
}

test("a Target built from a preset says what its order means as the cutoff moves, and lands in the list", async ({
  adminPage: page,
}, testInfo) => {
  await openTargets(page);
  await page.getByRole("button", { name: "New Target" }).click();
  await expect(page).toHaveURL(/\/settings\/targets\/new$/u);

  await page.getByRole("textbox", { name: "Name" }).fill("HD only");
  await page.getByRole("button", { name: "Fill from a preset" }).click();
  await choose(page, "Presets", "Upgrade for disc or resolution");
  await untick(page, ["Remux-2160p", "Bluray-2160p", "WEBDL-2160p", "WEBRip-2160p", "HDTV-2160p"]);
  await untick(page, ["WEBRip-480p", "DVD", "SDTV"]);
  const sentence = page.getByText(/^Grabs /u);
  await expect(sentence).toHaveText(
    "Grabs HDTV-720p, WEBRip-720p or WEBDL-720p or better. Upgrades until Bluray-1080p or Remux-1080p, then stops.",
  );

  await tickAcross(page);
  await moveCutoffDown(page);
  await expect(sentence).toHaveText(
    "Grabs HDTV-720p, WEBRip-720p or WEBDL-720p or better. Upgrades until HDTV-1080p, WEBRip-1080p or WEBDL-1080p, then stops.",
  );
  await expect(
    page
      .getByRole("list", { name: "Sample moves" })
      .getByText("Bluray-720p ranks above HDTV-720p."),
  ).toBeVisible();
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("editor.png"));

  await page.getByRole("button", { name: "Create Target" }).click();
  await expect(page).toHaveURL(/\/settings\/targets$/u);
  const row = page
    .getByRole("table", { name: "Targets" })
    .getByRole("row")
    .filter({ hasText: "HD only" });
  await expect(row).toContainText("HDTV-720p to Remux-1080p");
  await expect(row).toContainText("HDTV-1080p");
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("list.png"));
});

const Saved = z.object({
  targets: z.array(
    z.object({
      name: z.string(),
      scores: z.object({
        formats: z.array(z.object({ formatId: z.string(), score: z.number() })),
        minimum: z.number(),
        upgradeUntil: z.number(),
        increment: z.number(),
      }),
      propers: z.string(),
    }),
  ),
});

async function open1080p(page: Page) {
  await page
    .getByRole("table", { name: "Targets" })
    .getByRole("link", { name: "1080p", exact: true })
    .click();
}

async function saved1080p(world: World) {
  const { targets } = Saved.parse((await world.api("GET", "/api/v1/targets")).body);
  return targets.find((target) => target.name === "1080p");
}

async function makeHdrFormat(world: World) {
  const made = await world.api("POST", "/api/v1/custom-formats", {
    name: "HDR",
    conditions: [
      {
        name: "HDR",
        kind: "ReleaseTitleSpecification",
        negate: false,
        required: false,
        fields: { value: String.raw`\bHDR\b` },
      },
    ],
  });
  return z.object({ id: z.string() }).parse(made.body).id;
}

test("a Target's scores and propers setting show in its sentence, and save", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const hdr = await makeHdrFormat(world);
  await openTargets(page);
  await open1080p(page);
  const scores = page.getByRole("region", { name: "Scores" });
  await expect(scores.getByText(/^Scores none of the 1 Custom Format, /u)).toBeVisible();
  await scores.getByRole("button", { name: "Details" }).click();
  await scores.getByRole("textbox", { name: "HDR score" }).fill("40");
  await scores.getByRole("textbox", { name: "Upgrade until" }).fill("100");
  await scores.getByRole("textbox", { name: "Increment" }).fill("5");
  await expect(page.getByText(/^Grabs /u)).toHaveText(
    "Grabs HDTV-720p, WEBRip-720p or WEBDL-720p or better. Upgrades until Bluray-1080p or Remux-1080p and a score of 100, then stops.",
  );
  await expect(
    scores.getByText(/^No file can reach the upgrade-until score of 100, because/u),
  ).toBeVisible();
  await page.getByRole("radio", { name: "Don't prefer" }).click();
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("scores.png"));

  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByText("Saved 1080p.")).toBeVisible();
  expect(await saved1080p(world)).toMatchObject({
    scores: {
      formats: [{ formatId: hdr, score: 40 }],
      minimum: 0,
      upgradeUntil: 100,
      increment: 5,
    },
    propers: "dont_prefer",
  });
  await open1080p(page);
  await expect(scores.getByText(/^Scores 1 of 1 Custom Format\./u)).toBeVisible();
  await scores.getByRole("button", { name: "Details" }).click();
  await expect(scores.getByRole("textbox", { name: "HDR score" })).toHaveValue("40");
  await expect(page.getByRole("radio", { name: "Don't prefer" })).toBeChecked();
  await expect(page.getByText(/^Grabs /u)).toContainText("and a score of 100, then stops.");
});
