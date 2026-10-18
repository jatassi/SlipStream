import { HttpResponse, http } from "msw";

import type {
  SettingChoiceField,
  SettingDurationField,
  TasksDetail,
  TasksOpenRuns,
  TasksOverview,
  TasksRun,
  TasksSchedule,
  TasksTask,
} from "../api/gen/model";
import { homeId, seedboxId } from "./nodes";

const londonSummerHoursAheadOfUTC = 1;

export const local = (day: number, hours: number, minutes = 0): string =>
  new Date(Date.UTC(2026, 9, day, hours - londonSummerHoursAheadOfUTC, minutes)).toISOString();

export const tasksNow = new Date(local(16, 23));

export const steps = { one: "step", many: "steps" } as const;

const about = {
  source: "database",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
} as const;

const day = 86_400;

export const dailyCadence: SettingDurationField = {
  ...about,
  key: "tasks.housekeeping.cadence",
  label: "Every",
  kind: "duration",
  value: day,
  running: day,
  default: day,
  min: day,
  max: 7 * day,
  step: day,
};

const waitOverdue: SettingChoiceField = {
  ...about,
  key: "tasks.housekeeping.overdue",
  label: "When overdue",
  kind: "choice",
  value: "wait",
  running: "wait",
  default: "wait",
  options: [
    { value: "catch-up", label: "Catch up" },
    { value: "wait", label: "Wait" },
  ],
};

export const succeededRun: TasksRun = {
  id: 41,
  trigger: "schedule",
  startedAt: local(16, 4, 0),
  endedAt: local(16, 4, 1),
  result: "succeeded",
  summary: "6 steps",
  cause: null,
  nodes: ["home", "seedbox"],
  units: [
    {
      key: "runs",
      label: "Run history",
      node: "home",
      nodeId: homeId,
      nodeRemoved: false,
      outcome: "succeeded",
      cause: null,
    },
    {
      key: "node:seedbox",
      label: "seedbox",
      node: "seedbox",
      nodeId: seedboxId,
      nodeRemoved: false,
      outcome: "succeeded",
      cause: null,
    },
  ],
};

export const housekeeping: TasksTask = {
  id: "housekeeping",
  name: "Housekeeping",
  about: "Prunes old runs, Join Codes and sessions, then tidies every database.",
  group: "controller",
  noun: steps,
  cadence: { words: "Daily in the Maintenance window", inWindow: true, field: dailyCadence },
  overdue: waitOverdue,
  state: { kind: "idle", next: local(17, 4) },
  lastRun: succeededRun,
  nodes: ["home", "seedbox"],
  runNow: { available: true },
};

export const schedule: TasksSchedule = {
  window: {
    ...about,
    key: "tasks.window",
    label: "Maintenance window",
    kind: "window",
    value: { start: "04:00", end: "06:00" },
    running: { start: "04:00", end: "06:00" },
    default: { start: "04:00", end: "06:00" },
    zone: "Europe/London",
  },
  overdue: {
    ...waitOverdue,
    key: "tasks.overdue",
    label: "Overdue tasks",
    value: "catch-up",
    running: "catch-up",
    default: "catch-up",
  },
  runsKept: {
    ...about,
    key: "tasks.runsKept",
    label: "Runs kept per task",
    kind: "count",
    value: 50,
    running: 50,
    default: 50,
    min: 1,
    max: 1000,
    unit: { one: "run", many: "runs" },
  },
  jitterPercent: {
    ...about,
    key: "tasks.jitterPercent",
    label: "Jitter",
    kind: "count",
    value: 10,
    running: 10,
    default: 10,
    min: 0,
    max: 50,
    unit: { one: "%", many: "%" },
  },
  jitterCap: {
    ...about,
    key: "tasks.jitterCap",
    label: "Longest jitter",
    kind: "duration",
    value: 300,
    running: 300,
    default: 300,
    min: 0,
    max: 3600,
    step: 1,
  },
  headerDelay: {
    ...about,
    key: "tasks.headerDelay",
    label: "Show a scheduled run in the header after",
    kind: "duration",
    value: 5,
    running: 5,
    default: 5,
    min: 0,
    max: 300,
    step: 1,
  },
  partialWarnIntervals: {
    ...about,
    key: "tasks.partialWarnIntervals",
    label: "Warn when a unit has not succeeded for",
    kind: "count",
    value: 3,
    running: 3,
    default: 3,
    min: 1,
    max: 100,
    unit: { one: "interval", many: "intervals" },
  },
};

