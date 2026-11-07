import { expect, test } from "vitest";

import {
  getGetDownloadClientSettingsQueryKey,
  getListDownloadClientsQueryKey,
  getListTorrentsQueryKey,
} from "../api/gen/query/downloadclients/downloadclients";
import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import { getListGrabsQueryKey } from "../api/gen/query/grabs/grabs";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

test("download-clients.changed invalidates the cards and only the named Nodes' client settings", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const seedbox = "6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e";
  const home = "0f9e8d7c-6b5a-4c3d-8e2f-1a0b9c8d7e6f";
  const invalidated = seeded(app, {
    cards: getListDownloadClientsQueryKey(),
    seedbox: getGetDownloadClientSettingsQueryKey(seedbox),
    home: getGetDownloadClientSettingsQueryKey(home),
  });

  server.clients[0]?.send({
    event: "download-clients.changed",
    data: { nodeIds: [seedbox], clientIds: [] },
  });
  await expect.poll(invalidated).toEqual({ cards: true, seedbox: true, home: false });
});

test("download-clients.changed naming a client invalidates the torrents and every Work's needs", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const work = "3c5e7a9b-1d2f-4a6c-8e0b-2d4f6a8c0e1a";
  const invalidated = seeded(app, {
    torrents: getListTorrentsQueryKey(),
    cells: getGetWorkCellsQueryKey(work),
  });

  server.clients[0]?.send({
    event: "download-clients.changed",
    data: { nodeIds: [], clientIds: ["7d3f9b1e-5a2c-4e8f-b6d0-1c3e5a7f9b2d"] },
  });
  await expect.poll(invalidated).toEqual({ torrents: true, cells: true });
});

test("download-clients.changed for a Node's settings alone leaves the torrents and Work needs", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    cards: getListDownloadClientsQueryKey(),
    torrents: getListTorrentsQueryKey(),
  });

  server.clients[0]?.send({
    event: "download-clients.changed",
    data: { nodeIds: ["6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e"], clientIds: [] },
  });
  await expect.poll(invalidated).toEqual({ cards: true, torrents: false });
});

test("torrents.changed invalidates the counts and the cards", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    counts: getGetCountsQueryKey(),
    cards: getListDownloadClientsQueryKey(),
    grabs: getListGrabsQueryKey(),
  });

  server.clients[0]?.send({
    event: "torrents.changed",
    data: { ids: ["1b3d5f7a-9c1e-4a3c-8e5a-7c9e1b3d5f7a"] },
  });
  await expect.poll(invalidated).toEqual({ counts: true, cards: true, grabs: false });
});
