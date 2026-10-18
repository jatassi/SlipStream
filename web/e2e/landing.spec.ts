import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 300_000 });

const Nodes = z.object({ nodes: z.array(z.object({ name: z.string() })) });

const namesOf = async (world: World): Promise<string[]> =>
  Nodes.parse((await world.api("GET", "/api/v1/nodes")).body).nodes.map(({ name }) => name);

const roster = (page: Page) => page.getByRole("list", { name: "Roster" });
const rowLink = (page: Page, name: string) =>
  roster(page).getByRole("link", { name: new RegExp(`^${name} Online`, "u") });

async function choose(page: Page, node: string, item: string): Promise<void> {
  await page.getByRole("button", { name: `${node} actions` }).click();
  if (isPhone(page)) {
    await page
      .getByRole("dialog", { name: `${node} actions` })
      .getByRole("button", { name: item })
      .click();
    return;
  }
  await page.getByRole("menuitem", { name: item }).click();
}

test("Nodes opens on the Diagram on a desktop browser and on the Roster on a phone, and a Roster row opens its Node", async ({
  adminPage: page,
}) => {
  await page.goto("/nodes");
  await expect(page.getByRole("heading", { name: "Nodes", level: 1 })).toBeVisible();
  const views = page.getByRole("navigation", { name: "View" });
  const opened = isPhone(page) ? "Roster" : "Diagram";
  await expect(views.getByRole("link", { name: opened })).toHaveAttribute("aria-current", "page");
  await expect(page.getByRole("button", { name: "Add a Node", exact: true })).toBeVisible();
  if (!isPhone(page)) {
    await expect(page.getByRole("figure", { name: "Diagram" })).toHaveAttribute(
      "data-takes-drags",
      "true",
    );
    await views.getByRole("link", { name: "Roster" }).click();
  }

  await expect(rowLink(page, "home")).toContainText("Online · Controller");
  await expect(rowLink(page, "seedbox")).toContainText(/Online · seedbox dials home/u);
  await rowLink(page, "seedbox").click();
  await expect(page).toHaveURL(/\/nodes\/[0-9a-f-]{36}$/u);
  await expect(page.getByRole("region", { name: "Overview" })).toContainText(
    "Reached byseedbox dials home",
  );
});

test("Rename on the Roster renames the Node everywhere", async ({ adminPage: page, world }) => {
  await page.goto("/nodes?view=roster");
  await choose(page, "seedbox", "Rename");
  const dialog = page.getByRole("dialog", { name: "Rename seedbox" });
  await dialog.getByRole("textbox", { name: "Name" }).fill("home");
  await dialog.getByRole("button", { name: "Rename" }).click();
  await expect(dialog.getByRole("textbox", { name: "Name" })).toHaveAccessibleDescription(
    "Another Node has this name",
  );
  await dialog.getByRole("textbox", { name: "Name" }).fill("box");
  await dialog.getByRole("button", { name: "Rename" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(rowLink(page, "box")).toBeVisible();
  expect((await namesOf(world)).toSorted()).toEqual(["box", "home"]);
});

test("Remove past the re-confirm window asks for the PIN, then the Worker leaves the Roster", async ({
  adminPage: page,
  world,
}) => {
  const shortened = await world.api("PATCH", "/api/v1/identity/settings", { reconfirmWindow: 60 });
  expect(shortened.status, "the re-confirm window drops to a minute").toBe(200);
  await page.goto("/nodes?view=roster");
  await expect(rowLink(page, "seedbox")).toBeVisible();
  const lapsesAt = Date.now() + 61_000;
  await expect
    .poll(() => Date.now(), {
      message: "the sign-in's confirm is past the window",
      timeout: 70_000,
    })
    .toBeGreaterThan(lapsesAt);

  await choose(page, "seedbox", "Remove");
  const dialog = page.getByRole("dialog", { name: "Remove seedbox?" });
  await expect(dialog).toContainText("seedbox leaves the Roster");
  await dialog.getByRole("button", { name: /^Remove/u }).click();
  const confirm = page.getByRole("dialog", { name: "Confirm it's you" });
  await expect(confirm).toContainText("Enter your PIN to continue.");
  await confirm.getByRole("textbox", { name: "PIN", exact: true }).fill(world.admin().pin);
  await expect(confirm).toHaveCount(0);
  await expect(dialog).toHaveCount(0);
  await expect(roster(page).getByRole("link", { name: /^seedbox/u })).toHaveCount(0);
  expect(await namesOf(world)).toEqual(["home"]);
});
