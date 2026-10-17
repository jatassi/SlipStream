import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TasksTask } from "../api/gen/model";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import {
  housekeeping,
  local,
  schedule,
  steps,
  succeededRun,
  taskServer,
  tasksNow,
} from "../test/tasks";
import { asPhone } from "../test/viewport";

afterEach(() => {
  vi.useRealTimers();
});

const atTasksNow = () => {
  vi.useFakeTimers({ toFake: ["Date"], now: tasksNow });
};

const backlog: TasksTask = {
  ...housekeeping,
  id: "backlog",
  name: "Backlog search",
  about: "Searches for every wanted Work that has no file yet.",
  group: "acquire",
  cadence: {
    ...housekeeping.cadence,
    words: "Every 8 h",
    inWindow: false,
    field: { ...housekeeping.cadence.field, key: "tasks.backlog.cadence", value: 28_800 },
  },
  state: { kind: "idle", next: local(17, 5) },
  lastRun: {
    ...succeededRun,
    result: "partial",
    summary: "5 of 6 steps, seedbox offline",
  },
};

const scan: TasksTask = {
  ...housekeeping,
  id: "scan",
  name: "Library scan",
  group: "store",
  cadence: { ...housekeeping.cadence, words: "Every 2 days in the Maintenance window" },
  state: { kind: "waiting", node: "seedbox", link: "down", since: local(16, 21, 31) },
  runNow: { available: false, reason: "waiting" },
};

const row = (name: string) => page.getByRole("row").filter({ hasText: name });

test("the sentence leads with a failure and a wait, and groups keep their fixed order", async () => {
  atTasksNow();
  const tasks = taskServer({
    tasks: [scan, backlog, housekeeping],
    sentence: [
      { kind: "failure", text: "Housekeeping failed at 04:12.", href: "/tasks/housekeeping" },
      { kind: "warning", text: "1 task is waiting for seedbox.", href: "/tasks/scan" },
    ],
  });
  renderApp({ path: "/tasks", handlers: tasks.handlers });
  await expect
    .element(page.getByRole("link", { name: "Housekeeping failed at 04:12." }))
    .toHaveAttribute("href", "/tasks/housekeeping");
  await expect
    .element(page.getByRole("link", { name: "1 task is waiting for seedbox." }))
    .toBeVisible();
  expect(
    page
      .getByRole("heading", { level: 2 })
      .all()
      .map((heading) => heading.element().textContent),
  ).toEqual(["Controller", "Acquire", "Store", "Schedule"]);
});

test("rows say the cadence, the last result and the next run in words", async () => {
  atTasksNow();
  renderApp({ path: "/tasks", handlers: taskServer({ tasks: [housekeeping, backlog] }).handlers });
  await expect.element(row("Housekeeping")).toMatchTextContent("Daily in the Maintenance window");
  await expect.element(row("Housekeeping")).toMatchTextContent("Tonight at 04:00");
  await expect.element(row("Housekeeping")).toMatchTextContent("Succeeded6 steps · 04:01");
  await expect.element(row("Backlog search")).toMatchTextContent("Every 8 h");
  await expect.element(row("Backlog search")).toMatchTextContent("In 6 h");
  await expect
    .element(row("Backlog search"))
    .toMatchTextContent("Partial5 of 6 steps, seedbox offline");
});

test("a row whose every unit waits is amber with the reason and no Run now", async () => {
  atTasksNow();
  renderApp({ path: "/tasks", handlers: taskServer({ tasks: [housekeeping, scan] }).handlers });
  await expect
    .element(row("Library scan"))
    .toMatchTextContent("Waiting for seedbox · offline since 21:31");
  expect(row("Library scan").element().dataset["tone"]).toBe("warn");
  expect(row("Housekeeping").element().dataset["tone"]).toBeUndefined();
  await expect.element(page.getByRole("button", { name: "Run now: Library scan" })).toBeDisabled();
  await expect.element(page.getByRole("button", { name: "Run now: Housekeeping" })).toBeEnabled();
});

test("the Node column shows only when there is more than one Node", async () => {
  renderApp({ path: "/tasks", handlers: taskServer({ tasks: [housekeeping] }).handlers });
  await expect.element(page.getByRole("columnheader", { name: "Nodes" })).toBeVisible();
  await expect.element(row("Housekeeping")).toMatchTextContent("home, seedbox");
});

test("a single Node has no Node column", async () => {
  renderApp({
    path: "/tasks",
    handlers: taskServer({ tasks: [housekeeping], multinode: false }).handlers,
  });
  await expect.element(page.getByRole("columnheader", { name: "Cadence" })).toBeVisible();
  expect(page.getByRole("columnheader", { name: "Nodes" }).query()).toBeNull();
});

