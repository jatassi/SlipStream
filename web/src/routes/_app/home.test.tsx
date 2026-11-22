import { afterEach, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsGrab } from "../../api/gen/model";
import { downloading } from "../../test/grabs";
import {
  countsAre,
  failedGrab,
  harvestWork,
  homeServer,
  job,
  jobRow,
  need,
  nodeTorrents,
  orchardWork,
  renderHome,
  summaryOf,
} from "../../test/home";
import { synced } from "../../test/live-server";
import { home } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { countRequests } from "../../test/requests";
import { torrent } from "../../test/torrents";
// oxlint-disable-next-line import/max-dependencies -- Home reads Grabs, torrents, counts and the live stream
import { worker } from "../../test/worker";

afterEach(() => {
  worker.events.removeAllListeners();
});

const orchardTorrent = torrent({ state: "downloading", tabs: ["downloading", "all"] });
const cancelable: GrabsGrab = {
  ...downloading,
  torrentId: orchardTorrent.id,
  actions: { cancel: true, blocklistable: false, sendTo: false },
};

test("Home never reads the Grabs, torrents, Library or Transfers, even on hover or a menu", async () => {
  const requests = countRequests();
  const server = homeServer(
    summaryOf({
      jobs: [job(cancelable, orchardWork, orchardTorrent)],
      moving: 12,
      needs: [need(failedGrab("2026-10-12T09:02:00Z"), harvestWork, null)],
    }),
    [nodeTorrents(home.id, { torrents: 3, failures: 1 })],
  );
  const app = renderApp({
    path: "/",
    handlers: [countsAre({ pendingImports: 2 }), ...server.handlers],
  });
  await expect.element(jobRow("The Quiet Orchard")).toBeVisible();
  await synced(app);
  await userEvent.hover(jobRow("The Quiet Orchard"));
  await userEvent.hover(page.getByRole("link", { name: "The Quiet Orchard" }));
  await userEvent.hover(page.getByRole("link", { name: "See all 12 jobs" }));
  await userEvent.hover(page.getByRole("link", { name: /pending imports/u }));
  await userEvent.hover(page.getByRole("list", { name: "Downloading Nodes" }).getByRole("link"));
  await userEvent.click(
    jobRow("The Quiet Orchard").getByRole("button", { name: "The Quiet Orchard actions" }),
  );
  await userEvent.click(page.getByRole("menuitem", { name: "Cancel Grab" }));
  await expect.element(page.getByRole("dialog")).toBeVisible();
  await expect.poll(() => app.queryClient.isFetching()).toBe(0);
  for (const list of ["grabs", "torrents", "library", "transfers"]) {
    expect(requests(`GET /api/v1/${list}`), list).toBe(0);
  }
});

test("a progress tick reads nothing; a stage change, a shown Job's Torrent or Rows on Home reads Home again", async () => {
  const requests = countRequests();
  const shown = job(downloading, orchardWork, orchardTorrent);
  const { app, server } = renderHome(summaryOf({ jobs: [shown] }));
  await expect.element(jobRow("The Quiet Orchard")).toBeVisible();
  await synced(app);
  const reads = () => requests("GET /api/v1/home");
  const counted = () => requests("GET /api/v1/torrents/counts");
  const [before, countedBefore] = [reads(), counted()];
  const send = (event: Parameters<NonNullable<(typeof server.clients)[0]>["send"]>[0]) => {
    server.clients[0]?.send(event);
  };
  const stage = (progress: boolean) => ({
    event: "grabs.changed" as const,
    data: { grabId: shown.grab.id, workId: orchardWork.id, progress },
  });
  send(stage(true));
  send({ event: "torrents.changed", data: { ids: ["7a000000-0000-4000-8000-ffffffffffff"] } });
  await expect.poll(counted).toBe(countedBefore + 1);
  expect(reads()).toBe(before);
  send({ event: "torrents.changed", data: { ids: [orchardTorrent.id] } });
  await expect.poll(reads).toBe(before + 1);
  send(stage(false));
  await expect.poll(reads).toBe(before + 2);
  send({ event: "home.settings-changed", data: {} });
  await expect.poll(reads).toBe(before + 3);
});

test("Rows on Home saves from Home's menu and Home reads its lists again", async () => {
  const requests = countRequests();
  const { app } = renderHome(summaryOf());
  await expect.element(page.getByRole("heading", { name: "Jobs" })).toBeVisible();
  await synced(app);
  const before = requests("GET /api/v1/home");
  await userEvent.click(page.getByRole("button", { name: "Home options" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Rows on Home…" }));
  const dialog = page.getByRole("dialog", { name: "Rows on Home" });
  await expect
    .element(dialog.getByText("How many entries Needs you, Jobs and Recent each show."))
    .toBeVisible();
  const field = dialog.getByRole("textbox", { name: "Rows on Home" });
  await userEvent.fill(field, "3");
  await userEvent.click(dialog.getByRole("button", { name: "Save" }));
  await expect.poll(() => requests("PATCH /api/v1/home-settings")).toBe(1);
  await expect.poll(() => requests("GET /api/v1/home")).toBe(before + 1);
  await expect.element(field).toHaveValue("3");
});
