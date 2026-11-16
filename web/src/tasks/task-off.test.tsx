import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { TasksTask } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { housekeeping, taskServer } from "../test/tasks";

const off: TasksTask = {
  ...housekeeping,
  state: { kind: "idle", next: "2026-11-17T04:00:00Z", off: true },
};

test("a task a setting turned off reads Off on its row in place of its next run", async () => {
  renderApp({ path: "/tasks", handlers: taskServer({ tasks: [off] }).handlers });
  const row = page.getByRole("row").filter({ hasText: off.name });
  await expect.element(row).toMatchTextContent(/Off/u);
  expect(row.getByText(/at 04:00/u).elements()).toHaveLength(0);
});

test("a task a setting turned off reads Off on its page in place of its next run", async () => {
  renderApp({ path: `/tasks/${off.id}`, handlers: taskServer({ tasks: [off] }).handlers });
  await expect.element(page.getByText("Off", { exact: true }).first()).toBeVisible();
  expect(page.getByText(/^Next run/u).elements()).toHaveLength(0);
});
