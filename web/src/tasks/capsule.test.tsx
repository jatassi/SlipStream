import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TasksOpenRun } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { housekeeping, steps, taskServer, tasksNow } from "../test/tasks";

afterEach(() => {
  vi.useRealTimers();
});

const backlogRun: TasksOpenRun = {
  runId: 7,
  taskId: "backlog",
  name: "Backlog search",
  noun: { one: "search", many: "searches" },
  trigger: "run-now",
  startedAt: tasksNow.toISOString(),
  nodes: ["home"],
  done: 3,
  total: 12,
};

const housekeepingRun: TasksOpenRun = {
  runId: 8,
  taskId: "housekeeping",
  name: "Housekeeping",
  noun: steps,
  trigger: "schedule",
  startedAt: tasksNow.toISOString(),
  nodes: ["home", "seedbox"],
  done: 0,
  total: 6,
};

const running = (...runs: TasksOpenRun[]) =>
  taskServer({ tasks: [housekeeping], open: { headerDelaySeconds: 5, runs } });

const capsule = () => page.getByRole("banner").getByRole("button", { name: /running/u });

test("the capsule is absent while no run is visible", async () => {
  vi.useFakeTimers({ toFake: ["Date", "setInterval", "clearInterval"], now: tasksNow });
  renderApp({ path: "/tasks", handlers: running(housekeepingRun).handlers });
  await expect.element(page.getByRole("link", { name: "Housekeeping" })).toBeVisible();
  vi.advanceTimersByTime(4900);
  await expect.element(capsule()).not.toBeInTheDocument();

  vi.advanceTimersByTime(100);
  await expect.element(capsule()).toHaveAccessibleName("1 task running: Housekeeping");
});

test("the capsule lists each run with its Nodes and progress, and Cancel deletes the run", async () => {
  const tasks = running(backlogRun, { ...housekeepingRun, trigger: "run-now" });
  renderApp({ path: "/tasks", handlers: tasks.handlers });
  await userEvent.click(capsule());
  const list = page.getByRole("dialog", { name: "Running tasks" });
  await expect
    .element(list.getByRole("listitem").nth(0))
    .toMatchTextContent("3 of 12 searcheshome");
  await expect
    .element(list.getByRole("listitem").nth(1))
    .toMatchTextContent("0 of 6 stepshome, seedbox");
  await expect
    .element(list.getByRole("progressbar", { name: "Backlog search progress" }))
    .toHaveAttribute("aria-valuenow", "25");

  await userEvent.click(list.getByRole("button", { name: "Cancel Backlog search" }));
  await expect
    .poll(() => tasks.writes)
    .toEqual([{ method: "DELETE", path: "runs/7", body: undefined }]);
});
