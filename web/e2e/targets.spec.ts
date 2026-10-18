import type { Page } from "@playwright/test";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

async function openTargets(page: Page) {
  await navigate(page, "Settings");
  const sections = isPhone(page)
    ? page.getByRole("list", { name: "Settings" })
    : page.getByRole("navigation", { name: "Settings" });
  await sections.getByRole("link", { name: "Targets" }).click();
  await expect(page).toHaveURL(/\/settings\/targets$/u);
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
    await page.getByRole("button", { name: "Move down" }).click();
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
  await page
    .getByRole(isPhone(page) ? "button" : "menuitem", { name: "Upgrade for disc or resolution" })
    .click();
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
  await page.screenshot({ path: testInfo.outputPath("editor.png"), fullPage: true });

  await page.getByRole("button", { name: "Create Target" }).click();
  await expect(page).toHaveURL(/\/settings\/targets$/u);
  const row = page
    .getByRole("table", { name: "Targets" })
    .getByRole("row")
    .filter({ hasText: "HD only" });
  await expect(row).toContainText("HDTV-720p to Remux-1080p");
  await expect(row).toContainText("HDTV-1080p");
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("list.png"), fullPage: true });
});
