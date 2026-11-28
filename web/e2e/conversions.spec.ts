import { z } from "zod";

import { type World, expect, test } from "./fixtures.ts";
import { add } from "./library.ts";

test.use({ scenario: "conversion" });
test.describe.configure({ timeout: 180_000 });

const Counts = z.object({
  counts: z.object({ waiting: z.number(), converting: z.number() }),
});
const Played = z.object({
  data: z.object({ result: z.object({ session: z.string() }) }),
});
const Cells = z.object({
  cells: z.array(z.object({ targetName: z.string(), state: z.string() })),
});

const movies = {
  "9000001": "The Quiet Orchard",
  "9000005": "The Salt Road",
  "9000006": "Meadowlark",
};

// Living room Plex transcodes video on home while every movie is added and
// searched as a script would, so each Conversion waits. Answers the play's
// session and the Works.
async function waiting(world: World) {
  const played = Played.parse(
    await world.fw.cue("plex-play", {
      path: "/media/shelf/Grandma's 80th Birthday (2019).mkv",
      user: "plex-admin",
      player: "Office PC",
      playerKey: "office-pc",
      video: "transcode",
      audio: "direct",
      subtitles: "none",
    }),
  );
  const works = await Promise.all(
    Object.keys(movies).map(async (key) => {
      const workId = await add(world, key, "all");
      expect((await world.api("POST", `/api/v1/works/${workId}/search`)).status).toBe(202);
      return workId;
    }),
  );
  await expect
    .poll(
      async () => Counts.parse((await world.api("GET", "/api/v1/conversions")).body).counts.waiting,
      { message: "every Conversion waits for the transcode", timeout: 90_000 },
    )
    .toBe(works.length);
  return { session: played.data.result.session, works };
}

test("the Conversions tab shows each band, and Move up moves a queued Conversion", async ({
  adminPage: page,
  world,
}) => {
  const { session, works } = await waiting(world);
  await page.goto("/activity/conversions");
  const show = page.getByRole("radiogroup", { name: "Show" });
  await expect(show.getByRole("radio", { name: /Up next/u })).toContainText("3");
  for (const title of Object.values(movies)) {
    // oxlint-disable-next-line no-await-in-loop -- each row is checked in turn
    await expect(
      page.getByRole("list", { name: "New" }).getByRole("link", { name: title }),
    ).toBeVisible();
  }
  const moveUp = page.getByRole("button", { name: /^Move up /u });
  await expect(moveUp, "only the Conversion not yet placed can move").toHaveCount(1);
  const queued = (await moveUp.getAttribute("aria-label"))?.replace(/^Move up /u, "");
  await moveUp.click();
  await expect(page.getByRole("list", { name: "Moved up" }).getByRole("link")).toHaveText([
    queued ?? "",
  ]);

  await world.fw.cue("plex-stop", { session });
  const converting = page.getByRole("region", {
    name: "Converting",
    exact: true,
  });
  await expect(converting.getByRole("link", { name: "Meadowlark" })).toBeVisible({
    timeout: 60_000,
  });
  await expect(show.getByRole("radio", { name: /Converting/u })).toContainText("1");
  for (const workId of works) {
    // oxlint-disable-next-line no-await-in-loop -- each Work's cells are read in turn
    await expect
      .poll(
        async () =>
          Cells.parse((await world.api("GET", `/api/v1/works/${workId}/cells`)).body).cells.map(
            (cell) => `${cell.targetName} ${cell.state}`,
          ),
        { message: "the made file fills 720p mobile", timeout: 90_000 },
      )
      .toContain("720p mobile met");
  }
});
