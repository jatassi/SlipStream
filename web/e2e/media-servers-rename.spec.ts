import { z } from "zod";

import { front } from "../harness/manifest.ts";
import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import {
  addedItem,
  change,
  imported,
  openChanges,
  openServer,
  openTab,
  orchard,
  serverRow,
  settles,
} from "./media-servers.ts";
import { openSettingsSection } from "./shell.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "media-servers" });
test.describe.configure({ timeout: 360_000 });

const renamed = "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p [WEBDL-1080p].mkv";
const withQuality = "{Movie Title} {(Year)} - {Edition Tag} {Target} {[Quality Full]}";

const Plan = z.object({ digest: z.string() });
const JellyfinItem = z.object({ id: z.string() });
const SignedIn = z.object({ AccessToken: z.string() });
const UserData = z.object({ Played: z.boolean(), PlayCount: z.number() });

// Signs in to Den Jellyfin as user, as that person's app would.
async function jellyfinUser(world: World, user: string, password: string) {
  const response = await fetch(
    `${front(world.manifest, "jellyfin", "world")}/Users/AuthenticateByName`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `MediaBrowser Client="Playwright", Device="Test", DeviceId="e2e-${user}", Version="1"`,
      },
      body: JSON.stringify({ Username: user, Pw: password }),
    },
  );
  return { name: user, token: SignedIn.parse(await response.json()).AccessToken };
}

async function markPlayed(world: World, token: string, item: string): Promise<void> {
  const at = `${front(world.manifest, "jellyfin", "world")}/UserPlayedItems/${item}?ApiKey=${token}`;
  expect((await fetch(at, { method: "POST" })).ok).toBe(true);
}

async function userData(world: World, token: string, item: string) {
  const at = `${front(world.manifest, "jellyfin", "world")}/UserItems/${item}/UserData?ApiKey=${token}`;
  return UserData.parse(await (await fetch(at)).json());
}

// The Work's file renamed under a naming template with its quality, as a
// script confirms the plan.
async function renameWithQuality(world: World, work: string): Promise<void> {
  const naming = await world.api("PATCH", "/api/v1/naming/movie", {
    values: { "movie.naming.file.standard": withQuality },
  });
  expect(naming.status).toBe(200);
  const { digest } = Plan.parse((await world.api("POST", `/api/v1/renames/plan/${work}`)).body);
  const renames = await world.api("POST", "/api/v1/renames", { works: [{ work, digest }] });
  expect(renames.status).toBe(200);
}

test("a rename on Den Jellyfin keeps both users' watch data", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const work = await imported(world, "den-movies");
  const first = await addedItem(world, "jellyfin", `/media/den-movies/${orchard}`);
  const before = JellyfinItem.parse(first.data).id;
  await openChanges(page, "Den Jellyfin");
  await expect(change(page, "Den Jellyfin", "The Quiet Orchard (2021) - 1080p.mkv")).toContainText(
    "Sent the new file.",
    { timeout: settles },
  );
  const users = await Promise.all([
    jellyfinUser(world, "admin", "fake-world-password"),
    jellyfinUser(world, "grace", ""),
  ]);
  await Promise.all(users.map(({ token }) => markPlayed(world, token, before)));

  await renameWithQuality(world, work);
  const rename = change(page, "Den Jellyfin", "The Quiet Orchard (2021) - 1080p [WEBDL-1080p].mkv");
  await expect(rename).toContainText("Rename");
  await expect(rename).toContainText(
    "Sent the rename. Kept who watched it and each user's play count, resume position, favorite and rating.",
    {
      timeout: settles,
    },
  );
  await expect(rename).toContainText(`From ${orchard}`);
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("jellyfin-rename.png"));

  const moved = await addedItem(world, "jellyfin", `/media/den-movies/${renamed}`, first.seq);
  const after = JellyfinItem.parse(moved.data).id;
  expect(after, "Jellyfin gives the renamed file a new item").not.toBe(before);
  const kept = await Promise.all(users.map(({ token }) => userData(world, token, after)));
  expect(kept, "admin's and grace's watch data").toMatchObject([
    { Played: true, PlayCount: 1 },
    { Played: true, PlayCount: 1 },
  ]);
});

test("with den stopped, the Controller refreshes Den Jellyfin through its fallback address", async ({
  adminPage: page,
  world,
}) => {
  await world.fw.node("stop", "den");
  await openSettingsSection(page, "Media Servers");
  await expect(serverRow(page, "Den Jellyfin")).toContainText(
    "Den Jellyfin can't be checked while den is offline. SlipStream sends changes through the Controller until den is back.",
  );
  await openServer(page, "Den Jellyfin");
  const scans = world.fw.capture("jellyfin-scans");
  const after = (await scans.list()).at(-1)?.seq ?? 0;
  await page.getByRole("button", { name: "Refresh libraries" }).click();
  await page
    .getByRole("dialog", { name: "Refresh every library on Den Jellyfin?" })
    .getByRole("button", { name: "Refresh" })
    .click();
  await openTab(page, "Den Jellyfin", "Changes");
  const refreshes = page
    .getByRole("list", { name: "Changes on Den Jellyfin" })
    .getByRole("listitem")
    .filter({ hasText: "Whole library" });
  // The refresh waits out Wait before sending, unless it folded into the
  // one the world's saved Path mappings queued at its start.
  await expect(refreshes.first()).toContainText("Refreshed the whole library.", {
    timeout: settles,
  });
  await scans.wait({ node: "home", after, timeout: 60_000 });
  await expect(page.getByText("Every recent change reached Den Jellyfin.")).toBeVisible();
  await expectAccessible(page);
});
