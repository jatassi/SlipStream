import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { DownloadclientsTorrentRow } from "../api/gen/model";
import { getGetWorkMockHandler } from "../api/gen/query/works/works.msw";
import { downloading as grab } from "../test/grabs";
import { orchard as orchardTree, orchardItemId } from "../test/library";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { actionsServer, detailOf, orchardId, torrent, torrentsServer } from "../test/torrents";
import { asPhone } from "../test/viewport";

const orchard = torrent({
  state: "downloading",
  tabs: ["downloading", "all"],
  workId: orchardId,
  name: "The.Quiet.Orchard.2021.1080p.WEB-DL",
  done: 250_000_000,
  uploaded: 50_000_000,
  ratio: 0.05,
  tracker: { host: "beacon.example", status: "working" },
});
const paused = torrent({
  state: "paused",
  hold: "wanted",
  tabs: ["downloading", "all"],
  name: "Paused.Release.2020.1080p",
});
const third = torrent({ state: "queued", tabs: ["downloading", "all"], name: "Third.2021.720p" });

const detail = (row: DownloadclientsTorrentRow) =>
  row.id === orchard.id
    ? detailOf(row, {
        announceAt: "2020-01-01T00:00:00Z",
        trackers: [
          {
            url: "https://beacon.example/announce/•••",
            tier: 0,
            status: "working",
            answeredAt: "2026-10-20T11:50:00Z",
            nextAnnounce: "2026-10-20T12:20:00Z",
            minIntervalSeconds: 1800,
          },
        ],
        files: [
          {
            index: 0,
            path: "The.Quiet.Orchard.2021.1080p.WEB-DL/orchard.mkv",
            size: 1_000_000_000,
            done: 250_000_000,
            itemId: orchardItemId,
          },
        ],
        grabs: [
          {
            id: grab.id,
            workId: orchardId,
            itemId: orchardItemId,
            release: grab.releaseTitle,
            stage: "downloading",
          },
        ],
      })
    : detailOf(row);

const table = () => page.getByRole("table", { name: "Torrents" });
const inspector = () => page.getByRole("complementary", { name: "The Quiet Orchard" });
const notices = () => page.getByRole("region", { name: "Notifications" });

function open(path = "/activity/torrents?tab=all") {
  const live = liveServer();
  const server = torrentsServer([orchard, paused, third], detail);
  const actions = actionsServer();
  const app = renderApp({
    path,
    handlers: [
      actions.handler,
      getGetWorkMockHandler(orchardTree),
      ...live.handlers,
      ...server.handlers,
    ],
  });
  return { app, live, server, actions };
}

test("a row opens the Inspector beside the table, leading with what the torrent is doing", async () => {
  const { app } = open();
  await userEvent.click(table().getByRole("button", { name: /^The Quiet Orchard(?! actions)/u }));
  await expect
    .poll(() => app.router.state.location.pathname)
    .toBe(`/activity/torrents/${orchard.id}`);
  expect(app.router.state.location.search).toEqual({ tab: "all" });
  await expect.element(inspector()).toMatchTextContent(/^The Quiet OrchardDownloading\./u);
  await expect.element(table()).toBeVisible();

  const section = (name: string) => inspector().getByRole("heading", { name, level: 3 });
  await Promise.all(
    ["Numbers", "Trackers", "Files", "Grabs", "Details"].map((name) =>
      expect.element(section(name)).toBeVisible(),
    ),
  );
  await expect.element(inspector()).toMatchTextContent("Progress25% of 954 MB");
  await expect.element(inspector()).toMatchTextContent("beacon.exampleWorking");
  await expect
    .element(inspector())
    .toMatchTextContent("orchard.mkv25%954 MB · Filled The Quiet Orchard");
  await expect
    .element(inspector().getByRole("link", { name: "The Quiet Orchard" }))
    .toHaveAttribute("href", `/works/${orchardId}`);
  await expect
    .element(inspector())
    .toMatchTextContent("Info hash0102030405060708090a0b0c0d0e0f1011121314");
  const address = inspector().getByText("https://beacon.example/announce/•••");
  await expect.element(address).not.toBeInTheDocument();
  await userEvent.click(inspector().getByRole("button", { name: "Details" }));
  await expect.element(address).toBeVisible();
});

test("the Inspector's numbers and sentence follow the live stream", async () => {
  const { live } = open(`/activity/torrents/${orchard.id}?tab=all`);
  await expect.element(inspector()).toMatchTextContent(/^The Quiet OrchardDownloading\./u);
  live.clients[0]?.send({
    event: "torrents.live",
    data: {
      at: "2026-10-20T12:00:00Z",
      rows: [
        {
          id: orchard.id,
          done: 500_000_000,
          wanted: 1_000_000_000,
          downBps: 4 * 1024 * 1024,
          upBps: 1024 * 1024,
          peers: 38,
          seeds: 4,
        },
      ],
      downloads: { done: 500_000_000, wanted: 1_000_000_000, allPaused: false },
    },
  });
  await expect
    .element(inspector())
    .toMatchTextContent("Downloading from 38 peers, 2 min left at 4.0 MB/s.");
  await expect.element(inspector()).toMatchTextContent("Peers38 peers, 4 seeding");
});

