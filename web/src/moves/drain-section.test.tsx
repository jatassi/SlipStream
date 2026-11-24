import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { drainWorld, fullerFilms, homeFilms, homeFilmsVolume, seedboxFilms } from "../test/drain";
import { mediaLevels } from "../test/health-levels";
import { homeId, seedboxId } from "../test/nodes";

const dialog = () => page.getByRole("dialog", { name: "Review the change" });
const option = (path: string) =>
  page.getByRole("option", { name: new RegExp(`${path} on home`, "u") });

async function review() {
  await userEvent.click(page.getByRole("checkbox", { name: "Downloading", exact: true }));
  await userEvent.click(page.getByRole("button", { name: "Review" }));
  await expect.element(dialog().getByText(/^seedbox takes no new Storage work/u)).toBeVisible();
}

test("a drain's Review offers to move what the Node keeps, and Apply changes the Roles, then starts the Move", async () => {
  const sent = drainWorld();
  await review();
  await userEvent.click(
    dialog().getByRole("checkbox", { name: "Also move seedbox's library files and torrents" }),
  );
  await expect
    .poll(() => sent.previews)
    .toEqual([
      {
        origin: "drain",
        works: [],
        roots: [{ rootId: seedboxFilms.id, to: homeFilms.id }],
        clients: [{ nodeId: seedboxId, to: { nodeId: homeId } }],
        options: { deleteEmptiedRoots: true, includeLinkedTorrents: true },
      },
    ]);
  await expect
    .element(dialog().getByRole("checkbox", { name: "Delete the roots it empties" }))
    .toBeChecked();
  await expect.element(dialog().getByText(/1 torrent is/u)).toBeVisible();
  await expect.element(dialog().getByText(/3 Works without files/u)).toBeVisible();

  await userEvent.click(dialog().getByRole("button", { name: "Apply" }));
  await expect.element(page.getByText(/2 Moves/u)).toBeVisible();
  expect(sent.order).toEqual(["roles", "move"]);
  expect(sent.starts).toEqual([{ token: "drain-1" }]);
});

test("a drain's Review left alone applies the Roles and starts no Move", async () => {
  const sent = drainWorld();
  await review();
  await userEvent.click(dialog().getByRole("button", { name: "Apply" }));
  await expect.element(dialog()).not.toBeInTheDocument();
  expect(sent.order).toEqual(["roles"]);
  expect(sent.previews).toEqual([]);
});

test("a drain's Review defaults to the root with the most free space above its Warning level, and its picker shows a root's level", async () => {
  const sent = drainWorld([
    { ...mediaLevels, ...homeFilmsVolume, warnBytes: 3 * 1024 ** 4, level: "warn" },
  ]);
  await review();
  await userEvent.click(
    dialog().getByRole("checkbox", { name: "Also move seedbox's library files and torrents" }),
  );
  await expect
    .poll(() => sent.previews.at(-1))
    .toMatchObject({ roots: [{ rootId: seedboxFilms.id, to: fullerFilms.id }] });
  await userEvent.click(dialog().getByRole("combobox", { name: `Move ${seedboxFilms.path} to` }));
  await expect.element(option(homeFilms.path).getByText("Warning", { exact: true })).toBeVisible();
  expect(option(fullerFilms.path).getByText("Warning", { exact: true }).query()).toBeNull();
});
