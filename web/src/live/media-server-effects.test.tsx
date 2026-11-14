import { expect, test } from "vitest";

import {
  getGetMediaServerQueryKey,
  getListMediaServerChangesQueryKey,
  getListMediaServerSessionsQueryKey,
  getListMediaServersQueryKey,
} from "../api/gen/query/mediaservers/mediaservers";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import { orchardId } from "../test/library";
import { liveServer, seeded, synced } from "../test/live-server";
import { denJellyfin, livingRoomPlex } from "../test/media-servers";
import { renderApp } from "../test/render-app";

test("media-servers.changed invalidates the list and only the named servers", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    list: getListMediaServersQueryKey(),
    den: getGetMediaServerQueryKey(denJellyfin.id),
    denChanges: getListMediaServerChangesQueryKey(denJellyfin.id),
    plex: getGetMediaServerQueryKey(livingRoomPlex.id),
  });

  server.clients[0]?.send({
    event: "media-servers.changed",
    data: { serverIds: [denJellyfin.id] },
  });
  await expect.poll(invalidated).toEqual({ list: true, den: true, denChanges: true, plex: false });
});

test("media-servers.sessions-changed invalidates only the named servers' sessions", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    den: getListMediaServerSessionsQueryKey(denJellyfin.id),
    plex: getListMediaServerSessionsQueryKey(livingRoomPlex.id),
    list: getListMediaServersQueryKey(),
  });

  server.clients[0]?.send({
    event: "media-servers.sessions-changed",
    data: { serverIds: [livingRoomPlex.id] },
  });
  await expect.poll(invalidated).toEqual({ den: false, plex: true, list: false });
});

test("media-servers.unhealthy invalidates the list and the server", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    list: getListMediaServersQueryKey(),
    plex: getGetMediaServerQueryKey(livingRoomPlex.id),
  });

  server.clients[0]?.send({
    event: "media-servers.unhealthy",
    data: { serverId: livingRoomPlex.id, kind: "plex", reason: "not_responding" },
  });
  await expect.poll(invalidated).toEqual({ list: true, plex: true });
});

test("media-servers.items-found invalidates each named Work", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, { work: getGetWorkQueryKey(orchardId) });

  server.clients[0]?.send({ event: "media-servers.items-found", data: { workIds: [orchardId] } });
  await expect.poll(invalidated).toEqual({ work: true });
});
