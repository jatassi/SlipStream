import type { Page } from "@playwright/test";
import { z } from "zod";

import { node } from "../harness/manifest.ts";
import { type World, expect, test } from "./fixtures.ts";

test.use({ scenario: "cgnat-home" });
test.describe.configure({ timeout: 180_000 });

const Claim = z.object({ state: z.string(), nodeAddress: z.string().optional() });
const Board = z.object({
  nodes: z.array(z.object({ name: z.string(), assigned: z.array(z.string()) })),
});

const dialog = (page: Page) => page.getByRole("dialog", { name: "Add a Node" });

async function homePage(page: Page) {
  await page.goto("/nodes");
  await expect(page.getByRole("button", { name: "Add a Node", exact: true })).toBeVisible();
}

// first-run.spec.ts drives the first-run page's Join form. These tests
// follow the dialog, so seedbox joins through the operation that form sends.
async function claimSeedbox(world: World, code: string) {
  const seedbox = node(world.manifest, "seedbox");
  const response = await fetch(new URL("/api/v1/setup/worker", seedbox.url), {
    method: "POST",
    headers: { "X-Setup-Code": await world.setupCode("seedbox") },
    body: new Blob([JSON.stringify({ code })], { type: "application/json" }),
  });
  return { status: response.status, body: (await response.json()) as unknown };
}

async function mint(page: Page): Promise<string> {
  await page.getByRole("button", { name: "Add a Node", exact: true }).click();
  const code = dialog(page).getByRole("textbox", { name: "Join Code" });
  await expect(code).toHaveValue(/^SLP1-/u);
  return code.inputValue();
}

async function expectPaired(page: Page, world: World, reached: string) {
  await expect(dialog(page).getByText("seedbox paired.")).toBeVisible();
  await expect(dialog(page).getByText(node(world.manifest, "seedbox").nodeKey)).toBeVisible();
  await expect(dialog(page).getByText(reached)).toBeVisible();
}

test("a Node paired with the dialog's Join Code shows its key, then takes its pre-ticked Roles", async ({
  adminPage: page,
  world,
}) => {
  await world.fw.fault.clear("home-behind-cgnat");
  await homePage(page);
  const code = await mint(page);
  await expect(dialog(page).getByText(`slipstream join ${code}`)).toBeVisible();
  expect((await claimSeedbox(world, code)).status).toBe(201);
  await expectPaired(page, world, "seedbox dials home");

  await dialog(page).getByRole("button", { name: "Choose its Roles" }).click();
  await expect(
    dialog(page).getByRole("checkbox", { name: "Searching", exact: true }),
  ).toBeChecked();
  await expect(
    dialog(page).getByRole("checkbox", { name: "Downloading", exact: true }),
  ).toBeChecked();
  await dialog(page).getByRole("button", { name: "Review", exact: true }).click();
  const review = page.getByRole("dialog", { name: "Review the change" });
  await expect(review.getByRole("list", { name: "What changes" })).toContainText("seedbox");
  await review.getByRole("button", { name: "Apply" }).click();
  await expect(page).toHaveURL(/\/nodes\/[0-9a-f-]+\/roles$/u);
  await expect(
    page.getByRole("main").getByRole("checkbox", { name: "Downloading", exact: true }),
  ).toBeChecked();
  const board = Board.parse((await world.api("GET", "/api/v1/nodes/roles")).body);
  expect(board.nodes.find(({ name }) => name === "seedbox")?.assigned).toEqual([
    "searching",
    "downloading",
  ]);
});

test("a new Node that can't reach home pairs once its Node address is pasted and home dials it", async ({
  adminPage: page,
  world,
}) => {
  await homePage(page);
  const code = await mint(page);
  const claim = await claimSeedbox(world, code);
  expect(claim.status).toBe(202);
  const { nodeAddress } = Claim.parse(claim.body);
  expect(nodeAddress).toMatch(/^SLN1-/u);

  await dialog(page).getByRole("button", { name: "The new Node can't reach home?" }).click();
  await dialog(page)
    .getByRole("textbox", { name: "Node address" })
    .fill(nodeAddress ?? "");
  await dialog(page).getByRole("button", { name: "home dials it" }).click();
  await expectPaired(page, world, "home dials seedbox");
});

test("the open Join Codes survive a reload and revoke a code so it can't pair", async ({
  adminPage: page,
  world,
}) => {
  await world.fw.fault.clear("home-behind-cgnat");
  const stray = z
    .object({ id: z.string(), code: z.string() })
    .parse((await world.api("POST", "/api/v1/join-codes")).body);
  await homePage(page);
  await mint(page);
  await page.reload();
  await expect(dialog(page).getByText(/^A Join Code shows only when it is made\./u)).toBeVisible();
  const open = dialog(page).getByRole("list", { name: "Open Join Codes" });
  await expect(open.getByRole("listitem")).toHaveCount(2);

  await open.getByRole("button", { name: /^Revoke the code made /u }).click();
  await expect(open.getByRole("listitem")).toHaveCount(1);
  expect(await claimSeedbox(world, stray.code)).toMatchObject({
    status: 403,
    body: { code: "join_code_unknown" },
  });

  await open.getByRole("button", { name: "Revoke this code" }).click();
  await expect(
    dialog(page).getByText("This Join Code was revoked, so no Node can use it."),
  ).toBeVisible();
  expect((await world.api("GET", "/api/v1/join-codes")).body).toEqual({ codes: [] });
});
