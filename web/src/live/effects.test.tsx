import { expect, test } from "vitest";

import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import {
  getGetGrabQueryKey,
  getListGrabsQueryKey,
  getListSearchesQueryKey,
} from "../api/gen/query/grabs/grabs";
import { getGetNodeQueryKey, getListNodesQueryKey } from "../api/gen/query/nodes/nodes";
import { liveServer, seeded } from "../test/live-server";
import { renderApp } from "../test/render-app";

const orchard = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
const harvest = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";
const grab = "7e9a1c3d-5f7b-4d9e-8a1c-5e7f9b1d3a4c";
const otherGrab = "9a1c3e5f-7b9d-4f1a-8c3e-7a9b1d3f5c6e";

test("nodes.changed invalidates the Node list and that Node", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await expect.poll(() => app.live.getState()).toBe("open");
  const { queryClient } = app;
  const seedbox = "6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e";
  const home = "0f9e8d7c-6b5a-4c3d-8e2f-1a0b9c8d7e6f";
  const keys = {
    list: getListNodesQueryKey(),
    seedbox: getGetNodeQueryKey(seedbox),
    home: getGetNodeQueryKey(home),
  };
  for (const key of Object.values(keys)) {
    queryClient.setQueryData(key, {});
  }
  const invalidated = () =>
    Object.fromEntries(
      Object.entries(keys).map(([name, key]) => [
        name,
        queryClient.getQueryState(key)?.isInvalidated,
      ]),
    );

  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedbox } });
  await expect.poll(invalidated).toEqual({ list: true, seedbox: true, home: false });
});

test("grabs.changed invalidates the Grabs, that Grab and only that Work's cells", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/library", handlers: server.handlers });
  await expect.poll(() => app.live.getState()).toBe("open");
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
  const app = renderApp({ path: "/library", handlers: server.handlers });
  await expect.poll(() => app.live.getState()).toBe("open");
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
