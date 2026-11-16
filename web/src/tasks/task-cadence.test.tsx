import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderAction } from "../test/actions";
import { housekeeping, taskServer } from "../test/tasks";
import { TaskCadence } from "./task-cadence";

test("a listed task's cadence reads from the task", async () => {
  await renderAction(
    <TaskCadence task="housekeeping" fallback="its owner's field" />,
    taskServer({ tasks: [housekeeping] }).handlers,
  );
  await expect
    .element(
      page.getByRole("button", {
        name: "Every: Daily in the Maintenance window",
      }),
    )
    .toBeVisible();
  expect(page.getByText("its owner's field").query()).toBeNull();
});

test("a task the Tasks page doesn't list shows its owner's field in its place", async () => {
  await renderAction(
    <TaskCadence task="rss-sync" fallback="its owner's field" />,
    taskServer({ tasks: [housekeeping] }).handlers,
  );
  await expect.element(page.getByText("its owner's field")).toBeVisible();
});
