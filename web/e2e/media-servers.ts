import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect } from "./fixtures.ts";
import { isPhone, openSettingsSection } from "./shell.ts";

export const orchard = "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv";
// Jellyfin takes a path report after its 60 s monitor delay, and a rename waits two.
export const settles = 240_000;

const Roots = z.object({ roots: z.array(z.object({ id: z.string(), path: z.string() })) });
const Servers = z.object({ servers: z.array(z.object({ id: z.string(), name: z.string() })) });
const Changes = z.object({ changes: z.array(z.object({ state: z.string() })) });
const Grabs = z.object({ grabs: z.array(z.object({ stage: z.string() })) });
const Added = z.object({ work: z.object({ id: z.string() }) });
const Item = z.object({ files: z.array(z.string()) });

export async function serverId(world: World, name: string): Promise<string> {
  const { servers } = Servers.parse((await world.api("GET", "/api/v1/media-servers")).body);
  const found = servers.find((server) => server.name === name);
  if (found === undefined) {
    throw new Error(`the API lists no Media Server named ${name}`);
  }
  return found.id;
}

async function allSent(world: World, id: string): Promise<boolean> {
  const { changes } = Changes.parse(
    (await world.api("GET", `/api/v1/media-servers/${id}/changes`)).body,
  );
  return changes.length > 0 && changes.every(({ state }) => state === "sent");
}

// Waits until every server has sent each of its changes, such as the whole
// library refresh its Path mappings started, so that the next change isn't
// covered by that refresh.
async function untilSent(world: World): Promise<void> {
  const { servers } = Servers.parse((await world.api("GET", "/api/v1/media-servers")).body);
  await expect
    .poll(() => Promise.all(servers.map(({ id }) => allSent(world, id))), {
      message: "every server sent its first whole library refresh",
      timeout: 60_000,
    })
    .toEqual(servers.map(() => true));
}

// The Quiet Orchard added into the root at folder and searched the way a
// script would, until its Grab is imported. It answers the Work's id.
export async function imported(world: World, folder: string): Promise<string> {
  await untilSent(world);
  const { roots } = Roots.parse((await world.api("GET", "/api/v1/roots")).body);
  const root = roots.find((each) => each.path.split(/[/\\]/u).at(-1) === folder);
  if (root === undefined) {
    throw new Error(`the API lists no root at ${folder}`);
  }
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    preset: "all",
    rootId: root.id,
  });
  const work = Added.parse(added.body).work.id;
  const search = await world.api("POST", `/api/v1/works/${work}/search`);
  expect(search.status, "its search starts").toBe(202);
  await expect
    .poll(
      async () =>
        Grabs.parse((await world.api("GET", "/api/v1/grabs")).body).grabs.map(({ stage }) => stage),
      { message: "its Grab is imported", timeout: 60_000 },
    )
    .toEqual(["imported"]);
  return work;
}

// The item the server of kind adds for file, the first one after seq.
export async function addedItem(world: World, kind: string, file: string, after = 0) {
  const record = await world.fw.capture(`${kind}-items`).wait({
    where: { event: "added" },
    after,
    timeout: 60_000,
  });
  const item = Item.parse(record.data);
  expect(item.files, `${kind} adds the file`).toEqual([file]);
  return record;
}

export const serverRow = (page: Page, name: string) =>
  page
    .getByRole("list", { name: "Media Servers" })
    .getByRole("link", { name: new RegExp(`^${name}`, "u") });

export async function openServer(page: Page, name: string): Promise<void> {
  await openSettingsSection(page, "Media Servers");
  await serverRow(page, name).click();
  await page.getByRole("heading", { name, level: 1 }).waitFor();
}

// The server page's tab, a row of the page on a phone.
export async function openTab(page: Page, name: string, tab: string): Promise<void> {
  const sections = isPhone(page) ? page.getByRole("main") : page.getByRole("navigation", { name });
  await sections.getByRole("link", { name: tab, exact: true }).click();
}

export async function openChanges(page: Page, name: string): Promise<void> {
  await openServer(page, name);
  await openTab(page, name, "Changes");
}

export const change = (page: Page, name: string, file: string) =>
  page
    .getByRole("list", { name: `Changes on ${name}` })
    .getByRole("listitem")
    .filter({ hasText: file });
