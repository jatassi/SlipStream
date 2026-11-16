import { expect, test } from "vitest";

import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import { getGetWorkSearchingQueryKey, getListWantedQueryKey } from "../api/gen/query/grabs/grabs";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const orchard = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
const harvest = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

const keys = {
  orchard: getGetWorkQueryKey(orchard),
  orchardSearching: getGetWorkSearchingQueryKey(orchard),
  orchardCells: getGetWorkCellsQueryKey(orchard),
  harvest: getGetWorkQueryKey(harvest),
  harvestSearching: getGetWorkSearchingQueryKey(harvest),
  wanted: getListWantedQueryKey(),
  wantedPages: [...getListWantedQueryKey({ sort: "next" }), "pages"],
};

test("grabs.queue-changed reads again the Works it names, their search state and cells, and Wanted", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, keys);

  server.clients[0]?.send({ event: "grabs.queue-changed", data: { workIds: [orchard] } });
  await expect.poll(invalidated).toEqual({
    orchard: true,
    orchardSearching: true,
    orchardCells: true,
    harvest: false,
    harvestSearching: false,
    wanted: true,
    wantedPages: true,
  });
});

test("grabs.queue-changed naming no Work reads every open Work again", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, keys);

  server.clients[0]?.send({ event: "grabs.queue-changed", data: { workIds: [] } });
  await expect.poll(invalidated).toEqual({
    orchard: true,
    orchardSearching: true,
    orchardCells: true,
    harvest: true,
    harvestSearching: true,
    wanted: true,
    wantedPages: true,
  });
});

test("grabs.changed for a Grab's stage reads again that Work's search state, since a failure can stop its cells", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, keys);

  server.clients[0]?.send({
    event: "grabs.changed",
    data: { grabId: harvest, workId: orchard, progress: false },
  });
  await expect.poll(invalidated).toMatchObject({ orchardSearching: true, harvestSearching: false });
});

test("grabs.changed for a progress tick leaves the search state alone", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, keys);

  server.clients[0]?.send({
    event: "grabs.changed",
    data: { grabId: orchard, workId: orchard, progress: true },
  });
  server.clients[0]?.send({
    event: "grabs.changed",
    data: { grabId: harvest, workId: harvest, progress: false },
  });
  await expect.poll(invalidated).toMatchObject({ orchardSearching: false, harvestSearching: true });
});
