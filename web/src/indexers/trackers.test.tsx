import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IndexersTracker } from "../api/gen/model";
import {
  aitherTrackerId,
  beyondHd,
  prowlarrId,
  torrentLeechTrackerId,
  trackers,
} from "../test/indexers";
import { indexersServer } from "../test/indexers-server";
import { renderApp } from "../test/render-app";

const path = "/settings/indexers";
const byName = (list: IndexersTracker[]) => list.toSorted((a, b) => a.name.localeCompare(b.name));
const heading = (name: string) => page.getByRole("heading", { name, exact: true });
const group = (name: string) => page.getByRole("list", { name });

async function choose(tracker: string, item: string) {
  await userEvent.click(page.getByRole("button", { name: `${tracker} actions` }));
  await userEvent.click(page.getByRole("menuitem", { name: item }));
}

test("Rename saves the new name and the group takes it", async () => {
  const server = indexersServer();
  renderApp({ path, handlers: server.handlers });
  await choose("TorrentLeech", "Rename");
  const dialog = page.getByRole("dialog", { name: "Rename TorrentLeech" });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Name" }), "TL");
  await userEvent.click(dialog.getByRole("button", { name: "Rename" }));
  await expect.element(heading("TL")).toBeVisible();
  await expect
    .element(group("Indexers reaching TL").getByRole("listitem", { name: "Prowlarr" }))
    .toBeVisible();
  expect(server.writes).toEqual([
    { method: "PATCH", id: torrentLeechTrackerId, body: { name: "TL" } },
  ]);
});

test("Rename to another Tracker's name says so on the field and keeps the dialog open", async () => {
  const server = indexersServer();
  renderApp({ path, handlers: server.handlers });
  await choose("TorrentLeech", "Rename");
  const dialog = page.getByRole("dialog", { name: "Rename TorrentLeech" });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Name" }), "Aither");
  await userEvent.click(dialog.getByRole("button", { name: "Rename" }));
  await expect.element(dialog.getByText("Another Tracker has that name")).toBeVisible();
  expect(server.state.trackers.map((tracker) => tracker.name)).toEqual(["Aither", "TorrentLeech"]);
});

test("Merge into offers every other Tracker but never the Tracker itself, and moves its Indexers to the one kept", async () => {
  const server = indexersServer({ trackerList: byName([...trackers, beyondHd]) });
  renderApp({ path, handlers: server.handlers });
  await choose("Aither", "Merge into…");
  const dialog = page.getByRole("dialog", { name: "Merge Aither" });
  await userEvent.click(dialog.getByRole("combobox", { name: "Merge into" }));
  await expect.element(page.getByRole("option", { name: "TorrentLeech" })).toBeVisible();
  expect(
    page
      .getByRole("option")
      .elements()
      .map((option) => option.textContent),
  ).toEqual(["BeyondHD", "TorrentLeech"]);
  await userEvent.click(page.getByRole("option", { name: "TorrentLeech" }));
  await expect
    .element(
      dialog.getByText("Aither's Indexers, Grabs and torrents move to TorrentLeech", {
        exact: false,
      }),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Merge" }));
  await expect.element(heading("Aither")).not.toBeInTheDocument();
  await expect
    .element(
      group("Indexers reaching TorrentLeech").getByRole("listitem", {
        name: "Prowlarr",
      }),
    )
    .toBeVisible();
  expect(server.writes).toEqual([
    {
      method: "MERGE",
      id: aitherTrackerId,
      body: { into: torrentLeechTrackerId },
    },
  ]);
  expect(server.state.indexers.find((indexer) => indexer.id === prowlarrId)?.trackerIds).toEqual([
    torrentLeechTrackerId,
  ]);
});

test("a merge the server refuses keeps the dialog open with its reason", async () => {
  const server = indexersServer();
  renderApp({ path, handlers: server.handlers });
  await choose("Aither", "Merge into…");
  const dialog = page.getByRole("dialog", { name: "Merge Aither" });
  server.state.trackers = server.state.trackers.filter(
    (tracker) => tracker.id !== torrentLeechTrackerId,
  );
  await userEvent.click(dialog.getByRole("button", { name: "Merge" }));
  await expect.element(dialog.getByRole("alert")).toHaveTextContent("No Tracker has that id");
});

test("a Tracker no Indexer reaches stays, says so and keeps its keys and hosts under Details", async () => {
  renderApp({
    path,
    handlers: indexersServer({ trackerList: [...trackers, beyondHd] }).handlers,
  });
  await expect.element(heading("BeyondHD")).toBeVisible();
  const section = page.getByRole("region", { name: "BeyondHD" });
  await expect
    .element(section.getByText("No Indexer reaches this Tracker.", { exact: false }))
    .toBeVisible();
  expect(section.getByRole("list").query()).toBeNull();
  await userEvent.click(section.getByRole("button", { name: "Details" }));
  await expect.element(section.getByText("definition:beyondhd")).toBeVisible();
  await expect.element(section.getByText("tracker.beyond-hd.me")).toBeVisible();
  await expect
    .element(
      page.getByRole("region", { name: "TorrentLeech" }).getByText("TorrentLeech is private."),
    )
    .toBeVisible();
});

test("with one Tracker its menu has Rename and no Merge into…", async () => {
  renderApp({
    path,
    handlers: indexersServer({ trackerList: [beyondHd], indexers: [] }).handlers,
  });
  await expect
    .element(
      page.getByText("No Indexers yet. Every search asks each Indexer added here that is on."),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "BeyondHD actions" }));
  await expect.element(page.getByRole("menuitem", { name: "Rename" })).toBeVisible();
  expect(page.getByRole("menuitem", { name: "Merge into…" }).query()).toBeNull();
});
