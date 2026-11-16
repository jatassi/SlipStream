import type { Page } from "@playwright/test";
import { z } from "zod";

import { front } from "../harness/manifest.ts";
import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import {
  addedItem,
  change,
  imported,
  openChanges,
  orchard,
  serverId,
  serverRow,
  settles,
} from "./media-servers.ts";
import { openSettingsSection } from "./shell.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "media-servers" });
test.describe.configure({ timeout: 300_000 });

const Report = z.object({
  updates: z.array(z.object({ path: z.string(), updateType: z.string() })),
});

async function openAddSheet(page: Page, kind: string) {
  await openSettingsSection(page, "Media Servers");
  await page.getByRole("button", { name: "Add a Media Server" }).click();
  const sheet = page.getByRole("dialog", { name: "Add a Media Server" });
  await sheet.getByRole("radio", { name: kind }).check();
  await sheet.getByRole("button", { name: "Next" }).click();
  return page.getByRole("dialog", { name: `Add ${kind}` });
}

// Saves the server's Path mappings with the roots on the Node it doesn't
// hold marked not shown.
async function saveMappings(page: Page, name: string, elsewhere: string): Promise<void> {
  const mappings = page.getByRole("dialog", { name: `Path mappings on ${name}` });
  const shown = (on: string) =>
    mappings
      .getByRole("group")
      .filter({ hasText: `On ${on}` })
      .getByRole("switch", { name: `In a ${name} library` });
  await expect(shown(elsewhere)).toHaveCount(2);
  await shown(elsewhere).nth(0).click();
  await shown(elsewhere).nth(1).click();
  await expect(shown(elsewhere).nth(0)).not.toBeChecked();
  await expect(shown(elsewhere).nth(1)).not.toBeChecked();
  await expectAccessible(page);
  await mappings.getByRole("button", { name: "Save" }).click();
  await expect(mappings).toHaveCount(0);
}

// The Add sheet's Jellyfin or Emby fields, signed in as the server's
// administrator from den, with home's front as the fallback address.
async function addThroughTheSheet(page: Page, world: World, kind: string, name: string) {
  const id = kind.toLowerCase();
  const fields = await openAddSheet(page, kind);
  await fields.getByRole("textbox", { name: "Name" }).fill(name);
  await fields.getByRole("textbox", { name: "Address" }).fill(front(world.manifest, id, "den"));
  await fields.getByRole("combobox", { name: "Connect from" }).click();
  await page.getByRole("option", { name: "den", exact: true }).click();
  await fields.getByRole("button", { name: "Sign in as an administrator" }).click();
  await fields.getByRole("textbox", { name: "User" }).fill("admin");
  await fields.getByLabel("Password").fill("fake-world-password");
  await fields.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(fields.getByText("Created a key named SlipStream.")).toBeVisible();
  await fields.getByRole("button", { name: "Details" }).click();
  await fields
    .getByRole("textbox", { name: "Fallback address" })
    .fill(front(world.manifest, id, "home"));
  await fields.getByRole("button", { name: "Save" }).click();
  await saveMappings(page, name, "home");
}

for (const [kind, name] of [
  ["Jellyfin", "Den Jellyfin"],
  ["Emby", "Den Emby"],
] as const) {
  test(`${name}, added through the Add sheet with an administrator sign-in, takes an imported movie's path`, async ({
    adminPage: page,
    world,
  }, testInfo) => {
    const id = kind.toLowerCase();
    const removed = await world.api(
      "DELETE",
      `/api/v1/media-servers/${await serverId(world, name)}`,
    );
    expect(removed.status).toBe(204);
    await addThroughTheSheet(page, world, kind, name);
    await expect(serverRow(page, name)).toContainText(`${kind} · Connects from den`);
    await expect(serverRow(page, name)).toContainText(`${name} responds to den.`);

    await imported(world, "den-movies");
    const file = `/media/den-movies/${orchard}`;
    const report = await world.fw.capture(`${id}-reports`).wait({ node: "den", timeout: 60_000 });
    expect(Report.parse(report.data).updates).toContainEqual({ path: file, updateType: "Created" });
    await addedItem(world, id, file);

    await openChanges(page, name);
    const sent = change(page, name, "The Quiet Orchard (2021) - 1080p.mkv");
    await expect(sent).toContainText("Sent the new file.", { timeout: settles });
    await expect(sent).toContainText("New file");
    await expectAccessible(page);
    await screenshotWholePage(page, testInfo.outputPath(`${id}-changes.png`));
  });
}

async function signOutOfPlex(page: Page): Promise<void> {
  await openSettingsSection(page, "Media Servers");
  const account = page.getByRole("list", { name: "Plex account" });
  await expect(account).toContainText("plex-admin");
  await account.getByRole("button", { name: "Sign out" }).click();
  const confirm = page.getByRole("dialog", { name: "Sign out of Plex?" });
  await expect(confirm).toContainText("Signing out also removes Living room Plex from SlipStream");
  await confirm.getByRole("button", { name: "Sign out" }).click();
  await expect(serverRow(page, "Living room Plex")).toHaveCount(0);
}

// Signs in with Plex in the tab the Add sheet opens on plex.tv's sign-in
// page, which returns to SlipStream once the Admin allows it.
async function signInInTab(page: Page, world: World) {
  const plex = await openAddSheet(page, "Plex");
  const opened = page.context().waitForEvent("page");
  await plex.getByRole("button", { name: "Sign in with Plex" }).click();
  const tab = await opened;
  await expect(
    plex.getByText("Waiting for you to sign in to Plex in the other tab."),
  ).toBeVisible();
  await expect(tab).toHaveURL(
    new RegExp(`^${front(world.manifest, "plextv-auth", "home")}/auth`, "u"),
  );
  await tab.getByRole("button", { name: "Allow" }).click();
  await expect(tab).toHaveURL(/\/settings\/media-servers$/u);
  await tab.close();
  await expect(
    plex.getByText("Signed in to Plex as plex-admin. The account has Plex Pass."),
  ).toBeVisible();
  return plex;
}

test.describe("Plex", () => {
  test.use({ allow: ["plextv-auth"] });

  test("Living room Plex, added again through a Plex sign-in in a new tab, refreshes an imported movie's folder", async ({
    adminPage: page,
    world,
  }, testInfo) => {
    await signOutOfPlex(page);
    const plex = await signInInTab(page, world);
    await expect(plex.getByRole("radio", { name: "Living room Plex" })).toBeChecked();
    await plex.getByRole("combobox", { name: "Runs on" }).click();
    await page.getByRole("option", { name: "home", exact: true }).click();
    await expect(plex.getByRole("combobox", { name: "Connect from" })).toContainText("home");
    await expectAccessible(page);
    await plex.getByRole("button", { name: "Save" }).click();
    await saveMappings(page, "Living room Plex", "den");
    await expect(serverRow(page, "Living room Plex")).toContainText("Plex · Connects from home");
    await expect(serverRow(page, "Living room Plex")).toContainText(
      "Living room Plex responds to home.",
    );

    await imported(world, "movies");
    await world.fw.capture("plex-refreshes").wait({
      node: "home",
      where: { path: "/media/movies/The Quiet Orchard (2021)" },
      timeout: 60_000,
    });
    await addedItem(world, "plex", `/media/movies/${orchard}`);
    await openChanges(page, "Living room Plex");
    await expect(
      change(page, "Living room Plex", "The Quiet Orchard (2021) - 1080p.mkv"),
    ).toContainText("Sent the new file.");
    await screenshotWholePage(page, testInfo.outputPath("plex-changes.png"));
  });
});
