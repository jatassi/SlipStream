import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { DownloadclientsTorrentRow } from "../api/gen/model";
import { getListWorksMockHandler } from "../api/gen/query/works/works.msw";
import { orchardListed } from "../test/library";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import {
  harvestId,
  orchardId,
  orchardPoster,
  seedboxId,
  torrent,
  torrentsServer,
} from "../test/torrents";
import { asPhone } from "../test/viewport";

const downloading: DownloadclientsTorrentRow = torrent({
  state: "downloading",
  tabs: ["downloading", "all"],
  workId: orchardId,
  edition: "directors",
  quality: "webdl-1080p",
  name: "The.Quiet.Orchard.2021.1080p.WEB-DL",
  done: 250_000_000,
  tracker: { host: "beacon.example", status: "working" },
});

const seeding = torrent({
  state: "seeding",
  tabs: ["seeding", "all"],
  workId: harvestId,
  nodeId: seedboxId,
  contentType: "zines",
  name: "Orchard.Harvest.2019.720p.HDTV",
  done: 1_000_000_000,
  ratio: 1.42,
  seedingSeconds: 6 * 86_400,
});

const unregistered = torrent({
  state: "tracker-error",
  severity: "failure",
  tabs: ["seeding", "problems", "all"],
  name: "Gone.Release.2020.1080p",
  done: 1_000_000_000,
  tracker: { host: "beacon.example", status: "unregistered" },
});

const sentence = () => page.getByRole("paragraph").filter({ hasText: "seeding" });
const table = () => page.getByRole("table", { name: "Torrents" });
const tabs = () => page.getByRole("radiogroup", { name: "Show" });

const posterOf = (name: RegExp) =>
  table().getByRole("row", { name }).element().querySelector<HTMLElement>("[data-poster]");

function open(rows: readonly DownloadclientsTorrentRow[], path = "/activity/torrents") {
  const live = liveServer();
  const server = torrentsServer(rows);
  const app = renderApp({ path, handlers: [...live.handlers, ...server.handlers] });
  return { app, live, server };
}

test("Torrents leads with its sentence and problems, then the downloads by default", async () => {
  open([downloading, seeding, unregistered]);
  await expect.element(sentence()).toHaveTextContent("1 downloading · 1 seeding · 1 failure");
  await expect
    .element(page.getByRole("link", { name: "1 failure" }))
    .toHaveAttribute("href", "#torrent-signal-0");
  await expect
    .element(page.getByRole("region", { name: "Needs you" }))
    .toMatchTextContent(
      "UnregisteredGone.Release.2020.1080pbeacon.example reports that the torrent is no longer registered.",
    );

  await expect.element(tabs().getByRole("radio", { name: "Downloading 1" })).toBeChecked();
  await expect.element(tabs().getByRole("radio", { name: "Seeding 2" })).toBeVisible();
  await expect.element(tabs().getByRole("radio", { name: "Problems 1" })).toBeVisible();
  await expect.element(tabs().getByRole("radio", { name: "All 3" })).toBeVisible();

  const row = table().getByRole("row", { name: /^The Quiet Orchard/u });
  await expect
    .element(row)
    .toHaveTextContent(
      "The Quiet OrchardDirector's Cut · WEBDL-1080pDownloading25%0.00↓ —↑ —beacon.exampleWorkinghomeBuilt-in",
    );
  expect(table().getByRole("row").all()).toHaveLength(2);
});

test("each row leads with its Work's poster, and a placeholder where there is none", async () => {
  open([downloading, seeding, unregistered], "/activity/torrents?tab=all");
  await expect.element(table().getByRole("row", { name: /^The Quiet Orchard/u })).toBeVisible();

  const art = () => posterOf(/^The Quiet Orchard/u)?.querySelector("img");
  await expect.poll(() => art()?.getAttribute("src")).toBe(`${orchardPoster}?size=small`);
  await expect.poll(() => art()?.naturalWidth ?? 0).toBeGreaterThan(0);
  expect(posterOf(/^Orchard Harvest/u)?.querySelector("img")).toBeNull();
  expect(posterOf(/^Gone\.Release/u)?.querySelector("img")).toBeNull();
  expect(posterOf(/^Gone\.Release/u)).not.toBeNull();
});

