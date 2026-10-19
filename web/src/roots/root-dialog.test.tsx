import { HttpResponse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../test/live-server";
import { act, gate, openRoots } from "../test/roots-page";
import {
  type RootsWorld,
  changedOver,
  filmsRoot,
  media,
  notStorage,
  refused,
  serveRoots,
} from "../test/roots";

test("Add a library root picks a folder through the browser, checks it on its Node and adds it", async () => {
  const live = liveServer();
  const { sent } = serveRoots({ changed: changedOver(live) });
  openRoots({ handlers: live.handlers });
  await userEvent.click(page.getByRole("button", { name: "Add a library root" }));
  const dialog = page.getByRole("dialog", { name: "Add a library root" });
  await expect
    .element(dialog.getByRole("combobox", { name: "Content Type" }))
    .toHaveTextContent("Zines");
  await expect
    .element(dialog.getByRole("combobox", { name: "Node" }))
    .toHaveTextContent("onlinehome");
  await expect
    .element(dialog.getByRole("link", { name: "its Roles" }))
    .toHaveAttribute("href", `/nodes/${notStorage.id}/roles`);
  await expect
    .element(dialog.getByText("It becomes the default for Zines, where new Zines go."))
    .toBeVisible();
  await expect.element(dialog.getByRole("button", { name: "Add" })).toBeDisabled();

  await userEvent.type(dialog.getByRole("textbox", { name: "Folder" }), "/srv/media");
  await userEvent.click(dialog.getByRole("button", { name: "Browse" }));
  const browser = page.getByRole("dialog", { name: "Choose a folder on home" });
  await userEvent.click(browser.getByRole("option", { name: "zines" }));
  await userEvent.click(browser.getByRole("button", { name: "Use zines" }));
  await expect
    .element(dialog.getByText("This folder is on /srv/media, 1.2 TB free."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Add" }));

  await expect.element(page.getByText("Added /srv/media/zines. New Zines go there.")).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent.created).toEqual([
    { contentType: "zine", nodeId: media.id, path: "/srv/media/zines", makeDefault: false },
  ]);
  await expect.element(page.getByRole("heading", { name: "Zines 1", level: 2 })).toBeVisible();
});

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
  await expect.element(page.getByText("Added /srv/media/more. New Films go there.")).toBeVisible();
  expect(sent.created).toEqual([
    { contentType: "cinema", nodeId: media.id, path: "/srv/media/more", makeDefault: true },
  ]);
});

/** A re-home that reports its check halfway, holds until released, finds 3 files missing, then forgets them. */
function threeMissing(world: RootsWorld, held: Promise<void>): RootsWorld["rehome"] {
  return async (_rootId, body) => {
    if (body.forget === 3) {
      world.roots = [{ ...filmsRoot, path: "/srv/new/films" }];
      return HttpResponse.json(world.roots[0]);
    }
    world.roots = [{ ...filmsRoot, rehome: { checked: 250, total: 500 } }];
    world.changed([filmsRoot.id]);
    await held;
    world.roots = [filmsRoot];
    return refused(409, "rehome_missing", "Recorded files aren't at the new place", {
      missing: {
        count: 3,
        first: [
          { path: "Heat (1995)/Heat (1995).mkv", reason: "missing" },
          { path: "Ran (1985)/Ran (1985).mkv", reason: "size_differs" },
        ],
      },
    });
  };
}

test("Edit re-homes a root: it shows the check's progress, the missing files, and Save anyway forgets them", async () => {
  const live = liveServer();
  const { held, open: release } = gate();
  const { world, sent } = serveRoots({ changed: changedOver(live) });
  world.rehome = threeMissing(world, held);
  openRoots({ handlers: live.handlers });
  await act(filmsRoot, "Edit");
  const dialog = page.getByRole("dialog", { name: "Edit the Films root" });
  await expect.element(dialog.getByRole("button", { name: "Save" })).toBeDisabled();
  const folder = dialog.getByRole("textbox", { name: "Folder" });
  await userEvent.clear(folder);
  await userEvent.type(folder, "/srv/new/films");
  await expect
    .element(dialog.getByText("This folder is on /srv/media, 1.2 TB free."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Save" }));

  await expect.element(dialog.getByText("Checked 250 of 500 files at the new place")).toBeVisible();
  release();
  await expect.element(dialog.getByText("3 files aren't at the new place.")).toBeVisible();
  expect(
    dialog
      .getByRole("list", { name: "Missing files" })
      .getByRole("listitem")
      .all()
      .map((item) => item.element().textContent),
  ).toEqual(["Heat (1995)/Heat (1995).mkvnot there", "Ran (1985)/Ran (1985).mkva different size"]);
  await expect.element(dialog.getByText("And 1 more.")).toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Save anyway, forget 3 files" }));

  await expect
    .element(page.getByText("The Films root is now /srv/new/films on home."))
    .toBeVisible();
  expect(sent.rehomed).toEqual([
    { nodeId: media.id, path: "/srv/new/films" },
    { nodeId: media.id, path: "/srv/new/films", forget: 3 },
  ]);
  expect(sent.checked.at(-1)).toEqual({
    nodeId: media.id,
    path: "/srv/new/films",
    rootId: filmsRoot.id,
  });
});
