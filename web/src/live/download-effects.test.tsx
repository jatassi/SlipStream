import { expect, test } from "vitest";

import {
  getGetDownloadClientSettingsQueryKey,
  getListDownloadClientsQueryKey,
} from "../api/gen/query/downloadclients/downloadclients";
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

  server.clients[0]?.send({ event: "download-clients.changed", data: { nodeIds: [seedbox] } });
  await expect.poll(invalidated).toEqual({ cards: true, seedbox: true, home: false });
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
