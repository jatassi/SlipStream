import { expect, test } from "vitest";

import { getGetTopologyQueryKey } from "../api/gen/query/nodes/nodes";
import {
  getGetRootSettingsQueryKey,
  getListRootsQueryKey,
  getListVolumesQueryKey,
} from "../api/gen/query/roots/roots";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const seedbox = "6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e";
const films = "0199f0a1-1111-7000-8000-000000000001";

async function opened() {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    roots: getListRootsQueryKey(),
    volumes: getListVolumesQueryKey(),
    topology: getGetTopologyQueryKey(),
    settings: getGetRootSettingsQueryKey(),
  });
  return { send: server.clients[0]?.send.bind(server.clients[0]), invalidated };
}

test("roots.changed invalidates the roots, the volumes they sit on and the topology", async () => {
  const { send, invalidated } = await opened();
  send?.({ event: "roots.changed", data: { rootIds: [films] } });
  await expect
    .poll(invalidated)
    .toEqual({ roots: true, volumes: true, topology: true, settings: false });
});

test("volumes.changed invalidates the volumes and the topology, not the roots", async () => {
  const { send, invalidated } = await opened();
  send?.({ event: "volumes.changed", data: { nodeId: seedbox, volumeIds: ["sda1"] } });
  await expect
    .poll(invalidated)
    .toEqual({ roots: false, volumes: true, topology: true, settings: false });
});

test("nodes.changed invalidates the roots and volumes, whose states follow their Node's standing", async () => {
  const { send, invalidated } = await opened();
  send?.({ event: "nodes.changed", data: { nodeId: seedbox } });
  await expect
    .poll(invalidated)
    .toEqual({ roots: true, volumes: true, topology: true, settings: false });
});
