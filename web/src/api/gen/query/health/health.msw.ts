import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  HealthChecked,
  HealthReport,
  HealthSettings,
  HealthVolume,
  HealthVolumeList,
} from "../../model";

export const getGetSystemHealthResponseMock = (
  overrideResponse: Partial<Extract<HealthReport, object>> = {},
): HealthReport => ({
  categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      canCheck: faker.datatype.boolean(),
      id: faker.helpers.arrayElement([
        "storage",
        "nodes",
        "download-clients",
        "indexers",
        "scheduled-tasks",
      ] as const),
      items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          action: faker.helpers.arrayElement([
            {
              ...(faker.datatype.boolean()
                ? { href: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              kind: faker.helpers.arrayElement(["check", "run", "link"] as const),
              ...(faker.datatype.boolean()
                ? { label: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean()
                ? { taskId: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
            },
            null,
          ]),
          changedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          detail: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            ...(faker.datatype.boolean()
              ? { at: faker.date.past().toISOString().slice(0, 19) + "Z" }
              : {}),
            ...(faker.datatype.boolean() ? { bytes: faker.number.int() } : {}),
            label: faker.string.alpha({ length: { min: 10, max: 20 } }),
            ...(faker.datatype.boolean()
              ? { text: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
          })),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          level: faker.helpers.arrayElement(["warn", "fail"] as const),
          since: faker.date.past().toISOString().slice(0, 19) + "Z",
          subject: faker.string.alpha({ length: { min: 10, max: 20 } }),
          text: faker.string.alpha({ length: { min: 10, max: 20 } }),
          unsure: faker.datatype.boolean(),
        }),
      ),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      level: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["warn", "fail"] as const),
        null,
      ]),
      phrase: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { unread: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    }),
  ),
  first: faker.helpers.arrayElement([
    {
      category: faker.helpers.arrayElement([
        "storage",
        "nodes",
        "download-clients",
        "indexers",
        "scheduled-tasks",
      ] as const),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      text: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  level: faker.helpers.arrayElement([faker.helpers.arrayElement(["warn", "fail"] as const), null]),
  ...overrideResponse,
});

export const getCheckSystemHealthResponseMock = (
  overrideResponse: Partial<Extract<HealthChecked, object>> = {},
): HealthChecked => ({
  report: {
    categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        canCheck: faker.datatype.boolean(),
        id: faker.helpers.arrayElement([
          "storage",
          "nodes",
          "download-clients",
          "indexers",
          "scheduled-tasks",
        ] as const),
        items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            action: faker.helpers.arrayElement([
              {
                ...(faker.datatype.boolean()
                  ? { href: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                kind: faker.helpers.arrayElement(["check", "run", "link"] as const),
                ...(faker.datatype.boolean()
                  ? { label: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                ...(faker.datatype.boolean()
                  ? { taskId: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
              },
              null,
            ]),
            changedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            detail: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => ({
              ...(faker.datatype.boolean()
                ? { at: faker.date.past().toISOString().slice(0, 19) + "Z" }
                : {}),
              ...(faker.datatype.boolean() ? { bytes: faker.number.int() } : {}),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              ...(faker.datatype.boolean()
                ? { text: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
            })),
            key: faker.string.alpha({ length: { min: 10, max: 20 } }),
            level: faker.helpers.arrayElement(["warn", "fail"] as const),
            since: faker.date.past().toISOString().slice(0, 19) + "Z",
            subject: faker.string.alpha({ length: { min: 10, max: 20 } }),
            text: faker.string.alpha({ length: { min: 10, max: 20 } }),
            unsure: faker.datatype.boolean(),
          }),
        ),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        level: faker.helpers.arrayElement([
          faker.helpers.arrayElement(["warn", "fail"] as const),
          null,
        ]),
        phrase: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? { unread: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
      }),
    ),
    first: faker.helpers.arrayElement([
      {
        category: faker.helpers.arrayElement([
          "storage",
          "nodes",
          "download-clients",
          "indexers",
          "scheduled-tasks",
        ] as const),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    level: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["warn", "fail"] as const),
      null,
    ]),
  },
  unreached: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean()
        ? { key: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getGetHealthSettingsResponseMock = (
  overrideResponse: Partial<Extract<HealthSettings, object>> = {},
): HealthSettings => ({
  errorBelow: {
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
  warnBelow: {
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

export const getUpdateHealthSettingsResponseMock = (
  overrideResponse: Partial<Extract<HealthSettings, object>> = {},
): HealthSettings => ({
  errorBelow: {
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
  warnBelow: {
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

export const getListHealthVolumesResponseMock = (
  overrideResponse: Partial<Extract<HealthVolumeList, object>> = {},
): HealthVolumeList => ({
  notReporting: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      errorBelow: faker.helpers.arrayElement([
        {
          amount: faker.number.int({ min: 0 }),
          unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
        },
        null,
      ]),
      mount: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      warnBelow: faker.helpers.arrayElement([
        {
          amount: faker.number.int({ min: 0 }),
          unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
        },
        null,
      ]),
    }),
  ),
  volumes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      errorBelow: {
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
      errorBytes: faker.number.int(),
      free: faker.number.int(),
      level: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["warn", "fail"] as const),
        null,
      ]),
      live: faker.datatype.boolean(),
      mount: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      sampledAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      skipsWarning: faker.datatype.boolean(),
      total: faker.number.int(),
      warnBelow: {
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
      warnBytes: faker.number.int(),
    }),
  ),
  ...overrideResponse,
});

export const getUpdateHealthVolumeResponseMock = (
  overrideResponse: Partial<Extract<HealthVolume, object>> = {},
): HealthVolume => ({
  errorBelow: {
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
  errorBytes: faker.number.int(),
  free: faker.number.int(),
  level: faker.helpers.arrayElement([faker.helpers.arrayElement(["warn", "fail"] as const), null]),
  live: faker.datatype.boolean(),
  mount: faker.string.alpha({ length: { min: 10, max: 20 } }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  nodeId: faker.string.uuid(),
  nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
  sampledAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  skipsWarning: faker.datatype.boolean(),
  total: faker.number.int(),
  warnBelow: {
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
  warnBytes: faker.number.int(),
  ...overrideResponse,
});

export const getCheckHealthCategoryResponseMock = (
  overrideResponse: Partial<Extract<HealthChecked, object>> = {},
): HealthChecked => ({
  report: {
    categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        canCheck: faker.datatype.boolean(),
        id: faker.helpers.arrayElement([
          "storage",
          "nodes",
          "download-clients",
          "indexers",
          "scheduled-tasks",
        ] as const),
        items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            action: faker.helpers.arrayElement([
              {
                ...(faker.datatype.boolean()
                  ? { href: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                kind: faker.helpers.arrayElement(["check", "run", "link"] as const),
                ...(faker.datatype.boolean()
                  ? { label: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                ...(faker.datatype.boolean()
                  ? { taskId: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
              },
              null,
            ]),
            changedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            detail: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => ({
              ...(faker.datatype.boolean()
                ? { at: faker.date.past().toISOString().slice(0, 19) + "Z" }
                : {}),
              ...(faker.datatype.boolean() ? { bytes: faker.number.int() } : {}),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              ...(faker.datatype.boolean()
                ? { text: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
            })),
            key: faker.string.alpha({ length: { min: 10, max: 20 } }),
            level: faker.helpers.arrayElement(["warn", "fail"] as const),
            since: faker.date.past().toISOString().slice(0, 19) + "Z",
            subject: faker.string.alpha({ length: { min: 10, max: 20 } }),
            text: faker.string.alpha({ length: { min: 10, max: 20 } }),
            unsure: faker.datatype.boolean(),
          }),
        ),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        level: faker.helpers.arrayElement([
          faker.helpers.arrayElement(["warn", "fail"] as const),
          null,
        ]),
        phrase: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? { unread: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
      }),
    ),
    first: faker.helpers.arrayElement([
      {
        category: faker.helpers.arrayElement([
          "storage",
          "nodes",
          "download-clients",
          "indexers",
          "scheduled-tasks",
        ] as const),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    level: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["warn", "fail"] as const),
      null,
    ]),
  },
  unreached: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean()
        ? { key: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getCheckHealthItemResponseMock = (
  overrideResponse: Partial<Extract<HealthChecked, object>> = {},
): HealthChecked => ({
  report: {
    categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        canCheck: faker.datatype.boolean(),
        id: faker.helpers.arrayElement([
          "storage",
          "nodes",
          "download-clients",
          "indexers",
          "scheduled-tasks",
        ] as const),
        items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            action: faker.helpers.arrayElement([
              {
                ...(faker.datatype.boolean()
                  ? { href: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                kind: faker.helpers.arrayElement(["check", "run", "link"] as const),
                ...(faker.datatype.boolean()
                  ? { label: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                ...(faker.datatype.boolean()
                  ? { taskId: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
              },
              null,
            ]),
            changedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            detail: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => ({
              ...(faker.datatype.boolean()
                ? { at: faker.date.past().toISOString().slice(0, 19) + "Z" }
                : {}),
              ...(faker.datatype.boolean() ? { bytes: faker.number.int() } : {}),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              ...(faker.datatype.boolean()
                ? { text: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
            })),
            key: faker.string.alpha({ length: { min: 10, max: 20 } }),
            level: faker.helpers.arrayElement(["warn", "fail"] as const),
            since: faker.date.past().toISOString().slice(0, 19) + "Z",
            subject: faker.string.alpha({ length: { min: 10, max: 20 } }),
            text: faker.string.alpha({ length: { min: 10, max: 20 } }),
            unsure: faker.datatype.boolean(),
          }),
        ),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        level: faker.helpers.arrayElement([
          faker.helpers.arrayElement(["warn", "fail"] as const),
          null,
        ]),
        phrase: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? { unread: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
      }),
    ),
    first: faker.helpers.arrayElement([
      {
        category: faker.helpers.arrayElement([
          "storage",
          "nodes",
          "download-clients",
          "indexers",
          "scheduled-tasks",
        ] as const),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    level: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["warn", "fail"] as const),
      null,
    ]),
  },
  unreached: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean()
        ? { key: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getGetSystemHealthMockHandler = (
  overrideResponse?:
    | HealthReport
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HealthReport> | HealthReport),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/system/health",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSystemHealthResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCheckSystemHealthMockHandler = (
  overrideResponse?:
    | HealthChecked
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<HealthChecked> | HealthChecked),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/system/health/check",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCheckSystemHealthResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetHealthSettingsMockHandler = (
  overrideResponse?:
    | HealthSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HealthSettings> | HealthSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/system/health/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetHealthSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateHealthSettingsMockHandler = (
  overrideResponse?:
    | HealthSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<HealthSettings> | HealthSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/system/health/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateHealthSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteHealthVolumeMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/system/health/volumes",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListHealthVolumesMockHandler = (
  overrideResponse?:
    | HealthVolumeList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HealthVolumeList> | HealthVolumeList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/system/health/volumes",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListHealthVolumesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateHealthVolumeMockHandler = (
  overrideResponse?:
    | HealthVolume
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<HealthVolume> | HealthVolume),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/system/health/volumes",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateHealthVolumeResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCheckHealthCategoryMockHandler = (
  overrideResponse?:
    | HealthChecked
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<HealthChecked> | HealthChecked),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/system/health/:category/check",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCheckHealthCategoryResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCheckHealthItemMockHandler = (
  overrideResponse?:
    | HealthChecked
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<HealthChecked> | HealthChecked),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/system/health/:category/items/:key/check",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCheckHealthItemResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getHealthMock = () => [
  getGetSystemHealthMockHandler(),
  getCheckSystemHealthMockHandler(),
  getGetHealthSettingsMockHandler(),
  getUpdateHealthSettingsMockHandler(),
  getDeleteHealthVolumeMockHandler(),
  getListHealthVolumesMockHandler(),
  getUpdateHealthVolumeMockHandler(),
  getCheckHealthCategoryMockHandler(),
  getCheckHealthItemMockHandler(),
];
