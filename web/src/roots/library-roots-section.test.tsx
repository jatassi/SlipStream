import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { RootsRoot } from "../api/gen/model";
import { act, openRoots } from "../test/roots-page";
import { asPhone } from "../test/viewport";
import {
  archiveRoot,
  filmsRoot,
  mediaVolume,
  mediaVolumes,
  notStorage,
  serveRoots,
} from "../test/roots";

const rowTexts = (table: string) =>
  page
    .getByRole("table", { name: table })
    .getByRole("row")
    .all()
    .slice(1)
    .map((row) => row.element().textContent);

test("Library roots lists each Content Type's roots with the default, the Node, free space and what each holds", async () => {
  serveRoots({ roots: [filmsRoot, archiveRoot] });
  openRoots();
  await expect.element(page.getByRole("heading", { name: "Films 2", level: 2 })).toBeVisible();
  expect(rowTexts("Films roots")).toEqual([
    "/srv/media/filmsDefaultonlinehome1.2 TB free12 Films, 910 GB",
    "/mnt/archive/filmsThe folder is there but looks unmountedonlinehomeFree space not read yet3 Films, 0 bytes",
  ]);
  await expect.element(page.getByRole("heading", { name: "Zines 0", level: 2 })).toBeVisible();
  await expect.element(page.getByText("Add a library root to hold Zines.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add a root for Zines" })).toBeVisible();
});

test("the Storage block shows each volume's used space, free of total and its roots, and an offline Node's as of", async () => {
  serveRoots({
    volumes: [
      mediaVolumes,
      {
        nodeId: notStorage.id,
        live: false,
        volumes: [
          {
            ...mediaVolume,
            id: "nas",
            name: "\\\\nas\\media",
            kind: "network",
            total: 4 * 1024 ** 4,
            free: 2 * 1024 ** 4,
            rootIds: [],
          },
        ],
      },
    ],
  });
  openRoots();
  const home = page.getByRole("list", { name: "Volumes on home" });
  await expect.element(home.getByText("1.2 TB free of 8.0 TB")).toBeVisible();
  await expect.element(home.getByText("/srv/media/films")).toBeVisible();
  await expect
    .element(home.getByRole("meter", { name: "/srv/media used" }))
    .toHaveAttribute("aria-valuenow", String(6.8 * 1024 ** 4));
  const seedbox = page.getByRole("list", { name: "Volumes on seedbox" });
  await expect.element(seedbox.getByText("Network")).toBeVisible();
  await expect.element(seedbox.getByText(/^2\.0 TB free of 4\.0 TB as of /u)).toBeVisible();
});

test("Make default makes another root where its type's new Works go", async () => {
  const { sent } = serveRoots({ roots: [filmsRoot, archiveRoot] });
  openRoots();
  await expect
    .element(page.getByRole("button", { name: `Actions for ${filmsRoot.path}` }))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: `Actions for ${filmsRoot.path}` }));
  await expect
    .poll(() =>
      page
        .getByRole("menuitem")
        .all()
        .map((item) => item.element().textContent),
    )
    .toEqual(["Edit", "Delete"]);
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("menu")).not.toBeInTheDocument();
  await act(archiveRoot, "Make default");
  await expect.element(page.getByText("New Films now go into /mnt/archive/films.")).toBeVisible();
  expect(sent.defaulted).toEqual([archiveRoot.id]);
});

test("a root on a written-off Node says so and leads its menu with Re-home", async () => {
  const lost: RootsRoot = { ...archiveRoot, state: "written_off" };
  serveRoots({ roots: [filmsRoot, lost] });
  openRoots();
  await expect.element(page.getByText("Node written off")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: `Actions for ${lost.path}` }));
  await expect
    .poll(() =>
      page
        .getByRole("menuitem")
        .all()
        .map((item) => item.element().textContent),
    )
    .toEqual(["Re-home", "Make default", "Delete"]);
  await userEvent.click(page.getByRole("menuitem", { name: "Re-home" }));
  await expect.element(page.getByRole("dialog", { name: "Edit the Films root" })).toBeVisible();
});

test("deleting a type's last root leaves its Works waiting without a root", async () => {
  const { sent } = serveRoots();
  openRoots();
  await act(filmsRoot, "Delete");
  const dialog = page.getByRole("dialog", { name: `Delete ${filmsRoot.path}?` });
  await expect
    .element(dialog.getByText("Its 12 Films wait without a root until you add another Films root."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent.deleted).toEqual([{ rootId: filmsRoot.id, worksTo: null }]);
});

test("Check storage every sits under Details beside the levels and saves seconds", async () => {
  const { sent } = serveRoots();
  openRoots();
  await expect
    .element(
      page.getByText(
        "A volume shows a warning below 100 GB free and an error below 20 GB. Storage Nodes check each volume every 5 min, and again after each import.",
      ),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  const every = page.getByRole("textbox", { name: "Check storage every" });
  await userEvent.clear(every);
  await userEvent.type(every, "15");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(
      page.getByText(
        "A volume shows a warning below 100 GB free and an error below 20 GB. Storage Nodes check each volume every 15 min, and again after each import.",
      ),
    )
    .toBeVisible();
  expect(sent.sampleEvery).toEqual([900]);
});

test("on phone the section is a pushed page with one root per row, and Add opens as a sheet", async () => {
  await asPhone();
  serveRoots({ roots: [filmsRoot, archiveRoot] });
  openRoots();
  await expect.element(page.getByRole("heading", { name: "Films 2", level: 2 })).toBeVisible();
  await expect
    .element(page.getByRole("banner").getByRole("link", { name: "Settings" }))
    .toBeVisible();
  await expect.element(page.getByText("The folder is there but looks unmounted")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Add a library root" }));
  const sheet = page.getByRole("dialog", { name: "Add a library root" });
  await expect.element(sheet.getByRole("textbox", { name: "Folder" })).toBeVisible();
  await expect.element(sheet.getByRole("textbox", { name: "Folder" })).not.toHaveFocus();
});

test("Library roots leads with a switch for each kind of media", async () => {
  serveRoots({ roots: [filmsRoot] });
  openRoots();
  const kinds = page.getByRole("list", { name: "Kinds of media" });
  await expect.element(kinds.getByRole("switch", { name: "Films" })).toBeChecked();
  await expect.element(kinds.getByRole("switch", { name: "Zines" })).toBeChecked();
});
