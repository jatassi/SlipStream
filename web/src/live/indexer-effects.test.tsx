import { expect, test } from "vitest";

import {
  getGetIndexerQueryKey,
  getListIndexerQueriesQueryKey,
  getListIndexersQueryKey,
  getListTrackersQueryKey,
} from "../api/gen/query/indexers/indexers";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const quarry = "4b6d8f0a-2c4e-4a6c-8e0a-2c4e6a8c0e2a";
const larkspur = "6d8f0a2c-4e6a-4c8e-8a2c-4e6a8c0e2a4c";

test("indexers.changed refreshes the list, every Indexer page and the Trackers", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    list: getListIndexersQueryKey(),
    quarry: getGetIndexerQueryKey(quarry),
    trackers: getListTrackersQueryKey(),
    queries: getListIndexerQueriesQueryKey(quarry),
  });
  server.clients[0]?.send({ event: "indexers.changed", data: { indexerId: quarry } });
  await expect
    .poll(invalidated)
    .toEqual({ list: true, quarry: true, trackers: true, queries: false });
});

test("indexers.status-changed refreshes the list, that Indexer and its queries", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    list: getListIndexersQueryKey(),
    quarry: getGetIndexerQueryKey(quarry),
    larkspur: getGetIndexerQueryKey(larkspur),
    queries: getListIndexerQueriesQueryKey(quarry),
  });
  server.clients[0]?.send({ event: "indexers.status-changed", data: { indexerId: quarry } });
  await expect
    .poll(invalidated)
    .toEqual({ list: true, quarry: true, larkspur: false, queries: true });
});
