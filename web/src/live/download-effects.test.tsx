import { expect, test } from "vitest";

import { getGetDownloadClientSettingsQueryKey } from "../api/gen/query/downloadclients/downloadclients";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

test("download-clients.changed invalidates only the named Nodes' client settings", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const seedbox = "6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e";
  const home = "0f9e8d7c-6b5a-4c3d-8e2f-1a0b9c8d7e6f";
  const invalidated = seeded(app, {
    seedbox: getGetDownloadClientSettingsQueryKey(seedbox),
    home: getGetDownloadClientSettingsQueryKey(home),
  });

  server.clients[0]?.send({ event: "download-clients.changed", data: { nodeIds: [seedbox] } });
  await expect.poll(invalidated).toEqual({ seedbox: true, home: false });
});
