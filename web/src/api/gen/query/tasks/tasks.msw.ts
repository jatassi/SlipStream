import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  SettingAddressField,
  SettingChoiceField,
  SettingCountField,
  SettingDurationField,
  SettingFolderField,
  SettingPortField,
  SettingRateField,
  SettingSpaceField,
  SettingSwitchField,
  SettingWindowField,
  TasksDetail,
  TasksFailurePart,
  TasksIdleState,
  TasksOpenRuns,
  TasksOverview,
  TasksPlainPart,
  TasksRunStarted,
  TasksRunningState,
  TasksSchedule,
  TasksTask,
  TasksWaitingState,
  TasksWarningPart,
} from "../../model";

export const getListTasksResponseTasksIdleStateMock = (
  overrideResponse: Partial<TasksIdleState> = {},
): TasksIdleState => ({
  ...{
    kind: faker.helpers.arrayElement(["idle"] as const),
    next: faker.date.past().toISOString().slice(0, 19) + "Z",
  },
  ...overrideResponse,
});

export const getListTasksResponseTasksRunningStateMock = (
  overrideResponse: Partial<TasksRunningState> = {},
): TasksRunningState => ({
  ...{
    done: faker.number.int({ min: 0 }),
    kind: faker.helpers.arrayElement(["running"] as const),
    runId: faker.number.int(),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    total: faker.number.int({ min: 0 }),
    trigger: faker.helpers.arrayElement(["schedule", "catch-up", "run-now", "returned"] as const),
  },
  ...overrideResponse,
});

export const getListTasksResponseTasksWaitingStateMock = (
  overrideResponse: Partial<TasksWaitingState> = {},
): TasksWaitingState => ({
  ...{
    kind: faker.helpers.arrayElement(["waiting"] as const),
    node: faker.string.alpha({ length: { min: 10, max: 20 } }),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    standing: faker.helpers.arrayElement([
      "reconnecting",
      "offline",
      "lapsed",
      "mismatched",
    ] as const),
  },
  ...overrideResponse,
});

