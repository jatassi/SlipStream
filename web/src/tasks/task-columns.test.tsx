import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { TasksTask } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { housekeeping, taskServer } from "../test/tasks";

const refresh: TasksTask = {
  ...housekeeping,
  id: "refresh",
  name: "Metadata refresh",
  about:
    "Keeps each Work's details, artwork and ratings current from the metadata sources, a few at a time, so that every page reads as it should.",
  group: "connect",
};

const columnEdges = (table: ReturnType<typeof page.getByRole>) =>
  table
    .getByRole("columnheader")
    .all()
    .map((header) => {
      const { left, width } = header.element().getBoundingClientRect();
      return { left, width };
    });

test("every group's table shares its column edges and fits its card", async () => {
  renderApp({ path: "/tasks", handlers: taskServer({ tasks: [housekeeping, refresh] }).handlers });
  const controller = page.getByRole("table", { name: "Controller" });
  const connect = page.getByRole("table", { name: "Connect" });
  await expect.element(connect).toBeVisible();
  expect(columnEdges(connect)).toEqual(columnEdges(controller));
  for (const table of [controller, connect]) {
    const { right } = table.element().getBoundingClientRect();
    const card = table.element().parentElement?.getBoundingClientRect();
    expect(right).toBeLessThanOrEqual(card?.right ?? 0);
  }
  const { scrollWidth, clientWidth } = document.documentElement;
  expect(scrollWidth).toBe(clientWidth);
});
