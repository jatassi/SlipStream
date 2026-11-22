import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { openRoots } from "../test/roots-page";
import { filmsRoot, media, serveRoots } from "../test/roots";

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

/** A re-home that reports its check halfway, holds until released, finds 3 files missing, then forgets them. */
