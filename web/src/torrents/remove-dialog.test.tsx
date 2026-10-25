import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { DownloadclientsPreview, DownloadclientsTorrentRow } from "../api/gen/model";
import { downloading as grab } from "../test/grabs";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { actionsServer, harvestId, orchardId, torrent, torrentsServer } from "../test/torrents";

const orchard = torrent({
  state: "downloading",
  tabs: ["downloading", "all"],
  workId: orchardId,
  name: "The.Quiet.Orchard.2021.1080p.WEB-DL",
});
const harvest = torrent({
  state: "seeding",
  tabs: ["seeding", "all"],
  workId: harvestId,
  name: "Orchard.Harvest.2019.720p.HDTV",
  done: 1_000_000_000,
});
const stalled = torrent({
  state: "stalled",
  severity: "warning",
  tabs: ["downloading", "problems", "all"],
  name: "Dead.Release.2020.1080p",
});

const table = () => page.getByRole("table", { name: "Torrents" });
const notices = () => page.getByRole("region", { name: "Notifications" });

function open(
  rows: readonly DownloadclientsTorrentRow[],
  actions: ReturnType<typeof actionsServer>,
) {
  renderApp({
    path: "/activity/torrents?tab=all",
    handlers: [actions.handler, ...liveServer().handlers, ...torrentsServer(rows).handlers],
  });
}

async function removeFromMenu(name: string) {
  await userEvent.click(table().getByRole("button", { name: `${name} actions` }));
  await userEvent.click(page.getByRole("menuitem", { name: "Remove" }));
}

const shared: DownloadclientsPreview = {
  cancels: [{ grabId: grab.id, workId: orchardId, release: grab.releaseTitle }],
  stopSeeding: [harvestId],
  frees: 15_246_000_000,
  linked: false,
};

test("remove names the torrent, keeps library files, and lists what it takes with it", async () => {
  const actions = actionsServer({ preview: shared });
  open([orchard, harvest], actions);
  await removeFromMenu("The Quiet Orchard");
  const dialog = page.getByRole("dialog", { name: "Remove The Quiet Orchard?" });
  await expect
    .element(dialog)
    .toMatchTextContent(
      "Files in your library stay." +
        "It cancels this Grab, still downloading, and its Item goes back to wanted:" +
        "The Quiet Orchard" +
        "The.Quiet.Orchard.2021.1080p.WEB-DL" +
        "Orchard Harvest stops seeding.",
    );
  const copy = dialog.getByRole("checkbox", {
    name: "Also delete the download copy (frees 14 GB)",
  });
  await expect.element(copy).not.toBeChecked();
  expect(actions.bodies).toEqual([{ action: "remove", ids: [orchard.id], dryRun: true }]);

  await userEvent.click(dialog.getByRole("button", { name: "Remove" }));
  await expect.element(dialog).not.toBeInTheDocument();
  await expect.element(notices().getByText("Removing The Quiet Orchard.")).toBeVisible();
  expect(actions.bodies.at(-1)).toEqual({ action: "remove", ids: [orchard.id] });
});