test("Pause and Announce now act on the torrent the Inspector shows", async () => {
  const { actions } = open(`/activity/torrents/${orchard.id}?tab=all`);
  await userEvent.click(inspector().getByRole("button", { name: "Pause" }));
  await expect.element(notices().getByText("Pausing The Quiet Orchard.")).toBeVisible();
  await userEvent.click(
    inspector().getByRole("button", { name: "More actions for The Quiet Orchard" }),
  );
  await userEvent.click(page.getByRole("menuitem", { name: "Announce now" }));
  await expect
    .element(notices().getByText("The Quiet Orchard is announcing to its trackers."))
    .toBeVisible();
  expect(actions.bodies).toEqual([
    { action: "pause", ids: [orchard.id] },
    { action: "announce", ids: [orchard.id] },
  ]);
});

test("a recheck of files the library shares asks first", async () => {
  const live = liveServer();
  const actions = actionsServer({
    preview: { cancels: [], stopSeeding: [], frees: 0, linked: true },
  });
  renderApp({
    path: `/activity/torrents/${orchard.id}?tab=all`,
    handlers: [actions.handler, ...live.handlers, ...torrentsServer([orchard], detail).handlers],
  });
  await userEvent.click(
    inspector().getByRole("button", { name: "More actions for The Quiet Orchard" }),
  );
  await userEvent.click(page.getByRole("menuitem", { name: "Recheck" }));
  const dialog = page.getByRole("dialog", { name: "Recheck The Quiet Orchard?" });
  await expect
    .element(dialog)
    .toMatchTextContent(
      /^Recheck The Quiet Orchard\?Some of its files are also files in your library\./u,
    );
  await userEvent.click(dialog.getByRole("button", { name: "Recheck" }));
  await expect.element(notices().getByText("Rechecking The Quiet Orchard.")).toBeVisible();
  expect(actions.bodies).toEqual([
    { action: "recheck", ids: [orchard.id], dryRun: true },
    { action: "recheck", ids: [orchard.id] },
  ]);
});

test("the arrow keys move the selection and the Inspector follows", async () => {
  const { app } = open();
  await userEvent.click(table().getByRole("button", { name: /^The Quiet Orchard(?! actions)/u }));
  await expect.element(inspector()).toBeVisible();
  const before = app.router.history.length;
  await userEvent.keyboard("{ArrowDown}");
  await expect
    .poll(() => app.router.state.location.pathname)
    .toBe(`/activity/torrents/${paused.id}`);
  await expect
    .element(page.getByRole("complementary", { name: "Paused.Release.2020.1080p" }))
    .toMatchTextContent("Paused. It stays paused until you resume it.");
  await expect
    .element(table().getByRole("button", { name: /^Paused.Release(?!.* actions)/u }))
    .toHaveFocus();
  await userEvent.keyboard("{End}");
  await expect
    .poll(() => app.router.state.location.pathname)
    .toBe(`/activity/torrents/${third.id}`);
  expect(app.router.history.length).toBe(before);

  await userEvent.click(page.getByRole("complementary").getByRole("button", { name: "Close" }));
  await expect.poll(() => app.router.state.location.pathname).toBe("/activity/torrents");
  await expect.element(page.getByRole("complementary")).not.toBeInTheDocument();
});

test("a torrent that leaves closes its Inspector", async () => {
  const { app, live, server } = open(`/activity/torrents/${orchard.id}?tab=all`);
  await expect.element(inspector()).toBeVisible();
  server.become([paused, third]);
  live.clients[0]?.send({ event: "torrents.changed", data: { ids: [orchard.id] } });
  await expect.poll(() => app.router.state.location.pathname).toBe("/activity/torrents");
  await expect.element(page.getByRole("complementary")).not.toBeInTheDocument();
});

test("on a phone a row opens the torrent as a pushed page with a way back", async () => {
  await asPhone();
  const { app } = open();
  await userEvent.click(table().getByRole("button", { name: /^The Quiet Orchard(?! actions)/u }));
  await expect
    .poll(() => app.router.state.location.pathname)
    .toBe(`/activity/torrents/${orchard.id}`);
  await expect
    .element(page.getByRole("banner").getByRole("heading", { name: "The Quiet Orchard" }))
    .toBeVisible();
  await expect.element(page.getByRole("main")).toMatchTextContent(/^Downloading/u);
  await expect.element(table()).not.toBeInTheDocument();
  await expect.element(page.getByRole("radiogroup", { name: "View" })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("banner").getByRole("link", { name: "Torrents" }));
  await expect.poll(() => app.router.state.location.pathname).toBe("/activity/torrents");
  await expect.element(table()).toBeVisible();
});

test("a link to a torrent that is gone is not found", async () => {
  open("/activity/torrents/7a000000-0000-4000-8000-ffffffffffff");
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
});
