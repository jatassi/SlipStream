import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TasksOpenRun } from "../api/gen/model";
import { getListGrabsMockHandler } from "../api/gen/query/grabs/grabs.msw";
import { renderApp } from "../test/render-app";
import { housekeeping, steps, taskServer, tasksNow } from "../test/tasks";
import { asPhone } from "../test/viewport";

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

test("a run started from Run now shows at once and a scheduled one after the header delay", async () => {
  vi.useFakeTimers({ toFake: ["Date", "setInterval", "clearInterval"], now: tasksNow });
  renderApp({ path: "/tasks", handlers: running(backlogRun, housekeepingRun).handlers });
  await expect.element(capsule()).toHaveAccessibleName("1 task running: Backlog search");

  vi.advanceTimersByTime(4900);
  await expect.element(capsule()).toHaveAccessibleName("1 task running: Backlog search");

  vi.advanceTimersByTime(100);
  await expect
    .element(capsule())
    .toHaveAccessibleName("2 tasks running: Backlog search and 1 more");
});

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
    .toMatchTextContent("home · 3 of 12 searches");
  await expect
    .element(list.getByRole("listitem").nth(1))
    .toMatchTextContent("home, seedbox · 0 of 6 steps");
  await expect
    .element(list.getByRole("meter", { name: "Backlog search progress" }))
    .toHaveAttribute("aria-valuenow", "25");

  await userEvent.click(list.getByRole("button", { name: "Cancel Backlog search" }));
  await expect
    .poll(() => tasks.writes)
    .toEqual([{ method: "DELETE", path: "runs/7", body: undefined }]);
});

test("the top bar's right side runs connection, tasks, then the page action", async () => {
  renderApp({ path: "/tasks/housekeeping", handlers: running(backlogRun).handlers });
  const bar = page.getByRole("banner");
  const action = bar.getByRole("button", { name: "Run now: Housekeeping" });
  await expect.element(action).toBeVisible();
  const slots = [...bar.element().querySelectorAll("output, button")];
  const output = slots.findIndex((slot) => slot.tagName === "OUTPUT");
  const tasks = slots.indexOf(capsule().element());
  expect([output < tasks, tasks < slots.indexOf(action.element())]).toEqual([true, true]);
});

test("on phone the running list heads Activity, even with nothing downloading", async () => {
  await asPhone();
  renderApp({
    path: "/activity",
    handlers: [getListGrabsMockHandler({ grabs: [] }), ...running(backlogRun).handlers],
  });
  const section = page.getByRole("region", { name: "Running tasks" });
  await expect.element(section).toMatchTextContent("Backlog search");
  const empty = page.getByText("Nothing is downloading.", { exact: false });
  await expect.element(empty).toBeVisible();
  expect(section.element().compareDocumentPosition(empty.element())).toBe(
    Node.DOCUMENT_POSITION_FOLLOWING,
  );
});