test("a poster the Node can't fetch leaves the placeholder", async () => {
  const gone = "/artwork/gone.png";
  renderApp({
    path: "/activity/torrents",
    handlers: [
      getListWorksMockHandler({
        works: [{ ...orchardListed, artwork: [{ kind: "poster", url: gone }] }],
      }),
      http.get(gone, () => HttpResponse.json({}, { status: 404 })),
      ...liveServer().handlers,
      ...torrentsServer([downloading]).handlers,
    ],
  });
  await expect.element(table().getByRole("row", { name: /^The Quiet Orchard/u })).toBeVisible();
  await expect.poll(() => posterOf(/^The Quiet Orchard/u)?.querySelector("img")).toBeNull();
  expect(posterOf(/^The Quiet Orchard/u)?.dataset["poster"]).toBe("art");
});

test("a tab, a Node and a search narrow the table", async () => {
  const { app } = open([downloading, seeding, unregistered]);
  await userEvent.click(tabs().getByRole("radio", { name: "All 3" }));
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(4);
  expect(app.router.state.location.search).toEqual({ tab: "all" });

  await userEvent.click(page.getByRole("combobox", { name: "Node" }));
  await userEvent.click(page.getByRole("option", { name: "seedbox" }));
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(2);
  await expect
    .element(table().getByRole("row", { name: /^Orchard Harvest/u }))
    .toMatchTextContent("Seeding");
  await expect
    .element(table().getByRole("row", { name: /^Orchard Harvest/u }))
    .toMatchTextContent("Seeded 6 days");

  await userEvent.click(page.getByRole("combobox", { name: "Node" }));
  await userEvent.click(page.getByRole("option", { name: "Every Node" }));
  await userEvent.fill(page.getByRole("searchbox", { name: "Search torrents" }), "quiet");
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(2);
  await expect.element(table().getByRole("row", { name: /^The Quiet Orchard/u })).toBeVisible();
  await expect.poll(() => app.router.state.location.search).toEqual({ tab: "all", q: "quiet" });

  await userEvent.fill(page.getByRole("searchbox", { name: "Search torrents" }), "nothing like it");
  await expect.element(page.getByText("No torrents match these filters.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Clear filters" }));
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(4);
});

test("the Node filter survives a switch to Jobs and back", async () => {
  const { app } = open([downloading, seeding], `/activity/torrents?tab=all&node=${seedboxId}`);
  const views = page.getByRole("main").getByRole("navigation", { name: "Activity" });
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(2);
  await userEvent.click(views.getByRole("link", { name: "Jobs" }));
  await expect.poll(() => app.router.state.location.pathname).toBe("/activity");
  expect(app.router.state.location.search).toEqual({ node: seedboxId });
  await userEvent.click(views.getByRole("link", { name: "Torrents" }));
  await expect.poll(() => app.router.state.location.pathname).toBe("/activity/torrents");
  await expect.element(page.getByRole("combobox", { name: "Node" })).toHaveTextContent("seedbox");
});

test("a Node filter from a link shows even when one Node holds every torrent", async () => {
  open([downloading], `/activity/torrents?tab=all&node=${seedboxId}`);
  await expect.element(page.getByRole("combobox", { name: "Node" })).toHaveTextContent("seedbox");
  await expect.element(page.getByText("No torrents match these filters.")).toBeVisible();
});

test("torrents.changed splices the named rows in without refetching the list", async () => {
  const { live, server } = open([downloading, seeding]);
  await expect.element(table().getByRole("row", { name: /^The Quiet Orchard/u })).toBeVisible();

  const paused: DownloadclientsTorrentRow = { ...downloading, state: "paused", pending: "pause" };
  const added = torrent({
    state: "queued",
    hold: "slot",
    queuePosition: 2,
    tabs: ["downloading", "all"],
    name: "Another.Release.2022.1080p",
  });
  server.become([paused, added]);
  live.clients[0]?.send({
    event: "torrents.changed",
    data: { ids: [downloading.id, seeding.id, added.id] },
  });

  await expect
    .element(table().getByRole("row", { name: /^The Quiet Orchard/u }))
    .toMatchTextContent("Pausing");
  await expect
    .element(table().getByRole("row", { name: /^Another.Release/u }))
    .toMatchTextContent("Queued2nd in line");
  await expect.element(tabs().getByRole("radio", { name: "Seeding 0" })).toBeVisible();
  expect(server.asked).toEqual([[downloading.id, seeding.id, added.id]]);
});

