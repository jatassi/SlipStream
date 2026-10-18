import type { Page } from "@playwright/test";
import { z } from "zod";

import { orchardCells } from "./cells.ts";
import { type World, expect, expectAccessible, test } from "./fixtures.ts";

test.use({ scenario: "targets" });
test.describe.configure({ timeout: 120_000 });

const Named = z.object({ id: z.string(), name: z.string() });
const Added = z.object({ work: z.object({ id: z.string() }) });
const Cells = z.object({
  cells: z.array(z.object({ targetName: z.string(), state: z.string() })),
});

async function idOf(world: World, path: "/api/v1/targets" | "/api/v1/profiles", name: string) {
  const reply = await world.api("GET", path);
  const list = z.record(z.string(), z.array(Named)).parse(reply.body);
  const found = Object.values(list)
    .flat()
    .find((entry) => entry.name === name);
  if (found === undefined) {
    throw new Error(`${path} has no ${name}`);
  }
  return found.id;
}

async function cellsOf(world: World, workId: string) {
  return Cells.parse((await world.api("GET", `/api/v1/works/${workId}/cells`)).body).cells.map(
    (cell) => `${cell.targetName} ${cell.state}`,
  );
}

// The Quiet Orchard, added and searched the way a script would, with its WEB-DL imported.
async function imported(world: World): Promise<string> {
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    monitor: true,
  });
  expect(added.status, "The Quiet Orchard is added").toBe(200);
  const workId = Added.parse(added.body).work.id;
  const search = await world.api("POST", `/api/v1/works/${workId}/search`);
  expect(search.status, "its search starts").toBe(202);
  await expect
    .poll(
      async () => (await cellsOf(world, workId)).every((cell) => cell.endsWith(" upgradable")),
      {
        message: "the WEB-DL fills every Target of the Work's Profile",
        timeout: 60_000,
      },
    )
    .toBe(true);
  return workId;
}

const impactLines = (page: Page, title: string) =>
  page
    .getByRole("dialog", { name: title })
    .getByRole("list", { name: "What changes" })
    .getByRole("listitem");

test("tightening a Target so a file fills nothing shows the Impact, and Delete removes the file", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const starter = await idOf(world, "/api/v1/targets", "1080p");
  const films = await idOf(world, "/api/v1/profiles", "Films");
  const holding = await world.api("PUT", `/api/v1/profiles/${films}`, {
    name: "Films",
    contentType: "movie",
    targets: [{ targetId: starter, sourcing: "download" }],
    editions: { kind: "any" },
  });
  expect(holding.status, "Films holds only the starter").toBe(200);
  const workId = await imported(world);

  await page.goto(`/settings/targets/${starter}`);
  await page.getByRole("checkbox", { name: "WEBDL-1080p", exact: true }).click();
  await page.getByRole("button", { name: "Save", exact: true }).click();

  const sheet = page.getByRole("dialog", { name: "Save 1080p" });
  await expect(impactLines(page, "Save 1080p")).toHaveText([
    "1 Work changes.",
    /^1 file would fill nothing \(\d+(\.\d)? (bytes|KB|MB)\)\.$/u,
  ]);
  await expect(sheet.getByRole("radio", { name: "Delete" })).toBeChecked();
  await expect(sheet.getByRole("radio", { name: "Keep as Kept files" })).not.toBeChecked();
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("impact-sheet.png") });
  await sheet.getByRole("button", { name: "Save and delete 1 file" }).click();

  await expect(page.getByRole("region", { name: "Notifications" })).toContainText("Saved 1080p.");
  await expect(page).toHaveURL(/\/settings\/targets$/u);
  expect(await cellsOf(world, workId)).toEqual(["1080p missing"]);
  await page.goto(`/works/${workId}`);
  await expect(orchardCells(page)).toHaveText("The Quiet OrchardMissingMissing");
  await expect(page.getByText("No files yet.", { exact: false })).toBeVisible();
});

// A UHD Profile beside Films, holding one 2160p Target the WEB-DL can't fill.
async function makeUHD(world: World) {
  const uhd = await world.api("POST", "/api/v1/targets", {
    name: "2160p",
    tiers: [["remux-2160p", "bluray-2160p"], ["webdl-2160p"]],
    cutoff: "remux-2160p",
  });
  expect(uhd.status, "the 2160p Target is created").toBe(201);
  const profile = await world.api("POST", "/api/v1/profiles", {
    name: "UHD",
    contentType: "movie",
    targets: [{ targetId: Named.parse(uhd.body).id, sourcing: "download" }],
    editions: { kind: "any" },
  });
  expect(profile.status, "the UHD Profile is created").toBe(201);
}

test("an Admin changes a Work's Profile through the Impact sheet and keeps the file it leaves", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await makeUHD(world);
  const workId = await imported(world);

  await page.goto(`/works/${workId}`);
  await expect(page.getByText("Profile Films. Editions from the Profile.")).toBeVisible();
  await page.getByRole("button", { name: "Profile and Editions" }).click();
  const sheet = page.getByRole("dialog", { name: "Profile and Editions" });
  await sheet.getByRole("combobox", { name: "Profile" }).click();
  await page.getByRole("option", { name: "UHD" }).click();
  await expect(sheet.getByRole("heading", { name: "Editions from Profile UHD" })).toBeVisible();
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("work-sheet.png") });
  await sheet.getByRole("button", { name: "Save", exact: true }).click();

  const impact = page.getByRole("dialog", { name: "Save The Quiet Orchard" });
  await expect(impactLines(page, "Save The Quiet Orchard")).toHaveText([
    "1 Work changes.",
    /^1 file would fill nothing /u,
    "1 cell becomes wanted.",
  ]);
  await impact.getByRole("radio", { name: "Keep as Kept files" }).click();
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("work-impact.png") });
  await impact.getByRole("button", { name: "Save and keep 1 file" }).click();

  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Saved what The Quiet Orchard wants.",
  );
  await expect(sheet).toBeHidden();
  await expect(page.getByText("Profile UHD. Editions from the Profile.")).toBeVisible();
  await expect(orchardCells(page)).toHaveText("The Quiet OrchardMissingMissing");
  const wants = await world.api("GET", `/api/v1/works/${workId}/wants`);
  expect(wants.body).toMatchObject({ profileName: "UHD", from: "profile" });
});