test("the delete choice removes the download copy too", async () => {
  const actions = actionsServer({ preview: shared });
  open([orchard], actions);
  await removeFromMenu("The Quiet Orchard");
  const dialog = page.getByRole("dialog", { name: "Remove The Quiet Orchard?" });
  await userEvent.click(dialog.getByRole("checkbox", { name: /^Also delete the download copy/u }));
  await userEvent.click(dialog.getByRole("button", { name: "Remove" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(actions.bodies.at(-1)).toEqual({ action: "remove", ids: [orchard.id], deleteFiles: true });
});

test("a copy the library shares says it frees little, and an offline Node can't say", async () => {
  open([orchard], actionsServer({ preview: { ...shared, frees: 2048, linked: true } }));
  await removeFromMenu("The Quiet Orchard");
  const dialog = page.getByRole("dialog", { name: "Remove The Quiet Orchard?" });
  await expect
    .element(dialog.getByRole("checkbox", { name: "Also delete the download copy (frees 2.0 KB)" }))
    .toBeVisible();
  await expect
    .element(dialog)
    .toMatchTextContent(
      "Your library shares some of these files, so the copy frees only what the library doesn't use.",
    );
});

test("with a Node that can't count, the choice stands without a size", async () => {
  open([orchard], actionsServer({ preview: { ...shared, frees: null } }));
  await removeFromMenu("The Quiet Orchard");
  const dialog = page.getByRole("dialog", { name: "Remove The Quiet Orchard?" });
  await expect
    .element(dialog.getByRole("checkbox", { name: "Also delete the download copy" }))
    .toBeVisible();
  await expect
    .element(dialog)
    .toMatchTextContent("The space this frees is unknown because its Node didn't respond.");
});

test("a torrent being imported refuses the remove in plain words", async () => {
  const actions = actionsServer({ refuse: () => ({ code: "importing", workId: orchardId }) });
  open([orchard], actions);
  await removeFromMenu("The Quiet Orchard");
  const dialog = page.getByRole("dialog", { name: "Remove The Quiet Orchard?" });
  await expect
    .element(dialog)
    .toMatchTextContent(
      "SlipStream is finishing the import of The Quiet Orchard. Try again once it's done.",
    );
  await expect.element(dialog.getByRole("button", { name: "Remove" })).toBeDisabled();
});

test("picked rows pause together and a bulk remove names the count", async () => {
  const actions = actionsServer({ preview: { ...shared, cancels: [], stopSeeding: [] } });
  open([orchard, harvest, stalled], actions);
  await userEvent.click(table().getByRole("checkbox", { name: "Pick The Quiet Orchard" }));
  await userEvent.click(table().getByRole("checkbox", { name: "Pick Dead.Release.2020.1080p" }), {
    modifiers: ["Shift"],
  });
  const bar = page.getByRole("toolbar", { name: "Picked torrents" });
  await expect.element(bar).toMatchTextContent("3 torrents picked");

  await userEvent.click(bar.getByRole("button", { name: "Pause" }));
  await expect.element(notices().getByText("Pausing 3 torrents.")).toBeVisible();
  expect(actions.bodies).toEqual([{ action: "pause", ids: [orchard.id, harvest.id, stalled.id] }]);

  await userEvent.click(bar.getByRole("button", { name: "Remove" }));
  const dialog = page.getByRole("dialog", { name: "Remove 3 torrents?" });
  await expect.element(dialog).toMatchTextContent("Files in your library stay.");
  await userEvent.click(dialog.getByRole("button", { name: "Remove" }));
  await expect.element(notices().getByText("Removing 3 torrents.")).toBeVisible();

  await userEvent.click(bar.getByRole("button", { name: "Clear" }));
  await expect.element(bar).not.toBeInTheDocument();
});

test("mark failed on a stalled download deletes its copy after asking", async () => {
  const actions = actionsServer({ preview: { ...shared, stopSeeding: [], frees: 1024 * 1024 } });
  open([stalled], actions);
  await userEvent.click(table().getByRole("button", { name: "Dead.Release.2020.1080p actions" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Mark failed" }));
  const dialog = page.getByRole("dialog", { name: "Mark Dead.Release.2020.1080p failed?" });
  await expect
    .element(dialog)
    .toMatchTextContent(
      "It is removed and its download copy deleted (frees 1.0 MB). Files in your library stay.",
    );
  await expect.element(dialog.getByRole("checkbox")).not.toBeInTheDocument();
  await userEvent.click(dialog.getByRole("button", { name: "Mark failed" }));
  await expect.element(notices().getByText("Marked Dead.Release.2020.1080p failed.")).toBeVisible();
  expect(actions.bodies.at(-1)).toEqual({
    action: "mark-failed",
    ids: [stalled.id],
    deleteFiles: true,
  });
});
