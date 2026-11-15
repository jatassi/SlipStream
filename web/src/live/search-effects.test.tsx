import { expect, test } from "vitest";

import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const orchard = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
const harvest = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

test("grabs.queue-changed invalidates only the Works it names", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    orchard: getGetWorkQueryKey(orchard),
    harvest: getGetWorkQueryKey(harvest),
  });

  server.clients[0]?.send({ event: "grabs.queue-changed", data: { workIds: [orchard] } });
  await expect.poll(invalidated).toEqual({ orchard: true, harvest: false });
});
