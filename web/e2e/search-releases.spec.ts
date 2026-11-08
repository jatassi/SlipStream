import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "targets" });
test.describe.configure({ timeout: 120_000 });

const Added = z.object({ work: z.object({ id: z.string() }) });
const Named = z.object({ id: z.string() });
const Searches = z.object({ searches: z.array(z.object({ outcome: z.string().nullable() })) });
const Target = z.object({
  id: z.string(),
  name: z.string(),
  tiers: z.array(z.array(z.string())),
  cutoff: z.string(),
  filters: z.array(z.unknown()),
  sizes: z.array(z.unknown()),
  propers: z.string(),
});

const bluray = "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW";
const extended = "The.Quiet.Orchard.2021.EXTENDED.1080p.WEB-DL.H264-FKW";
const superfan = "The.Quiet.Orchard.2021.Superfan.1080p.WEB-DL.H264-FKW";
const webdl = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW";

// Every Target scores a format for the Superfan cut at -10, under its minimum of 0.
async function superfanScoredDown(world: World): Promise<void> {
  const made = await world.api("POST", "/api/v1/custom-formats", {
    name: "Superfan cut",
    conditions: [
      {
        name: "Superfan",
        kind: "ReleaseTitleSpecification",
        negate: false,
        required: false,
        fields: { value: String.raw`\bSuperfan\b` },
      },
    ],
  });
  expect(made.status, "the Superfan cut format is made").toBe(201);
  const formatId = Named.parse(made.body).id;
  const { targets } = z
    .object({ targets: z.array(Target) })
    .parse((await world.api("GET", "/api/v1/targets")).body);
  for (const { id, name, tiers, cutoff, filters, sizes, propers } of targets) {
    // oxlint-disable-next-line no-await-in-loop -- each Target's change converges its Works before the next
    const scored = await world.api("PUT", `/api/v1/targets/${id}`, {
      name,
      tiers,
      cutoff,
      filters,
      sizes,
      propers,
      scores: { minimum: 0, upgradeUntil: 0, increment: 1, formats: [{ formatId, score: -10 }] },
    });
    expect(scored.status, `${name} scores the Superfan cut`).toBe(200);
  }
}

// The Quiet Orchard, searched once fixture-site lists its Bluray, Extended and Superfan Releases.
async function searchedFourReleases(world: World): Promise<string> {
  for (const release of [bluray, extended, superfan]) {
    // oxlint-disable-next-line no-await-in-loop -- fixture-site lists them in this order
    await world.fw.cue("fixture-site-list", { release });
  }
  await superfanScoredDown(world);
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    preset: "all",
  });
  expect(added.status, "The Quiet Orchard is added").toBe(200);
  const workId = Added.parse(added.body).work.id;
  expect((await world.api("POST", `/api/v1/works/${workId}/search`)).status).toBe(202);
  await expect
    .poll(
      async () =>
        Searches.parse((await world.api("GET", `/api/v1/works/${workId}/searches`)).body)
          .searches[0]?.outcome,
      { message: "the search finishes", timeout: 60_000 },
    )
    .toBe("grabbed");
  return workId;
}

test("a search's Details list every Release it saw, the grabbed one first, and why each other lost or was refused", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const workId = await searchedFourReleases(world);
  await page.goto(`/works/${workId}`);
  const run = page.getByRole("list", { name: "Searches" });
  await expect(run).toContainText("Grabbed a Release");
  await run.getByRole("button", { name: "Details" }).click();

  const lines = run.getByRole("list", { name: "Releases" }).locator(":scope > li");
  await expect(lines).toHaveCount(4);
  await expect(lines.nth(0)).toContainText(`Bluray-1080p · FKW · 9.8 GB${bluray}`);
  await expect(lines.nth(0)).toContainText("Grabbed for 1080p and HD.");
  const tier =
    "Its quality is WEBDL-1080p. The grabbed Release is Bluray-1080p, a higher tier at 1080p.";
  await expect(lines.nth(1)).toContainText(extended);
  await expect(lines.nth(1)).toContainText(tier);
  await expect(lines.nth(2)).toContainText(webdl);
  await expect(lines.nth(2)).toContainText(tier);
  const refused = lines.nth(3);
  await expect(refused).toContainText(superfan);
  await expect(refused).toContainText("An automatic search skips it. A Manual search can grab it.");
  await expect(refused.getByRole("list", { name: "Reasons" }).getByRole("listitem")).toHaveText([
    "Score -10 is below the minimum of 0 at 1080p.",
    "Score -10 is below the minimum of 0 at HD.",
  ]);
  await expect(refused).toContainText("Matches Superfan cut. Scores -10 at 1080p and -10 at HD.");

  await expectAccessible(page);
  await refused.scrollIntoViewIfNeeded();
  await screenshotWholePage(page, testInfo.outputPath("releases.png"));

  await refused.getByRole("button", { name: "Open in tester" }).click();
  const tester = page.getByRole("dialog", { name: "Tester" });
  await expect(tester.getByRole("textbox", { name: "Release names" })).toHaveValue(superfan);
  await expect(tester).toContainText("Each name is judged against The Quiet Orchard");
  await expect(tester).toContainText("Score -10 is below the minimum of 0 at");
  await expect(page).toHaveURL(new RegExp(`/works/${workId}$`, "u"));
  await expectAccessible(page);
});
