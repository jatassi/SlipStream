import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { grabOnSeedbox, nodeId } from "./grabs.ts";
import { isPhone, navigate } from "./shell.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "home" });

const Settings = z.object({ sendLimit: z.object({ value: z.number() }) });
const Imports = z.object({ imports: z.tuple([z.object({ id: z.string() })]) });
const Transfers = z.object({
  transfers: z.array(z.object({ purpose: z.string(), job: z.string() })),
});

const pace = / of 36 MB · [\d.]+ [KM]B\/s/u;

async function openTransfers(page: Page, id: string, name: string): Promise<void> {
  await page.goto(`/nodes/${id}`);
  const sections = isPhone(page)
    ? page.getByRole("list", { name })
    : page.getByRole("navigation", { name });
  await sections.getByRole("link", { name: "Transfers" }).click();
  await expect(page).toHaveURL(/\/nodes\/[\da-f-]+\/transfers$/u);
}

async function importOf(world: World, workId: string): Promise<string> {
  const listed = await world.api("GET", `/api/v1/imports?source=grab&work=${workId}`);
  const [only] = Imports.parse(listed.body).imports;
  return only.id;
}

async function purposeOf(world: World, job: string): Promise<string | undefined> {
  const { transfers } = Transfers.parse((await world.api("GET", "/api/v1/transfers")).body);
  return transfers.find((transfer) => transfer.job === job)?.purpose;
}

test("seedbox's Transfers say nothing moves and save a send limit the Node reads back", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const seedbox = await nodeId(world, "seedbox");
  await openTransfers(page, seedbox, "seedbox");
  await expect(page.getByText("Nothing running", { exact: true })).toBeVisible();

  await page.getByRole("textbox", { name: "Send limit" }).fill("10");
  await page.getByRole("main").getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("region", { name: "Messages" })).toContainText(
    "Transfer settings saved",
  );
  const saved = await world.api("GET", `/api/v1/transfers/nodes/${seedbox}/settings`);
  expect(Settings.parse(saved.body).sendLimit.value, "10 MB/s in bytes").toBe(10 * 1024 * 1024);
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("transfers.png"));
});

test.describe("between two Workers", () => {
  test.use({ scenario: "seedbox-nas" });
  test.describe.configure({ timeout: 180_000 });

  test("a Grab on seedbox reads Transferring from seedbox on nas, its Import's Transfer, then imports", async ({
    adminPage: page,
    world,
  }, testInfo) => {
    const workId = await grabOnSeedbox(world);
    const nas = await nodeId(world, "nas");
    await openTransfers(page, nas, "nas");
    const row = page
      .getByRole("list", { name: "Transfers of nas" })
      .getByRole("listitem")
      .filter({ hasText: "The Quiet Orchard" });
    const copying = row.getByText("Transferring from seedbox", { exact: true });
    await expect(copying).toBeVisible({ timeout: 90_000 });
    await expect(row.getByText(pace)).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText(/^Receiving The Quiet Orchard from seedbox at /u)).toBeVisible();
    await expectAccessible(page);
    await screenshotWholePage(page, testInfo.outputPath("nas-transfers.png"));

    const importId = await importOf(world, workId);
    expect(await purposeOf(world, importId), "the Import asked for the Transfer").toBe("import");

    await navigate(page, "Activity");
    const job = page.getByRole("table", { name: "Grabs" }).getByRole("row", { name: /Quiet/u });
    await expect(job.getByText("Importing", { exact: true })).toBeVisible();
    await expectAccessible(page);
    await screenshotWholePage(page, testInfo.outputPath("activity.png"));

    await expect(job).toContainText("Imported", { timeout: 90_000 });
    await openTransfers(page, nas, "nas");
    await expect(page.getByText("Nothing running", { exact: true })).toBeVisible();
  });
});