export const overviewOf = (
  tasks: readonly TasksTask[],
  {
    multinode = true,
    sentence = [{ kind: "plain", text: "Every task ran as scheduled." }],
  }: Partial<Pick<TasksOverview, "multinode" | "sentence">> = {},
): TasksOverview => ({
  multinode,
  sentence,
  groups: (["controller", "acquire", "store", "process", "connect", "reach"] as const).flatMap(
    (group) => {
      const shown = tasks.filter((task) => task.group === group);
      return shown.length === 0 ? [] : [{ group, tasks: shown }];
    },
  ),
  settings: schedule,
});

export const noOpenRuns: TasksOpenRuns = { headerDelaySeconds: 5, runs: [] };

const detailUnits: TasksDetail["units"] = [
  {
    key: "runs",
    label: "Run history",
    node: "home",
    nodeId: homeId,
    nodeRemoved: false,
    outcome: "succeeded",
    cause: null,
    endedAt: local(16, 4),
    lastSuccessAt: local(16, 4),
    dueOnReturn: false,
  },
  {
    key: "node:seedbox",
    label: "seedbox",
    node: "seedbox",
    nodeId: seedboxId,
    nodeRemoved: false,
    outcome: "skipped",
    cause: "seedbox is offline",
    endedAt: local(16, 4),
    lastSuccessAt: local(15, 4),
    dueOnReturn: true,
  },
];

type TaskState = {
  tasks: TasksTask[];
  open: TasksOpenRuns;
  runs: TasksRun[];
  settings: TasksSchedule;
};

const detailOf = (state: TaskState, id: string): TasksDetail | undefined => {
  const task = state.tasks.find((candidate) => candidate.id === id);
  return task && { task, units: detailUnits, runs: state.runs };
};

export function taskServer({
  tasks,
  multinode = true,
  sentence,
  open = noOpenRuns,
}: {
  tasks: TasksTask[];
  multinode?: boolean;
  sentence?: TasksOverview["sentence"];
  open?: TasksOpenRuns;
}) {
  const state: TaskState = { tasks, open, runs: [succeededRun], settings: schedule };
  const writes: { method: string; path: string; body: unknown }[] = [];
  const record = async (request: Request, path: string) => {
    const text = await request.text();
    writes.push({ method: request.method, path, body: text === "" ? undefined : JSON.parse(text) });
  };
  const handlers = [
    http.get("*/api/v1/tasks", () =>
      HttpResponse.json(overviewOf(state.tasks, { multinode, ...(sentence && { sentence }) })),
    ),
    http.get("*/api/v1/tasks/runs/open", () => HttpResponse.json(state.open)),
    http.get("*/api/v1/tasks/settings", () => HttpResponse.json(state.settings)),
    http.patch("*/api/v1/tasks/settings", async ({ request }) => {
      await record(request, "settings");
      return HttpResponse.json(state.settings);
    }),
    http.delete("*/api/v1/tasks/runs/:runId", async ({ request, params }) => {
      await record(request, `runs/${String(params["runId"])}`);
      return new HttpResponse(null, { status: 204 });
    }),
    http.get("*/api/v1/tasks/:taskId", ({ params }) =>
      HttpResponse.json(detailOf(state, String(params["taskId"]))),
    ),
    http.patch("*/api/v1/tasks/:taskId", async ({ request, params }) => {
      await record(request, String(params["taskId"]));
      return HttpResponse.json(state.tasks.find((task) => task.id === params["taskId"]));
    }),
    http.post("*/api/v1/tasks/:taskId/runs", async ({ request, params }) => {
      await record(request, `${String(params["taskId"])}/runs`);
      return HttpResponse.json({ runId: 42 }, { status: 202 });
    }),
  ];
  return { handlers, state, writes };
}