test("Run now posts the task's run and Cancel deletes the open run", async () => {
  atTasksNow();
  const running: TasksTask = {
    ...backlog,
    state: {
      kind: "running",
      runId: 7,
      trigger: "schedule",
      since: local(16, 14, 2),
      done: 3,
      total: 12,
    },
    runNow: { available: false, reason: "running" },
  };
  const tasks = taskServer({ tasks: [housekeeping, running] });
  renderApp({ path: "/tasks", handlers: tasks.handlers });
  await userEvent.click(page.getByRole("button", { name: "Run now: Housekeeping" }));
  await userEvent.click(page.getByRole("button", { name: "Cancel Backlog search" }));
  await expect
    .poll(() => tasks.writes)
    .toEqual([
      { method: "POST", path: "housekeeping/runs", body: undefined },
      { method: "DELETE", path: "runs/7", body: undefined },
    ]);
  await expect.element(row("Backlog search")).toMatchTextContent("Running since 14:02");
});

test("a saved schedule setting shows from the save's answer, without waiting for the live event", async () => {
  const tasks = taskServer({ tasks: [housekeeping] });
  renderApp({ path: "/tasks", handlers: tasks.handlers });
  const runsKept = page.getByRole("textbox", { name: "Runs kept per task" });
  await expect.element(runsKept).toHaveValue("50");
  tasks.state.settings = {
    ...schedule,
    runsKept: { ...schedule.runsKept, value: 80, running: 80, isDefault: false },
  };
  await userEvent.fill(runsKept, "80");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => tasks.writes)
    .toEqual([{ method: "PATCH", path: "settings", body: { runsKept: 80 } }]);
  await expect.element(page.getByRole("button", { name: "Save" })).not.toBeInTheDocument();
  await expect.element(runsKept).toHaveValue("80");
});

test("a tasks.progress frame moves a running row's bar", async () => {
  const server = liveServer();
  const running: TasksTask = {
    ...backlog,
    state: {
      kind: "running",
      runId: 7,
      trigger: "schedule",
      since: local(16, 14, 2),
      done: 3,
      total: 12,
    },
  };
  const tasks = taskServer({
    tasks: [running],
    open: {
      headerDelaySeconds: 5,
      runs: [
        {
          runId: 7,
          taskId: "backlog",
          name: "Backlog search",
          noun: steps,
          trigger: "schedule",
          startedAt: local(16, 14, 2),
          nodes: ["home"],
          done: 3,
          total: 12,
        },
      ],
    },
  });
  const app = renderApp({ path: "/tasks", handlers: [...server.handlers, ...tasks.handlers] });
  const bar = page.getByRole("progressbar", { name: "Backlog search progress" });
  await expect.element(bar).toHaveAttribute("aria-valuenow", "25");
  await expect.poll(() => app.live.getState()).toBe("open");

  server.clients[0]?.send({
    event: "tasks.progress",
    data: { runId: 7, taskId: "backlog", done: 9, total: 12 },
  });
  await expect.element(bar).toHaveAttribute("aria-valuenow", "75");
  await expect.element(row("Backlog search")).toMatchTextContent("9 of 12 steps");
});

test("one cadence, two surfaces: a change on the row shows on the detail page after tasks.changed", async () => {
  const server = liveServer();
  const tasks = taskServer({ tasks: [housekeeping] });
  const list = renderApp({ path: "/tasks", handlers: [...server.handlers, ...tasks.handlers] });
  const detail = renderApp({ path: "/tasks/housekeeping" });
  await expect.poll(() => [list.live.getState(), detail.live.getState()]).toEqual(["open", "open"]);
  const when = page.getByRole("region", { name: "When it runs" });
  await expect.element(when).toMatchTextContent("Daily in the Maintenance window");

  await userEvent.click(
    page.getByRole("button", { name: "Every: Daily in the Maintenance window" }),
  );
  const popover = page.getByRole("dialog", { name: "Every" });
  await userEvent.fill(popover.getByRole("textbox", { name: "Every" }), "2");
  await userEvent.click(popover.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => tasks.writes)
    .toEqual([{ method: "PATCH", path: "housekeeping", body: { cadence: 172_800 } }]);

  tasks.state.tasks = [
    {
      ...housekeeping,
      cadence: {
        ...housekeeping.cadence,
        words: "Every 2 days in the Maintenance window",
        field: { ...housekeeping.cadence.field, value: 172_800, isDefault: false },
      },
    },
  ];
  for (const client of server.clients) {
    client.send({ event: "tasks.changed", data: { taskId: "housekeeping" } });
  }
  await expect.element(when).toMatchTextContent("Every 2 days in the Maintenance window");
  await expect.element(when.getByRole("textbox", { name: "Every" })).toHaveValue("2");
  await expect
    .element(page.getByRole("button", { name: "Every: Every 2 days in the Maintenance window" }))
    .toBeVisible();
});

test("on phone the tasks are a list with the result, the next run and Run now", async () => {
  atTasksNow();
  await asPhone();
  renderApp({ path: "/tasks", handlers: taskServer({ tasks: [housekeeping, scan] }).handlers });
  const list = page.getByRole("list", { name: "Controller" });
  await expect.element(list).toMatchTextContent("Succeeded·Tonight at 04:00");
  await expect.element(page.getByRole("table")).not.toBeInTheDocument();
  await expect
    .element(page.getByRole("list", { name: "Store" }))
    .toMatchTextContent("Waiting for seedbox · offline since 21:31");
});
