import { expect, test } from "vitest";

import { getListNotificationConnectionsQueryKey } from "../api/gen/query/notifications/notifications";
import { feedQuery, unreadKey } from "../notifications/feed-keys";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

test("history.changed refreshes the bell's feed and count and nothing else", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    feed: feedQuery.queryKey,
    unread: unreadKey,
    connections: getListNotificationConnectionsQueryKey(),
  });

  server.clients[0]?.send({ event: "history.changed", data: {} });
  await expect.poll(invalidated).toEqual({ feed: true, unread: true, connections: false });
});
