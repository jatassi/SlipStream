import type { Locator, Page } from "@playwright/test";
import { z } from "zod";

import { workAction } from "./cells.ts";
import { type World, expect } from "./fixtures.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

export const lanternKeepers = "1001";
export const harborNights = "1002";
export const signalNine = "1003";

const Added = z.object({ work: z.object({ id: z.string() }) });
const Tree = z.object({
  monitorNew: z.boolean(),
  monitorAsides: z.boolean(),
  levelFolders: z.boolean(),
  items: z.array(
    z.object({ label: z.string(), monitored: z.boolean(), withdrawnAt: z.string().nullable() }),
  ),
});

type Adding = {
  preset: "all" | "future" | "missing" | "existing" | "first" | "latest" | "none";
  includeAsides?: boolean;
};

export async function addSeries(world: World, key: string, adding: Adding): Promise<string> {
  const added = await world.api("POST", "/api/v1/works", { contentType: "series", key, ...adding });
  expect(added.status, `series ${key} is added`).toBe(200);
  return Added.parse(added.body).work.id;
}

export const treeOf = async (world: World, workId: string) =>
  Tree.parse((await world.api("GET", `/api/v1/works/${workId}`)).body);

export async function refreshed(world: World, workId: string): Promise<void> {
  const refresh = await world.api("POST", "/api/v1/works/refresh", { workIds: [workId] });
  expect(refresh.status, "the refresh runs").toBe(200);
  expect(refresh.body, "the refresh is done").toMatchObject({ outcomes: [{ outcome: "done" }] });
}

export const seasons = (page: Page) => page.getByRole("region", { name: "Seasons" });

export const levelHeader = (page: Page, name: string) =>
  seasons(page).getByRole("heading", { level: 3, name: new RegExp(`^${name} `, "u") });

/** Scrolls the window down until the virtualized list renders the locator, then into view. */
export async function reveal(page: Page, target: Locator): Promise<Locator> {
  await seasons(page).waitFor();
  await page.evaluate(() => {
    window.scrollTo(0, 0);
  });
  await expect
    .poll(
      async () => {
        if ((await target.count()) > 0) {
          return true;
        }
        await page.evaluate(() => {
          window.scrollBy(0, 300);
        });
        return false;
      },
      { message: "the list renders it", intervals: [100] },
    )
    .toBe(true);
  await target.first().scrollIntoViewIfNeeded();
  return target;
}

export async function openLevel(page: Page, name: string): Promise<void> {
  const toggle = await reveal(page, seasons(page).getByRole("button", { name, exact: true }));
  if ((await toggle.getAttribute("aria-expanded")) !== "true") {
    await toggle.click();
  }
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
}

const monitoringButton = (page: Page) =>
  page.getByRole("button", { name: /^Monitoring \d+ of \d+ episodes$/u });

/** The hero's count of monitored episodes, which a phone's hero leaves out, so there the API's. */
export async function expectMonitoring(
  page: Page,
  world: World,
  workId: string,
  [monitored, of]: readonly [number, number],
): Promise<void> {
  if (!isPhone(page)) {
    await expect(monitoringButton(page)).toHaveText(`Monitoring ${monitored} of ${of} episodes`);
    return;
  }
  await expect
    .poll(
      async () =>
        (await treeOf(world, workId)).items.filter(
          (item) => item.monitored && item.withdrawnAt === null,
        ).length,
    )
    .toBe(monitored);
}

export async function openMonitoring(page: Page, label: string): Promise<Locator> {
  if (isPhone(page)) {
    await workAction(page, "Set monitoring…", label);
  } else {
    await monitoringButton(page).click();
  }
  const dialog = page.getByRole("dialog", { name: `Set monitoring for ${label}` });
  await expect(dialog).toBeVisible();
  return dialog;
}

/** The row of the episode with this code: its line on desktop, its sheet's button on phone. */
export const episodeRow = (page: Page, code: string): Locator =>
  isPhone(page)
    ? seasons(page).getByRole("button", { name: new RegExp(`^${code} `, "u") })
    : seasons(page).getByText(code, { exact: true }).locator("..");

export const workIdOf = (page: Page): string =>
  new URL(page.url()).pathname.split("/").at(-1) ?? "";

// A phone row opens the episode's sheet, which holds its Monitor switch.
export async function setEpisodeMonitored(page: Page, code: string, on: boolean): Promise<void> {
  if (!isPhone(page)) {
    const toggle = seasons(page).getByRole("switch", { name: `Monitor ${code}` });
    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-checked", String(on));
    return;
  }
  await clickWhenStill(episodeRow(page, code));
  const sheet = page.getByRole("dialog", { name: new RegExp(`^${code} `, "u") });
  const toggle = sheet.getByRole("switch", { name: "Monitor" });
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-checked", String(on));
  await sheet.getByRole("button", { name: "Cancel" }).click();
  await expect(sheet).toBeHidden();
}
