import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

test.use({ scenario: "home" });

const Nodes = z.object({ nodes: z.array(z.object({ id: z.string(), name: z.string() })) });
const Settings = z.object({ sendLimit: z.object({ value: z.number() }) });
const Added = z.object({ work: z.object({ id: z.string() }) });

const pace = / of 36 MB · [\d.]+ [KM]B\/s/u;

async function nodeId(world: World, name: string): Promise<string> {
  const { nodes } = Nodes.parse((await world.api("GET", "/api/v1/nodes")).body);
  const node = nodes.find((candidate) => candidate.name === name);
  if (node === undefined) {
    throw new Error(`the Scenario has no ${name}`);
  }
  return node.id;
}

async function openTransfers(page: Page, id: string, name: string): Promise<void> {
  await page.goto(`/nodes/${id}`);
  const sections = isPhone(page)
    ? page.getByRole("list", { name })
    : page.getByRole("navigation", { name });
  await sections.getByRole("link", { name: "Transfers" }).click();
  await expect(page).toHaveURL(/\/nodes\/[\da-f-]+\/transfers$/u);
}

// The Quiet Orchard, added and searched the way a script would, with
// seedbox's send limit holding its Transfer to about a minute.
async function grabOnSeedbox(world: World): Promise<void> {
  const settings = `/api/v1/transfers/nodes/${await nodeId(world, "seedbox")}/settings` as const;
  const limited = await world.api("PATCH", settings, { sendLimit: 512 * 1024 });
  expect(limited.status, "seedbox's send limit is saved").toBe(200);
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    monitor: true,
  });
  expect(added.status, "The Quiet Orchard is added").toBe(200);
  const workId = Added.parse(added.body).work.id;
  const search = await world.api("POST", `/api/v1/works/${workId}/search`);
  expect(search.status, "its search starts").toBe(202);
}

test("seedbox's Transfers say nothing moves and save a send limit the Node reads back", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const seedbox = await nodeId(world, "seedbox");
  await openTransfers(page, seedbox, "seedbox");
  await expect(page.getByText("Nothing moving", { exact: true })).toBeVisible();

  await page.getByRole("textbox", { name: "Send limit" }).fill("10");
  await page.getByRole("main").getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Transfer settings saved",
  );
  const saved = await world.api("GET", `/api/v1/transfers/nodes/${seedbox}/settings`);
  expect(Settings.parse(saved.body).sendLimit.value, "10 MB/s in bytes").toBe(10 * 1024 * 1024);
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("transfers.png"), fullPage: true });
});

test.describe("between two Workers", () => {
  test.use({ scenario: "seedbox-nas" });
  test.describe.configure({ timeout: 180_000 });

  test("a Grab on seedbox reads Transferring from seedbox on nas and in Activity, then imports", async ({
    adminPage: page,
    world,
  }, testInfo) => {
    await grabOnSeedbox(world);
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
    await page.screenshot({ path: testInfo.outputPath("nas-transfers.png"), fullPage: true });

    await navigate(page, "Activity");
    const job = page.getByRole("table", { name: "Grabs" }).getByRole("row", { name: /Quiet/u });
    await expect(job.getByText("Transferring from seedbox", { exact: true })).toBeVisible();
    await expect(job.getByText(pace)).toBeVisible();
    await expectAccessible(page);
    await page.screenshot({ path: testInfo.outputPath("activity.png"), fullPage: true });

    await expect(job).toContainText("Imported", { timeout: 90_000 });
    await openTransfers(page, nas, "nas");
    await expect(page.getByText("Nothing moving", { exact: true })).toBeVisible();
  });
});