export const getListTasksResponseTasksPlainPartMock = (
  overrideResponse: Partial<TasksPlainPart> = {},
): TasksPlainPart => ({
  ...{
    kind: faker.helpers.arrayElement(["plain"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getListTasksResponseTasksWarningPartMock = (
  overrideResponse: Partial<TasksWarningPart> = {},
): TasksWarningPart => ({
  ...{
    href: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["warning"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getListTasksResponseTasksFailurePartMock = (
  overrideResponse: Partial<TasksFailurePart> = {},
): TasksFailurePart => ({
  ...{
    href: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["failure"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getListTasksResponseMock = (
  overrideResponse: Partial<Extract<TasksOverview, object>> = {},
): TasksOverview => ({
  groups: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      group: faker.helpers.arrayElement([
        "controller",
        "acquire",
        "store",
        "process",
        "connect",
        "reach",
      ] as const),
      tasks: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          about: faker.string.alpha({ length: { min: 10, max: 20 } }),
          cadence: {
            field: {
              applies: faker.helpers.arrayElement(["live", "restart"] as const),
              default: faker.number.int(),
              ...(faker.datatype.boolean()
                ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean()
                ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              isDefault: faker.datatype.boolean(),
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              kind: faker.helpers.arrayElement(["duration"] as const),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              ...(faker.datatype.boolean()
                ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              max: faker.number.int(),
              min: faker.number.int(),
              pendingRestart: faker.datatype.boolean(),
              running: faker.number.int(),
              source: faker.helpers.arrayElement([
                "database",
                "default",
                "config.yaml",
                ".env",
                "env",
              ] as const),
              step: faker.number.int(),
              value: faker.number.int(),
            },
            inWindow: faker.datatype.boolean(),
            words: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          group: faker.helpers.arrayElement([
            "controller",
            "acquire",
            "store",
            "process",
            "connect",
            "reach",
          ] as const),
          id: faker.string.alpha({ length: { min: 10, max: 20 } }),
          ...(faker.datatype.boolean()
            ? {
                lastRun: {
                  cause: faker.helpers.arrayElement([
                    faker.string.alpha({ length: { min: 10, max: 20 } }),
                    null,
                  ]),
                  endedAt: faker.helpers.arrayElement([
                    faker.date.past().toISOString().slice(0, 19) + "Z",
                    null,
                  ]),
                  id: faker.number.int(),
                  nodes: Array.from(
                    { length: faker.number.int({ min: 1, max: 10 }) },
                    (_, i) => i + 1,
                  ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
                  result: faker.helpers.arrayElement([
                    faker.helpers.arrayElement([
                      "succeeded",
                      "partial",
                      "failed",
                      "skipped",
                      "canceled",
                    ] as const),
                    null,
                  ]),
                  startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
                  summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  trigger: faker.helpers.arrayElement([
                    "schedule",
                    "catch-up",
                    "run-now",
                    "returned",
                  ] as const),
                  units: Array.from(
                    { length: faker.number.int({ min: 1, max: 10 }) },
                    (_, i) => i + 1,
                  ).map(() => ({
                    cause: faker.helpers.arrayElement([
                      faker.string.alpha({ length: { min: 10, max: 20 } }),
                      null,
                    ]),
                    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
                    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
                    node: faker.string.alpha({ length: { min: 10, max: 20 } }),
                    nodeId: faker.string.uuid(),
                    nodeRemoved: faker.datatype.boolean(),
                    outcome: faker.helpers.arrayElement([
                      "succeeded",
                      "failed",
                      "skipped",
                      "canceled",
                    ] as const),
                  })),
                },
              }
            : {}),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
          noun: {
            many: faker.string.alpha({ length: { min: 10, max: 20 } }),
            one: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          overdue: {
            applies: faker.helpers.arrayElement(["live", "restart"] as const),
            default: faker.string.alpha({ length: { min: 10, max: 20 } }),
            ...(faker.datatype.boolean()
              ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            isDefault: faker.datatype.boolean(),
            key: faker.string.alpha({ length: { min: 10, max: 20 } }),
            kind: faker.helpers.arrayElement(["choice"] as const),
            label: faker.string.alpha({ length: { min: 10, max: 20 } }),
            ...(faker.datatype.boolean()
              ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            options: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => ({
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              value: faker.string.alpha({ length: { min: 10, max: 20 } }),
            })),
            pendingRestart: faker.datatype.boolean(),
            running: faker.string.alpha({ length: { min: 10, max: 20 } }),
            source: faker.helpers.arrayElement([
              "database",
              "default",
              "config.yaml",
              ".env",
              "env",
            ] as const),
            value: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          runNow: {
            available: faker.datatype.boolean(),
            ...(faker.datatype.boolean()
              ? { reason: faker.helpers.arrayElement(["running", "waiting"] as const) }
              : {}),
          },
          state: faker.helpers.arrayElement([
            { ...getListTasksResponseTasksIdleStateMock() },
            { ...getListTasksResponseTasksRunningStateMock() },
            { ...getListTasksResponseTasksWaitingStateMock() },
          ]),
        }),
      ),
    }),
  ),
  multinode: faker.datatype.boolean(),
  sentence: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getListTasksResponseTasksPlainPartMock() },
      { ...getListTasksResponseTasksWarningPartMock() },
      { ...getListTasksResponseTasksFailurePartMock() },
    ]),
  ),
  settings: {
    headerDelay: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["duration"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      max: faker.number.int(),
      min: faker.number.int(),
      pendingRestart: faker.datatype.boolean(),
      running: faker.number.int(),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      step: faker.number.int(),
      value: faker.number.int(),
    },
    jitterCap: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["duration"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      max: faker.number.int(),
      min: faker.number.int(),
      pendingRestart: faker.datatype.boolean(),
      running: faker.number.int(),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      step: faker.number.int(),
      value: faker.number.int(),
    },
    jitterPercent: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["count"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      max: faker.number.int(),
      min: faker.number.int(),
      pendingRestart: faker.datatype.boolean(),
      running: faker.number.int(),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      unit: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      value: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    },
    overdue: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["choice"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          value: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      pendingRestart: faker.datatype.boolean(),
      running: faker.string.alpha({ length: { min: 10, max: 20 } }),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      value: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    partialWarnIntervals: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["count"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      max: faker.number.int(),
      min: faker.number.int(),
      pendingRestart: faker.datatype.boolean(),
      running: faker.number.int(),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      unit: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      value: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    },
    runsKept: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["count"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      max: faker.number.int(),
      min: faker.number.int(),
      pendingRestart: faker.datatype.boolean(),
      running: faker.number.int(),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      unit: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      value: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    },
    window: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: {
        end: faker.string.alpha({ length: { min: 10, max: 20 } }),
        start: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["window"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      pendingRestart: faker.datatype.boolean(),
      running: {
        end: faker.string.alpha({ length: { min: 10, max: 20 } }),
        start: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      value: {
        end: faker.string.alpha({ length: { min: 10, max: 20 } }),
        start: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
  },
  ...overrideResponse,
});

export const getListOpenTaskRunsResponseMock = (
  overrideResponse: Partial<Extract<TasksOpenRuns, object>> = {},
): TasksOpenRuns => ({
  headerDelaySeconds: faker.number.int(),
  runs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    done: faker.number.int({ min: 0 }),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    noun: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    runId: faker.number.int(),
    startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    taskId: faker.string.alpha({ length: { min: 10, max: 20 } }),
    total: faker.number.int({ min: 0 }),
    trigger: faker.helpers.arrayElement(["schedule", "catch-up", "run-now", "returned"] as const),
  })),
  ...overrideResponse,
});

export const getGetTaskSettingsResponseMock = (
  overrideResponse: Partial<Extract<TasksSchedule, object>> = {},
): TasksSchedule => ({
  headerDelay: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  jitterCap: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  jitterPercent: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  overdue: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  partialWarnIntervals: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  runsKept: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  window: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateTaskSettingsResponseMock = (
  overrideResponse: Partial<Extract<TasksSchedule, object>> = {},
): TasksSchedule => ({
  headerDelay: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  jitterCap: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  jitterPercent: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  overdue: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  partialWarnIntervals: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  runsKept: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  window: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetTaskResponseTasksIdleStateMock = (
  overrideResponse: Partial<TasksIdleState> = {},
): TasksIdleState => ({
  ...{
    kind: faker.helpers.arrayElement(["idle"] as const),
    next: faker.date.past().toISOString().slice(0, 19) + "Z",
  },
  ...overrideResponse,
});

export const getGetTaskResponseTasksRunningStateMock = (
  overrideResponse: Partial<TasksRunningState> = {},
): TasksRunningState => ({
  ...{
    done: faker.number.int({ min: 0 }),
    kind: faker.helpers.arrayElement(["running"] as const),
    runId: faker.number.int(),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    total: faker.number.int({ min: 0 }),
    trigger: faker.helpers.arrayElement(["schedule", "catch-up", "run-now", "returned"] as const),
  },
  ...overrideResponse,
});

export const getGetTaskResponseTasksWaitingStateMock = (
  overrideResponse: Partial<TasksWaitingState> = {},
): TasksWaitingState => ({
  ...{
    kind: faker.helpers.arrayElement(["waiting"] as const),
    node: faker.string.alpha({ length: { min: 10, max: 20 } }),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    standing: faker.helpers.arrayElement([
      "reconnecting",
      "offline",
      "lapsed",
      "mismatched",
    ] as const),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingCountFieldMock = (
  overrideResponse: Partial<SettingCountField> = {},
): SettingCountField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingChoiceFieldMock = (
  overrideResponse: Partial<SettingChoiceField> = {},
): SettingChoiceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingWindowFieldMock = (
  overrideResponse: Partial<SettingWindowField> = {},
): SettingWindowField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingPortFieldMock = (
  overrideResponse: Partial<SettingPortField> = {},
): SettingPortField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["port"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingSwitchFieldMock = (
  overrideResponse: Partial<SettingSwitchField> = {},
): SettingSwitchField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["switch"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingRateFieldMock = (
  overrideResponse: Partial<SettingRateField> = {},
): SettingRateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["rate"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingAddressFieldMock = (
  overrideResponse: Partial<SettingAddressField> = {},
): SettingAddressField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["address"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingFolderFieldMock = (
  overrideResponse: Partial<SettingFolderField> = {},
): SettingFolderField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["folder"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    pendingRestart: faker.datatype.boolean(),
    purpose: faker.helpers.arrayElement(["downloads"] as const),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetTaskResponseSettingSpaceFieldMock = (
  overrideResponse: Partial<SettingSpaceField> = {},
): SettingSpaceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["space"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    maxBytes: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
  },
  ...overrideResponse,
});

export const getGetTaskResponseMock = (
  overrideResponse: Partial<Extract<TasksDetail, object>> = {},
): TasksDetail => ({
  runs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    cause: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    endedAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    id: faker.number.int(),
    nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    result: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "succeeded",
        "partial",
        "failed",
        "skipped",
        "canceled",
      ] as const),
      null,
    ]),
    startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
    trigger: faker.helpers.arrayElement(["schedule", "catch-up", "run-now", "returned"] as const),
    units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        cause: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        node: faker.string.alpha({ length: { min: 10, max: 20 } }),
        nodeId: faker.string.uuid(),
        nodeRemoved: faker.datatype.boolean(),
        outcome: faker.helpers.arrayElement([
          "succeeded",
          "failed",
          "skipped",
          "canceled",
        ] as const),
      }),
    ),
  })),
  task: {
    about: faker.string.alpha({ length: { min: 10, max: 20 } }),
    cadence: {
      field: {
        applies: faker.helpers.arrayElement(["live", "restart"] as const),
        default: faker.number.int(),
        ...(faker.datatype.boolean()
          ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        isDefault: faker.datatype.boolean(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        kind: faker.helpers.arrayElement(["duration"] as const),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        max: faker.number.int(),
        min: faker.number.int(),
        pendingRestart: faker.datatype.boolean(),
        running: faker.number.int(),
        source: faker.helpers.arrayElement([
          "database",
          "default",
          "config.yaml",
          ".env",
          "env",
        ] as const),
        step: faker.number.int(),
        value: faker.number.int(),
      },
      inWindow: faker.datatype.boolean(),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    group: faker.helpers.arrayElement([
      "controller",
      "acquire",
      "store",
      "process",
      "connect",
      "reach",
    ] as const),
    id: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? {
          lastRun: {
            cause: faker.helpers.arrayElement([
              faker.string.alpha({ length: { min: 10, max: 20 } }),
              null,
            ]),
            endedAt: faker.helpers.arrayElement([
              faker.date.past().toISOString().slice(0, 19) + "Z",
              null,
            ]),
            id: faker.number.int(),
            nodes: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            result: faker.helpers.arrayElement([
              faker.helpers.arrayElement([
                "succeeded",
                "partial",
                "failed",
                "skipped",
                "canceled",
              ] as const),
              null,
            ]),
            startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
            trigger: faker.helpers.arrayElement([
              "schedule",
              "catch-up",
              "run-now",
              "returned",
            ] as const),
            units: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => ({
              cause: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              node: faker.string.alpha({ length: { min: 10, max: 20 } }),
              nodeId: faker.string.uuid(),
              nodeRemoved: faker.datatype.boolean(),
              outcome: faker.helpers.arrayElement([
                "succeeded",
                "failed",
                "skipped",
                "canceled",
              ] as const),
            })),
          },
        }
      : {}),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    noun: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    overdue: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["choice"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          value: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      pendingRestart: faker.datatype.boolean(),
      running: faker.string.alpha({ length: { min: 10, max: 20 } }),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      value: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    runNow: {
      available: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? { reason: faker.helpers.arrayElement(["running", "waiting"] as const) }
        : {}),
    },
    state: faker.helpers.arrayElement([
      { ...getGetTaskResponseTasksIdleStateMock() },
      { ...getGetTaskResponseTasksRunningStateMock() },
      { ...getGetTaskResponseTasksWaitingStateMock() },
    ]),
  },
  tuning: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetTaskResponseSettingDurationFieldMock() },
      { ...getGetTaskResponseSettingCountFieldMock() },
      { ...getGetTaskResponseSettingChoiceFieldMock() },
      { ...getGetTaskResponseSettingWindowFieldMock() },
      { ...getGetTaskResponseSettingPortFieldMock() },
      { ...getGetTaskResponseSettingSwitchFieldMock() },
      { ...getGetTaskResponseSettingRateFieldMock() },
      { ...getGetTaskResponseSettingAddressFieldMock() },
      { ...getGetTaskResponseSettingFolderFieldMock() },
      { ...getGetTaskResponseSettingSpaceFieldMock() },
    ]),
  ),
  units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    cause: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    dueOnReturn: faker.datatype.boolean(),
    endedAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    lastSuccessAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    node: faker.string.alpha({ length: { min: 10, max: 20 } }),
    nodeId: faker.string.uuid(),
    nodeRemoved: faker.datatype.boolean(),
    outcome: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["succeeded", "failed", "skipped", "canceled"] as const),
      null,
    ]),
  })),
  ...overrideResponse,
});

export const getUpdateTaskResponseTasksIdleStateMock = (
  overrideResponse: Partial<TasksIdleState> = {},
): TasksIdleState => ({
  ...{
    kind: faker.helpers.arrayElement(["idle"] as const),
    next: faker.date.past().toISOString().slice(0, 19) + "Z",
  },
  ...overrideResponse,
});

export const getUpdateTaskResponseTasksRunningStateMock = (
  overrideResponse: Partial<TasksRunningState> = {},
): TasksRunningState => ({
  ...{
    done: faker.number.int({ min: 0 }),
    kind: faker.helpers.arrayElement(["running"] as const),
    runId: faker.number.int(),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    total: faker.number.int({ min: 0 }),
    trigger: faker.helpers.arrayElement(["schedule", "catch-up", "run-now", "returned"] as const),
  },
  ...overrideResponse,
});

export const getUpdateTaskResponseTasksWaitingStateMock = (
  overrideResponse: Partial<TasksWaitingState> = {},
): TasksWaitingState => ({
  ...{
    kind: faker.helpers.arrayElement(["waiting"] as const),
    node: faker.string.alpha({ length: { min: 10, max: 20 } }),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    standing: faker.helpers.arrayElement([
      "reconnecting",
      "offline",
      "lapsed",
      "mismatched",
    ] as const),
  },
  ...overrideResponse,
});

export const getUpdateTaskResponseMock = (
  overrideResponse: Partial<Extract<TasksTask, object>> = {},
): TasksTask => ({
  about: faker.string.alpha({ length: { min: 10, max: 20 } }),
  cadence: {
    field: {
      applies: faker.helpers.arrayElement(["live", "restart"] as const),
      default: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      isDefault: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["duration"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      max: faker.number.int(),
      min: faker.number.int(),
      pendingRestart: faker.datatype.boolean(),
      running: faker.number.int(),
      source: faker.helpers.arrayElement([
        "database",
        "default",
        "config.yaml",
        ".env",
        "env",
      ] as const),
      step: faker.number.int(),
      value: faker.number.int(),
    },
    inWindow: faker.datatype.boolean(),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  group: faker.helpers.arrayElement([
    "controller",
    "acquire",
    "store",
    "process",
    "connect",
    "reach",
  ] as const),
  id: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? {
        lastRun: {
          cause: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          endedAt: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          id: faker.number.int(),
          nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
          result: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
              "succeeded",
              "partial",
              "failed",
              "skipped",
              "canceled",
            ] as const),
            null,
          ]),
          startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
          trigger: faker.helpers.arrayElement([
            "schedule",
            "catch-up",
            "run-now",
            "returned",
          ] as const),
          units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => ({
              cause: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              node: faker.string.alpha({ length: { min: 10, max: 20 } }),
              nodeId: faker.string.uuid(),
              nodeRemoved: faker.datatype.boolean(),
              outcome: faker.helpers.arrayElement([
                "succeeded",
                "failed",
                "skipped",
                "canceled",
              ] as const),
            }),
          ),
        },
      }
    : {}),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  noun: {
    many: faker.string.alpha({ length: { min: 10, max: 20 } }),
    one: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  overdue: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  runNow: {
    available: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { reason: faker.helpers.arrayElement(["running", "waiting"] as const) }
      : {}),
  },
  state: faker.helpers.arrayElement([
    { ...getUpdateTaskResponseTasksIdleStateMock() },
    { ...getUpdateTaskResponseTasksRunningStateMock() },
    { ...getUpdateTaskResponseTasksWaitingStateMock() },
  ]),
  ...overrideResponse,
});

