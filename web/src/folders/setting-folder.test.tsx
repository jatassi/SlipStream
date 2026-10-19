import { Suspense } from "react";
import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingFolderField } from "../api/gen/model";
import { serveFolderSetting as serve } from "../test/folder-setting";
import { renderQueried } from "../test/folders";
import { media } from "../test/roots";
import { SettingField } from "../ui/setting-field";

const downloads: SettingFolderField = {
  key: "torrent.downloads",
  label: "Downloads folder",
  kind: "folder",
  source: "config.yaml",
  applies: "restart",
  pendingRestart: false,
  isDefault: false,
  value: "/srv/torrents",
  running: "/srv/torrents",
  default: "/var/lib/slipstream/downloads",
  nodeId: media.id,
  purpose: "downloads",
};

function render(field: SettingFolderField) {
  const onSave = vi.fn<(value: unknown) => Promise<unknown>>(() => Promise.resolve());
  renderQueried(
    <Suspense>
      <SettingField field={field} onSave={onSave} />
    </Suspense>,
  );
  return onSave;
}

const folder = () => page.getByRole("textbox", { name: "Downloads folder" });

test("a folder setting is a path on its Node, judged there for its purpose", async () => {
  const asked = serve();
  render(downloads);
  await expect.element(folder()).toHaveValue("/srv/torrents");
  await expect.element(page.getByText("This folder is on /srv/media, 1.2 TB free.")).toBeVisible();
  await expect.element(page.getByText("Default /var/lib/slipstream/downloads.")).toBeVisible();
  await expect.element(page.getByText("Applies after restart.")).toBeVisible();

  await userEvent.clear(folder());
  await userEvent.type(folder(), "/srv/media/films/new");
  await expect
    .element(page.getByText("Can't use this folder because it's inside the Films library root."))
    .toBeVisible();
  expect(asked.at(-1)).toEqual({ path: "/srv/media/films/new", purpose: "downloads" });
});

test("Browse picks a folder on the setting's Node and Save sends its path", async () => {
  serve();
  const onSave = render(downloads);
  expect(page.getByRole("button", { name: "Save" }).query()).toBeNull();
  await userEvent.clear(folder());
  await userEvent.type(folder(), "/srv/media");
  await userEvent.click(page.getByRole("button", { name: "Browse" }));
  const dialog = page.getByRole("dialog", { name: "Choose a folder on home" });
  await userEvent.click(dialog.getByRole("option", { name: "incoming" }));
  await userEvent.click(dialog.getByRole("button", { name: "Use incoming" }));
  await expect.element(folder()).toHaveValue("/srv/media/incoming");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  expect(onSave).toHaveBeenCalledExactlyOnceWith("/srv/media/incoming");
});

test("an empty folder is refused before any request", async () => {
  serve();
  const onSave = render(downloads);
  await userEvent.clear(folder());
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Choose a folder.");
  expect(onSave).not.toHaveBeenCalled();
});

test("a folder set by the environment can be neither typed nor browsed", async () => {
  serve();
  render({ ...downloads, source: "env", lockedBy: "SLIPSTREAM_TORRENT_DOWNLOADS" });
  await expect.element(folder()).toBeDisabled();
  await expect.element(page.getByRole("button", { name: "Browse" })).toBeDisabled();
  await expect.element(page.getByText("SLIPSTREAM_TORRENT_DOWNLOADS")).toBeVisible();
});
