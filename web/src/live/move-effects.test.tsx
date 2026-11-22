import { expect, test } from "vitest";

import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const moving = "0190a000-0000-7000-8000-000000000001";
const still = "0190a000-0000-7000-8000-000000000002";

test("moves.changed invalidates the Work views it names and no other", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    moving: getGetWorkQueryKey(moving),
    still: getGetWorkQueryKey(still),
  });

  server.clients[0]?.send({
    event: "moves.changed",
    data: { batchIds: [], unitIds: [], workIds: [moving] },
  });
  await expect.poll(invalidated).toEqual({ moving: true, still: false });
});