export const getRunTaskResponseMock = (
  overrideResponse: Partial<Extract<TasksRunStarted, object>> = {},
): TasksRunStarted => ({ runId: faker.number.int(), ...overrideResponse });

export const getListTasksMockHandler = (
  overrideResponse?:
    | TasksOverview
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TasksOverview> | TasksOverview),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/tasks",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListTasksResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListOpenTaskRunsMockHandler = (
  overrideResponse?:
    | TasksOpenRuns
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TasksOpenRuns> | TasksOpenRuns),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/tasks/runs/open",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListOpenTaskRunsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelTaskRunMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/tasks/runs/:runId",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetTaskSettingsMockHandler = (
  overrideResponse?:
    | TasksSchedule
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TasksSchedule> | TasksSchedule),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/tasks/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTaskSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateTaskSettingsMockHandler = (
  overrideResponse?:
    | TasksSchedule
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<TasksSchedule> | TasksSchedule),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/tasks/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateTaskSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetTaskMockHandler = (
  overrideResponse?:
    | TasksDetail
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<TasksDetail> | TasksDetail),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/tasks/:taskId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTaskResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateTaskMockHandler = (
  overrideResponse?:
    | TasksTask
    | ((info: Parameters<Parameters<typeof http.patch>[1]>[0]) => Promise<TasksTask> | TasksTask),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/tasks/:taskId",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateTaskResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRunTaskMockHandler = (
  overrideResponse?:
    | TasksRunStarted
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TasksRunStarted> | TasksRunStarted),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/tasks/:taskId/runs",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRunTaskResponseMock(),
        { status: 202 },
      );
    },
    options,
  );
};
export const getTasksMock = () => [
  getListTasksMockHandler(),
  getListOpenTaskRunsMockHandler(),
  getCancelTaskRunMockHandler(),
  getGetTaskSettingsMockHandler(),
  getUpdateTaskSettingsMockHandler(),
  getGetTaskMockHandler(),
  getUpdateTaskMockHandler(),
  getRunTaskMockHandler(),
];
