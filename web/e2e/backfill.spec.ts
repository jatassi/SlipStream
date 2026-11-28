import { z } from "zod";

import { type World, expect, test } from "./fixtures.ts";
import { add } from "./library.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "conversion" });
test.describe.configure({ timeout: 180_000 });

const Named = z.object({ id: z.string(), name: z.string() });
const Targets = z.object({ targets: z.array(Named) });
const Profiles = z.object({ profiles: z.array(Named) });
const Cells = z.object({ cells: z.array(z.object({ state: z.string() })) });
const Conversions = z.object({
  next: z.array(
    z.object({
      lane: z.string(),
      rows: z.array(z.object({ kind: z.string() })),
    }),
  ),
});

function id(list: readonly z.infer<typeof Named>[], name: string) {
  const found = list.find((entry) => entry.name === name);
  if (found === undefined) {
    throw new Error(`the Scenario has no ${name}`);
  }
  return found.id;
}

async function ids(world: World) {
  const { targets } = Targets.parse((await world.api("GET", "/api/v1/targets")).body);
  const { profiles } = Profiles.parse((await world.api("GET", "/api/v1/profiles")).body);
  return { starter: id(targets, "1080p"), films: id(profiles, "Films") };
}

// Films holds only the starter while The Quiet Orchard and The Salt Road
// are added and searched as a script would, so each has a 1080p file and
// nothing made from it.
async function downloadedOnly(world: World) {
  const { starter, films } = await ids(world);
  const held = await world.api("PUT", `/api/v1/profiles/${films}`, {
    name: "Films",
    contentType: "movie",
    targets: [{ targetId: starter, sourcing: "download" }],
    editions: { kind: "any" },
    languages: [],
  });
  expect(held.status, "Films holds only the starter").toBe(200);
  for (const key of ["9000001", "9000005"]) {
    // oxlint-disable-next-line no-await-in-loop -- each Work is added and imported in turn
    const workId = await add(world, key, "all");
    // oxlint-disable-next-line no-await-in-loop -- as above
    expect((await world.api("POST", `/api/v1/works/${workId}/search`)).status).toBe(202);
    // oxlint-disable-next-line no-await-in-loop -- as above
    await expect
      .poll(
        async () =>
          Cells.parse((await world.api("GET", `/api/v1/works/${workId}/cells`)).body).cells.every(
            (cell) => cell.state === "upgradable",
          ),
        { message: "the WEB-DL is imported", timeout: 60_000 },
      )
      .toBe(true);
  }
  return films;
}

test("adding a made-from Target to a Profile with files asks what to convert in three steps", async ({
  adminPage: page,
  world,
}) => {
  const films = await downloadedOnly(world);
  await page.goto(`/settings/profiles/${films}`);
  await page.getByRole("button", { name: "Add Target" }).click();
  await clickWhenStill(
    page.getByRole(isPhone(page) ? "button" : "menuitem", {
      name: "720p mobile",
      exact: true,
    }),
  );
  await page.getByRole("combobox", { name: "How 720p mobile gets its file" }).click();
  await clickWhenStill(page.getByRole("option", { name: "Made from 1080p", exact: true }));
  await page.getByRole("button", { name: "Save", exact: true }).click();

  const dialog = page.getByRole("dialog", {
    name: "Convert existing Movies to 720p mobile",
  });
  const steps = dialog.getByRole("list", { name: "Steps" }).getByRole("listitem");
  await expect(steps).toHaveText(["1. Cost", "2. Movies", "3. The rest"]);
  await expect(steps.nth(0)).toHaveAttribute("aria-current", "step");
  await expect(dialog.getByRole("list", { name: "What changes" })).toBeVisible();
  await dialog.getByRole("button", { name: "Next" }).click();
  await expect(dialog.getByRole("radio", { name: "Convert all" })).toBeChecked();
  await dialog.getByRole("button", { name: "Next" }).click();
  await expect(steps.nth(2)).toHaveAttribute("aria-current", "step");
  await expect(dialog.getByRole("status")).toContainText("Save Films with 720p mobile.");
  await dialog.getByRole("button", { name: "Save", exact: true }).click();
  await expect(dialog).toBeHidden();

  await expect
    .poll(
      async () =>
        Conversions.parse((await world.api("GET", "/api/v1/conversions")).body).next.flatMap(
          (lane) => lane.rows.map(() => lane.lane),
        ),
      { message: "both Works' Conversions queue in Backfill", timeout: 30_000 },
    )
    .toEqual(expect.arrayContaining(["backfill"]));
});
