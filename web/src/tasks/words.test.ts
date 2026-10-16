import { expect, test } from "vitest";

import type { TasksOpenRun } from "../api/gen/model";
import { visibleRuns } from "./words";

const startedAt = new Date(2026, 9, 16, 23, 0);

const run = (trigger: TasksOpenRun["trigger"]): TasksOpenRun => ({
  runId: 1,
  taskId: "housekeeping",
  name: "Housekeeping",
  noun: { one: "step", many: "steps" },
  trigger,
  startedAt: startedAt.toISOString(),
  nodes: [],
  done: 0,
  total: 0,
});

const after = (ms: number) => new Date(startedAt.getTime() + ms);

test.each([
  ["schedule", 4_900, false],
  ["schedule", 5_000, true],
  ["catch-up", 4_999, false],
  ["returned", 5_000, true],
  ["run-now", 0, true],
] as const)("a %s run open %i ms is shown: %s", (trigger, open, shown) => {
  const runs = visibleRuns({ headerDelaySeconds: 5, runs: [run(trigger)] }, after(open));
  expect(runs.length === 1).toBe(shown);
});
