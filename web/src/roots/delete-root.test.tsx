import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { RootsRoot } from "../api/gen/model";
import { liveServer } from "../test/live-server";
import { act, openRoots } from "../test/roots-page";
import { archiveRoot, changedOver, filmsRoot, refused, serveRoots } from "../test/roots";

test("Delete moves the root's Works to the root picked and leaves the folder on disk", async () => {
  const live = liveServer();
  const { sent } = serveRoots({ roots: [filmsRoot, archiveRoot], changed: changedOver(live) });
  openRoots({ handlers: live.handlers });
  await act(archiveRoot, "Delete");
  const dialog = page.getByRole("dialog", { name: `Delete ${archiveRoot.path}?` });
  await expect
    .element(
      dialog.getByText(
        "SlipStream forgets this root. The folder and everything in it stay on disk.",
      ),
    )
    .toBeVisible();
  await expect.element(dialog.getByText("Its 3 Films move to the root you pick.")).toBeVisible();
  await expect
    .element(dialog.getByRole("combobox", { name: "Move its Films to" }))
    .toHaveTextContent(filmsRoot.path);
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect
    .element(page.getByText(`Deleted the ${archiveRoot.path} root. The folder is still on disk.`))
    .toBeVisible();
  expect(sent.deleted).toEqual([{ rootId: archiveRoot.id, worksTo: filmsRoot.id }]);
  await expect.element(page.getByRole("heading", { name: "Films 1", level: 2 })).toBeVisible();
});

test("Delete is refused with the counts while the root holds files or a Grab", async () => {
  const empty: RootsRoot = { ...archiveRoot, works: 0 };
  serveRoots({
    roots: [filmsRoot, empty],
    remove: () =>
      refused(409, "root_in_use", "The root holds files or has a Grab in flight", {
        use: { files: 40, grabs: 1 },
      }),
  });
  openRoots();
  await act(empty, "Delete");
  const dialog = page.getByRole("dialog", { name: `Delete ${empty.path}?` });
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect
    .element(
      dialog.getByText("It can't be deleted while it holds 40 files and has a Grab in flight."),
    )
    .toBeVisible();
});

test("the default root can't be deleted while its type has another root", async () => {
  const { sent } = serveRoots({ roots: [filmsRoot, archiveRoot] });
  openRoots();
  await act(filmsRoot, "Delete");
  const dialog = page.getByRole("dialog", { name: `Delete ${filmsRoot.path}?` });
  await expect
    .element(
      dialog.getByText(
        "New Films go into this root. Make another Films root the default first, then delete this one.",
      ),
    )
    .toBeVisible();
  await expect.element(dialog.getByRole("button", { name: "Delete" })).not.toBeInTheDocument();
  await userEvent.click(dialog.getByRole("button", { name: "Close" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent.deleted).toEqual([]);
});
