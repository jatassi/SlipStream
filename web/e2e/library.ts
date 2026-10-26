import { mkdir } from "node:fs/promises";
import { join } from "node:path";

import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect } from "./fixtures.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

const Created = z.object({ id: z.string() });
const Added = z.object({ work: z.object({ id: z.string() }) });
const Library = z.object({
  rows: z.array(
    z.object({
      label: z.string(),
      monitored: z.string(),
      profileId: z.string(),
      rootId: z.string().nullable(),
    }),
  ),
});
const Cells = z.object({ cells: z.array(z.object({ state: z.string() })) });

export const orchard = "The Quiet Orchard";
export const saltRoad = "The Salt Road";

export async function rootAt(world: World, folder: string) {
  const path = join(world.manifest.scratchDir, folder);
  await mkdir(path, { recursive: true });
  const made = await world.api("POST", "/api/v1/roots", {
    contentType: "movie",
    nodeId: world.node.nodeId,
    path,
  });
  expect(made.status, `the ${folder} root is made`).toBe(201);
  return Created.parse(made.body).id;
}

export async function add(world: World, key: string, monitor: boolean) {
  const added = await world.api("POST", "/api/v1/works", { contentType: "movie", key, monitor });
  expect(added.status, `movie ${key} is added`).toBe(200);
  return Added.parse(added.body).work.id;
}

// A UHD Profile beside the default one, holding a 2160p Target.
export async function makeUHD(world: World) {
  const uhd = await world.api("POST", "/api/v1/targets", {
    name: "2160p",
    tiers: [["remux-2160p", "bluray-2160p"], ["webdl-2160p"]],
    cutoff: "remux-2160p",
  });
  expect(uhd.status, "the 2160p Target is made").toBe(201);
  const profile = await world.api("POST", "/api/v1/profiles", {
    name: "UHD",
    contentType: "movie",
    targets: [{ targetId: Created.parse(uhd.body).id, sourcing: "download" }],
    editions: { kind: "any" },
  });
  expect(profile.status, "the UHD Profile is made").toBe(201);
  return Created.parse(profile.body).id;
}

export const libraryRows = async (world: World) =>
  Library.parse((await world.api("GET", "/api/v1/library")).body).rows;

export const bar = (page: Page) => page.getByRole("toolbar", { name: "Picked Works" });
const barMenu = "Actions for the picked Works";

// A phone's bar holds its actions in a sheet.
export async function barAction(page: Page, name: string) {
  if (isPhone(page)) {
    await bar(page).getByRole("button", { name: barMenu }).click();
    await clickWhenStill(
      page.getByRole("dialog", { name: barMenu }).getByRole("button", { name, exact: true }),
    );
  } else {
    await bar(page).getByRole("button", { name, exact: true }).click();
  }
}

export const toasts = (page: Page) => page.getByRole("region", { name: "Notifications" });

export async function pick(page: Page, ...labels: string[]) {
  for (const label of labels) {
    // oxlint-disable-next-line no-await-in-loop -- each pick is a click after the last
    await page.getByRole("checkbox", { name: `Pick ${label}` }).click();
  }
}

// The Quiet Orchard, added and searched as a script would, with its WEB-DL imported.
export async function imported(world: World): Promise<string> {
  const workId = await add(world, "9000001", true);
  const search = await world.api("POST", `/api/v1/works/${workId}/search`);
  expect(search.status, "its search starts").toBe(202);
  await expect
    .poll(
      async () =>
        Cells.parse((await world.api("GET", `/api/v1/works/${workId}/cells`)).body).cells.every(
          (cell) => cell.state !== "missing",
        ),
      { message: "the WEB-DL is imported", timeout: 60_000 },
    )
    .toBe(true);
  return workId;
}
