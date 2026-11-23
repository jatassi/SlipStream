import { readdir, stat } from "node:fs/promises";
import { join } from "node:path";

import { z } from "zod";

import { workAction } from "./cells.ts";
import { expect, expectAccessible, test } from "./fixtures.ts";
import { imported, orchard, toasts } from "./library.ts";
import { clickWhenStill } from "./still.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "moves" });
test.describe.configure({ timeout: 360_000 });

const Batch = z.object({
  units: z.array(z.object({ kind: z.string(), name: z.string(), phase: z.string() })),
});
const Moves = z.object({ batches: z.array(z.object({ id: z.string() })) });

// The video files under folder, by their paths.
async function films(folder: string): Promise<string[]> {
  const entries = await readdir(folder, { recursive: true, withFileTypes: true });
  return entries
    .filter((entry) => entry.isFile() && entry.name.endsWith(".mkv"))
    .map((entry) => join(entry.parentPath, entry.name));
}

test("a Work moves from seedbox to den with its linked torrent, its file still a hard link of the payload", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const { mediaDir } = world.manifest;
  if (mediaDir === undefined) {
    throw new Error("the moves Scenario has no media folder");
  }
  const workId = await imported(world);
  const [before] = await films(join(mediaDir, "movies-d"));
  expect(before, "the import placed the film in seedbox's movies-d").toBeDefined();
  expect((await stat(before ?? "")).nlink, "the film is a hard link of seedbox's payload").toBe(2);

  await page.goto(`/works/${workId}`);
  await workAction(page, "Move to…", orchard);
  const dialog = page.getByRole("dialog", { name: `Move ${orchard}` });
  await dialog.getByRole("combobox", { name: "Move to" }).click();
  await expect(
    page.getByRole("option", { name: /movies-d on seedbox/u }),
    "the root the film is in isn't a destination",
  ).toHaveCount(0);
  await clickWhenStill(page.getByRole("option", { name: /movies-e on den/u }));
  await expect(dialog.getByRole("checkbox", { name: "Also move its torrent" })).toBeChecked();
  await expect(dialog.getByText("Move 1 movie and 1 torrent to den.")).toBeVisible();
  await dialog.getByRole("button", { name: "Details" }).click();
  await expect(
    dialog.getByRole("listitem").filter({ hasText: "the built-in client on den" }),
    "Details names the clients the torrent copies between",
  ).toContainText("the built-in client on seedbox");
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("preview.png"));
  await dialog.getByRole("button", { name: /^Move \d/u }).click();
  await expect(toasts(page)).toContainText("Moving 1 movie and 1 torrent to den.");

  const { batches } = Moves.parse((await world.api("GET", "/api/v1/moves")).body);
  const [batch] = batches;
  expect(batch, "the Start made one batch").toBeDefined();
  await expect
    .poll(
      async () =>
        Batch.parse((await world.api("GET", `/api/v1/moves/${batch?.id ?? ""}`)).body).units.map(
          ({ kind, phase }) => `${kind}:${phase}`,
        ),
      { message: "the torrent and the Work both move", timeout: 240_000, intervals: [2000] },
    )
    .toEqual(["torrent:done", "work:done"]);

  expect(await films(join(mediaDir, "movies-d")), "seedbox's movies-d keeps no film").toEqual([]);
  const [after] = await films(join(mediaDir, "movies-e"));
  expect(after, "the film is in den's movies-e").toBeDefined();
  expect((await stat(after ?? "")).nlink, "den's film is a hard link of den's payload").toBe(2);
  await screenshotWholePage(page, testInfo.outputPath("moved.png"));
});
