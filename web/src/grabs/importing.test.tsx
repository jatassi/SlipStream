import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsGrab } from "../api/gen/model";
import { getListGrabsMockHandler } from "../api/gen/query/grabs/grabs.msw";
import { getListIndexersMockHandler } from "../api/gen/query/indexers/indexers.msw";
import { getListTrackersMockHandler } from "../api/gen/query/trackers/trackers.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { importingPack } from "../test/grabs";
import { packEntry, severanceRow } from "../test/import-entries";
import { serveImports } from "../test/imports";
// oxlint-disable-next-line import/max-dependencies -- Activity reads Grabs, Indexers, Trackers, Torrents, Targets and the Grab's Import
import { trackers } from "../test/indexers";
import { cinema, orchardRow } from "../test/library";
import { shows } from "../test/pending-imports";
import { renderApp } from "../test/render-app";
import { starter, targetHandlers } from "../test/targets";
import { torrentsServer } from "../test/torrents";

function activity(grab: GrabsGrab) {
  serveImports({ imports: [packEntry], rows: [orchardRow, severanceRow] });
  renderApp({
    path: "/activity",
    handlers: [
      ...targetHandlers([{ ...starter, id: grab.targetIds[0] ?? "" }]),
      getListGrabsMockHandler({ grabs: [grab] }),
      getListContentTypesMockHandler({ contentTypes: [cinema, shows] }),
      getListIndexersMockHandler({ indexers: [] }),
      getListTrackersMockHandler({ trackers }),
      ...torrentsServer([]).handlers,
    ],
  });
}

const row = () => page.getByRole("row", { name: /The Quiet Orchard/u });

async function open(item: string) {
  await userEvent.click(page.getByRole("button", { name: "The Quiet Orchard actions" }));
  await userEvent.click(page.getByRole("menuitem", { name: item }));
}

test("a Grab at Import reads its Import's progress and links Waiting for you to its Pending import", async () => {
  activity(importingPack);
  await expect.element(row()).toMatchTextContent(/Importing 6 of 10 episodes/u);
  await expect
    .element(row().getByRole("link", { name: "Waiting for you" }))
    .toHaveAttribute("href", `/library/pending/${importingPack.id}`);
  await expect
    .element(row().getByRole("progressbar", { name: "Import The Quiet Orchard" }))
    .toBeVisible();
});

test("a Grab whose Import sends a file lights Transfer", async () => {
  if (importingPack.import === null) {
    throw new Error("importingPack has an Import");
  }
  activity({ ...importingPack, import: { ...importingPack.import, sending: 2, waiting: 0 } });
  await expect
    .element(row().getByRole("progressbar", { name: "Transfer The Quiet Orchard" }))
    .toBeVisible();
  await expect
    .element(row().getByRole("link", { name: "Waiting for you" }))
    .not.toBeInTheDocument();
});

test("a Grab's Details list each file of its download with where it got to and why", async () => {
  activity(importingPack);
  await open("Details");
  const files = page
    .getByRole("dialog", { name: "Grab of The Quiet Orchard" })
    .getByRole("region", { name: "Files" });
  await expect
    .element(files)
    .toMatchTextContent(
      "Severance.S01E01.1080p.WEB-DL.mkvWaiting for you. This isn't better than the file you have for 1080p.Severance.S01E02.1080p.WEB-DL.mkvImported",
    );
});

test("Import by hand opens the Grab's own Import in Import files, with no Move files switch", async () => {
  activity(importingPack);
  await open("Import by hand");
  const sheet = page.getByRole("dialog", { name: "Import files" });
  await expect
    .element(sheet)
    .toMatchTextContent(
      /Severance\.S01E01\.1080p\.WEB-DL\.mkv.*This isn't better than the file you have for 1080p\./u,
    );
  await expect.element(sheet.getByRole("switch", { name: "Move files" })).not.toBeInTheDocument();
});

test("Cancel at Import says what stays and what goes back to wanted", async () => {
  activity({ ...importingPack, actions: { cancel: true, blocklistable: true, sendTo: false } });
  await open("Cancel Grab");
  await expect
    .element(page.getByRole("dialog", { name: "Cancel this Grab?" }))
    .toMatchTextContent(
      "SlipStream stops importing the download and keeps its torrent. Files already in your library stay, and Items it hasn't imported go back to wanted.",
    );
});
