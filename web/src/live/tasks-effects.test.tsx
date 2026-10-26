import { HttpResponse, delay, http } from "msw";
import { expect, test } from "vitest";

import { getListOpenTaskRunsMockHandler } from "../api/gen/query/tasks/tasks.msw";
import {
  getGetTaskQueryKey,
  getGetTaskSettingsQueryKey,
  getListOpenTaskRunsQueryKey,
  getListTasksQueryKey,
} from "../api/gen/query/tasks/tasks";
import { getListLibraryQueryKey } from "../api/gen/query/works/works";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

test("tasks.changed invalidates every tasks query and nothing else", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    list: getListTasksQueryKey(),
    task: getGetTaskQueryKey("housekeeping"),
    settings: getGetTaskSettingsQueryKey(),
    open: getListOpenTaskRunsQueryKey(),
    library: getListLibraryQueryKey(),
  });

  server.clients[0]?.send({ event: "tasks.changed", data: { taskId: "" } });
  await expect
    .poll(invalidated)
    .toEqual({ list: true, task: true, settings: true, open: true, library: false });
});

test("a burst of tasks.changed starts one refetch and one more after it, not one per event", async () => {
  const server = liveServer();
  let asked = 0;
  const open = http.get("*/api/v1/tasks/runs/open", async () => {
    asked += 1;
    const answer = asked;
    await delay(200);
    return HttpResponse.json({ headerDelaySeconds: answer, runs: [] });
  });
  const app = renderApp({ path: "/nodes", handlers: [...server.handlers, open] });
  const fetchStatus = () =>
    app.queryClient.getQueryState(getListOpenTaskRunsQueryKey())?.fetchStatus;
  await expect.poll(() => app.live.getState()).toBe("open");
  await expect.poll(() => asked > 0 && fetchStatus() === "idle").toBe(true);
  const before = asked;

  for (let i = 0; i < 5; i += 1) {
    server.clients[0]?.send({ event: "tasks.changed", data: { taskId: "housekeeping" } });
  }
  await expect.poll(fetchStatus).toBe("fetching");
  await expect.poll(fetchStatus).toBe("idle");
  expect(asked - before).toBe(2);
  expect(app.queryClient.getQueryData(getListOpenTaskRunsQueryKey())).toEqual({
    headerDelaySeconds: asked,
    runs: [],
  });
});

const openRun = (runId: number, taskId: string) => ({
  runId,
  taskId,
  name: taskId,
  noun: { one: "step", many: "steps" },
  trigger: "run-now" as const,
  startedAt: "2026-10-16T14:02:00Z",
  nodes: ["home"],
  done: 0,
  total: 6,
});

test("tasks.progress moves that open run in place", async () => {
  const server = liveServer();
  const open = { headerDelaySeconds: 5, runs: [openRun(1, "housekeeping"), openRun(2, "tidy")] };
  const app = renderApp({
    path: "/nodes",
    handlers: [...server.handlers, getListOpenTaskRunsMockHandler(open)],
  });
  await expect.poll(() => app.live.getState()).toBe("open");
  await expect
    .poll(() => app.queryClient.getQueryData(getListOpenTaskRunsQueryKey()))
    .toEqual(open);

  server.clients[0]?.send({
    event: "tasks.progress",
    data: { runId: 1, taskId: "housekeeping", done: 3, total: 6 },
  });
  await expect
    .poll(() => app.queryClient.getQueryData(getListOpenTaskRunsQueryKey()))
    .toEqual({
      headerDelaySeconds: 5,
      runs: [{ ...openRun(1, "housekeeping"), done: 3 }, openRun(2, "tidy")],
    });
  expect(app.queryClient.getQueryState(getListOpenTaskRunsQueryKey())?.isInvalidated).toBe(false);
});