test("torrents.live moves a row's progress and speed", async () => {
  const { live } = open([downloading]);
  const row = table().getByRole("row", { name: /^The Quiet Orchard/u });
  await expect.element(row).toMatchTextContent("25%");
  live.clients[0]?.send({
    event: "torrents.live",
    data: {
      at: "2026-10-20T12:00:00Z",
      rows: [
        {
          id: downloading.id,
          done: 500_000_000,
          wanted: 1_000_000_000,
          downBps: 5 * 1024 * 1024,
          upBps: 0,
          peers: 38,
          seeds: 4,
        },
      ],
      downloads: { done: 500_000_000, wanted: 1_000_000_000, allPaused: false },
    },
  });
  await expect.element(row).toMatchTextContent("Downloading38 peers");
  await expect.element(row).toMatchTextContent("50%2 min left · 5.0 MB/s");
  await expect
    .element(row.getByRole("progressbar", { name: `Downloading ${downloading.name}` }))
    .toHaveAttribute("aria-valuenow", "50");

  live.clients[0]?.send({ event: "live.resync", data: {} });
  await expect.element(row).toMatchTextContent("Downloading25%");
  await expect.element(row).toMatchTextContent("↓ —↑ —");
});

test("a paused row keeps its percent but says no time left while its speed winds down", async () => {
  const paused = { ...downloading, state: "paused", hold: "wanted" } as const;
  const { live } = open([paused]);
  const row = table().getByRole("row", { name: /^The Quiet Orchard/u });
  await expect.element(row).toMatchTextContent("Paused");
  live.clients[0]?.send({
    event: "torrents.live",
    data: {
      at: "2026-10-20T12:00:00Z",
      rows: [
        {
          id: paused.id,
          done: 500_000_000,
          wanted: 1_000_000_000,
          downBps: 512 * 1024,
          upBps: 0,
          peers: 0,
          seeds: 0,
        },
      ],
      downloads: { done: 500_000_000, wanted: 1_000_000_000, allPaused: true },
    },
  });
  await expect.element(row).toMatchTextContent("↓ 512 KB/s");
  await expect.element(row).toMatchTextContent("50%");
  await expect.element(row).not.toMatchTextContent("left");
});

test("thousands of torrents draw only the rows near the view", async () => {
  const many = Array.from({ length: 3000 }, () =>
    torrent({ state: "seeding", tabs: ["seeding", "all"], done: 1_000_000_000 }),
  );
  open(many, "/activity/torrents?tab=seeding");
  await expect.element(tabs().getByRole("radio", { name: "Seeding 3000" })).toBeChecked();
  await expect.poll(() => table().getByRole("row").all().length).toBeGreaterThan(1);
  expect(table().getByRole("row").all().length).toBeLessThan(60);
  await expect.element(table()).toHaveAttribute("aria-rowcount", "3001");
});

test("with no torrents the view says how one arrives", async () => {
  open([]);
  await expect.element(page.getByText(/^No torrents yet\./u)).toBeVisible();
  await expect
    .element(page.getByRole("main").getByRole("button", { name: "Open the Library" }))
    .toHaveAttribute("href", "/library");
});

test("on a phone the tabs lead with All, which shows first, and rows become cards", async () => {
  await asPhone();
  open([downloading, seeding], "/activity/torrents");
  const options = tabs().getByRole("radio");
  await expect.element(tabs().getByRole("radio", { name: "All 2" })).toBeChecked();
  expect(options.all().map((option) => option.element().textContent)).toEqual([
    "All2",
    "Downloading1",
    "Seeding1",
    "Problems0",
  ]);
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(2);
  await expect.element(page.getByRole("columnheader", { name: "Work" })).not.toBeInTheDocument();
  await expect.element(page.getByRole("button", { name: "Filter" })).toBeVisible();
});
