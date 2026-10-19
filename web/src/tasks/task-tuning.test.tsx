import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingDurationField, SettingSwitchField } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { housekeeping, taskServer } from "../test/tasks";

const about = {
  source: "database",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
} as const;

const day = 86_400;

const othersEvery: SettingDurationField = {
  ...about,
  key: "metadata.refresh.others",
  label: "Every other Work refresh every",
  kind: "duration",
  value: 30 * day,
  running: 30 * day,
  default: 30 * day,
  min: day,
  max: 365 * day,
  step: day,
};

const followChanges: SettingSwitchField = {
  ...about,
  key: "metadata.refresh.changes",
  label: "Refresh what the provider reports changed",
  kind: "switch",
  value: true,
  running: true,
  default: true,
};

test("a task's tuning shows through the shared control and saves under tuning by key", async () => {
  const tasks = taskServer({ tasks: [housekeeping] });
  tasks.state.tuning = [othersEvery, followChanges];
  renderApp({ path: "/tasks/housekeeping", handlers: tasks.handlers });
  const when = page.getByRole("region", { name: "When it runs" });
  const every = when.getByRole("textbox", { name: "Every other Work refresh every" });
  await expect.element(every).toHaveValue("30");

  await userEvent.fill(every, "14");
  await userEvent.click(when.getByRole("button", { name: "Save" }));
  await userEvent.click(
    when.getByRole("switch", { name: "Refresh what the provider reports changed" }),
  );
  await expect
    .poll(() => tasks.writes)
    .toEqual([
      {
        method: "PATCH",
        path: "housekeeping",
        body: { tuning: { "metadata.refresh.others": 14 * day } },
      },
      {
        method: "PATCH",
        path: "housekeeping",
        body: { tuning: { "metadata.refresh.changes": false } },
      },
    ]);
});

test("a task with no tuning shows only its cadence and overdue rule", async () => {
  renderApp({
    path: "/tasks/housekeeping",
    handlers: taskServer({ tasks: [housekeeping] }).handlers,
  });
  const when = page.getByRole("region", { name: "When it runs" });
  await expect.element(when.getByRole("textbox", { name: "Every" })).toBeVisible();
  expect(when.getByRole("switch").query()).toBeNull();
  expect(when.getByRole("textbox").all()).toHaveLength(1);
});
