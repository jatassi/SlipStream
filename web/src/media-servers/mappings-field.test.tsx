import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { mediaServersWorld } from "../test/media-server-writes";
import { denJellyfin } from "../test/media-servers";
import { renderApp } from "../test/render-app";
import { archiveRoot, filmsRoot } from "../test/roots";

const path = `/settings/media-servers/${denJellyfin.id}/mappings`;
const editor = () => page.getByRole("form", { name: "Path mappings on Den Jellyfin" });
const archive = () => editor().getByRole("group", { name: "Archive (/mnt/archive/films)" });
const films = () => editor().getByRole("group", { name: "Films (/srv/media/films)" });

test("the editor shows each root's pre-fill, marks a guess until it's edited and offers the server's folders", async () => {
  renderApp({ path, handlers: mediaServersWorld().handlers });
  await expect
    .element(films().getByRole("combobox", { name: "Folder on Den Jellyfin" }))
    .toHaveValue("/data/films");
  await expect.element(films().getByText("In Films.")).toBeVisible();
  expect(films().getByText("Guess").query()).toBeNull();
  await expect.element(archive().getByText("Guess")).toBeVisible();
  await expect
    .element(
      archive().getByText(
        "Den Jellyfin won't get changes for files in Archive (/mnt/archive/films). Confirm the Path mapping, or mark the root as not shown on this server.",
      ),
    )
    .toBeVisible();
  const folder = archive().getByRole("combobox", { name: "Folder on Den Jellyfin" });
  const list = folder.element().getAttribute("list") ?? "";
  expect(
    [...(document.querySelector(`[id="${list}"]`)?.querySelectorAll("option") ?? [])].map(
      (o) => o.value,
    ),
  ).toEqual(["/data/films", "/data/archive"]);

  await userEvent.fill(folder, "/data/archive/old");
  expect(archive().getByText("Guess").query()).toBeNull();
  await expect
    .element(
      archive().getByText(
        "Save checks that the folder is inside one of Den Jellyfin's library folders.",
      ),
    )
    .toBeVisible();
});

test("Save shows a folder outside the libraries on its row, then saves a root marked as not shown", async () => {
  const world = mediaServersWorld();
  renderApp({ path, handlers: world.handlers });
  await userEvent.fill(
    archive().getByRole("combobox", { name: "Folder on Den Jellyfin" }),
    "/srv/elsewhere",
  );
  await userEvent.click(editor().getByRole("button", { name: "Save" }));
  await expect
    .element(
      archive().getByText(
        "/srv/elsewhere isn't inside a library folder on Den Jellyfin. Its library folders are /data/films, /data/archive. Add the folder to a library on the server first.",
      ),
    )
    .toBeVisible();

  await userEvent.click(archive().getByRole("switch", { name: "Shown on Den Jellyfin" }));
  expect(archive().getByRole("combobox").query()).toBeNull();
  await userEvent.click(editor().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved the Path mappings of Den Jellyfin")).toBeVisible();
  expect(world.writes.at(-1)).toEqual({
    path: "mappings",
    body: {
      mappings: [
        { rootId: filmsRoot.id, shown: true, serverFolder: "/data/films" },
        { rootId: archiveRoot.id, shown: false },
      ],
    },
  });
  expect(archive().getByText("Guess").query()).toBeNull();
});
