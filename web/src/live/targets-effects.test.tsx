import { expect, test } from "vitest";

import { getGetWorkCellsQueryKey, getListWorkFilesQueryKey } from "../api/gen/query/files/files";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const orchard = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
const harvest = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

test("targets.changed invalidates every Work's cells, and the Works it names with their files", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    orchard: getGetWorkCellsQueryKey(orchard),
    harvest: getGetWorkCellsQueryKey(harvest),
    orchardWork: getGetWorkQueryKey(orchard),
    harvestWork: getGetWorkQueryKey(harvest),
    orchardFiles: getListWorkFilesQueryKey(orchard),
    harvestFiles: getListWorkFilesQueryKey(harvest),
  });

  server.clients[0]?.send({ event: "targets.changed", data: { workIds: [orchard] } });
  await expect.poll(invalidated).toEqual({
    orchard: true,
    harvest: true,
    orchardWork: true,
    harvestWork: false,
    orchardFiles: true,
    harvestFiles: false,
  });
});
