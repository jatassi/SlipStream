import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { act, openRoots } from "../test/roots-page";
import { filmsRoot, media, refused, serveRoots } from "../test/roots";

test("a folder the root rules refuse says why in words and keeps Add off", async () => {
  serveRoots({
    check: ({ path: folder }) => ({
      usable: false,
      path: folder,
      refusal: { reason: "inside_root", rootId: filmsRoot.id },
    }),
  });
  openRoots();
  await userEvent.click(page.getByRole("button", { name: "Add a root for Zines" }));
  const dialog = page.getByRole("dialog", { name: "Add a library root" });
  await userEvent.type(dialog.getByRole("textbox", { name: "Folder" }), "/srv/media/films/zines");
  await expect
    .element(dialog.getByText("Can't use this folder because it's inside the Films library root."))
    .toBeVisible();
  await expect.element(dialog.getByRole("button", { name: "Add" })).toBeDisabled();
});

test("a second root of a type offers to become its default", async () => {
  const { sent } = serveRoots();
  openRoots();
  await userEvent.click(page.getByRole("button", { name: "Add a library root" }));
  const dialog = page.getByRole("dialog", { name: "Add a library root" });
  await userEvent.click(dialog.getByRole("combobox", { name: "Content Type" }));
  await userEvent.click(page.getByRole("option", { name: "Films" }));
  await userEvent.click(dialog.getByRole("checkbox", { name: "Make this the default for Films" }));
  await userEvent.type(dialog.getByRole("textbox", { name: "Folder" }), "/srv/media/more");
  await expect
    .element(dialog.getByText("This folder is on /srv/media, 1.2 TB free."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Add" }));
  await expect.element(page.getByText("Added /srv/media/more. New films go there.")).toBeVisible();
  expect(sent.created).toEqual([
    { contentType: "cinema", nodeId: media.id, path: "/srv/media/more", makeDefault: true },
  ]);
});

test("Edit on a root that Moves, Imports and Transfers use says what to wait for", async () => {
  const { world } = serveRoots();
  world.rehome = () =>
    refused(409, "root_busy", "Moves, Imports or Transfers are using this root", {
      busy: { moves: 3, imports: 1, transfers: 2 },
    });
  openRoots();
  await act(filmsRoot, "Edit");
  const dialog = page.getByRole("dialog", { name: "Edit the Films root" });
  const folder = dialog.getByRole("textbox", { name: "Folder" });
  await userEvent.clear(folder);
  await userEvent.type(folder, "/srv/new/films");
  await expect
    .element(dialog.getByText("This folder is on /srv/media, 1.2 TB free."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Save" }));
  await expect
    .element(
      dialog.getByText(
        "The Films root can't be re-homed while 3 Moves, 1 Import and 2 Transfers use it. Wait for them to finish, or cancel the Moves in Activity.",
      ),
    )
    .toBeVisible();
});
