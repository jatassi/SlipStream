import { QueryObserver } from "@tanstack/react-query";
import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { getCounts, getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";

test("grabs.changed refetches the Activity count", async () => {
  const server = liveServer();
  let activity = 2;
  const { queryClient } = renderApp({
    path: "/system",
    handlers: [
      ...server.handlers,
      http.get("*/api/v1/counts", () => HttpResponse.json({ activity })),
    ],
  });
  await expect.element(page.getByText(/^Live updates: /u)).toHaveTextContent("Live updates: on");
  const counts = new QueryObserver(queryClient, {
    queryKey: getGetCountsQueryKey(),
    queryFn: () => getCounts(),
  });
  const unsubscribe = counts.subscribe(() => {});
  await expect.poll(() => counts.getCurrentResult().data).toEqual({ activity: 2 });

  activity = 3;
  server.clients[0]?.send({
    event: "grabs.changed",
    data: {
      grabId: "7e9a1c3d-5f7b-4d9e-8a1c-5e7f9b1d3a4c",
      workId: "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03",
    },
  });
  await expect.poll(() => counts.getCurrentResult().data).toEqual({ activity: 3 });
  unsubscribe();
});
