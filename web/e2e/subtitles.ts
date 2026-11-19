import { stat } from "node:fs/promises";
import { join } from "node:path";

import type { Locator, Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect } from "./fixtures.ts";
import { workId } from "./manual-search.ts";
import { menuAction } from "./manual-search-entries.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

/** The subtitles Scenario's films and series. */
export const amber = "Amber Signal";
export const brick = "Brick Lane Echo";
export const cinder = "Cinder Bloom";

const Profiles = z.object({
  profiles: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      contentType: z.string(),
      targets: z.unknown(),
      editions: z.unknown(),
    }),
  ),
});

const WorkSubtitles = z.object({
  files: z.array(
    z.object({
      fileId: z.string(),
      path: z.string(),
      wants: z.array(
        z.object({
          want: z.string(),
          state: z.object({ kind: z.string(), missing: z.string().nullable() }),
        }),
      ),
    }),
  ),
});

async function ok(
  reply: Promise<{ status: number; body: unknown }>,
  what: string,
): Promise<unknown> {
  const { status, body } = await reply;
  if (status >= 300) {
    throw new Error(`${what} = ${status} ${JSON.stringify(body)}`);
  }
  return body;
}

/** Keys SubDL and SubSource, which the fakes take. */
export async function keyProviders(world: World): Promise<void> {
  await Promise.all(
    ["subdl", "subsource"].map((id) =>
      ok(
        world.api("PUT", `/api/v1/subtitles/providers/${id}/key`, {
          key: `e2e-${id}`,
        }),
        `put ${id}'s key`,
      ),
    ),
  );
}

/** Turns a Subtitle Provider off, so no search asks it. */
export async function providerOff(world: World, id: string): Promise<void> {
  await ok(
    world.api("PATCH", `/api/v1/subtitles/providers/${id}`, { on: false }),
    `turn ${id} off`,
  );
}

/** Turns Subtitle search off, so nothing fetches before the test acts. */
export async function searchOff(world: World): Promise<void> {
  await ok(
    world.api("PATCH", "/api/v1/subtitles/settings", {
      values: { "subtitles.searchOn": false },
    }),
    "turn Subtitle search off",
  );
}

async function profileNamed(world: World, profile: string) {
  const listed = Profiles.parse(await ok(world.api("GET", "/api/v1/profiles"), "list Profiles"));
  const found = listed.profiles.find((candidate) => candidate.name === profile);
  if (found === undefined) {
    throw new Error(`no Profile is named ${profile}`);
  }
  return found;
}

/** The id of the Profile named profile. */
export async function profileId(world: World, profile: string): Promise<string> {
  return (await profileNamed(world, profile)).id;
}

/** Puts English, full, whatever the audio, on the Profile through the API. */
export async function wantEnglish(world: World, profile: string): Promise<void> {
  const { id, name, contentType, targets, editions } = await profileNamed(world, profile);
  await ok(
    world.api("PUT", `/api/v1/profiles/${id}?confirm=0&confirmSubtitles=0`, {
      name,
      contentType,
      targets,
      editions,
      languages: [
        {
          language: "en",
          forced: false,
          hearingImpaired: "allow",
          audio: "always",
        },
      ],
    }),
    `put ${profile}'s languages`,
  );
}

/** The Work's first file's English want, as "kind" or "kind missing". */
export async function englishOf(world: World, label: string): Promise<string> {
  const work = await workId(world, label);
  const { files } = WorkSubtitles.parse(
    await ok(world.api("GET", `/api/v1/works/${work}/subtitles`), "read the Work's subtitles"),
  );
  const want = files[0]?.wants.find((candidate) => candidate.want === "en");
  if (want === undefined) {
    return "unwanted";
  }
  return want.state.missing === null ? want.state.kind : `${want.state.kind} ${want.state.missing}`;
}

/** Searches the Work's files' English now, as a row's Search does, which answers 202 with no body. */
export async function searchNow(world: World, label: string): Promise<void> {
  const work = await workId(world, label);
  const { files } = WorkSubtitles.parse(
    await ok(world.api("GET", `/api/v1/works/${work}/subtitles`), "read the Work's subtitles"),
  );
  const statuses = await Promise.all(
    files.map(
      async ({ fileId }) =>
        (
          await fetch(new URL(`/api/v1/files/${fileId}/subtitles/en/search`, world.node.url), {
            method: "POST",
            headers: { "X-Api-Key": world.manifest.apiKey ?? "" },
          })
        ).status,
    ),
  );
  expect(statuses, `searching ${label} now`).toEqual(files.map(() => 202));
}

/** Waits until the Work's audio is scanned, so a pick or an upload can be timed. */
export async function scanned(world: World, label: string): Promise<void> {
  await expect
    .poll(() => englishOf(world, label), {
      timeout: 120_000,
      message: `${label}'s audio is scanned`,
    })
    .not.toMatch(/map_due|unwanted/u);
}

/** The Work's file's subtitle rows. */
export const subtitleRows = (page: Page): Locator =>
  page.getByRole("main").getByRole("list", { name: "Subtitles" }).first();

/** Runs a subtitle row's action: a button on desktop, its "…" menu for Mark bad…, and an action sheet on phone. */
export async function rowAction(page: Page, action: string): Promise<void> {
  const more = "More for English subtitles";
  if (isPhone(page) || action === "Mark bad…") {
    await clickWhenStill(subtitleRows(page).getByRole("button", { name: more }));
    await menuAction(page, more, action);
    return;
  }
  await clickWhenStill(subtitleRows(page).getByRole("button", { name: action, exact: true }));
}

/** The one subtitle dialog or sheet open, Manual search's or Upload's. */
export const subtitleSheet = (page: Page, file: string): Locator =>
  page.getByRole("dialog", { name: `English subtitle for ${file}` });

/** The toast that says words. */
export const toast = (page: Page, words: string): Locator =>
  page.getByRole("region", { name: "Messages" }).getByText(words);

/** Whether a file is at path. */
export const exists = (path: string): Promise<boolean> =>
  stat(path).then(
    () => true,
    () => false,
  );

/** Where Cinder Bloom's English Sidecar lands. */
export const cinderSidecar = (world: World): string =>
  join(
    world.manifest.mediaDir ?? "",
    "movies",
    "Cinder Bloom (2017) {tmdb-9000203}",
    "Cinder.Bloom.2017.1080p.WEB-DL.H264-FKW.en.srt",
  );
