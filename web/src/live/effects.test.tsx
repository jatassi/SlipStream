import { HttpResponse, delay, http } from "msw";
import { expect, test } from "vitest";

import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import {
  getGetGrabQueryKey,
  getListGrabsQueryKey,
  getListSearchesQueryKey,
} from "../api/gen/query/grabs/grabs";
import {
  getListPasskeysQueryKey,
  getListSessionsQueryKey,
} from "../api/gen/query/identity/identity";
import { getListIndexersQueryKey } from "../api/gen/query/indexers/indexers";
import {
  getGetJoinCodeQueryKey,
  getGetNodeQueryKey,
  getGetRoleBoardQueryKey,
  getListJoinCodesQueryKey,
  getListNodesQueryKey,
} from "../api/gen/query/nodes/nodes";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const orchard = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
const harvest = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";
const grab = "7e9a1c3d-5f7b-4d9e-8a1c-5e7f9b1d3a4c";
const otherGrab = "9a1c3e5f-7b9d-4f1a-8c3e-7a9b1d3f5c6e";

test("nodes.changed invalidates the Nodes, that Node, the Role board, the counts, the Indexers and the Join Codes", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const seedbox = "6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e";
  const home = "0f9e8d7c-6b5a-4c3d-8e2f-1a0b9c8d7e6f";
  const code = "2c4e6a8b-0d1f-4a3c-9e5b-7d9f1b3c5e7a";
  const invalidated = seeded(app, {
    list: getListNodesQueryKey(),
    seedbox: getGetNodeQueryKey(seedbox),
    home: getGetNodeQueryKey(home),
    board: getGetRoleBoardQueryKey(),
    counts: getGetCountsQueryKey(),
    indexers: getListIndexersQueryKey(),
    codes: getListJoinCodesQueryKey(),
    code: getGetJoinCodeQueryKey(code),
  });

  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedbox } });
  await expect.poll(invalidated).toEqual({
    list: true,
    seedbox: true,
    home: false,
    board: true,
    counts: true,
    indexers: true,
    codes: true,
    code: true,
  });
});

test("a lapse's burst of nodes.changed refetches the counts once and once more after it, not one per event", async () => {
  const server = liveServer();
  let asked = 0;
  const counts = http.get("*/api/v1/counts", async () => {
    asked += 1;
    const answer = asked;
    await delay(200);
    return HttpResponse.json({ activity: answer, alerts: { nodes: "fail" } });
  });
  const app = renderApp({ path: "/", handlers: [...server.handlers, counts] });
  const fetchStatus = () => app.queryClient.getQueryState(getGetCountsQueryKey())?.fetchStatus;
  await expect.poll(() => app.live.getState()).toBe("open");
  await expect.poll(() => asked > 0 && fetchStatus() === "idle").toBe(true);
  const before = asked;
  const seedbox = "6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e";
  const home = "0f9e8d7c-6b5a-4c3d-8e2f-1a0b9c8d7e6f";

  for (const nodeId of [home, seedbox, home]) {
    server.clients[0]?.send({ event: "nodes.changed", data: { nodeId } });
  }
  await expect.poll(fetchStatus).toBe("fetching");
  await expect.poll(fetchStatus).toBe("idle");
  expect(asked - before).toBe(2);
  expect(app.queryClient.getQueryData(getGetCountsQueryKey())).toEqual({
    activity: asked,
    alerts: { nodes: "fail" },
  });
});

test("grabs.changed invalidates the Grabs, that Grab and only that Work's cells", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    grabs: getListGrabsQueryKey(),
    grab: getGetGrabQueryKey(grab),
    other: getGetGrabQueryKey(otherGrab),
    orchard: getGetWorkCellsQueryKey(orchard),
    harvest: getGetWorkCellsQueryKey(harvest),
  });

  server.clients[0]?.send({ event: "grabs.changed", data: { grabId: grab, workId: orchard } });
  await expect
    .poll(invalidated)
    .toEqual({ grabs: true, grab: true, other: false, orchard: true, harvest: false });
});

test("grabs.search-changed invalidates only that Work's searches", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    orchard: getListSearchesQueryKey(orchard),
    harvest: getListSearchesQueryKey(harvest),
  });

  server.clients[0]?.send({
    event: "grabs.search-changed",
    data: { searchId: grab, workId: orchard },
  });
  await expect.poll(invalidated).toEqual({ orchard: true, harvest: false });
});

test("identity.passkeys-changed invalidates your passkey list", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    passkeys: getListPasskeysQueryKey(),
    sessions: getListSessionsQueryKey(),
  });

  server.clients[0]?.send({
    event: "identity.passkeys-changed",
    data: { account: "1b3d5f7a-9c1e-4a3c-8e5a-7c9e1b3d5f7a" },
  });
  await expect.poll(invalidated).toEqual({ passkeys: true, sessions: false });
});
