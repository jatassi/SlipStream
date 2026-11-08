import { expect, test } from "vitest";

import {
  getGetIndexerQueryKey,
  getGetIndexerSettingsQueryKey,
  getListIndexerQueriesQueryKey,
  getListIndexersQueryKey,
} from "../api/gen/query/indexers/indexers";
import {
  getGetDefaultGoalQueryKey,
  getGetSeedingSettingsQueryKey,
  getGetTrackerQueryKey,
  getListTrackerWindowsQueryKey,
  getListTrackersQueryKey,
  getPreviewSeedingGoalsQueryKey,
} from "../api/gen/query/trackers/trackers";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const quarry = "4b6d8f0a-2c4e-4a6c-8e0a-2c4e6a8c0e2a";
const larkspur = "6d8f0a2c-4e6a-4c8e-8a2c-4e6a8c0e2a4c";
const seedpool = "8f0a2c4e-6a8c-4e0a-8c4e-6a8c0e2a4c6d";
const aither = "0a2c4e6a-8c0e-4a2c-8e6a-8c0e2a4c6d8f";

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

test("indexers.settings-changed refreshes the Indexer settings", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    settings: getGetIndexerSettingsQueryKey(),
    list: getListIndexersQueryKey(),
  });
  server.clients[0]?.send({ event: "indexers.settings-changed", data: {} });
  await expect.poll(invalidated).toEqual({ settings: true, list: false });
});

test("trackers.seeding-changed for a Tracker refreshes the list and that Tracker", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    list: getListTrackersQueryKey(),
    seedpool: getGetTrackerQueryKey(seedpool),
    aither: getGetTrackerQueryKey(aither),
    defaultGoal: getGetDefaultGoalQueryKey(),
    seedpoolHistory: getListTrackerWindowsQueryKey(seedpool, { offset: 50 }),
    aitherWindows: getListTrackerWindowsQueryKey(aither),
    preview: getPreviewSeedingGoalsQueryKey(),
  });
  server.clients[0]?.send({ event: "trackers.seeding-changed", data: { trackerId: seedpool } });
  await expect.poll(invalidated).toEqual({
    list: true,
    seedpool: true,
    aither: false,
    defaultGoal: false,
    seedpoolHistory: true,
    aitherWindows: false,
    preview: true,
  });
});

test("trackers.seeding-changed for no Tracker refreshes the list, the default goal and the settings", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    list: getListTrackersQueryKey(),
    seedpool: getGetTrackerQueryKey(seedpool),
    defaultGoal: getGetDefaultGoalQueryKey(),
    settings: getGetSeedingSettingsQueryKey(),
  });
  server.clients[0]?.send({ event: "trackers.seeding-changed", data: { trackerId: null } });
  await expect
    .poll(invalidated)
    .toEqual({ list: true, seedpool: false, defaultGoal: true, settings: true });
});
