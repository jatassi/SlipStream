import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { RootsRoot } from "../api/gen/model";
import { act, openRoots } from "../test/roots-page";
import { archiveRoot, filmsRoot, refused, serveRoots } from "../test/roots";

test("the default root can't be deleted while its type has another root", async () => {
  const { sent } = serveRoots({ roots: [filmsRoot, archiveRoot] });
  openRoots();
  await act(filmsRoot, "Delete");
  const dialog = page.getByRole("dialog", {
    name: `Delete ${filmsRoot.path}?`,
  });
  await expect
    .element(
      dialog.getByText(
        "New films go into this root. Make another Films root the default first, then delete this one.",
      ),
    )
    .toBeVisible();
  await expect.element(dialog.getByRole("button", { name: "Delete" })).not.toBeInTheDocument();
  await userEvent.click(dialog.getByRole("button", { name: "Close" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent.deleted).toEqual([]);
});

test("a refused Delete counts the Moves that use the root", async () => {
  const empty: RootsRoot = { ...archiveRoot, works: 0 };
  serveRoots({
    roots: [filmsRoot, empty],
    remove: () =>
      refused(409, "root_in_use", "Root in use", {
        use: { files: 0, grabs: 0, moves: 2 },
      }),
  });
  openRoots();
  await act(empty, "Delete");
  const dialog = page.getByRole("dialog", { name: `Delete ${empty.path}?` });
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect.element(dialog.getByText(/\b2 Moves\b/u)).toBeVisible();
});
