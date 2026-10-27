import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IndexersIndexer } from "../api/gen/model";

import { prowlarr, prowlarrId, torrentLeech, torrentLeechId } from "../test/indexers";
import { indexersServer } from "../test/indexers-server";
import { renderApp } from "../test/render-app";

const path = "/settings/indexers";
const group = (name: string) => page.getByRole("list", { name });
const row = (list: string, name: string) => group(list).getByRole("listitem", { name });

test("the list groups each Indexer under every Tracker it reaches, and leads with who needs the Admin", async () => {
  renderApp({ path, handlers: indexersServer().handlers });
  await expect
    .element(page.getByText("Larkspur needs you. Open it to see what to do."))
    .toBeVisible();
  await expect.element(page.getByRole("heading", { name: "Aither" })).toBeVisible();
  expect(group("Indexers reaching Aither").getByRole("listitem").elements()).toHaveLength(1);
  await expect.element(row("Indexers reaching Aither", "Prowlarr")).toBeVisible();
  await expect.element(row("Indexers reaching TorrentLeech", "Prowlarr")).toBeVisible();
  await expect
    .element(
      row("Indexers reaching TorrentLeech", "TorrentLeech").getByText("Catalog site on seedbox"),
    )
    .toBeVisible();
  await expect.element(page.getByRole("heading", { name: "No Tracker yet" })).toBeVisible();
  await expect
    .element(
      row("Indexers with no Tracker yet", "Larkspur").getByText(
        "Stopped because Larkspur rejected the username or password.",
      ),
    )
    .toBeVisible();
});

test("a switch saves through PATCH and every row of that Indexer follows", async () => {
  const server = indexersServer();
  renderApp({ path, handlers: server.handlers });
  const automatic = (list: string) =>
    row(list, "Prowlarr").getByRole("switch", { name: "Automatic search" });
  await expect.element(automatic("Indexers reaching Aither")).toBeChecked();
  await userEvent.click(automatic("Indexers reaching Aither"));
  await expect.element(automatic("Indexers reaching Aither")).not.toBeChecked();
  await expect.element(automatic("Indexers reaching TorrentLeech")).not.toBeChecked();
  expect(server.writes).toEqual([
    { method: "PATCH", id: prowlarrId, body: { automaticSearch: false } },
  ]);

  await userEvent.click(
    row("Indexers reaching TorrentLeech", "TorrentLeech").getByRole("switch", { name: "On" }),
  );
  await expect
    .poll(() => server.writes.at(-1))
    .toEqual({
      method: "PATCH",
      id: torrentLeechId,
      body: { on: false },
    });
  await expect
    .element(
      row("Indexers reaching TorrentLeech", "TorrentLeech").getByText("TorrentLeech is off."),
    )
    .toBeVisible();
});

test("while no Indexer reaches a Tracker the list has no groups", async () => {
  const lone = [
    { ...prowlarr, trackerIds: [] },
    { ...torrentLeech, trackerIds: [] },
  ];
  renderApp({ path, handlers: indexersServer({ indexers: lone, trackerList: [] }).handlers });
  await expect.element(page.getByText("Searches ask all 2 Indexers that are on.")).toBeVisible();
  await expect.element(group("Indexers").getByRole("listitem", { name: "Prowlarr" })).toBeVisible();
  expect(page.getByRole("heading", { name: "No Tracker yet" }).query()).toBeNull();
});

test("the summary says how many Indexers a search asks while some wait", async () => {
  const limited: IndexersIndexer = {
    ...prowlarr,
    trackerIds: [],
    sentence: "Prowlarr is limiting requests until 14:05.",
    waits: {
      kind: "limited",
      releases: 0,
      until: "2026-10-26T14:05:00Z",
      since: null,
      nodeId: null,
      lapsed: false,
      cause: null,
      text: null,
    },
  };
  const lone = { ...torrentLeech, trackerIds: [] };
  renderApp({
    path,
    handlers: indexersServer({ indexers: [limited, lone], trackerList: [] }).handlers,
  });
  await expect
    .element(
      page.getByText(
        "Searches ask 1 of the 2 Indexers that are on. The others are waiting, and each row says why.",
      ),
    )
    .toBeVisible();
  await expect.element(page.getByText("Prowlarr is limiting requests until 14:05.")).toBeVisible();
});

test("with no Indexers the section says what one is for", async () => {
  renderApp({ path, handlers: indexersServer({ indexers: [], trackerList: [] }).handlers });
  await expect
    .element(
      page.getByText("No Indexers yet. Every search asks each Indexer added here that is on."),
    )
    .toBeVisible();
});

test("the long backoff saves in hours", async () => {
  const server = indexersServer();
  renderApp({ path, handlers: server.handlers });
  await userEvent.fill(
    page.getByRole("textbox", { name: "Warn when every request has failed for" }),
    "12",
  );
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => server.writes.at(-1))
    .toEqual({
      method: "PATCH",
      id: "settings",
      body: { values: { "indexers.longBackoff": 43_200 } },
    });
  await expect.element(page.getByRole("button", { name: "Reset to default" })).toBeVisible();
});
