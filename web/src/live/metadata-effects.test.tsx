import { expect, test } from "vitest";

import { getListMetadataProvidersQueryKey } from "../api/gen/query/metadata/metadata";
import { getListLibraryQueryKey } from "../api/gen/query/works/works";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

test("metadata.changed invalidates the providers' status and nothing else", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    providers: getListMetadataProvidersQueryKey(),
    library: getListLibraryQueryKey(),
  });

  server.clients[0]?.send({ event: "metadata.changed", data: { provider: "tmdb" } });
  await expect.poll(invalidated).toEqual({ providers: true, library: false });
});
